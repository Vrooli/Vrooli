package flow

import (
	"testing"

	"asset-studio/internal/renders/flow/generated"
)

func TestRenderFormalReplay(t *testing.T) {
	generated.RunReplay(t, func(state generated.RenderStatus, event generated.RenderEvent) (generated.RenderStatus, error) {
		renderState := RenderState{
			Status:             RenderStatus(state),
			ProvenanceRecorded: true,
			ActualCostRecorded: true,
		}
		next, err := TransitionRender(renderState, RenderEvent(event))
		return generated.RenderStatus(next.Status), err
	})
}

func TestRenderSuccessRequiresProvenanceAndCost(t *testing.T) {
	state := InitialRenderState()
	var err error
	state, err = TransitionRender(state, RenderStart)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = TransitionRender(state, RenderSucceed); err == nil {
		t.Fatal("success without provenance/cost accepted")
	}
	state.ProvenanceRecorded = true
	state.ActualCostRecorded = true
	state, err = TransitionRender(state, RenderSucceed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = TransitionRender(state, RenderCancel); err == nil {
		t.Fatal("terminal transition accepted")
	}
}
