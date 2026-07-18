package backlog

import (
	"errors"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
)

type workflowStartErrorKind int

const (
	workflowStartInternal workflowStartErrorKind = iota
	workflowStartUnavailable
	workflowStartBusy
)

func classifyWorkflowStartError(err error) workflowStartErrorKind {
	switch {
	case errors.Is(err, agentmanager.ErrNotAvailable):
		return workflowStartUnavailable
	case errors.Is(err, agentactivity.ErrBacklogItemBusy):
		return workflowStartBusy
	default:
		return workflowStartInternal
	}
}
