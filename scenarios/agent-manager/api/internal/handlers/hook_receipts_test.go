package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/hookreceipt"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
)

type hookReceiptRuns struct{ run *domain.Run }

func (r hookReceiptRuns) GetStoredRun(context.Context, uuid.UUID) (*domain.Run, error) {
	return r.run, nil
}

func TestHookReceiptHandlerRequiresRunIdentityAndConsumesExactLaunchCredential(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	store := hookreceipt.NewStore(db)
	runID, invocationID := uuid.New(), uuid.New()
	run := &domain.Run{ID: runID, Status: domain.RunStatusRunning, ExecutionMode: domain.ExecutionModeCodecPipe, SessionID: "native-session"}
	credential, err := store.Issue(context.Background(), runID, invocationID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	handler := HookReceiptHandler{Store: store, Runs: hookReceiptRuns{run: run}}
	body := fmt.Sprintf(`{"receipt_id":%q,"invocation_id":%q,"session_id":"native-session","runner":"codex","evaluator_id":"policy:deny","duration_ms":0,"occurred_at":%q}`,
		uuid.NewString(), invocationID.String(), now.Format(time.RFC3339Nano))
	foreignRunID := uuid.New()
	verifier := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		subject := runID.String()
		if token == "foreign-run-token" {
			subject = foreignRunID.String()
		}
		if token != "run-token" && token != "foreign-run-token" {
			return nil, nil
		}
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: subject, Subject: "owner", ExpiresAt: time.Now().Add(time.Hour).Unix()}}, nil
	})
	route := provenance.Middleware(verifier)(authn.Middleware(authn.Config{})(http.HandlerFunc(handler.Receive)))
	call := func(identityToken, receiptToken, payload string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/runtime/hook-receipts", strings.NewReader(payload))
		if identityToken != "" {
			req.Header.Set(cliutil.HeaderAgentIdentityToken, identityToken)
		}
		if receiptToken != "" {
			req.Header.Set(hookReceiptTokenHeader, receiptToken)
		}
		response := httptest.NewRecorder()
		route.ServeHTTP(response, req)
		return response
	}
	if got := call("foreign-run-token", credential, body); got.Code != http.StatusConflict {
		t.Fatalf("foreign run status = %d, want conflict", got.Code)
	}
	if got := call("", credential, body); got.Code != http.StatusUnauthorized {
		t.Fatalf("missing receipt token status = %d, want unauthorized", got.Code)
	}
	if got := call("run-token", credential, strings.TrimSuffix(body, "}")+`,"transcript":"forged"}`); got.Code != http.StatusBadRequest {
		t.Fatalf("unknown-field status = %d, want bad request", got.Code)
	}
	wrongSession := strings.Replace(body, "native-session", "other-session", 1)
	if got := call("run-token", credential, wrongSession); got.Code != http.StatusConflict {
		t.Fatalf("wrong-session status = %d, want conflict", got.Code)
	}
	if got := call("run-token", credential, body); got.Code != http.StatusCreated {
		t.Fatalf("valid receipt status = %d, want created: %s", got.Code, got.Body.String())
	}
	if got := call("run-token", credential, body); got.Code != http.StatusConflict {
		t.Fatalf("replay status = %d, want conflict", got.Code)
	}
}
