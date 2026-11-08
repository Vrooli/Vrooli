package vrooli

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"
	integration "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/integrations/vrooli"
)

// Feature: autoheal tells degraded apart from down
//
//	As autoheal
//	I want a resource that is serving on the wrong device treated as degraded
//	So that I do not restart something that is working, in a loop that cannot
//	succeed because a restart cannot reach a device the host does not have.

func boolPtr(v bool) *bool { return &v }

// Scenario: a resource serving below its declared backend is a warning.
func TestResourceStatusIsDegradedSeparatesDegradedFromDown(t *testing.T) {
	cases := []struct {
		scenario string
		status   integration.ResourceStatus
		want     bool
	}{
		{
			scenario: "Given healthy false with serving true, Then it is degraded",
			status:   integration.ResourceStatus{Healthy: boolPtr(false), Serving: boolPtr(true)},
			want:     true,
		},
		{
			scenario: "Given healthy false with mode drift, Then it is degraded",
			status:   integration.ResourceStatus{Healthy: boolPtr(false), ModeDrift: true},
			want:     true,
		},
		{
			scenario: "Given healthy false with serving false, Then it is down, not degraded",
			status:   integration.ResourceStatus{Healthy: boolPtr(false), Serving: boolPtr(false)},
			want:     false,
		},
		{
			scenario: "Given healthy true, Then it is not degraded",
			status:   integration.ResourceStatus{Healthy: boolPtr(true), Serving: boolPtr(true)},
			want:     false,
		},
		{
			scenario: "Given no health verdict at all, Then it is not degraded",
			status:   integration.ResourceStatus{},
			want:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// When the degraded classification is read
			// Then it matches the contract
			if got := tc.status.IsDegraded(); got != tc.want {
				t.Fatalf("IsDegraded() = %v, want %v", got, tc.want)
			}
		})
	}
}

// [REQ:AUTOHEAL-P0-012]
func TestResourceRecoveryAcceptsServingResourceWithDownCompanion(t *testing.T) {
	executor := testutil.NewMockExecutor()
	executor.Responses["vrooli resource status reranker --health-only --json"] = testutil.MockResponse{Output: []byte(`{"success":true,"name":"reranker","installed":true,"running":true,"healthy":false,"serving":true,"status":"degraded: companion down"}`)}
	check := NewResourceCheck(
		"reranker",
		WithResourceExecutor(executor),
		WithResourceRecoveryPolling(20*time.Millisecond, time.Millisecond, 0),
	)

	result := check.verifyRecovery(context.Background(), checks.ActionResult{ActionID: "start", CheckID: check.ID()}, "start", time.Now())
	if !result.Success {
		t.Fatalf("verification = %#v, want successful availability recovery", result)
	}
	if result.Warning == "" || !strings.Contains(result.Warning, "degraded") {
		t.Fatalf("warning = %q, want degraded companion warning", result.Warning)
	}
}

// [REQ:AUTOHEAL-P0-012]
func TestResourceRecoveryRejectsStoppedResourceWithOKStatusCode(t *testing.T) {
	executor := testutil.NewMockExecutor()
	executor.Responses["vrooli resource status reranker --health-only --json"] = testutil.MockResponse{Output: []byte(`{"success":true,"name":"reranker","installed":true,"running":false,"healthy":false,"serving":false,"status":"stopped","resource":{"statusCode":"ok"}}`)}
	check := NewResourceCheck(
		"reranker",
		WithResourceExecutor(executor),
		WithResourceRecoveryPolling(5*time.Millisecond, time.Millisecond, 0),
	)

	status := check.Run(context.Background())
	if status.Status != checks.StatusCritical {
		t.Fatalf("stopped status = %s, want critical despite status_code=ok", status.Status)
	}
	result := check.verifyRecovery(context.Background(), checks.ActionResult{ActionID: "start", CheckID: check.ID()}, "start", time.Now())
	if result.Success {
		t.Fatalf("verification = %#v, want failed recovery for not-serving resource", result)
	}
}

// Scenario: the re-acquire action is offered only when the artifact drifted.
func TestResourceCheckOffersReacquireOnlyOnArtifactDrift(t *testing.T) {
	check := NewResourceCheck("reranker")

	// Given a check result with no artifact drift
	clean := &checks.Result{Details: map[string]interface{}{"running": true}}

	// When recovery actions are listed
	// Then re-acquire is not among them, because there is nothing to re-acquire
	if hasAction(check.RecoveryActions(clean), "reacquire-artifact") {
		t.Fatal("reacquire-artifact offered for a resource with no artifact drift")
	}

	// Given a check result reporting needs_reacquire
	drifted := &checks.Result{Details: map[string]interface{}{"running": false, "needsReacquire": true}}

	// When recovery actions are listed
	actions := check.RecoveryActions(drifted)

	// Then re-acquire is offered first, ahead of the generic lifecycle actions
	if !hasAction(actions, "reacquire-artifact") {
		t.Fatalf("reacquire-artifact not offered for a drifted artifact: %+v", actions)
	}
	if actions[0].ID != "reacquire-artifact" {
		t.Fatalf("first action = %q, want reacquire-artifact; a restart cannot fix a moved host", actions[0].ID)
	}
	// And it is not marked dangerous, because it replaces bytes rather than
	// interrupting service
	if actions[0].Dangerous {
		t.Fatal("reacquire-artifact marked dangerous; it re-downloads an artifact, it does not stop a service")
	}
	if !strings.Contains(actions[0].Description, "current facts") {
		t.Fatalf("description = %q, want it to say the artifact is re-resolved under current host facts", actions[0].Description)
	}
}

func hasAction(actions []checks.RecoveryAction, id string) bool {
	for _, action := range actions {
		if action.ID == id {
			return true
		}
	}
	return false
}
