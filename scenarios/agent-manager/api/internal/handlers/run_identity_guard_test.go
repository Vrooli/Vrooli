package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/cli-core/cliutil"
)

type guardIdentityService struct {
	result *orchestration.IdentityVerifyResult
}

func TestRunIdentityGuardProtectsAllOperatorLifecycleHandlers(t *testing.T) {
	h := New(orchestration.HandlerServices{IdentityService: guardIdentityService{
		result: &orchestration.IdentityVerifyResult{Valid: true, Claims: &identity.Claims{RunID: uuid.New()}},
	}})

	handlers := map[string]http.HandlerFunc{
		"delete":          h.DeleteRun,
		"continue":        h.ContinueRun,
		"wake":            h.WakeRun,
		"recover":         h.RecoverRun,
		"delete-message":  h.DeleteRunMessage,
		"stop-by-tag":     h.StopRunByTag,
		"approve":         h.ApproveRun,
		"reject":          h.RejectRun,
		"partial-approve": h.PartialApproveRun,
		"sandbox-sync":    h.SyncRunFromSandbox,
	}
	for name, handler := range handlers {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+uuid.NewString(), nil)
			req.Header.Set(cliutil.HeaderAgentIdentityToken, "run-token")
			req = mux.SetURLVars(req, map[string]string{"id": uuid.NewString(), "tag": "tag", "event_id": uuid.NewString()})
			rr := httptest.NewRecorder()
			handler(rr, req)
			if rr.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
			}
		})
	}
}

func (s guardIdentityService) VerifyIdentityToken(context.Context, string) (*orchestration.IdentityVerifyResult, error) {
	return s.result, nil
}

func TestRunIdentityGuardRefusesRunAuthorityAndUnverifiedClaims(t *testing.T) {
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

	for _, result := range []*orchestration.IdentityVerifyResult{
		nil,
		{Valid: false},
		{Valid: true}, // A verdict without signed claims is not an identity.
	} {
		h = New(orchestration.HandlerServices{IdentityService: guardIdentityService{result: result}})
		rr = httptest.NewRecorder()
		if !h.denyRunInitiatedLifecycleOperation(rr, req, "create-run") || rr.Code != http.StatusUnauthorized {
			t.Fatalf("unverified run credential became operator authority: status=%d result=%+v", rr.Code, result)
		}
	}
	// Absence of a run credential retains the existing operator-auth path.
	// This guard does not itself authenticate that operator.
	req.Header.Del(cliutil.HeaderAgentIdentityToken)
	rr = httptest.NewRecorder()
	if h.denyRunInitiatedLifecycleOperation(rr, req, "create-run") {
		t.Fatal("run-specific guard intercepted an operator request")
	}
}
