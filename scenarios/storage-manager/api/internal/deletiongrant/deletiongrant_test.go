package deletiongrant

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/operatordecision"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	handle := dbtest.NewSQLite(t)
	if _, err := handle.ExecContext(context.Background(), schemaSQL); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	return NewStore(handle)
}

// fakeRaiser captures what Request would raise, and implements DecisionRaiser.
type fakeRaiser struct {
	ask   operatordecision.PendingAsk
	facts operatordecision.Facts
	store *Store
}

func (f *fakeRaiser) Raise(ctx context.Context, ask operatordecision.PendingAsk, facts operatordecision.Facts) (bool, error) {
	f.ask, f.facts = ask, facts
	ask.State = operatordecision.PendingStatePending
	ask.RaisedAt = time.Unix(0, 0).UTC()
	ask.ExpiresAt = time.Unix(0, 0).UTC().Add(24 * time.Hour)
	if err := f.store.Put(ctx, &ask); err != nil {
		return false, err
	}
	return true, nil
}

func TestRequestRaisesValidFacts(t *testing.T) {
	store := newStore(t)
	raiser := &fakeRaiser{store: store}
	id, raised, err := Request(context.Background(), raiser, []string{"/home/u/.cache/x", "/home/u/.cache/x"}, "agent:claude", "", "", "cleanup", 24*time.Hour)
	if err != nil || !raised {
		t.Fatalf("Request raised=%v err=%v", raised, err)
	}
	if id == "" {
		t.Fatal("expected a correlation id")
	}
	if err := raiser.facts.Validate(); err != nil {
		t.Fatalf("facts invalid: %v", err)
	}
	if raiser.facts.Recommended != "deny" || raiser.facts.Reversible {
		t.Fatalf("deletion must recommend deny and be irreversible: %+v", raiser.facts)
	}
}

