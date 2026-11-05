package authn_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
)

type pairedVerifier struct {
	calls atomic.Int32
	valid bool
}

func (pairedVerifier) Verify(string) (*cliutil.VerifyResult, error) {
	return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: "baseline-run", Subject: "owner", Scopes: []string{"prompt-manager:read"}, ExpiresAt: time.Now().Add(time.Hour).Unix()}}, nil
}

func (v *pairedVerifier) VerifyPair(_ context.Context, _, _ string) (*cliutil.VerifyPairResult, error) {
	v.calls.Add(1)
	if !v.valid {
		return &cliutil.VerifyPairResult{Error: "invalid_pair"}, nil
	}
	expires := time.Now().Add(time.Hour).UTC()
	grantJSON, _ := json.Marshal(identity.OperationGrant{ID: "grant-1", Policy: "effort-metadata", Approver: "owner", Operations: []string{"effort.reconcile"}, Targets: []string{"agent-manager:effort/target"}, ExpiresAt: expires})
	return &cliutil.VerifyPairResult{Valid: true, Operation: &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: "grant-run", Subject: "owner", Scopes: []string{"agent-manager:write"}, ExpiresAt: expires.Unix(), Meta: map[string]string{"operation_grant": string(grantJSON)}}}}, nil
}

func TestRequireOperationUsesFreshSeparatePairWithoutChangingBaselinePrincipal(t *testing.T) {
	verifier := &pairedVerifier{valid: true}
	var principal identity.Principal
	var ordinaryCapabilityErr error
	var exactOperationErr error
	var wrongTargetErr error
	var wrongCapabilityErr error
	var wrongOperationErr error
	var forwardedOperation string
	handler := provenance.Middleware(verifier)(authn.Middleware(authn.Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, _ = identity.PrincipalFromContext(r.Context())
		forwardedOperation = provenance.ForwardedOperationToken(r.Context())
		_, ordinaryCapabilityErr = authn.RequireCapability(r.Context(), "prompt-manager:read")
		_, exactOperationErr = authn.RequireOperation(r.Context(), "agent-manager:write", "effort.reconcile", "agent-manager:effort/target")
		_, wrongTargetErr = authn.RequireOperation(r.Context(), "agent-manager:write", "effort.reconcile", "agent-manager:effort/other")
		_, wrongCapabilityErr = authn.RequireOperation(r.Context(), "prompt-manager:read", "effort.reconcile", "agent-manager:effort/target")
		_, wrongOperationErr = authn.RequireOperation(r.Context(), "agent-manager:write", "effort.delete", "agent-manager:effort/target")
		w.WriteHeader(http.StatusNoContent)
	})))
	req := httptest.NewRequest(http.MethodPost, "http://localhost/effort", nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "baseline-token")
	req.Header.Set(cliutil.HeaderAgentOperationToken, "operation-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusNoContent {
		t.Fatalf("handler status = %d", response.Code)
	}
	if principal.Kind != identity.ActorAgent || principal.Subject != "baseline-run" || principal.Grant != nil || !reflect.DeepEqual(principal.Scopes, []string{"prompt-manager:read"}) {
		t.Fatalf("supplement changed baseline principal: %+v", principal)
	}
	if ordinaryCapabilityErr != nil || exactOperationErr != nil || wrongTargetErr == nil || wrongCapabilityErr == nil || wrongOperationErr == nil {
		t.Fatalf("unexpected authorization results: ordinary=%v exact=%v wrong-target=%v wrong-capability=%v wrong-operation=%v", ordinaryCapabilityErr, exactOperationErr, wrongTargetErr, wrongCapabilityErr, wrongOperationErr)
	}
	if forwardedOperation != "operation-token" {
		t.Fatalf("verified operation proof was not retained for trusted forwarding: %q", forwardedOperation)
	}
	if verifier.calls.Load() != 5 {
		t.Fatalf("expected middleware pair check and one fresh check per protected effect, calls=%d", verifier.calls.Load())
	}
}

func TestRequireOperationRejectsUnavailablePairAndDoesNotUnionScopes(t *testing.T) {
	verifier := &pairedVerifier{valid: false}
	var baselineOperationErr error
	var grantCapabilityErr error
	handler := provenance.Middleware(verifier)(authn.Middleware(authn.Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, baselineOperationErr = authn.RequireOperation(r.Context(), "agent-manager:write", "effort.reconcile", "agent-manager:effort/target")
		_, grantCapabilityErr = authn.RequireCapability(r.Context(), "agent-manager:write")
		w.WriteHeader(http.StatusNoContent)
	})))
	req := httptest.NewRequest(http.MethodPost, "http://localhost/effort", nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "baseline-token")
	req.Header.Set(cliutil.HeaderAgentOperationToken, "revoked-operation-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusNoContent || baselineOperationErr == nil || grantCapabilityErr == nil {
		t.Fatalf("unavailable pair authorized or unioned scopes: status=%d operation=%v capability=%v", response.Code, baselineOperationErr, grantCapabilityErr)
	}
}
