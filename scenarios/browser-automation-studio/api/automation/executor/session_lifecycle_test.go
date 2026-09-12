package executor

import (
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
)

type priorNavigationState string

const (
	noNavigation     priorNavigationState = "none"
	navigated        priorNavigationState = "navigated"
	failedNavigation priorNavigationState = "failed_navigate"
)

func TestDecideLifecycleNavigationStateMatrix(t *testing.T) {
	t.Parallel()

	for _, mode := range []engine.SessionReuseMode{engine.ReuseModeReuse, engine.ReuseModeClean, engine.ReuseModeFresh} {
		for _, tc := range []struct {
			name       string
			boundary   executionBoundary
			hasSession bool
			prior      priorNavigationState
			reset      bool
		}{
			{name: "start", boundary: executionStart, hasSession: false, prior: noNavigation, reset: true},
			{name: "start_after_navigation", boundary: executionStart, hasSession: true, prior: navigated, reset: true},
			{name: "between_live", boundary: betweenSteps, hasSession: true, prior: navigated, reset: false},
			{name: "between_live_failed_navigation", boundary: betweenSteps, hasSession: true, prior: failedNavigation, reset: false},
			{name: "between_missing_session", boundary: betweenSteps, hasSession: false, prior: navigated, reset: true},
			{name: "end", boundary: executionEnd, hasSession: true, prior: navigated, reset: false},
		} {
			tc := tc
			t.Run(string(mode)+"/"+tc.name, func(t *testing.T) {
				t.Parallel()
				state := navigationFor(tc.prior)
				decision := decideLifecycle(mode, tc.boundary, tc.hasSession)
				if decision.ResetNavigation != tc.reset {
					t.Fatalf("reset navigation = %t, want %t", decision.ResetNavigation, tc.reset)
				}
				if decision.ResetNavigation {
					resetNavigation(state)
				}
				if tc.reset && (state.hasNavigated || state.lastAttempt != nil) {
					t.Fatalf("reset decision left navigation state behind: %+v", state)
				}
				if !tc.reset && tc.prior == navigated && !state.hasNavigated {
					t.Fatal("live session lost successful navigation between workflow steps")
				}
				if !tc.reset && tc.prior == failedNavigation && state.lastAttempt == nil {
					t.Fatal("live session lost failed-navigation diagnosis between workflow steps")
				}
			})
		}
	}
}

func navigationFor(prior priorNavigationState) *navigationState {
	state := &navigationState{}
	switch prior {
	case navigated:
		markNavigation(state)
	case failedNavigation:
		recordFailedNavigate(state, contracts.CompiledInstruction{NodeID: "navigate"}, "network unavailable")
	}
	return state
}
