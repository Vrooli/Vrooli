package wiring

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"agent-manager/internal/fallback"
	healthstore "agent-manager/internal/health"
	"agent-manager/internal/permissionpolicy"
	"agent-manager/internal/rolepolicy"

	"github.com/vrooli/api-core/health"
)

func TestLifecycleRefusalsAreFindingsNotReadinessFailures(t *testing.T) {
	for _, fatal := range []bool{false, true} {
		builder := health.New("agent-manager").Check(health.Func("database", func(context.Context) error {
			if fatal {
				return errors.New("database unavailable")
			}
			return nil
		}), health.Critical)
		handler := withLifecycleRefusalObservation(builder, func() (bool, string) { return false, "repeated run-identity lifecycle refusals detected" }).Handler()
		response := httptest.NewRecorder()
		handler(response, httptest.NewRequest(http.MethodGet, "/health", nil))
		var body health.Response
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		wantStatus, wantCode := health.StatusHealthy, http.StatusOK
		if fatal {
			wantStatus, wantCode = health.StatusUnhealthy, http.StatusServiceUnavailable
		}
		if body.Status != wantStatus || response.Code != wantCode || body.Readiness == fatal || body.Functional != nil || body.Metrics["lifecycle_refusals"] == nil {
			t.Fatalf("refusal/fatal dimensions coupled: fatal=%t HTTP=%d body=%+v", fatal, response.Code, body)
		}
	}
}

func TestPolicyHealthCheckersExposeUnavailableRequiredCatalogs(t *testing.T) {
	if RolePolicyHealthChecker(nil) != nil || PermissionPolicyHealthChecker(nil, nil) != nil {
		t.Fatal("nil policy state should not register a checker")
	}
	role, _ := rolepolicy.NewState(filepath.Join(t.TempDir(), "missing-role.json"), rolepolicy.Requirement{Required: true, Reason: "run safety"})
	roleResult := RolePolicyHealthChecker(role).Check(context.Background())
	if roleResult.Connected || roleResult.Name != "role_policy_catalog" || roleResult.Error == nil {
		t.Fatalf("role health=%+v", roleResult)
	}
	permission, _ := permissionpolicy.NewState(filepath.Join(t.TempDir(), "missing-permission.json"), permissionpolicy.Requirement{Required: true, Reason: "run safety"})
	permissionResult := PermissionPolicyHealthChecker(permission, nil).Check(context.Background())
	if permissionResult.Connected || permissionResult.Name != "permission_policy_catalog" || permissionResult.Error == nil {
		t.Fatalf("permission health=%+v", permissionResult)
	}
}

func TestFallbackHealthClassifierPreservesFailureObservation(t *testing.T) {
	adapter := fallbackHealthClassifier{classifier: fallback.NewTextClassifier()}
	got := adapter.Classify(healthstore.FailureInput{RunnerType: "codex", Stderr: "rate limit exceeded", Cause: errors.New("rate limit exceeded")})
	if got == nil {
		t.Fatal("expected classified health failure")
	}
	if got.Reason != string(fallback.ReasonRateLimit) {
		t.Fatalf("reason = %q, want %q", got.Reason, fallback.ReasonRateLimit)
	}
}
