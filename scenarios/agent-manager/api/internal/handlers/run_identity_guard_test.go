package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
)

type guardIdentityService struct {
	result *orchestration.IdentityVerifyResult
}

func (s guardIdentityService) VerifyIdentityToken(context.Context, string) (*orchestration.IdentityVerifyResult, error) {
	return s.result, nil
}

func TestRunIdentityGuardRejectsOnlyValidRunIdentities(t *testing.T) {
	h := New(orchestration.HandlerServices{IdentityService: guardIdentityService{
		result: &orchestration.IdentityVerifyResult{Valid: true, Claims: &identity.Claims{RunID: uuid.New()}},
	}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "run-token")
	rr := httptest.NewRecorder()
	if !h.denyRunInitiatedLifecycleOperation(rr, req, "create-run") {
		t.Fatal("guard did not reject a valid run identity")
	}
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}

	h = New(orchestration.HandlerServices{IdentityService: guardIdentityService{
		result: &orchestration.IdentityVerifyResult{Valid: false},
	}})
	rr = httptest.NewRecorder()
	if h.denyRunInitiatedLifecycleOperation(rr, req, "create-run") {
		t.Fatal("guard rejected an invalid/non-run identity")
	}
}
