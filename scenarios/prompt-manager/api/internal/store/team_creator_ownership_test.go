package store

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
	"prompt-manager/internal/teamconfig"
)

func preparationActor() identity.Principal {
	return identity.Principal{Kind: identity.ActorAgent, Subject: "creator-run", Owner: "owner", Source: identity.SourceAgentProvenance, Verified: true, Scopes: []string{"prompt-manager:write"}}
}

// preparationFixture creates a team "draft" with (attributed=true) or without
// (attributed=false) a verified creation proof, isolating the creator-ownership
// gate from the finite machinery that formerly shared this helper.
func preparationFixture(t *testing.T, attributed bool) (*FileTeamStore, context.Context, struct{}) {
	t.Helper()
	s := setupStateTestStore(t)
	ctx := identity.WithPrincipal(context.Background(), preparationActor())
	team, err := s.Get(context.Background(), "team-1")
	if err != nil {
		t.Fatal(err)
	}
	team.ID = "draft"
	team.Enabled = false
	team.Purpose = "delivery"
	team.Lifetime = "finite"
	team.EffortRefs = []string{"new:arbitrary-effort"}
	team.Coordination.Pattern = teamconfig.CoordinationPatternLeaderLed
	team.Coordination.LeadAgentID = "leader"
	team.Coordination.ReportingMode = teamconfig.ReportingModeLeader
	team.Execution.QueuePolicy = teamconfig.QueuePolicySerialized
	team.Execution.MaxConcurrentRuns = 1
	team.OperatingContract.Members["leader"] = team.OperatingContract.Members["lead"]
	createCtx := context.Background()
	if attributed {
		createCtx = ctx
	}
	if err = s.Create(createCtx, team); err != nil {
		t.Fatal(err)
	}
	return s, ctx, struct{}{}
}

// Creator-owns-while-alive: the team's own verified creator may update what it
// created without a separate team.manage grant. Every verified non-creator agent
// must satisfy the exact operation check, whether authority is in its identity
// grant or a separately verified operation proof.
func TestTeamUpdateCreatorOwnsWhileAlive(t *testing.T) { // [REQ:PM-TEAM-CREATOR-OWNERSHIP]
	withUnrelatedGrant := func(p identity.Principal) context.Context {
		p.Grant = &identity.OperationGrant{
			ID: "g", Approver: "owner",
			Operations: []string{"team.prepare"},
			Targets:    []string{"prompt-manager:team/other"},
			ExpiresAt:  time.Now().Add(time.Hour),
		}
		return identity.WithPrincipal(context.Background(), p)
	}

	t.Run("verified creator bypasses the operation gate", func(t *testing.T) {
		s, _, _ := preparationFixture(t, true) // team "draft" created WITH creator-run attribution
		ctx := withUnrelatedGrant(preparationActor())
		if err := s.Update(ctx, "draft", &Team{Mission: "updated by its creator"}); err != nil {
			t.Fatalf("creator updating its own team must not need a team.manage grant: %v", err)
		}
		got, err := s.Get(context.Background(), "draft")
		if err != nil || got.Mission != "updated by its creator" {
			t.Fatalf("creator update did not persist: mission=%q err=%v", got.Mission, err)
		}
	})

	t.Run("verified creator baseline bypasses without an embedded grant", func(t *testing.T) {
		s, ctx, _ := preparationFixture(t, true)
		if err := s.Update(ctx, "draft", &Team{Mission: "creator baseline update"}); err != nil {
			t.Fatalf("verified creator baseline should retain its owner bypass: %v", err)
		}
		got, err := s.Get(context.Background(), "draft")
		if err != nil || got.Mission != "creator baseline update" {
			t.Fatalf("creator baseline update did not persist: mission=%q err=%v", got.Mission, err)
		}
	})

	t.Run("a different agent is still blocked", func(t *testing.T) {
		s, _, _ := preparationFixture(t, true)
		other := preparationActor()
		other.Subject = "other-run"
		other.Owner = "other-owner"
		if err := s.Update(withUnrelatedGrant(other), "draft", &Team{Mission: "updated by an intruder"}); err == nil {
			t.Fatal("a non-creator agent must still satisfy the team.manage operation gate")
		}
	})

	t.Run("a different agent with exact team.manage target may update", func(t *testing.T) {
		s, _, _ := preparationFixture(t, true)
		other := preparationActor()
		other.Subject = "other-run"
		other.Owner = "other-owner"
		other.Grant = &identity.OperationGrant{
			ID: "team-manager", Approver: "other-owner", Operations: []string{"team.manage"},
			Targets: []string{"prompt-manager:team/draft"}, ExpiresAt: time.Now().Add(time.Hour),
		}
		if err := s.Update(identity.WithPrincipal(context.Background(), other), "draft", &Team{Mission: "updated by exact manager"}); err != nil {
			t.Fatalf("exact target team.manage grant should pass the existing owner gate: %v", err)
		}
		got, err := s.Get(context.Background(), "draft")
		if err != nil || got.Mission != "updated by exact manager" {
			t.Fatalf("authorized team.manage update did not persist: mission=%q err=%v", got.Mission, err)
		}
	})

	t.Run("legacy team with no creation proof is still gated", func(t *testing.T) {
		s, _, _ := preparationFixture(t, false) // created WITHOUT attribution -> CreationProof nil
		if err := s.Update(withUnrelatedGrant(preparationActor()), "draft", &Team{Mission: "updated"}); err == nil {
			t.Fatal("a team with no recorded creator must not be bypassable")
		}
	})
}

