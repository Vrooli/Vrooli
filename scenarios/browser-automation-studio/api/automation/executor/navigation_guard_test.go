package executor

import (
	"context"
	"strings"
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

func engineSpec() engine.SessionSpec { return engine.SessionSpec{} }

// ensureNavigation's error message must distinguish two failure modes:
//
//  1. No navigate step has run at all → "no prior navigate step".
//  2. A navigate step ran but failed → name the failing node, URL, and
//     summary so the caller can actually fix it.
//
// Before the fix, both cases produced the same misleading "no prior
// navigate step" message even when a navigate had already failed.

func TestEnsureNavigation_NoPriorNavigate_GivesGenericGuidance(t *testing.T) {
	exec := NewSimpleExecutor(nil)
	req := Request{}
	execCtx := executionContext{navigation: &navigationState{}}

	_, err := exec.ensureNavigation(context.Background(), req, execCtx, nil, engineSpec(), nil, "node-assert", "assert")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "no prior navigate step") {
		t.Errorf("expected 'no prior navigate step' phrasing, got: %s", msg)
	}
	if strings.Contains(msg, "failed:") {
		t.Errorf("did not expect failure detail when no navigate ran, got: %s", msg)
	}
}

func TestEnsureNavigation_PriorNavigateFailed_NamesTheFailure(t *testing.T) {
	exec := NewSimpleExecutor(nil)
	req := Request{}
	nav := &navigationState{}
	recordFailedNavigate(nav, contracts.CompiledInstruction{
		NodeID: "node-nav-1",
		Action: &basactions.ActionDefinition{
			Type: basactions.ActionType_ACTION_TYPE_NAVIGATE,
			Params: &basactions.ActionDefinition_Navigate{
				Navigate: &basactions.NavigateParams{Url: "http://localhost/"},
			},
		},
	}, "net::ERR_CONNECTION_REFUSED")
	execCtx := executionContext{navigation: nav}

	_, err := exec.ensureNavigation(context.Background(), req, execCtx, nil, engineSpec(), nil, "node-assert", "assert")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if strings.Contains(msg, "no prior navigate step") {
		t.Errorf("should not claim 'no prior navigate step' when one ran and failed, got: %s", msg)
	}
	for _, want := range []string{"node-nav-1", "http://localhost/", "net::ERR_CONNECTION_REFUSED"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error missing %q: %s", want, msg)
		}
	}
}

func TestMarkNavigation_ClearsFailedAttempt(t *testing.T) {
	nav := &navigationState{}
	recordFailedNavigate(nav, contracts.CompiledInstruction{NodeID: "n"}, "boom")
	if nav.lastAttempt == nil {
		t.Fatal("precondition: lastAttempt should be set")
	}
	markNavigation(nav)
	if !nav.hasNavigated {
		t.Error("markNavigation must set hasNavigated")
	}
	if nav.lastAttempt != nil {
		t.Error("a successful navigate should clear the stored failure")
	}
}
