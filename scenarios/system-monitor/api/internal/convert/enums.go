package convert

import (
	"strings"

	investigationspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/investigations"
	scriptspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/scripts"
)

func investigationStatusToProto(status string) investigationspb.InvestigationStatus {
	switch strings.ToLower(status) {
	case "queued":
		return investigationspb.InvestigationStatus_INVESTIGATION_STATUS_QUEUED
	case "in_progress":
		return investigationspb.InvestigationStatus_INVESTIGATION_STATUS_IN_PROGRESS
	case "completed":
		return investigationspb.InvestigationStatus_INVESTIGATION_STATUS_COMPLETED
	case "failed":
		return investigationspb.InvestigationStatus_INVESTIGATION_STATUS_FAILED
	case "stopped":
		return investigationspb.InvestigationStatus_INVESTIGATION_STATUS_STOPPED
	case "cancelled":
		return investigationspb.InvestigationStatus_INVESTIGATION_STATUS_CANCELLED
	default:
		return investigationspb.InvestigationStatus_INVESTIGATION_STATUS_UNSPECIFIED
	}
}

func investigationStepStatusToProto(status string) investigationspb.InvestigationStepStatus {
	switch strings.ToLower(status) {
	case "pending":
		return investigationspb.InvestigationStepStatus_INVESTIGATION_STEP_STATUS_PENDING
	case "in_progress":
		return investigationspb.InvestigationStepStatus_INVESTIGATION_STEP_STATUS_IN_PROGRESS
	case "completed":
		return investigationspb.InvestigationStepStatus_INVESTIGATION_STEP_STATUS_COMPLETED
	case "failed":
		return investigationspb.InvestigationStepStatus_INVESTIGATION_STEP_STATUS_FAILED
	case "skipped":
		return investigationspb.InvestigationStepStatus_INVESTIGATION_STEP_STATUS_SKIPPED
	default:
		return investigationspb.InvestigationStepStatus_INVESTIGATION_STEP_STATUS_UNSPECIFIED
	}
}

func triggerConditionToProto(cond string) investigationspb.TriggerCondition {
	switch strings.ToLower(cond) {
	case "above":
		return investigationspb.TriggerCondition_TRIGGER_CONDITION_ABOVE
	case "below":
		return investigationspb.TriggerCondition_TRIGGER_CONDITION_BELOW
	default:
		return investigationspb.TriggerCondition_TRIGGER_CONDITION_UNSPECIFIED
	}
}

func scriptExecutionStatusToProto(status string) scriptspb.ScriptExecutionStatus {
	switch strings.ToLower(status) {
	case "running":
		return scriptspb.ScriptExecutionStatus_SCRIPT_EXECUTION_STATUS_RUNNING
	case "completed":
		return scriptspb.ScriptExecutionStatus_SCRIPT_EXECUTION_STATUS_COMPLETED
	case "failed":
		return scriptspb.ScriptExecutionStatus_SCRIPT_EXECUTION_STATUS_FAILED
	default:
		return scriptspb.ScriptExecutionStatus_SCRIPT_EXECUTION_STATUS_UNSPECIFIED
	}
}
