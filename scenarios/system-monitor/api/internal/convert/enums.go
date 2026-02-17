package convert

import (
	"strings"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/domain"
)

func investigationStatusToProto(status string) domain.InvestigationStatus {
	switch strings.ToLower(status) {
	case "queued":
		return domain.InvestigationStatus_INVESTIGATION_STATUS_QUEUED
	case "in_progress":
		return domain.InvestigationStatus_INVESTIGATION_STATUS_IN_PROGRESS
	case "completed":
		return domain.InvestigationStatus_INVESTIGATION_STATUS_COMPLETED
	case "failed":
		return domain.InvestigationStatus_INVESTIGATION_STATUS_FAILED
	case "stopped":
		return domain.InvestigationStatus_INVESTIGATION_STATUS_STOPPED
	case "cancelled":
		return domain.InvestigationStatus_INVESTIGATION_STATUS_CANCELLED
	default:
		return domain.InvestigationStatus_INVESTIGATION_STATUS_UNSPECIFIED
	}
}

func investigationStepStatusToProto(status string) domain.InvestigationStepStatus {
	switch strings.ToLower(status) {
	case "pending":
		return domain.InvestigationStepStatus_INVESTIGATION_STEP_STATUS_PENDING
	case "in_progress":
		return domain.InvestigationStepStatus_INVESTIGATION_STEP_STATUS_IN_PROGRESS
	case "completed":
		return domain.InvestigationStepStatus_INVESTIGATION_STEP_STATUS_COMPLETED
	case "failed":
		return domain.InvestigationStepStatus_INVESTIGATION_STEP_STATUS_FAILED
	case "skipped":
		return domain.InvestigationStepStatus_INVESTIGATION_STEP_STATUS_SKIPPED
	default:
		return domain.InvestigationStepStatus_INVESTIGATION_STEP_STATUS_UNSPECIFIED
	}
}

func triggerConditionToProto(cond string) domain.TriggerCondition {
	switch strings.ToLower(cond) {
	case "above":
		return domain.TriggerCondition_TRIGGER_CONDITION_ABOVE
	case "below":
		return domain.TriggerCondition_TRIGGER_CONDITION_BELOW
	default:
		return domain.TriggerCondition_TRIGGER_CONDITION_UNSPECIFIED
	}
}

func scriptExecutionStatusToProto(status string) domain.ScriptExecutionStatus {
	switch strings.ToLower(status) {
	case "running":
		return domain.ScriptExecutionStatus_SCRIPT_EXECUTION_STATUS_RUNNING
	case "completed":
		return domain.ScriptExecutionStatus_SCRIPT_EXECUTION_STATUS_COMPLETED
	case "failed":
		return domain.ScriptExecutionStatus_SCRIPT_EXECUTION_STATUS_FAILED
	default:
		return domain.ScriptExecutionStatus_SCRIPT_EXECUTION_STATUS_UNSPECIFIED
	}
}
