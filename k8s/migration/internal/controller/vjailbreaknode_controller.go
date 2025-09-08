/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/pkg/errors"
	migratev1alpha1 "github.com/kashyapshashankv/stellaris-migrate/k8s/migration/api/v1alpha1"
	"github.com/kashyapshashankv/stellaris-migrate/k8s/migration/pkg/constants"
	"github.com/kashyapshashankv/stellaris-migrate/k8s/migration/pkg/scope"
	"github.com/kashyapshashankv/stellaris-migrate/k8s/migration/pkg/utils"
)

// StellarisMigrateNodeReconciler reconciles a StellarisMigrateNode object
type StellarisMigrateNodeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Local  bool
}

// +kubebuilder:rbac:groups=migrate.k8s.stellaris.io,resources=stellaris-migrate-nodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=migrate.k8s.stellaris.io,resources=stellaris-migrate-nodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=migrate.k8s.stellaris.io,resources=stellaris-migrate-nodes/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;delete

// Reconcile handles the reconciliation of StellarisMigrateNode resources
func (r *StellarisMigrateNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := log.FromContext(ctx).WithName(constants.StellarisMigrateNodeControllerName)

	// Fetch the StellarisMigrateNode instance.
	vjailbreakNode := migratev1alpha1.StellarisMigrateNode{}
	client := r.Client
	err := client.Get(ctx, req.NamespacedName, &vjailbreakNode)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	vjailbreakNodeScope, err := scope.NewStellarisMigrateNodeScope(scope.StellarisMigrateNodeScopeParams{
		Logger:         log,
		Client:         r.Client,
		StellarisMigrateNode: &vjailbreakNode,
	})
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to create stellaris-migrate node scope")
	}

	// Always close the scope when exiting this function such that we can persist any StellarisMigrateNode changes.
	defer func() {
		if err := vjailbreakNodeScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// Quick path for just updating ActiveMigrations if node is ready
	if vjailbreakNode.Status.Phase == constants.StellarisMigrateNodePhaseNodeReady {
		result, err := r.updateActiveMigrations(ctx, vjailbreakNodeScope)
		if err != nil {
			return result, errors.Wrap(err, "failed to update active migrations")
		}
	}

	// Handle deleted StellarisMigrateNode
	if !vjailbreakNode.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, vjailbreakNodeScope)
	}

	// Handle regular StellarisMigrateNode reconcile
	return r.reconcileNormal(ctx, vjailbreakNodeScope)
}

// reconcileNormal handles regular StellarisMigrateNode reconcile
//
//nolint:unparam //future use
func (r *StellarisMigrateNodeReconciler) reconcileNormal(ctx context.Context,
	scope *scope.StellarisMigrateNodeScope) (ctrl.Result, error) {
	log := scope.Logger
	log.Info("Reconciling StellarisMigrateNode")
	var vmip string
	var node *corev1.Node

	vjNode := scope.StellarisMigrateNode
	controllerutil.AddFinalizer(vjNode, constants.StellarisMigrateNodeFinalizer)

	if vjNode.Spec.NodeRole == constants.NodeRoleMaster {
		err := utils.UpdateMasterNodeImageID(ctx, r.Client, r.Local)
		if err != nil {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, errors.Wrap(err, "failed to update master node image id")
		}
		log.Info("Skipping master node, updating flavor", "name", vjNode.Name)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	vjNode.Status.Phase = constants.StellarisMigrateNodePhaseVMCreating

	uuid, err := utils.GetOpenstackVMByName(ctx, vjNode.Name, r.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to get openstack vm by name")
	}

	if uuid != "" {
		if vjNode.Status.OpenstackUUID == "" {
			// This will error until the the IP is available
			vmip, err = utils.GetOpenstackVMIP(ctx, uuid, r.Client)
			if err != nil {
				return ctrl.Result{}, errors.Wrap(err, "failed to get vm ip from openstack uuid")
			}

			vjNode.Status.OpenstackUUID = uuid
			vjNode.Status.VMIP = vmip

			// Update the StellarisMigrateNode status
			err = r.Client.Status().Update(ctx, vjNode)
			if err != nil {
				return ctrl.Result{}, errors.Wrap(err, "failed to update stellaris-migrate node status")
			}
		}
		node, err = utils.GetNodeByName(ctx, r.Client, vjNode.Name)
		if err != nil {
			if apierrors.IsNotFound(err) {
				log.Info("Node not found, waiting for node to be created", "name", vjNode.Name)
				return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
			}
			return ctrl.Result{}, errors.Wrap(err, "failed to get node by name")
		}
		vjNode.Status.Phase = constants.StellarisMigrateNodePhaseVMCreated
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" {
				vjNode.Status.Phase = constants.StellarisMigrateNodePhaseNodeReady
				break
			}
		}
		// Update the StellarisMigrateNode status
		err = r.Client.Status().Update(ctx, vjNode)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to update stellaris-migrate node status")
		}

		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	// Create Openstack VM for worker node
	vmid, err := utils.CreateOpenstackVMForWorkerNode(ctx, r.Client, scope)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to create openstack vm for worker node")
	}

	vjNode.Status.OpenstackUUID = uuid
	vjNode.Status.Phase = constants.StellarisMigrateNodePhaseVMCreated
	vjNode.Status.VMIP = vmip

	// Update the StellarisMigrateNode status
	err = r.Client.Status().Update(ctx, vjNode)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to update stellaris-migrate node status")
	}

	log.Info("Successfully created openstack vm for worker node", "vmid", vmid)
	return ctrl.Result{}, nil
}

