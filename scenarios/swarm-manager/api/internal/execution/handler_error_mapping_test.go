package execution

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"
)

func TestMapMutationError_CancelRestoreLoadFailureReturnsConflict(t *testing.T) {
	h := &Handler{}
	rec := httptest.NewRecorder()

	h.mapMutationError(rec, "[execution] cancel", errWithMessage("failed to load backlog item for cancel restore: open /tmp/spec.json: no such file or directory"))

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "execution canceled but backlog status restore failed") {
		t.Fatalf("expected conflict guidance message, got %q", rec.Body.String())
	}
}

func TestMapMutationError_CancelRestoreStatusWriteFailureReturnsConflict(t *testing.T) {
	h := &Handler{}
	rec := httptest.NewRecorder()

	h.mapMutationError(rec, "[execution] cancel", errWithMessage("failed to restore backlog status after cancel: permission denied"))

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "execution canceled but backlog status restore failed") {
		t.Fatalf("expected conflict guidance message, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "fix the backlog item status and retry") {
		t.Fatalf("expected remediation guidance, got %q", rec.Body.String())
	}
}

func TestMapCreateError_AgentManagerRequestFailureReturnsBadGateway(t *testing.T) {
	h := &Handler{}
	rec := httptest.NewRecorder()

	h.mapCreateError(rec, fmt.Errorf("%w: status 500", agentmanager.ErrRequestFailed))

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "check agent-manager health/logs and retry") {
		t.Fatalf("expected remediation guidance, got %q", rec.Body.String())
	}
}

func TestMapCreateError_ProcessPreflightFailureReturnsBadRequest(t *testing.T) {
	h := &Handler{}
	rec := httptest.NewRecorder()

	h.mapCreateError(rec, errWithMessage("process preflight failed: 3 critical clarify question(s) remain unanswered"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "process preflight failed") {
		t.Fatalf("expected preflight failure details, got %q", rec.Body.String())
	}
}

func TestSummarizeCreateError_TruncatesLongMessages(t *testing.T) {
	longErr := errWithMessage(strings.Repeat("x", 500))
	got := summarizeCreateError(longErr)
	if len(got) > 243 {
		t.Fatalf("expected summarized error to be truncated, got len=%d", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected truncated message to end with ellipsis, got %q", got)
	}
}

type errWithMessage string

func (e errWithMessage) Error() string {
	return string(e)
}
