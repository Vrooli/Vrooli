package store

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
	"prompt-manager/internal/teamconfig"
)

func preparationActor() identity.Principal {
	return identity.Principal{Kind: identity.ActorAgent, Subject: "creator-run", Owner: "owner", Source: identity.SourceAgentProvenance, Verified: true, Scopes: []string{"prompt-manager:write"}}
}

func preparationFixture(t *testing.T, attributed bool) (*FileTeamStore, context.Context, *teamconfig.FiniteLeader) {
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
	cfg := finiteConfig()
	cfg.Enabled = false
	if err = s.SetHeartbeatConfig(ctx, "draft", "leader", cfg); err != nil {
		t.Fatal(err)
	}
	next := *cfg.FiniteLeader
	next.AcceptedRevision = "revision:8"
	next.SourceRefs = []string{"owner:amended-source"}
	return s, ctx, &next
}

func TestFinitePreparationCreatorAmendsWithoutActivationAndReplays(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	s, ctx, next := preparationFixture(t, true)
	if err := s.AmendFinitePreparation(ctx, "draft", "leader", "revision:7", next, "owner:evidence"); err != nil {
		t.Fatal(err)
	}
	if err := s.AmendFinitePreparation(ctx, "draft", "leader", "revision:7", next, "owner:evidence"); err != nil {
		t.Fatal("lost receipt replay:", err)
	}
	cfg, _ := s.GetHeartbeatConfig(ctx, "draft", "leader")
	state, err := s.ReadFiniteLeader(ctx, "draft", "leader")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled || state.TaskStarted || state.DispatchStarted || state.ID != "" || len(cfg.PreparationAmendments) != 1 || !reflect.DeepEqual(cfg.FiniteLeader, next) {
		t.Fatalf("unexpected admission/receipt: %+v %+v", cfg, state)
	}
	if cfg.PreparationAmendments[0].Actor.Subject != "creator-run" || cfg.PreparationAmendments[0].Previous.AcceptedRevision != "revision:7" {
		t.Fatal("lost actor or original")
	}
}

func TestFinitePreparationRejectsWrongAuthorityAndLifecycle(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	for _, tc := range []string{"anonymous", "unverified", "other-agent", "same-owner-other-run", "wrong-scope", "expired-principal", "legacy", "wrong-target-grant", "expired-grant", "enabled-team", "enabled-heartbeat", "reserved", "dispatch-uncertain", "retired", "stale", "effort", "policy"} {
		t.Run(tc, func(t *testing.T) {
			s, ctx, next := preparationFixture(t, tc != "legacy")
			p := preparationActor()
			expected := "revision:7"
			switch tc {
			case "anonymous":
				ctx = context.Background()
			case "unverified":
				p.Verified = false
				ctx = identity.WithPrincipal(ctx, p)
			case "other-agent":
				p.Subject = "other"
				p.Owner = "other-owner"
				ctx = identity.WithPrincipal(ctx, p)
			case "same-owner-other-run":
				p.Subject = "other"
				ctx = identity.WithPrincipal(ctx, p)
			case "wrong-scope":
				p.Scopes = []string{"prompt-manager:read"}
				ctx = identity.WithPrincipal(ctx, p)
			case "expired-principal":
				p.ExpiresAt = time.Now().Add(-time.Minute)
				ctx = identity.WithPrincipal(ctx, p)
			case "wrong-target-grant", "expired-grant":
				p.Grant = &identity.OperationGrant{ID: "g", Approver: "owner", Operations: []string{"team.prepare"}, Targets: []string{"prompt-manager:team/other"}, ExpiresAt: time.Now().Add(time.Hour)}
				if tc == "expired-grant" {
					p.Grant.Targets = []string{"prompt-manager:team/draft"}
					p.Grant.ExpiresAt = time.Now().Add(-time.Minute)
				}
				ctx = identity.WithPrincipal(ctx, p)
			case "enabled-team":
				if err := s.Update(ctx, "draft", &Team{Enabled: true, EnabledSet: true}); err != nil {
					t.Fatal(err)
				}
			case "enabled-heartbeat", "retired":
				cfg, _ := s.GetHeartbeatConfig(ctx, "draft", "leader")
				if tc == "enabled-heartbeat" {
					cfg.Enabled = true
				} else {
					cfg.FiniteLeader.Retired = true
				}
				if err := s.SetHeartbeatConfig(ctx, "draft", "leader", cfg); err != nil {
					t.Fatal(err)
				}
			case "reserved", "dispatch-uncertain":
				if err := s.WithFiniteLeader(ctx, "draft", "leader", func(_ *HeartbeatConfig, st *FiniteLeaderState, save func() error) error {
					st.ID = "reserved"
					if tc == "dispatch-uncertain" {
						st.TaskID = "task"
						st.TaskStarted = true
						st.DispatchStarted = true
					}
					return save()
				}); err != nil {
					t.Fatal(err)
				}
			case "stale":
				expected = "revision:6"
			case "effort":
				next.EffortRef = "different-effort"
			case "policy":
				next.KeepAlive = true
			}
			before, _ := s.GetHeartbeatConfig(context.Background(), "draft", "leader")
			b, _ := json.Marshal(before)
			if err := s.AmendFinitePreparation(ctx, "draft", "leader", expected, next, "owner:evidence"); err == nil {
				t.Fatal("unauthorized or unsafe mutation admitted")
			}
			after, _ := s.GetHeartbeatConfig(context.Background(), "draft", "leader")
			a, _ := json.Marshal(after)
			if string(a) != string(b) {
				t.Fatal("refusal wrote configuration")
			}
		})
	}
}

