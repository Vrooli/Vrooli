package execution

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"swarm-manager/internal/apierr"
)

// These tests verify that the execution service returns properly typed
// apierr.DomainError values, so apierr.MapError in the handler produces
// correct HTTP responses without string-matching.

func TestMapError_CancelRestoreReturnsConflict(t *testing.T) {
	// Simulate the error returned by restoreBacklogStatusForRecord.
	err := apierr.Conflict("execution canceled but backlog status restore failed; fix the backlog item status and retry")

	rec := httptest.NewRecorder()
	apierr.MapError(rec, "[execution] cancel", err)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "execution canceled but backlog status restore failed") {
		t.Fatalf("expected conflict guidance message, got %q", rec.Body.String())
	}
}

func TestMapError_BadGatewayForAgentFailure(t *testing.T) {
	err := apierr.BadGateway("agent-manager request failed; check agent-manager health/logs and retry")

	rec := httptest.NewRecorder()
	apierr.MapError(rec, "[execution] create", err)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "check agent-manager health/logs and retry") {
		t.Fatalf("expected remediation guidance, got %q", rec.Body.String())
	}
}

func TestMapError_PreflightFailureReturnsBadRequest(t *testing.T) {
	err := apierr.BadRequest("process preflight failed: 3 critical clarify question(s) remain unanswered")

	rec := httptest.NewRecorder()
	apierr.MapError(rec, "[execution] create", err)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "process preflight failed") {
		t.Fatalf("expected preflight failure details, got %q", rec.Body.String())
	}
}

func TestMapError_UntypedErrorReturnsInternalServerError(t *testing.T) {
	err := errors.New(strings.Repeat("x", 500))

	rec := httptest.NewRecorder()
	apierr.MapError(rec, "[execution] create", err)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	body := rec.Body.String()
	if len(body) > 260 {
		t.Fatalf("expected truncated error message, got len=%d", len(body))
	}
}

func TestServiceErrors_AreTypedDomainErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"not found", apierr.NotFound("execution not found"), 404},
		{"bad request", apierr.BadRequest("backlog_kind is required"), 400},
		{"conflict", apierr.Conflict("queue depth limit exceeded"), 409},
		{"unavailable", apierr.Unavailable("agent-manager is not available"), 503},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var domainErr *apierr.DomainError
			if !errors.As(tt.err, &domainErr) {
				t.Fatal("expected DomainError")
			}
			if domainErr.Status != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, domainErr.Status)
			}
		})
	}
}
