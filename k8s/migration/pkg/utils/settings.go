package utils

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	"github.com/kashyapshashankv/stellaris-migrate/k8s/migration/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctxlog "sigs.k8s.io/controller-runtime/pkg/log"
)

// VjailbreakSettings holds the settings for stellaris-migrate components
type VjailbreakSettings struct {
	// ChangedBlocksCopyIterationThreshold is the number of iterations to copy changed blocks
	ChangedBlocksCopyIterationThreshold int
	// VMActiveWaitIntervalSeconds is the interval to wait for VM to become active
	VMActiveWaitIntervalSeconds int
	// VMActiveWaitRetryLimit is the number of retries to wait for VM to become active
	VMActiveWaitRetryLimit int
	// DefaultMigrationMethod is the default migration method (hot/cold)
	DefaultMigrationMethod string
	// VCenterScanConcurrencyLimit is the max number of vms to scan at the same time
	VCenterScanConcurrencyLimit int
	// CleanupVolumesAfterConvertFailure is whether to cleanup volumes after disk convert failure
	CleanupVolumesAfterConvertFailure bool
	// PopulateVMwareMachineFlavors is whether to automatically populate VMwareMachine objects with OpenStack flavors
	PopulateVMwareMachineFlavors bool
}

// atoi is a helper function to convert string to int with a default value of 0
func atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

// GetMigrateSettings retrieves the stellaris-migrate settings from the configmap
func GetMigrateSettings(ctx context.Context, k8sClient client.Client) (*VjailbreakSettings, error) {
	log := ctxlog.FromContext(ctx)

	// Get the stellaris-migrate settings configmap
	vjailbreakSettingsCM := &corev1.ConfigMap{}
	if err := k8sClient.Get(ctx, k8stypes.NamespacedName{
		Name:      constants.StellarisMigrateSettingsConfigMapName,
		Namespace: constants.NamespaceMigrationSystem,
	}, vjailbreakSettingsCM); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("stellaris-migrate settings configmap not found, using default settings")
			return getDefaultSettings(), nil
		}
		return nil, errors.Wrap(err, "failed to get stellaris-migrate settings configmap")
	}

	// Set default values if not present in the configmap
	if vjailbreakSettingsCM.Data == nil {
		vjailbreakSettingsCM.Data = make(map[string]string)
	}

	if vjailbreakSettingsCM.Data["CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD"] == "" {
		vjailbreakSettingsCM.Data["CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD"] = "20"
	}

	if vjailbreakSettingsCM.Data["VM_ACTIVE_WAIT_INTERVAL_SECONDS"] == "" {
		vjailbreakSettingsCM.Data["VM_ACTIVE_WAIT_INTERVAL_SECONDS"] = "20"
	}

	if vjailbreakSettingsCM.Data["VM_ACTIVE_WAIT_RETRY_LIMIT"] == "" {
		vjailbreakSettingsCM.Data["VM_ACTIVE_WAIT_RETRY_LIMIT"] = "15"
	}

	if vjailbreakSettingsCM.Data["DEFAULT_MIGRATION_METHOD"] == "" {
		vjailbreakSettingsCM.Data["DEFAULT_MIGRATION_METHOD"] = "hot"
	}

	if vjailbreakSettingsCM.Data["VCENTER_SCAN_CONCURRENCY_LIMIT"] == "" {
		vjailbreakSettingsCM.Data["VCENTER_SCAN_CONCURRENCY_LIMIT"] = "10"
	}

	if vjailbreakSettingsCM.Data["CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE"] == "" {
		vjailbreakSettingsCM.Data["CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE"] = "false"
	}

	if vjailbreakSettingsCM.Data["POPULATE_VMWARE_MACHINE_FLAVORS"] == "" {
		vjailbreakSettingsCM.Data["POPULATE_VMWARE_MACHINE_FLAVORS"] = trueString
	}

	return &VjailbreakSettings{
		ChangedBlocksCopyIterationThreshold: atoi(vjailbreakSettingsCM.Data["CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD"]),
		VMActiveWaitIntervalSeconds:         atoi(vjailbreakSettingsCM.Data["VM_ACTIVE_WAIT_INTERVAL_SECONDS"]),
		VMActiveWaitRetryLimit:              atoi(vjailbreakSettingsCM.Data["VM_ACTIVE_WAIT_RETRY_LIMIT"]),
		DefaultMigrationMethod:              vjailbreakSettingsCM.Data["DEFAULT_MIGRATION_METHOD"],
		VCenterScanConcurrencyLimit:         atoi(vjailbreakSettingsCM.Data["VCENTER_SCAN_CONCURRENCY_LIMIT"]),
		CleanupVolumesAfterConvertFailure:   vjailbreakSettingsCM.Data["CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE"] == "true",
		PopulateVMwareMachineFlavors:        vjailbreakSettingsCM.Data["POPULATE_VMWARE_MACHINE_FLAVORS"] == "true",
	}, nil
}

// getDefaultSettings returns default stellaris-migrate settings
func getDefaultSettings() *VjailbreakSettings {
	return &VjailbreakSettings{
		ChangedBlocksCopyIterationThreshold: 20,
		VMActiveWaitIntervalSeconds:         20,
		VMActiveWaitRetryLimit:              15,
		DefaultMigrationMethod:              "hot",
		VCenterScanConcurrencyLimit:         10,
		CleanupVolumesAfterConvertFailure:   false,
		PopulateVMwareMachineFlavors:        true,
	}
}
