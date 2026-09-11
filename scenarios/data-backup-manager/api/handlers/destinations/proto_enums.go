package destinations

import (
	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/destinations"
	destinationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations"
)

// usageStateToProto translates the domain UsageState to the proto enum.
func usageStateToProto(s destinations.UsageState) destinationsv1.UsageState {
	switch s {
	case destinations.UsageStateWithin:
		return destinationsv1.UsageState_USAGE_STATE_WITHIN
	case destinations.UsageStateNear:
		return destinationsv1.UsageState_USAGE_STATE_NEAR
	case destinations.UsageStateOver:
		return destinationsv1.UsageState_USAGE_STATE_OVER
	default:
		return destinationsv1.UsageState_USAGE_STATE_UNSPECIFIED
	}
}

func readinessSeverityToProto(s destinationreadiness.CheckSeverity) destinationsv1.ReadinessSeverity {
	switch s {
	case destinationreadiness.SeverityPass:
		return destinationsv1.ReadinessSeverity_READINESS_SEVERITY_PASS
	case destinationreadiness.SeverityWarning:
		return destinationsv1.ReadinessSeverity_READINESS_SEVERITY_WARNING
	case destinationreadiness.SeverityFail:
		return destinationsv1.ReadinessSeverity_READINESS_SEVERITY_FAIL
	case destinationreadiness.SeverityUnknown:
		return destinationsv1.ReadinessSeverity_READINESS_SEVERITY_UNKNOWN
	default:
		return destinationsv1.ReadinessSeverity_READINESS_SEVERITY_UNSPECIFIED
	}
}

func preparationActionToProto(a destinationreadiness.PreparationAction) destinationsv1.PreparationAction {
	switch a {
	case destinationreadiness.ActionCreateSubdir:
		return destinationsv1.PreparationAction_PREPARATION_ACTION_CREATE_SUBDIR
	case destinationreadiness.ActionRelabel:
		return destinationsv1.PreparationAction_PREPARATION_ACTION_RELABEL
	case destinationreadiness.ActionClearDirectory:
		return destinationsv1.PreparationAction_PREPARATION_ACTION_CLEAR_DIRECTORY
	case destinationreadiness.ActionFormat:
		return destinationsv1.PreparationAction_PREPARATION_ACTION_FORMAT
	case destinationreadiness.ActionUnmount:
		return destinationsv1.PreparationAction_PREPARATION_ACTION_UNMOUNT
	case destinationreadiness.ActionCheckFilesystem:
		return destinationsv1.PreparationAction_PREPARATION_ACTION_CHECK_FILESYSTEM
	case destinationreadiness.ActionRepairFilesystem:
		return destinationsv1.PreparationAction_PREPARATION_ACTION_REPAIR_FILESYSTEM
	case destinationreadiness.ActionMountReadWrite:
		return destinationsv1.PreparationAction_PREPARATION_ACTION_MOUNT_READ_WRITE
	default:
		return destinationsv1.PreparationAction_PREPARATION_ACTION_UNSPECIFIED
	}
}

func protoToPreparationAction(a destinationsv1.PreparationAction) destinationreadiness.PreparationAction {
	switch a {
	case destinationsv1.PreparationAction_PREPARATION_ACTION_CREATE_SUBDIR:
		return destinationreadiness.ActionCreateSubdir
	case destinationsv1.PreparationAction_PREPARATION_ACTION_RELABEL:
		return destinationreadiness.ActionRelabel
	case destinationsv1.PreparationAction_PREPARATION_ACTION_CLEAR_DIRECTORY:
		return destinationreadiness.ActionClearDirectory
	case destinationsv1.PreparationAction_PREPARATION_ACTION_UNMOUNT:
		return destinationreadiness.ActionUnmount
	case destinationsv1.PreparationAction_PREPARATION_ACTION_CHECK_FILESYSTEM:
		return destinationreadiness.ActionCheckFilesystem
	case destinationsv1.PreparationAction_PREPARATION_ACTION_REPAIR_FILESYSTEM:
		return destinationreadiness.ActionRepairFilesystem
	case destinationsv1.PreparationAction_PREPARATION_ACTION_MOUNT_READ_WRITE:
		return destinationreadiness.ActionMountReadWrite
	case destinationsv1.PreparationAction_PREPARATION_ACTION_FORMAT:
		return destinationreadiness.ActionFormat
	default:
		return ""
	}
}