func TestRequestStampsTieBackCorrelation(t *testing.T) {
	store := newStore(t)
	raiser := &fakeRaiser{store: store}
	id, _, err := Request(context.Background(), raiser, []string{"/home/u/.cache/x"}, "run-123", "run-123", "wc-sess-1", "cleanup", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	// raised_by_run_id must carry the requester's run id so agent-manager can
	// deliver the answer back to it (parked or running).
	if got := raiser.facts.Correlation[operatordecision.CorrelationRaisedByRunID]; got != "run-123" {
		t.Fatalf("raised_by_run_id must be the requester run id, got %q", got)
	}
	// The stable web-console session id must be carried as the fallback handle.
	if got := raiser.facts.Correlation[operatordecision.CorrelationWebConsoleSessionID]; got != "wc-sess-1" {
		t.Fatalf("web_console_session_id must be carried, got %q", got)
	}
	// The optional park companion must equal the decision id.
	if got := raiser.facts.Correlation[operatordecision.CorrelationAgentAwaitKey]; got != id {
		t.Fatalf("agent_await_key must equal the decision id %q, got %q", id, got)
	}
}

func TestRequestOmitsRunIDForNonAgentCaller(t *testing.T) {
	store := newStore(t)
	raiser := &fakeRaiser{store: store}
	// A human/browser caller has no run id: no tie-back key is stamped.
	if _, _, err := Request(context.Background(), raiser, []string{"/home/u/.cache/x"}, "operator@host", "", "", "cleanup", time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, ok := raiser.facts.Correlation[operatordecision.CorrelationRaisedByRunID]; ok {
		t.Fatal("raised_by_run_id must be absent when the caller has no run id")
	}
}

func TestRequestCorrelationStableRegardlessOfOrder(t *testing.T) {
	store := newStore(t)
	a := &fakeRaiser{store: store}
	b := &fakeRaiser{store: store}
	idA, _, _ := Request(context.Background(), a, []string{"/a", "/b"}, "agent", "", "", "r", time.Hour)
	idB, _, _ := Request(context.Background(), b, []string{"/b", "/a", "/a"}, "agent", "", "", "r", time.Hour)
	if idA != idB {
		t.Fatalf("correlation id must be stable per path set: %q vs %q", idA, idB)
	}
}

func TestRequestRejectsEmptyOrRelative(t *testing.T) {
	store := newStore(t)
	raiser := &fakeRaiser{store: store}
	if _, _, err := Request(context.Background(), raiser, []string{"relative/path", ""}, "agent", "", "", "r", time.Hour); err == nil {
		t.Fatal("expected error when no absolute path is given")
	}
}

func TestApproveProducesActiveGrantThenExpires(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	raiser := &fakeRaiser{store: store}
	id, _, err := Request(ctx, raiser, []string{"/home/u/.cache/x"}, "agent", "", "", "cleanup", 2*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	resolver := NewResolver(store).WithClock(func() time.Time { return now })

	// Before resolution: no active grant.
	if rules, _ := store.ActiveGrantRoots(ctx, now); len(rules) != 0 {
		t.Fatalf("pending ask must not grant anything, got %+v", rules)
	}

	handled, err := resolver.Apply(ctx, operatordecision.DecisionResolution{Correlation: map[string]string{correlationAskID: id}, Answer: "approve", AnsweredBy: "operator@host"})
	if err != nil || !handled {
		t.Fatalf("apply handled=%v err=%v", handled, err)
	}
	rules, err := store.ActiveGrantRoots(ctx, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].Root != "/home/u/.cache/x" || rules[0].Approver != "operator@host" {
		t.Fatalf("expected one active grant for the path approved by operator, got %+v", rules)
	}
	// After the 2h TTL lapses, the grant is gone.
	if later, _ := store.ActiveGrantRoots(ctx, now.Add(3*time.Hour)); len(later) != 0 {
		t.Fatalf("expired grant must not project, got %+v", later)
	}
}

func TestDenyProducesNoGrantAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	raiser := &fakeRaiser{store: store}
	id, _, _ := Request(ctx, raiser, []string{"/home/u/keep"}, "agent", "", "", "", time.Hour)
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	resolver := NewResolver(store).WithClock(func() time.Time { return now })

	handled, err := resolver.Apply(ctx, operatordecision.DecisionResolution{Correlation: map[string]string{correlationAskID: id}, Answer: "deny", AnsweredBy: "operator@host"})
	if err != nil || !handled {
		t.Fatalf("apply handled=%v err=%v", handled, err)
	}
	if rules, _ := store.ActiveGrantRoots(ctx, now); len(rules) != 0 {
		t.Fatalf("deny must grant nothing, got %+v", rules)
	}
	// Replay of the same resolution is a benign no-op.
	handled, err = resolver.Apply(ctx, operatordecision.DecisionResolution{Correlation: map[string]string{correlationAskID: id}, Answer: "approve", AnsweredBy: "someone-else"})
	if err != nil || !handled {
		t.Fatalf("replay handled=%v err=%v", handled, err)
	}
	if rules, _ := store.ActiveGrantRoots(ctx, now); len(rules) != 0 {
		t.Fatal("an already-decided ask must not be re-decided by a replay")
	}
}

func TestOnApproveFiresOnlyForApproval(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

	// Approve → callback fires once.
	store := newStore(t)
	raiser := &fakeRaiser{store: store}
	id, _, _ := Request(ctx, raiser, []string{"/home/u/a"}, "agent", "", "", "", time.Hour)
	fired := 0
	resolver := NewResolver(store).WithClock(func() time.Time { return now }).WithOnApprove(func() { fired++ })
	resolver.Apply(ctx, operatordecision.DecisionResolution{Correlation: map[string]string{correlationAskID: id}, Answer: "approve", AnsweredBy: "op"})
	if fired != 1 {
		t.Fatalf("onApprove should fire once on approval, fired=%d", fired)
	}

	// Deny → callback does not fire.
	store2 := newStore(t)
	raiser2 := &fakeRaiser{store: store2}
	id2, _, _ := Request(ctx, raiser2, []string{"/home/u/b"}, "agent", "", "", "", time.Hour)
	fired2 := 0
	resolver2 := NewResolver(store2).WithClock(func() time.Time { return now }).WithOnApprove(func() { fired2++ })
	resolver2.Apply(ctx, operatordecision.DecisionResolution{Correlation: map[string]string{correlationAskID: id2}, Answer: "deny", AnsweredBy: "op"})
	if fired2 != 0 {
		t.Fatalf("onApprove must not fire on denial, fired=%d", fired2)
	}
}

func TestUnknownAskIsBenignNoOp(t *testing.T) {
	store := newStore(t)
	resolver := NewResolver(store)
	handled, err := resolver.Apply(context.Background(), operatordecision.DecisionResolution{Correlation: map[string]string{correlationAskID: "missing"}, Answer: "approve"})
	if err != nil || !handled {
		t.Fatalf("unknown ask must be a benign no-op, got handled=%v err=%v", handled, err)
	}
}

func TestCountOutstandingIgnoresResolved(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	raiser := &fakeRaiser{store: store}
	id, _, _ := Request(ctx, raiser, []string{"/home/u/p"}, "agent:x", "", "", "", time.Hour)
	if n, _ := store.CountOutstandingForSubject(ctx, "agent:x", time.Unix(0, 0).UTC()); n != 1 {
		t.Fatalf("expected 1 outstanding, got %d", n)
	}
	NewResolver(store).Apply(ctx, operatordecision.DecisionResolution{Correlation: map[string]string{correlationAskID: id}, Answer: "deny"})
	if n, _ := store.CountOutstandingForSubject(ctx, "agent:x", time.Unix(0, 0).UTC()); n != 0 {
		t.Fatalf("resolved ask must not count as outstanding, got %d", n)
	}
}

func dataProtection(path string) (string, string, bool, error) {
	const root = "/home/u/.vrooli/data"
	switch {
	case path == root:
		return "repo-contract runtime_home/data", root, true, nil
	case strings.HasPrefix(path, root+"/"):
		return "repo-contract runtime_home/data", root, false, nil
	}
	return "", "", false, nil
}

func TestRequestRefusesWhenProtectionIsUnknown(t *testing.T) {
	raiser := &fakeRaiser{store: newStore(t)}
	unknown := func(string) (string, string, bool, error) {
		return "", "", false, errors.New("declarations unreadable")
	}
	if _, raised, err := RequestProtected(context.Background(), raiser, unknown, []string{"/home/u/x"}, "agent", "", "", "r", time.Hour); err == nil || raised {
		t.Fatalf("raised=%v err=%v, want a refusal instead of an ask without the protection warning", raised, err)
	}
}

func TestRequestFlagsOwnerProtectedPathsForTheOperator(t *testing.T) {
	raiser := &fakeRaiser{store: newStore(t)}
	_, raised, err := RequestProtected(context.Background(), raiser, dataProtection, []string{"/home/u/.vrooli/data/app@old", "/home/u/.cache/x"}, "agent", "", "", "orphaned copy", time.Hour)
	if err != nil || !raised {
		t.Fatalf("raised=%v err=%v", raised, err)
	}
	note := raiser.facts.Note
	for _, want := range []string{"WARNING: 1 path(s) are owner-protected", "/home/u/.vrooli/data/app@old (PROTECTED: inside /home/u/.vrooli/data, which repo-contract runtime_home/data protects)", "  - /home/u/.cache/x\n"} {
		if !strings.Contains(note, want) {
			t.Errorf("note %q is missing %q", note, want)
		}
	}
	if !strings.Contains(raiser.facts.Question, "owner-protected") {
		t.Errorf("question %q hides that the deletion reaches protected data", raiser.facts.Question)
	}
}

func TestRequestRefusesAProtectedRootItself(t *testing.T) {
	raiser := &fakeRaiser{store: newStore(t)}
	if _, raised, err := RequestProtected(context.Background(), raiser, dataProtection, []string{"/home/u/.vrooli/data"}, "agent", "", "", "r", time.Hour); err == nil || raised {
		t.Fatalf("raised=%v err=%v, want a refusal no grant could satisfy", raised, err)
	}
	if raiser.facts.Subject != "" {
		t.Fatal("a refused request must not reach the operator's inbox")
	}
}
