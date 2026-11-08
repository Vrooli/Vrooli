package vrooli

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"
)

// Readiness refusals must retain their actual cause through monitoring and
// action verification; a bare degraded enum conceals a startup/restart loop.
func TestScenarioHealthEvidenceRetainsStartupReadinessCause(t *testing.T) {
	executor := testutil.NewMockExecutor()
	executor.Responses["vrooli scenario status test-scenario --json"] = testutil.MockResponse{Output: []byte(`{"success":true,"scenario":{"name":"test-scenario","status":"running","health_status":"degraded","health_error":"functional: startup initializing: in_flight_runs"}}`)}
	check := NewScenarioCheck("test-scenario", true, WithScenarioExecutor(executor), WithScenarioRecoveryPolling(5*time.Millisecond, time.Millisecond, 0))
	result := check.Run(t.Context())
	if result.Details["healthError"] != "functional: startup initializing: in_flight_runs" || result.Status != checks.StatusWarning {
		t.Fatalf("startup evidence lost: %+v", result)
	}
	verification := check.verifyRecovery(context.Background(), checks.ActionResult{ActionID: "restart", CheckID: check.ID()}, "restart", time.Now())
	if verification.Success || !strings.Contains(verification.Output, "functional: startup initializing: in_flight_runs") {
		t.Fatalf("failed readiness verification lost actual cause: %+v", verification)
	}
}
