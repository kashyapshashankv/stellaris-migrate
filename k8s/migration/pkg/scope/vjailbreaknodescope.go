package scope

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	migratev1alpha1 "github.com/kashyapshashankv/stellaris-migrate/k8s/migration/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// StellarisMigrateNodeScopeParams defines the input parameters used to create a new Scope.
type StellarisMigrateNodeScopeParams struct {
	Logger         logr.Logger
	Client         client.Client
	StellarisMigrateNode *migratev1alpha1.StellarisMigrateNode
}

// NewStellarisMigrateNodeScope creates a new StellarisMigrateNodeScope from the supplied parameters.
// This is meant to be called for each reconcile iteration only on StellarisMigrateNodeReconciler.
func NewStellarisMigrateNodeScope(params StellarisMigrateNodeScopeParams) (*StellarisMigrateNodeScope, error) {
	if reflect.DeepEqual(params.Logger, logr.Logger{}) {
		params.Logger = ctrl.Log
	}

	return &StellarisMigrateNodeScope{
		Logger:         params.Logger,
		Client:         params.Client,
		StellarisMigrateNode: params.StellarisMigrateNode,
	}, nil
}

// StellarisMigrateNodeScope defines the basic context for an actuator to operate upon.
type StellarisMigrateNodeScope struct {
	logr.Logger
	Client         client.Client
	StellarisMigrateNode *migratev1alpha1.StellarisMigrateNode
}

// Close closes the current scope persisting the StellarisMigrateNode configuration and status.
func (s *StellarisMigrateNodeScope) Close() error {
	err := s.Client.Update(context.TODO(), s.StellarisMigrateNode, &client.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// Name returns the StellarisMigrateNode name.
func (s *StellarisMigrateNodeScope) Name() string {
	return s.StellarisMigrateNode.GetName()
}

// Namespace returns the StellarisMigrateNode namespace.
func (s *StellarisMigrateNodeScope) Namespace() string {
	return s.StellarisMigrateNode.GetNamespace()
}
