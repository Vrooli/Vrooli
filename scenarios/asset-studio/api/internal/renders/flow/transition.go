package flow

import (
	"fmt"

	"asset-studio/internal/renders/flow/generated"
)

type (
	RenderStatus = generated.RenderStatus
	RenderEvent  = generated.RenderEvent
)

const (
	RenderQueued    = generated.RenderJobQueued
	RenderRunning   = generated.RenderJobRunning
	RenderSucceeded = generated.RenderJobSucceeded
	RenderFailed    = generated.RenderJobFailed
	RenderCancelled = generated.RenderJobCancelled
	RenderStart     = generated.RenderJobStart
	RenderSucceed   = generated.RenderJobSucceed
	RenderFail      = generated.RenderJobFail
	RenderCancel    = generated.RenderJobCancel
)

type RenderState struct {
	Status             RenderStatus
	ProvenanceRecorded bool
	ActualCostRecorded bool
}

func InitialRenderState() RenderState { return RenderState{Status: RenderQueued} }
func TransitionRender(state RenderState, event RenderEvent) (RenderState, error) {
	if err := CheckRenderInvariants(state); err != nil {
		return state, err
	}
	next, err := generated.TransitionRenderJobStatus(state.Status, event)
	if err != nil {
		return state, err
	}
	if next == RenderSucceeded && !state.ProvenanceRecorded {
		return state, fmt.Errorf("successful render requires provenance")
	}
	if (next == RenderSucceeded || next == RenderFailed || next == RenderCancelled) && !state.ActualCostRecorded {
		return state, fmt.Errorf("terminal render requires actual cost")
	}
	state.Status = next
	return state, nil
}

func CheckRenderInvariants(state RenderState) error {
	for _, status := range generated.AllRenderJobStatuses() {
		if state.Status == status {
			return nil
		}
	}
	return fmt.Errorf("unknown render status %q", state.Status)
}
