package effortauthority

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"errors"
	_ "modernc.org/sqlite"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type fixtureOwners struct {
	state       OwnerState
	ceiling     []string
	unavailable bool
	expired     bool
}

func (o *fixtureOwners) CurrentEffortOwner(context.Context, string) (OwnerState, error) {
	if o.unavailable {
		return OwnerState{}, errors.New("unavailable")
	}
	return o.state, nil
}
func (o *fixtureOwners) ApproveEffortOwner(context.Context, string) (OwnerState, error) {
	if o.expired {
		return OwnerState{}, ErrRefused
	}
	s := o.state
	s.Ceiling = o.ceiling
	return s, nil
}

type fixtureScope struct{ invalid bool }

func (s *fixtureScope) CheckEffortScope(context.Context, Policy) error {
	if s.invalid {
		return ErrRefused
	}
	return nil
}
func authorityFixture(t *testing.T) (*Authority, *fixtureOwners, *fixtureScope, Policy, ed25519.PrivateKey, *time.Time) {
	t.Helper()
	db, e := sql.Open("sqlite", filepath.Join(t.TempDir(), "authority.db"))
	if e != nil {
		t.Fatal(e)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	store := &SQLStore{DB: db}
	if e = store.Ensure(context.Background()); e != nil {
		t.Fatal(e)
	}
	pub, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	now := time.Now().UTC().Truncate(time.Second)
	owner := &fixtureOwners{state: OwnerState{Subject: "fixture-owner", Enabled: true, Scopes: []string{"agent-manager:*"}}, ceiling: []string{"agent-manager:*"}}
	scope := &fixtureScope{}
	p := Policy{ID: "fixture-grant", Owner: "fixture-owner", Client: "fixture-client", ClientKey: pub, Epoch: 1, Repository: t.TempDir(), Effort: "fixture-effort", Revision: "revision-1", ContentDigest: Digest("approved contents"), Team: "fixture-team", TeamDigest: Digest("team"), BindingDigest: Digest("binding"), Members: []string{"leader", "worker"}, Profiles: map[string]string{"bounded": Digest("profile")}, Scopes: []string{"agent-manager:write", "agent-manager:orchestrate"}, Effects: []string{"run.create", "run.child", "run.recover"}, NotBefore: now, Deadline: now.Add(8 * time.Hour), MaxStarts: 4, MaxConcurrent: 4, MaxTurns: 10, MaxToolCalls: 20, MaxRunSeconds: 3600, TotalTurns: 40, TotalToolCalls: 80, TotalRunSeconds: 14400, AllowChildren: true, AllowRecovery: true, SurviveLogout: true}
	a := &Authority{Store: store, Owners: owner, Scope: scope, Approver: owner, Now: func() time.Time { return now }}
	if e = a.Approve(context.Background(), "fixture-human-proof", p); e != nil {
		t.Fatal(e)
	}
	return a, owner, scope, p, key, &now
}
func nativeIntent(p Policy, key string) Intent {
	return Intent{Audience: Audience, Method: "POST", Endpoint: "/api/v1/runs", PolicyID: p.ID, Epoch: p.Epoch, Client: p.Client, Repository: p.Repository, Effort: p.Effort, Revision: p.Revision, ContentDigest: p.ContentDigest, Team: p.Team, Member: "leader", Profile: "bounded", ProfileDigest: p.Profiles["bounded"], Effect: "run.create", IdempotencyKey: key, InputDigest: Digest("exact native input"), Turns: 10, ToolCalls: 20, RunSeconds: 3600}
}
func signedFixture(t *testing.T, a *Authority, p Policy, key ed25519.PrivateKey, i Intent, nonce string) Proof {
	t.Helper()
	proof, e := Sign(Proof{Intent: i, Nonce: nonce, IssuedAt: a.now()}, key)
	if e != nil {
		t.Fatal(e)
	}
	return proof
}
func unchangedRecord(t *testing.T, a *Authority, id, before string) {
	t.Helper()
	r, e := a.Store.Get(context.Background(), id)
	if e != nil || Digest(r) != before {
		t.Fatalf("rejected request changed ledger: %v", e)
	}
}
func TestFiniteAuthorityOvernightIndependentOfExpiredHumanAndBrowser(t *testing.T) {
	a, owner, _, p, key, now := authorityFixture(t)
	owner.expired = true
	*now = now.Add(5 * time.Hour)
	proof := signedFixture(t, a, p, key, nativeIntent(p, "overnight-start"), "new-valid-nonce-after-browser-close")
	b, e := a.CheckProof(context.Background(), proof)
	if e != nil || b.Deadline != p.Deadline {
		t.Fatalf("overnight authority: %v", e)
	}
	if _, _, e = a.Reserve(context.Background(), b, proof.Intent, &proof); e != nil {
		t.Fatal(e)
	}
	child := nativeIntent(p, "late-child")
	child.Effect = "run.child"
	child.ParentRunID = "retained-exact-parent"
	if _, _, e = a.Reserve(context.Background(), b, child, nil); e != nil {
		t.Fatal(e)
	}
	*now = p.Deadline
	if _, e = a.CheckBinding(context.Background(), b); e == nil {
		t.Fatal("expiry equality accepted")
	}
}
func TestFiniteAuthorityRefusalHasZeroLedgerEffects(t *testing.T) {
	for _, name := range []string{"signature", "client", "purpose", "endpoint", "owner-grant", "disabled", "unavailable", "scope-change", "revision", "content", "repo", "member", "profile", "epoch", "effect", "budget", "nonce", "future-proof", "old-proof", "expired", "revoked", "wrong-binding"} {
		t.Run(name, func(t *testing.T) {
			a, owner, scope, p, key, now := authorityFixture(t)
			i := nativeIntent(p, "refuse")
			proof := signedFixture(t, a, p, key, i, "valid-disposable-nonce")
			switch name {
			case "signature":
				proof.Signature[0] ^= 1
			case "client":
				proof.Intent.Client = "foreign"
			case "purpose":
				proof.Intent.Audience = "human"
			case "endpoint":
				proof.Intent.Endpoint = "/grants"
			case "owner-grant":
				owner.state.Scopes = []string{"agent-manager:read"}
			case "disabled":
				owner.state.Enabled = false
			case "unavailable":
				owner.unavailable = true
			case "scope-change":
				scope.invalid = true
			case "revision":
				proof.Intent.Revision = "changed"
			case "content":
				proof.Intent.ContentDigest = Digest("changed")
			case "repo":
				proof.Intent.Repository = filepath.Dir(p.Repository)
			case "member":
				proof.Intent.Member = "not-enrolled"
			case "profile":
				proof.Intent.ProfileDigest = Digest("changed")
			case "epoch":
				proof.Intent.Epoch++
			case "effect":
				proof.Intent.Effect = "scope.grant"
			case "budget":
				proof.Intent.Turns++
			case "nonce":
				proof.Nonce = "short"
			case "future-proof":
				proof.IssuedAt = now.Add(time.Second)
			case "old-proof":
				proof.IssuedAt = now.Add(-3 * time.Minute)
			case "expired":
				*now = p.Deadline
			case "revoked":
				if e := a.Revoke(context.Background(), "fixture-human", p.ID); e != nil {
					t.Fatal(e)
				}
			case "wrong-binding":
				owner.state.Subject = "other"
			}
			r, _ := a.Store.Get(context.Background(), p.ID)
			before := Digest(r)
			// Re-sign changed data so every domain refusal is independent of bad crypto.
			if name != "signature" {
				var e error
				proof, e = Sign(proof, key)
				if e != nil {
					t.Fatal(e)
				}
			}
			if _, e := a.CheckProof(context.Background(), proof); e == nil {
				t.Fatal("accepted invalid proof")
			}
			unchangedRecord(t, a, p.ID, before)
		})
	}
}
func TestFiniteApprovalCannotExpandNarrowHumanCeiling(t *testing.T) {
	a, owner, _, p, _, _ := authorityFixture(t)
	owner.ceiling = []string{"agent-manager:read"}
	p.ID = "narrowed-proof-attempt"
	if e := a.Approve(context.Background(), "narrow-token", p); e == nil {
		t.Fatal("current broad account grants expanded offered proof")
	}
	if _, e := a.Store.Get(context.Background(), p.ID); e == nil {
		t.Fatal("rejected grant persisted")
	}
}
func TestFiniteReplayUncertaintyAndAggregateBound(t *testing.T) {
	a, _, _, p, key, _ := authorityFixture(t)
	proof := signedFixture(t, a, p, key, nativeIntent(p, "same-intent"), "original-nonce-12345")
	b, e := a.CheckProof(context.Background(), proof)
	if e != nil {
		t.Fatal(e)
	}
	if _, replay, e := a.Reserve(context.Background(), b, proof.Intent, &proof); e != nil || replay {
		t.Fatal(e)
	}
	proof.Nonce = "new-retry-nonce-12345"
	proof, _ = Sign(proof, key)
	if _, replay, e := a.Reserve(context.Background(), b, proof.Intent, &proof); e != nil || !replay {
		t.Fatal("same-key retry did not reconcile", e)
	}
	r, _ := a.Store.Get(context.Background(), p.ID)
	before := Digest(r)
	proof.Intent.InputDigest = Digest("changed payload")
	proof, _ = Sign(proof, key)
	if _, _, e = a.Reserve(context.Background(), b, proof.Intent, &proof); e == nil {
		t.Fatal("changed same-key accepted")
	}
	unchangedRecord(t, a, p.ID, before)
	// Only exact accepted native terminal evidence can release concurrency;
	// unknown usage still conservatively consumes total allowance.
	if e = a.Settle(context.Background(), b, "same-intent", "unaccepted-run"); e == nil {
		t.Fatal("unknown run released budget")
	}
	if e = a.Accept(context.Background(), b, "same-intent", "native-run"); e != nil {
		t.Fatal(e)
	}
	if e = a.Settle(context.Background(), b, "same-intent", "native-run"); e != nil {
		t.Fatal(e)
	}
	for n := 0; n < 3; n++ {
		i := nativeIntent(p, string(rune('a'+n)))
		if _, _, e = a.Reserve(context.Background(), b, i, nil); e != nil {
			t.Fatal(e)
		}
	}
	r, _ = a.Store.Get(context.Background(), p.ID)
	before = Digest(r)
	if _, _, e = a.Reserve(context.Background(), b, nativeIntent(p, "over-budget"), nil); e == nil {
		t.Fatal("extra start accepted")
	}
	unchangedRecord(t, a, p.ID, before)
}
func TestFiniteConcurrentSameIntentOneReservationAndNoSecrets(t *testing.T) {
	a, _, _, p, key, _ := authorityFixture(t)
	proof := signedFixture(t, a, p, key, nativeIntent(p, "concurrent"), "same-disposable-nonce")
	b, _ := a.CheckProof(context.Background(), proof)
	var wg sync.WaitGroup
	for n := 0; n < 12; n++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _, _ = a.Reserve(context.Background(), b, proof.Intent, &proof) }()
	}
	wg.Wait()
	r, e := a.Store.Get(context.Background(), p.ID)
	if e != nil || len(r.Reservations) != 1 {
		t.Fatalf("concurrent duplicate count=%d %v", len(r.Reservations), e)
	}
	if len(r.Policy.ClientKey) != ed25519.PublicKeySize {
		t.Fatal("unexpected key custody")
	}
	// New authority instance reads original policy/reservations; no auto renewal.
	fresh := &Authority{Store: a.Store, Owners: a.Owners, Scope: a.Scope, Now: a.Now}
	if p2, e := fresh.CheckBinding(context.Background(), b); e != nil || p2.Deadline != p.Deadline {
		t.Fatal("restart renewed/forgot authority", e)
	}
}
func TestFiniteBudgetExtremeValuesRefused(t *testing.T) {
	a, _, _, p, _, _ := authorityFixture(t)
	p.ID = "overflow-attempt"
	p.MaxTurns = int(^uint(0) >> 1)
	p.TotalTurns = int64(^uint64(0) >> 1)
	if e := a.Approve(context.Background(), "fixture", p); e == nil {
		t.Fatal("unbounded arithmetic policy accepted")
	}
}

