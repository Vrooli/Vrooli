package requestpolicy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	memberflowconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/memberflow/memberflow_v1connect"
)

func TestBoundedOperatingModelListRequiresVerifiedIdentityAndKeepsMutationsClosed(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	grant := &identity.OperationGrant{
		ID: "unrelated-agent-management", Policy: "agent-management", Approver: "owner",
		ExpiresAt: time.Now().Add(time.Hour),
		Rules:     []identity.OperationRule{{Policy: "agent-management", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:run/run-123"}}},
	}
	verifiedWithGrant := provenance.NewContext(context.Background(), provenance.Provenance{
		Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified,
		RunID: "run-123", Subject: "owner", Scopes: []string{"prompt-manager:read"}, Grant: grant,
	})
	verifiedWithAuthority := authn.WithRequestOperationAuthority(provenance.NewContext(context.Background(), provenance.Provenance{
		Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified,
		RunID: "run-123", Subject: "owner", Scopes: []string{"prompt-manager:read"},
	}), denyAuthority{})
	unverifiedBounded := authn.WithRequestOperationAuthority(identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorAgent, Subject: "run-123", Grant: grant,
	}), denyAuthority{})
	ordinaryVerified := provenance.NewContext(context.Background(), provenance.Provenance{
		Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified,
		RunID: "run-123", Subject: "owner", Scopes: []string{"prompt-manager:read"},
	})

	cases := []struct {
		name   string
		method string
		path   string
		ctx    context.Context
		allow  bool
	}{
		{name: "verified identity with unrelated grant can list operating models", method: http.MethodPost, path: memberflowconnect.MemberflowServiceGetOperatingModelsProcedure, ctx: verifiedWithGrant, allow: true},
		{name: "verified identity with request authority can list operating models", method: http.MethodPost, path: memberflowconnect.MemberflowServiceGetOperatingModelsProcedure, ctx: verifiedWithAuthority, allow: true},
		{name: "ordinary verified caller remains with the existing handler", method: http.MethodPost, path: memberflowconnect.MemberflowServiceGetOperatingModelsProcedure, ctx: ordinaryVerified, allow: true},
		{name: "unverified bounded caller cannot list operating models", method: http.MethodPost, path: memberflowconnect.MemberflowServiceGetOperatingModelsProcedure, ctx: unverifiedBounded},
		{name: "member topic mutation remains closed", method: http.MethodPost, path: memberflowconnect.MemberflowServiceUpdateMemberTopicsProcedure, ctx: verifiedWithGrant},
		{name: "unknown memberflow RPC remains closed", method: http.MethodPost, path: "/vrooli.prompt_manager.v1.memberflow.MemberflowService/FutureRead", ctx: verifiedWithGrant},
		{name: "wrong method remains closed", method: http.MethodGet, path: memberflowconnect.MemberflowServiceGetOperatingModelsProcedure, ctx: verifiedWithGrant},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entered := false
			wire := authn.Middleware(authn.Config{})(PreparationBoundary(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				entered = true
				w.WriteHeader(http.StatusNoContent)
			})))
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`)).WithContext(tc.ctx)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Connect-Protocol-Version", "1")
			rec := httptest.NewRecorder()
			wire.ServeHTTP(rec, req)
			if entered != tc.allow {
				t.Fatalf("handler entered=%v, want %v; status=%d body=%s", entered, tc.allow, rec.Code, rec.Body.String())
			}
			if tc.allow && rec.Code != http.StatusNoContent {
				t.Fatalf("allowed read returned status=%d body=%s", rec.Code, rec.Body.String())
			}
			if !tc.allow && (rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "permission_denied")) {
				t.Fatalf("bounded refusal lost: status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}
