package workflows

import "errors"

var (
	errInvalidWorkflowID     = errors.New("invalid workflow id")
	errWorkflowNotFound      = errors.New("workflow not found")
	errWorkflowVersionConfl  = errors.New("workflow version conflict")
	errWorkflowNameConflict  = errors.New("workflow name already exists in this project")
	errInvalidWorkflowFormat = errors.New("invalid workflow format: nodes must have 'action' field with typed action definitions")
	errModifyPromptRequired  = errors.New("modification_prompt is required")
	errCurrentFlowRequired   = errors.New("current_flow is required")
	errWorkflowPayload       = errors.New("workflow payload required")
	errUnsupportedSeedMode   = errors.New("unsupported seed_mode")
	errSeedSelfReference     = errors.New("seed_mode=needs-applying is not supported when targeting browser-automation-studio directly; run via test-genie or the CLI handshake")
	errSeedCleanupUnavail    = errors.New("seed cleanup manager unavailable")
)