func TestPreparationChargesBeforePlannerEffectsAndBindsNativeOnce(t *testing.T) {
	a, _, _, p, key, _ := authorityFixture(t)
	i := nativeIntent(p, "finite-leader-prepared")
	i.Endpoint = "/api/v1/teams/fixture-team/members/leader/finite/start"
	i.InputDigest = Digest("protected finite leader inputs")
	proof := signedFixture(t, a, p, key, i, "disposable-preparation-nonce")
	b, _ := a.CheckProof(context.Background(), proof)
	prepared, replay, e := a.Prepare(context.Background(), proof, nil)
	if e != nil || replay || prepared.NativeBound || prepared.Preparation == nil {
		t.Fatalf("prepare: %+v %v", prepared, e)
	}
	if e = a.Accept(context.Background(), b, i.IdempotencyKey, "premature"); e == nil {
		t.Fatal("unbound preparation accepted as native receipt")
	}
	actual := nativeIntent(p, i.IdempotencyKey)
	signed := signedFixture(t, a, p, key, actual, "disposable-native-bind-nonce")
	bound, replay, e := a.Reserve(context.Background(), b, actual, &signed)
	if e != nil || replay || !bound.NativeBound || bound.Preparation == nil {
		t.Fatalf("native bind: %+v %v", bound, e)
	}
	if _, replay, e = a.Reserve(context.Background(), b, actual, &signed); e != nil || !replay {
		t.Fatal("pending native outcome not reconciled", e)
	}
	changed := actual
	changed.InputDigest = Digest("different native task")
	signed = signedFixture(t, a, p, key, changed, "disposable-changed-nonce")
	r, _ := a.Store.Get(context.Background(), p.ID)
	before := Digest(r)
	if _, _, e = a.Reserve(context.Background(), b, changed, &signed); e == nil {
		t.Fatal("rebound task")
	}
	unchangedRecord(t, a, p.ID, before)
	// Preparation itself occupies all counts. No task/PM effect is needed to
	// consume capacity, and a failure before native acceptance never frees it.
	for n := 0; n < 3; n++ {
		i.IdempotencyKey = string(rune('a' + n))
		proof = signedFixture(t, a, p, key, i, "preparation-nonce-"+i.IdempotencyKey)
		if _, _, e = a.Prepare(context.Background(), proof, nil); e != nil {
			t.Fatal(e)
		}
	}
	r, _ = a.Store.Get(context.Background(), p.ID)
	before = Digest(r)
	i.IdempotencyKey = "refused-before-effects"
	proof = signedFixture(t, a, p, key, i, "preparation-exhausted-nonce")
	if _, _, e = a.Prepare(context.Background(), proof, nil); e == nil {
		t.Fatal("exhausted preparation accepted")
	}
	unchangedRecord(t, a, p.ID, before)
}
func TestExternalIngressAndPreparationOneTransaction(t *testing.T) {
	a, _, _, p, key, _ := authorityFixture(t)
	i := nativeIntent(p, "finite-leader-external")
	i.Endpoint = "/api/v1/teams/fixture-team/members/leader/finite/start"
	i.InputDigest = Digest("external team/member")
	external := signedFixture(t, a, p, key, i, "external-start-nonce-123")
	i.InputDigest = Digest("owner-prepared leader")
	prepared := signedFixture(t, a, p, key, i, "owner-preparation-nonce-123")
	if _, _, e := a.Prepare(context.Background(), prepared, &external); e != nil {
		t.Fatal(e)
	}
	if _, replay, e := a.Prepare(context.Background(), prepared, &external); e != nil || !replay {
		t.Fatal("same ingress failed reconciliation", e)
	}
	r, _ := a.Store.Get(context.Background(), p.ID)
	if len(r.Ingress) != 1 || len(r.Reservations) != 1 {
		t.Fatal("split ingress transaction")
	}
	// Exhaust all remaining capacity, then verify neither ingress nor nonce is
	// persisted for an exhausted external start.
	for n := 0; n < 3; n++ {
		i.IdempotencyKey = string(rune('a' + n))
		proof := signedFixture(t, a, p, key, i, "budget-fill-nonce-"+i.IdempotencyKey)
		if _, _, e := a.Prepare(context.Background(), proof, nil); e != nil {
			t.Fatal(e)
		}
	}
	r, _ = a.Store.Get(context.Background(), p.ID)
	before := Digest(r)
	i.IdempotencyKey = "finite-leader-refused"
	external = signedFixture(t, a, p, key, i, "external-refused-nonce-123")
	i.InputDigest = Digest("prepared refused")
	prepared = signedFixture(t, a, p, key, i, "prepared-refused-nonce-123")
	if _, _, e := a.Prepare(context.Background(), prepared, &external); e == nil {
		t.Fatal("exhausted ingress admitted")
	}
	unchangedRecord(t, a, p.ID, before)
}
