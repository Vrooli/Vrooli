package requestpolicy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	teamsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams/teams_v1connect"
)

type ordinaryBaselinePairVerifier struct{}

func (ordinaryBaselinePairVerifier) Verify(string) (*cliutil.VerifyResult, error) {
	return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{
		RunID: "ordinary-baseline-run", Subject: "owner", Scopes: []string{"prompt-manager:read"}, ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}}, nil
}

func (ordinaryBaselinePairVerifier) VerifyPair(context.Context, string, string) (*cliutil.VerifyPairResult, error) {
	grant, _ := json.Marshal(identity.OperationGrant{
		ID: "unrelated-grant", Policy: "agent-management", Approver: "owner", ExpiresAt: time.Now().Add(time.Hour),
		Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:run/another-run"},
	})
	return &cliutil.VerifyPairResult{Valid: true, Operation: &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{
		RunID: "grant-run", Subject: "owner", Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour).Unix(),
		Meta: map[string]string{"operation_grant": string(grant)},
	}}}, nil
}

func TestPairedSupplementLeavesOrdinaryPromptManagerBoundaryUnbounded(t *testing.T) {
	entered := false
	var bounded bool
	var capabilityErr error
	var principal identity.Principal
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered = true
		bounded = IsBounded(r.Context())
		principal, _ = identity.PrincipalFromContext(r.Context())
		_, capabilityErr = authn.RequireCapability(r.Context(), "prompt-manager:read")
		if _, present := authn.RequestOperationAuthorityFromContext(r.Context()); present {
			t.Error("paired supplement was promoted to the global operation-authority marker")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	handler := provenance.Middleware(ordinaryBaselinePairVerifier{})(
		authn.Middleware(authn.Config{})(PreparationBoundary(inner)),
	)
	req := httptest.NewRequest(http.MethodPost, teamsconnect.TeamsServiceGetTeamProcedure, nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "baseline-proof")
	req.Header.Set(cliutil.HeaderAgentOperationToken, "unrelated-operation-proof")
	req.Header.Set("Connect-Protocol-Version", "1")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusNoContent || !entered {
		t.Fatalf("ordinary owner read was constrained by unrelated grant: status=%d body=%s entered=%v", response.Code, response.Body.String(), entered)
	}
	if bounded || capabilityErr != nil || principal.Kind != identity.ActorAgent || principal.Subject != "ordinary-baseline-run" || principal.Grant != nil || len(principal.Scopes) != 1 || principal.Scopes[0] != "prompt-manager:read" {
		t.Fatalf("supplement changed ordinary Prompt Manager authority: bounded=%v capability=%v principal=%+v", bounded, capabilityErr, principal)
	}
}
