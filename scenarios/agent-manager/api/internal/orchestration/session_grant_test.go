package orchestration

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
)

// memStandingGrants is a real in-memory StandingGrantRepository for Phase-5
// session-grant tests (the shared fakeStandingGrants is intentionally no-op on
// Put/Revoke). It stores grants and honours state/expiry so mint + revoke +
// lookup behave like the durable store.
type memStandingGrants struct {
	grants map[string]*domain.StandingGrant
}

func newMemStandingGrants() *memStandingGrants {
	return &memStandingGrants{grants: map[string]*domain.StandingGrant{}}
}

func (m *memStandingGrants) Put(_ context.Context, g *domain.StandingGrant) error {
	cp := *g
	m.grants[g.ID] = &cp
	return nil
}

func (m *memStandingGrants) ActiveForSubject(_ context.Context, subject string, now time.Time) ([]*domain.StandingGrant, error) {
	var out []*domain.StandingGrant
	for _, g := range m.grants {
		if g.Subject == subject && g.Active(now) {
			out = append(out, g)
		}
	}
	return out, nil
}

func (m *memStandingGrants) Revoke(_ context.Context, id string, now time.Time) (bool, error) {
	g, ok := m.grants[id]
	if !ok || g.State != domain.StandingGrantApproved {
		return false, nil
	}
	g.State = domain.StandingGrantRevoked
	g.RevokedAt = &now
	return true, nil
}

func (m *memStandingGrants) List(_ context.Context, _ int) ([]*domain.StandingGrant, error) {
	var out []*domain.StandingGrant
	for _, g := range m.grants {
		out = append(out, g)
	}
	return out, nil
}

// A human-session grant is minted keyed to (human subject, session id), bounded,
// attributed to the human, and authorizes run.create for that human.
func TestCreateSessionGrantMintsBoundedHumanGrant(t *testing.T) {
	now := time.Now()
	store := newMemStandingGrants()
	o := &Orchestrator{clock: func() time.Time { return now }, standingGrants: store}

	g, err := o.CreateSessionGrant(context.Background(), "operator", "sess-1", time.Hour)
	if err != nil || g == nil {
		t.Fatalf("session grant mint failed: %v", err)
	}
	if g.Subject != "operator" || g.SessionID != "sess-1" || g.Grant.Approver != "operator" {
		t.Fatalf("session grant not keyed/attributed to the human+session: %+v", g)
	}
	if !g.Grant.Allows("run.create", standingGrantRunCreateTarget, now) {
		t.Fatal("session grant does not authorize run.create")
	}
	// Found by a subject lookup for that human.
	active, _ := store.ActiveForSubject(context.Background(), "operator", now)
	if len(active) != 1 || active[0].SessionID != "sess-1" {
		t.Fatalf("session grant not active for the human: %+v", active)
	}
}

// The lifetime is capped at MaxSessionGrantLifetime even if a longer TTL is
// requested — a forgotten-open session cannot confer indefinite authority.
func TestCreateSessionGrantCapsLifetime(t *testing.T) {
	now := time.Now()
	o := &Orchestrator{clock: func() time.Time { return now }, standingGrants: newMemStandingGrants()}
	g, err := o.CreateSessionGrant(context.Background(), "operator", "sess-1", 30*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if g.ExpiresAt.After(now.Add(MaxSessionGrantLifetime)) {
		t.Fatalf("session grant lifetime exceeded the cap: %v > %v", g.ExpiresAt, now.Add(MaxSessionGrantLifetime))
	}
}

// A requires-both-keys guard: missing subject or session id is refused.
func TestCreateSessionGrantRequiresSubjectAndSession(t *testing.T) {
	o := &Orchestrator{clock: time.Now, standingGrants: newMemStandingGrants()}
	if _, err := o.CreateSessionGrant(context.Background(), "", "sess-1", time.Hour); err == nil {
		t.Fatal("missing human subject should be refused")
	}
	if _, err := o.CreateSessionGrant(context.Background(), "operator", "", time.Hour); err == nil {
		t.Fatal("missing session id should be refused")
	}
}

// Revoke-on-close denies the grant on the next lookup, and only for that
// (subject, session) — a different session's grant for the same human survives.
func TestRevokeSessionGrantsIsSessionScoped(t *testing.T) {
	now := time.Now()
	store := newMemStandingGrants()
	o := &Orchestrator{clock: func() time.Time { return now }, standingGrants: store}
	if _, err := o.CreateSessionGrant(context.Background(), "operator", "sess-1", time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, err := o.CreateSessionGrant(context.Background(), "operator", "sess-2", time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := o.RevokeSessionGrants(context.Background(), "operator", "sess-1"); err != nil {
		t.Fatal(err)
	}
	active, _ := store.ActiveForSubject(context.Background(), "operator", now)
	if len(active) != 1 || active[0].SessionID != "sess-2" {
		t.Fatalf("revoke-on-close was not session-scoped: %+v", active)
	}
}

// A session grant for human A / session S does NOT authorize human B or a
// different session: ActiveForSubject is keyed to the subject, and the lookup
// the admission path uses (standingGrantOwner) returns the grant's own approver.
func TestSessionGrantDoesNotCrossHumanOrSession(t *testing.T) {
	now := time.Now()
	store := newMemStandingGrants()
	o := &Orchestrator{clock: func() time.Time { return now }, standingGrants: store}
	if _, err := o.CreateSessionGrant(context.Background(), "operator-A", "sess-A", time.Hour); err != nil {
		t.Fatal(err)
	}
	// human B has no grant.
	ownerB, _ := o.standingGrantOwner(context.Background(), "operator-B")
	if ownerB != "" {
		t.Fatalf("a session grant for operator-A authorized operator-B: %q", ownerB)
	}
	// human A resolves to their own grant.
	ownerA, _ := o.standingGrantOwner(context.Background(), "operator-A")
	if ownerA != "operator-A" {
		t.Fatalf("operator-A's session grant did not resolve to operator-A: %q", ownerA)
	}
}

// humanSpawnOwner gates the X-Human-Owner-Subject header strictly on a
// verified-human admission (CreateCaller.Kind == "human"); a service/agent/effort
// or caller-less run yields empty so a non-human subject is never forwarded.
func TestHumanSpawnOwnerGatesOnHumanCallerKind(t *testing.T) {
	humanRun := &domain.Run{OwnerSubject: "operator", ResolvedConfig: &domain.RunConfig{Admission: &domain.RunAdmission{CreateCaller: &domain.CreateRunCaller{Kind: "human", Subject: "operator"}}}}
	if got := humanSpawnOwner(humanRun); got != "operator" {
		t.Fatalf("human spawn should yield the operator subject, got %q", got)
	}
	for _, kind := range []string{"service-standing", "run", "effort-serial-native-owner", ""} {
		r := &domain.Run{OwnerSubject: "someone", ResolvedConfig: &domain.RunConfig{Admission: &domain.RunAdmission{CreateCaller: &domain.CreateRunCaller{Kind: kind, Subject: "someone"}}}}
		if got := humanSpawnOwner(r); got != "" {
			t.Fatalf("non-human caller kind %q must not forward a human owner, got %q", kind, got)
		}
	}
	// No admission / no caller → empty.
	if got := humanSpawnOwner(&domain.Run{OwnerSubject: "x"}); got != "" {
		t.Fatalf("run without admission must not forward a human owner, got %q", got)
	}
}
