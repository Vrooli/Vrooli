package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration"
	"github.com/google/uuid"
	requestidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
)

func TestRunCreateGrantGuardUsesExactCanonicalGrantRule(t *testing.T) {
	grantFor := func(withRunCreate bool) *requestidentity.OperationGrant {
		grant := &requestidentity.OperationGrant{
			ID: "approved-multi-policy", Policy: "agent-management", Approver: "owner-a",
			Operations: []string{"agent.manage", "run.create"}, Targets: []string{"agent-manager:*"},
			Rules:     []requestidentity.OperationRule{{Policy: "agent-management", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:*"}}},
			ExpiresAt: time.Now().Add(time.Hour),
		}
		if withRunCreate {
			grant.Rules = append(grant.Rules, requestidentity.OperationRule{Policy: "run-creation", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}})
		}
		return grant
	}
	for _, tc := range []struct {
		name        string
		grant       *requestidentity.OperationGrant
		expires     time.Time
		wantAllowed bool
	}{
		{name: "approved multi-policy run-create rule", grant: grantFor(true), expires: time.Now().Add(time.Hour), wantAllowed: true},
		{name: "unrelated agent-manage rule only", grant: grantFor(false), expires: time.Now().Add(time.Hour)},
		{name: "expired grant principal", grant: grantFor(true), expires: time.Now().Add(-time.Second)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			meta, err := json.Marshal(tc.grant)
			if err != nil {
				t.Fatal(err)
			}
			claims := &identity.Claims{RunID: uuid.New(), Meta: map[string]string{"operation_grant": string(meta)}}
			h := New(orchestration.HandlerServices{IdentityService: guardIdentityService{result: &orchestration.IdentityVerifyResult{Valid: true, Claims: claims}}})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", nil)
			req.Header.Set(cliutil.HeaderAgentIdentityToken, "run-token")
			principal := requestidentity.Principal{Kind: requestidentity.ActorAgent, Subject: uuid.NewString(), Owner: tc.grant.Approver, Verified: true, Scopes: []string{"agent-manager:write"}, Grant: tc.grant, ExpiresAt: tc.expires}
			req = req.WithContext(requestidentity.WithPrincipal(req.Context(), principal))
			rr := httptest.NewRecorder()
			refused := h.denyRunInitiatedLifecycleOperation(rr, req, "create-run")
			if allowed := !refused; allowed != tc.wantAllowed {
				t.Fatalf("create-run guard allowed=%v, want %v (status=%d)", allowed, tc.wantAllowed, rr.Code)
			}
			if tc.wantAllowed && rr.Code != 0 && rr.Code != http.StatusOK {
				t.Fatalf("unexpected response status for allowed run.create: %d", rr.Code)
			}
			if !tc.wantAllowed && rr.Code != http.StatusForbidden {
				t.Fatalf("refusal status = %d, want %d", rr.Code, http.StatusForbidden)
			}
		})
	}
}