type teamOperationPairVerifier struct {
	grant identity.OperationGrant
}

func (teamOperationPairVerifier) Verify(string) (*cliutil.VerifyResult, error) {
	return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{
		RunID: "worker-run", Subject: "team-owner", Scopes: []string{"prompt-manager:read", "prompt-manager:write"}, ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}}, nil
}

func (v teamOperationPairVerifier) VerifyPair(_ context.Context, _, _ string) (*cliutil.VerifyPairResult, error) {
	grantJSON, err := json.Marshal(v.grant)
	if err != nil {
		return nil, err
	}
	expires := v.grant.ExpiresAt
	return &cliutil.VerifyPairResult{Valid: true, Operation: &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{
		RunID: "grant-run", Subject: "team-owner", Scopes: []string{"prompt-manager:write"}, ExpiresAt: expires.Unix(), Meta: map[string]string{"operation_grant": string(grantJSON)},
	}}}, nil
}

type teamOperationPrincipalVerifier struct{}

func (teamOperationPrincipalVerifier) Verify(context.Context, string) (identity.Principal, error) {
	return identity.Principal{Kind: identity.ActorAgent, Subject: "worker-run", Owner: "team-owner", Source: identity.SourceAgentProvenance, Verified: true, Scopes: []string{"prompt-manager:read", "prompt-manager:write"}}, nil
}

func TestTeamUpdateRequiresExactOperationForVerifiedNonCreator(t *testing.T) {
	t.Run("bare verified baseline is not enough", func(t *testing.T) {
		s, _, _ := preparationFixture(t, true)
		other := preparationActor()
		other.Subject = "other-run"
		other.Owner = "other-owner"
		err := s.Update(identity.WithPrincipal(context.Background(), other), "draft", &Team{Mission: "must not change"})
		if err == nil {
			t.Fatal("verified non-creator baseline without an operation grant updated the team")
		}
		got, getErr := s.Get(context.Background(), "draft")
		if getErr != nil || got.Mission == "must not change" {
			t.Fatalf("refused update changed the team: mission=%q err=%v", got.Mission, getErr)
		}
	})

	for _, tc := range []struct {
		name   string
		target string
		wantOK bool
	}{
		{name: "exact paired team.manage proof", target: "prompt-manager:team/draft", wantOK: true},
		{name: "unrelated paired proof", target: "prompt-manager:team/other", wantOK: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := preparationFixture(t, true)
			verifier := teamOperationPairVerifier{grant: identity.OperationGrant{
				ID: "paired-team-manager", Approver: "team-owner", Operations: []string{"team.manage"},
				Targets: []string{tc.target}, ExpiresAt: time.Now().Add(time.Hour),
			}}
			provider := authn.TokenProvider{ProviderSource: identity.SourceAgentProvenance, Extract: func(r *http.Request) string {
				return r.Header.Get(cliutil.HeaderAgentIdentityToken)
			}, Verifier: teamOperationPrincipalVerifier{}}
			var updateErr error
			mux := provenance.Middleware(verifier)(authn.Middleware(authn.Config{Providers: []authn.Provider{provider}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				updateErr = s.Update(r.Context(), "draft", &Team{Mission: "paired update"})
				w.WriteHeader(http.StatusNoContent)
			})))
			req := httptest.NewRequest(http.MethodPost, "/teams/draft", nil)
			req.Header.Set(cliutil.HeaderAgentIdentityToken, "verified-baseline")
			req.Header.Set(cliutil.HeaderAgentOperationToken, "verified-operation")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, req)
			if response.Code != http.StatusNoContent {
				t.Fatalf("middleware returned %d", response.Code)
			}
			if tc.wantOK && updateErr != nil {
				t.Fatalf("exact paired team.manage proof refused: %v", updateErr)
			}
			if !tc.wantOK && updateErr == nil {
				t.Fatal("unrelated paired proof authorized team update")
			}
			got, err := s.Get(context.Background(), "draft")
			if err != nil {
				t.Fatal(err)
			}
			if (got.Mission == "paired update") != tc.wantOK {
				t.Fatalf("paired proof effect mismatch: mission=%q wantAllowed=%v", got.Mission, tc.wantOK)
			}
		})
	}
}
