package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	runidentity "agent-manager/internal/identity"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"

	"github.com/vrooli/api-core/authn"
	requestidentity "github.com/vrooli/api-core/identity"
)

func TestVerifyIdentityTokenHTTPTransportsRealCredentialPair(t *testing.T) {
	const secret = "identity-pair-handler-test-secret"
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	service := orchestration.New(nil, nil, repos.Runs,
		orchestration.WithEvents(events),
		orchestration.WithIdentitySecret([]byte(secret)),
		orchestration.WithAuthorizations(database.NewAuthorizationRepository(db)),
	)
	policy, ok := authn.InteractivePolicy("swarm-bookkeeping")
	if !ok {
		t.Fatal("swarm-bookkeeping policy missing")
	}
	owner := requestidentity.Principal{
		Kind: requestidentity.ActorHuman, Subject: "identity-pair-owner",
		Source: requestidentity.SourceCloudflareAccess, Verified: true,
		Scopes: append([]string{"agent-manager:write"}, policy.Scopes...), ExpiresAt: time.Now().Add(time.Hour),
	}
	ownerCtx := requestidentity.WithPrincipal(context.Background(), owner)
	attached, err := service.AttachRun(ownerCtx, orchestration.AttachRunRequest{HarnessKind: "codex", HarnessSession: "identity-pair-handler", ProcessID: os.Getpid()})
	if err != nil {
		t.Fatalf("attach baseline run: %v", err)
	}
	grant, err := service.RequestAuthorization(context.Background(), orchestration.AuthorizationRequest{
		RunID: attached.Run.ID, PID: os.Getpid(), HarnessKind: "codex", HarnessSession: attached.Run.HarnessSessionID,
		Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 3600,
	})
	if err != nil {
		t.Fatalf("request operation grant: %v", err)
	}
	approved, err := service.ApproveAuthorization(ownerCtx, grant.ID)
	if err != nil {
		t.Fatalf("approve operation grant: %v", err)
	}
	var claims runidentity.Claims
	if err := json.Unmarshal([]byte(approved.ClaimsJSON), &claims); err != nil {
		t.Fatalf("decode approved grant: %v", err)
	}
	operationToken, err := runidentity.GenerateToken(&claims, []byte(secret))
	if err != nil {
		t.Fatalf("mint approved grant token: %v", err)
	}
	handler := New(orchestration.NewHandlerServices(service))
	body, err := json.Marshal(map[string]string{"token": attached.Token, "operation_token": operationToken})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/identity/verify", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.VerifyIdentityToken(response, request)
	var result orchestration.IdentityPairVerifyResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode pair response: status=%d body=%s err=%v", response.Code, response.Body.String(), err)
	}
	if response.Code != http.StatusOK || !result.Valid || result.Operation == nil || !result.Operation.Valid || result.Operation.Claims == nil {
		t.Fatalf("HTTP identity pair was not verified by the real owner service: status=%d result=%+v body=%s", response.Code, result, response.Body.String())
	}
}