func TestFinitePreparationLegacyNeedsExactApprovedGrant(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	s, ctx, next := preparationFixture(t, false)
	p := preparationActor()
	p.Grant = &identity.OperationGrant{ID: "approved", Approver: "owner", Operations: []string{"team.prepare"}, Targets: []string{"prompt-manager:team/draft"}, ExpiresAt: time.Now().Add(time.Hour)}
	ctx = identity.WithPrincipal(ctx, p)
	if err := s.AmendFinitePreparation(ctx, "draft", "leader", "revision:7", next, "owner:evidence"); err != nil {
		t.Fatal(err)
	}
	team, _ := s.Get(ctx, "draft")
	if team.CreationProof != nil {
		t.Fatal("retroactive creator invented")
	}
}

func TestFinitePreparationAtomicSnapshotRecoversIdleState(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	s, ctx, next := preparationFixture(t, true)
	if err := s.WithFiniteLeader(ctx, "draft", "leader", func(_ *HeartbeatConfig, _ *FiniteLeaderState, save func() error) error { return save() }); err != nil {
		t.Fatal(err)
	}
	path := s.finiteLeaderPath(next.EffortRef)
	before, _ := os.ReadFile(path)
	if err := s.AmendFinitePreparation(ctx, "draft", "leader", "revision:7", next, "owner:evidence"); err != nil {
		t.Fatal(err)
	}
	restarted := NewFileTeamStore(s.configRoot, s.runtimeDataRoot, s.relationStore)
	state, err := restarted.ReadFiniteLeader(ctx, "draft", "leader")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(&state.Binding, next) {
		t.Fatal("restart did not recover committed amendment")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("read mutated historical state")
	}
	if err = restarted.WithFiniteLeader(ctx, "draft", "leader", func(_ *HeartbeatConfig, st *FiniteLeaderState, save func() error) error {
		st.ID = "new-reservation"
		return save()
	}); err != nil {
		t.Fatal(err)
	}
	state, err = restarted.ReadFiniteLeader(ctx, "draft", "leader")
	if err != nil || state.ID != "new-reservation" || !reflect.DeepEqual(&state.Binding, next) {
		t.Fatal("first reservation lost revised binding", err)
	}
}

func TestFinitePreparationConcurrentCompareAndSwap(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	s, ctx, next := preparationFixture(t, true)
	out := make(chan error, 2)
	var wg sync.WaitGroup
	for _, rev := range []string{"revision:8", "revision:9"} {
		wg.Add(1)
		go func(rev string) {
			defer wg.Done()
			n := *next
			n.AcceptedRevision = rev
			out <- s.AmendFinitePreparation(ctx, "draft", "leader", "revision:7", &n, "owner:evidence")
		}(rev)
	}
	wg.Wait()
	close(out)
	passed := 0
	for err := range out {
		if err == nil {
			passed++
		}
	}
	if passed != 1 {
		t.Fatalf("CAS admitted %d writers", passed)
	}
}

func TestTeamCreationProofCannotBeSuppliedOrReplaced(t *testing.T) { // [REQ:PM-TEAM-PREPARATION]
	s, ctx, _ := preparationFixture(t, true)
	team, _ := s.Get(ctx, "draft")
	if team.CreationProof == nil || team.CreationProof.Subject != "creator-run" {
		t.Fatal("verified creator not recorded")
	}
	if err := s.Update(ctx, "draft", &Team{CreationProof: &TeamCreationProof{Subject: "forged"}, Mission: "new mission"}); err != nil {
		t.Fatal(err)
	}
	team, _ = s.Get(ctx, "draft")
	if team.CreationProof.Subject != "creator-run" {
		t.Fatal("partial update replaced proof")
	}
	team.ID = "legacy-forged"
	if err := s.Create(context.Background(), team); err != nil {
		t.Fatal(err)
	}
	team, _ = s.Get(ctx, "legacy-forged")
	if team.CreationProof != nil {
		t.Fatal("input proof used as authority")
	}
}
