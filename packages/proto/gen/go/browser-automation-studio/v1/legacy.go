package browser_automation_studio_v1

import (
	base "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	timeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

// Legacy root-package compatibility shim for build-tagged tests.
type ExecutionTimeline = timeline.ExecutionTimeline
type ExecutionStatus = base.ExecutionStatus

const (
	ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED = base.ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED
	ExecutionStatus_EXECUTION_STATUS_PENDING     = base.ExecutionStatus_EXECUTION_STATUS_PENDING
	ExecutionStatus_EXECUTION_STATUS_RUNNING     = base.ExecutionStatus_EXECUTION_STATUS_RUNNING
	ExecutionStatus_EXECUTION_STATUS_COMPLETED   = base.ExecutionStatus_EXECUTION_STATUS_COMPLETED
	ExecutionStatus_EXECUTION_STATUS_FAILED      = base.ExecutionStatus_EXECUTION_STATUS_FAILED
	ExecutionStatus_EXECUTION_STATUS_CANCELLED   = base.ExecutionStatus_EXECUTION_STATUS_CANCELLED
)
