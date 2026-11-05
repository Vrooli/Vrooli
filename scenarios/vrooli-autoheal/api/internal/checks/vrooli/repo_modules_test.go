package vrooli

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"
)

// The verbatim text the 2026-09-01 outage produced. If a refactor stops this
// matching, the detection gap silently reopens.
const repoDriftOutput = `../../../packages/api-core/nodereach/client.go:19:2: missing go.sum entry ` +
	`for module providing package github.com/gorilla/websocket (imported by github.com/vrooli/api-core/nodereach); to add:
	go get github.com/vrooli/api-core/nodereach@v0.0.0`

func newRepoCheck(t *testing.T, output string, err error) *RepoModulesCheck {
	t.Helper()
	mockExecutor := testutil.NewMockExecutor()
	mockExecutor.Responses["go -C /repo list -e ./..."] = testutil.MockResponse{
		Output: []byte(output),
		Error:  err,
	}
	return NewRepoModulesCheck(
		WithRepoModulesExecutor(mockExecutor),
		WithRepoModulesRoot(func() string { return "/repo" }),
	)
}

func TestRepoModulesCheckDetectsMissingGoSum(t *testing.T) {
	result := newRepoCheck(t, repoDriftOutput, nil).Run(context.Background())

	if result.Status != checks.StatusCritical {
		t.Fatalf("module drift must be critical, got %v", result.Status)
	}
	if result.Details["rootCause"] != rootCauseRepoRootModuleDrift {
		t.Errorf("rootCause = %v", result.Details["rootCause"])
	}
	if result.Details["recommendedAction"] != recommendedActionRecoverRepoRootGo {
		t.Errorf("recommendedAction = %v", result.Details["recommendedAction"])
	}
	if result.Details["signature"] != "missing-go-sum-entry" {
		t.Errorf("signature = %v, want missing-go-sum-entry", result.Details["signature"])
	}
}

func TestRepoModulesCheckHealthyWhenGraphResolves(t *testing.T) {
	result := newRepoCheck(t, "github.com/vrooli/vrooli/internal/app\n", nil).Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Fatalf("resolving graph must be OK, got %v (%s)", result.Status, result.Message)
	}
	if _, ok := result.Details["rootCause"]; ok {
		t.Error("healthy result must carry no root cause")
	}
}

// An unclassifiable probe failure must not invite a dependency recovery that
// may be entirely irrelevant to the real cause.
func TestRepoModulesCheckUnclassifiedFailureIsNotAutoHealEligible(t *testing.T) {
	result := newRepoCheck(t, "some unrelated toolchain explosion", testutil.ErrCommandNotFound).
		Run(context.Background())

	if result.Status != checks.StatusWarning {
		t.Fatalf("unclassified failure should warn, got %v", result.Status)
	}
	if eligible, _ := result.Details["autoHealEligible"].(bool); eligible {
		t.Error("unclassified failure must not be auto-heal eligible")
	}
}

func TestRepoModulesCheckHandlesUnresolvableRoot(t *testing.T) {
	check := NewRepoModulesCheck(
		WithRepoModulesExecutor(testutil.NewMockExecutor()),
		WithRepoModulesRoot(func() string { return "" }),
	)
	result := check.Run(context.Background())

	if result.Status != checks.StatusWarning {
		t.Fatalf("unresolvable root should warn, got %v", result.Status)
	}
	if eligible, _ := result.Details["autoHealEligible"].(bool); eligible {
		t.Error("measuring nothing must never be auto-heal eligible")
	}
}

// The recovery rewrites go.mod and go.sum, so it must be offered only when a
// drift signature was actually observed.
func TestRepoModulesRecoveryActionGatedOnSignature(t *testing.T) {
	check := newRepoCheck(t, repoDriftOutput, nil)

	drifted := check.Run(context.Background())
	actions := check.RecoveryActions(&drifted)
	if len(actions) != 1 || !actions[0].Available {
		t.Fatalf("recovery must be offered after drift, got %+v", actions)
	}
	if actions[0].ID != recommendedActionRecoverRepoRootGo {
		t.Errorf("unexpected action id %q", actions[0].ID)
	}

	healthy := checks.Result{Details: map[string]interface{}{}}
	if actions := check.RecoveryActions(&healthy); actions[0].Available {
		t.Error("recovery must not be offered without a drift signature")
	}
	if actions := check.RecoveryActions(nil); actions[0].Available {
		t.Error("recovery must not be offered with no prior result")
	}
}

func TestRepoModulesRejectsUnknownAction(t *testing.T) {
	check := newRepoCheck(t, repoDriftOutput, nil)
	result := check.ExecuteAction(context.Background(), "definitely-not-an-action")
	if result.Success {
		t.Fatal("unknown action must not report success")
	}
}

// Interface conformance: the registry only schedules things that satisfy Check.
func TestRepoModulesCheckImplementsCheck(t *testing.T) {
	var _ checks.Check = (*RepoModulesCheck)(nil)
}