// reconcileDelete handles deleted StellarisMigrateNode
//
//nolint:unparam //future use
func (r *StellarisMigrateNodeReconciler) reconcileDelete(ctx context.Context,
	scope *scope.StellarisMigrateNodeScope) (ctrl.Result, error) {
	log := scope.Logger
	log.Info("Reconciling StellarisMigrateNode Delete")

	if scope.StellarisMigrateNode.Spec.NodeRole == constants.NodeRoleMaster {
		controllerutil.RemoveFinalizer(scope.StellarisMigrateNode, constants.StellarisMigrateNodeFinalizer)
		return ctrl.Result{}, nil
	}

	scope.StellarisMigrateNode.Status.Phase = constants.StellarisMigrateNodePhaseDeleting
	// Update the StellarisMigrateNode status
	err := r.Client.Status().Update(ctx, scope.StellarisMigrateNode)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to update stellaris-migrate node status")
	}

	uuid, err := utils.GetOpenstackVMByName(ctx, scope.StellarisMigrateNode.Name, r.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to get openstack vm by name")
	}

	if uuid == "" {
		log.Info("node already deleted", "name", scope.StellarisMigrateNode.Name)
		controllerutil.RemoveFinalizer(scope.StellarisMigrateNode, constants.StellarisMigrateNodeFinalizer)
		return ctrl.Result{}, nil
	}

	err = utils.DeleteOpenstackVM(ctx, scope.StellarisMigrateNode.Status.OpenstackUUID, r.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to delete openstack vm")
	}

	err = utils.DeleteNodeByName(ctx, r.Client, scope.StellarisMigrateNode.Name)
	if err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, errors.Wrap(err, "failed to delete node by name")
	}
	controllerutil.RemoveFinalizer(scope.StellarisMigrateNode, constants.StellarisMigrateNodeFinalizer)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StellarisMigrateNodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&migratev1alpha1.StellarisMigrateNode{}).
		Complete(r)
}

// updateActiveMigrations efficiently updates just the ActiveMigrations field
func (r *StellarisMigrateNodeReconciler) updateActiveMigrations(ctx context.Context,
	scope *scope.StellarisMigrateNodeScope) (ctrl.Result, error) {
	vjNode := scope.StellarisMigrateNode

	// Get active migrations happening on the node
	activeMigrations, err := utils.GetActiveMigrations(ctx, vjNode.Name, r.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to get active migrations")
	}
	// Create a patch to update only the ActiveMigrations field
	patch := client.MergeFrom(vjNode.DeepCopy())
	vjNode.Status.ActiveMigrations = activeMigrations

	err = r.Client.Status().Patch(ctx, vjNode, patch)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to patch stellaris-migrate node status")
	}

	// Always requeue after one minute
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}
