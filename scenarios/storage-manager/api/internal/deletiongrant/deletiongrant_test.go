package deletiongrant

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	id, raised, err := Request(context.Background(), raiser, nil, RequestInput{Paths: []string{"/home/u/.cache/x", "/home/u/.cache/x"}, RequestedBy: "agent:claude", Reason: "cleanup", TTL: 24 * time.Hour})
	if err != nil || !raised {
		t.Fatalf("Request raised=%v err=%v", raised, err)
	}
	if id == "" {
		t.Fatal("expected a correlation id")
	}
	if err := raiser.facts.Validate(); err != nil {
		t.Fatalf("facts invalid: %v", err)
	}
	if problems := operatordecision.CoverProblems(raiser.facts); len(problems) > 0 {
		t.Errorf("a cold reader cannot parse the cover: %v", problems)
	}
	if raiser.facts.Recommended == "approve" || raiser.facts.Reversible || raiser.facts.DefaultAnswer != "" {
		t.Fatalf("deleting data nobody declares rebuildable must be irreversible, with no default and no push to approve: %+v", raiser.facts)
	}
}

func TestRequestStampsTieBackCorrelation(t *testing.T) {
	store := newStore(t)
	raiser := &fakeRaiser{store: store}
	id, _, err := Request(context.Background(), raiser, nil, RequestInput{Paths: []string{"/home/u/.cache/x"}, RequestedBy: "run-123", RaisedByRunID: "run-123", WebConsoleSessionID: "wc-sess-1", Reason: "cleanup", TTL: time.Hour})
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
	if _, _, err := Request(context.Background(), raiser, nil, RequestInput{Paths: []string{"/home/u/.cache/x"}, RequestedBy: "operator@host", Reason: "cleanup", TTL: time.Hour}); err != nil {
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
	idA, _, _ := Request(context.Background(), a, nil, RequestInput{Paths: []string{"/a", "/b"}, RequestedBy: "agent", Reason: "r", TTL: time.Hour})
	idB, _, _ := Request(context.Background(), b, nil, RequestInput{Paths: []string{"/b", "/a", "/a"}, RequestedBy: "agent", Reason: "r", TTL: time.Hour})
	if idA != idB {
		t.Fatalf("correlation id must be stable per path set: %q vs %q", idA, idB)
	}
}

func TestRequestRejectsEmptyOrRelative(t *testing.T) {
	store := newStore(t)
	raiser := &fakeRaiser{store: store}
	if _, _, err := Request(context.Background(), raiser, nil, RequestInput{Paths: []string{"relative/path", ""}, RequestedBy: "agent", Reason: "r", TTL: time.Hour}); err == nil {
		t.Fatal("expected error when no absolute path is given")
	}
}

func TestApproveProducesActiveGrantThenExpires(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	raiser := &fakeRaiser{store: store}
	id, _, err := Request(ctx, raiser, nil, RequestInput{Paths: []string{"/home/u/.cache/x"}, RequestedBy: "agent", Reason: "cleanup", TTL: 2 * time.Hour})
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
	id, _, _ := Request(ctx, raiser, nil, RequestInput{Paths: []string{"/home/u/keep"}, RequestedBy: "agent", TTL: time.Hour})
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
	id, _, _ := Request(ctx, raiser, nil, RequestInput{Paths: []string{"/home/u/a"}, RequestedBy: "agent", TTL: time.Hour})
	fired := 0
	resolver := NewResolver(store).WithClock(func() time.Time { return now }).WithOnApprove(func() { fired++ })
	resolver.Apply(ctx, operatordecision.DecisionResolution{Correlation: map[string]string{correlationAskID: id}, Answer: "approve", AnsweredBy: "op"})
	if fired != 1 {
		t.Fatalf("onApprove should fire once on approval, fired=%d", fired)
	}

	// Deny → callback does not fire.
	store2 := newStore(t)
	raiser2 := &fakeRaiser{store: store2}
	id2, _, _ := Request(ctx, raiser2, nil, RequestInput{Paths: []string{"/home/u/b"}, RequestedBy: "agent", TTL: time.Hour})
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
	id, _, _ := Request(ctx, raiser, nil, RequestInput{Paths: []string{"/home/u/p"}, RequestedBy: "agent:x", TTL: time.Hour})
	if n, _ := store.CountOutstandingForSubject(ctx, "agent:x", time.Unix(0, 0).UTC()); n != 1 {
		t.Fatalf("expected 1 outstanding, got %d", n)
	}
	NewResolver(store).Apply(ctx, operatordecision.DecisionResolution{Correlation: map[string]string{correlationAskID: id}, Answer: "deny"})
	if n, _ := store.CountOutstandingForSubject(ctx, "agent:x", time.Unix(0, 0).UTC()); n != 0 {
		t.Fatalf("resolved ask must not count as outstanding, got %d", n)
	}
}

// fixture is a home folder with real files, so the page is written from
// measured sizes and times, as on the host.
type fixture struct {
	t    *testing.T
	home string
	now  time.Time
	decl map[string]Declaration // by path prefix
}

func newFixture(t *testing.T) *fixture {
	return &fixture{t: t, home: t.TempDir(), now: time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC), decl: map[string]Declaration{}}
}

// folder creates rel under home holding one file of size bytes, last changed
// age ago, and returns its absolute path.
func (f *fixture) folder(rel string, size int, age time.Duration) string {
	f.t.Helper()
	dir := filepath.Join(f.home, rel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		f.t.Fatal(err)
	}
	file := filepath.Join(dir, "blob")
	if err := os.WriteFile(file, make([]byte, size), 0o644); err != nil {
		f.t.Fatal(err)
	}
	when := f.now.Add(-age)
	for _, p := range []string{file, dir} {
		if err := os.Chtimes(p, when, when); err != nil {
			f.t.Fatal(err)
		}
	}
	return dir
}

func (f *fixture) declare(path string) Declaration {
	best, found := "", false
	for prefix := range f.decl {
		if within(path, prefix) && len(prefix) > len(best) {
			best, found = prefix, true
		}
	}
	if !found {
		return Declaration{Disposition: Undeclared}
	}
	d := f.decl[best]
	d.IsRoot = d.Disposition == Protected && path == best
	return d
}

func (f *fixture) request(paths []string, reason string) operatordecision.Facts {
	f.t.Helper()
	raiser := &fakeRaiser{store: newStore(f.t)}
	if _, raised, err := Request(context.Background(), raiser, f.declare, RequestInput{Paths: paths, Reason: reason, RequestedBy: "run-7", TTL: 2 * time.Hour, now: f.now, home: f.home}); err != nil || !raised {
		f.t.Fatalf("raised=%v err=%v", raised, err)
	}
	if err := raiser.facts.Validate(); err != nil {
		f.t.Fatalf("facts invalid: %v", err)
	}
	return raiser.facts
}

func assertContains(t *testing.T, where, text string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Errorf("%s is missing %q:\n%s", where, want, text)
		}
	}
}

// assertColdReadable checks the cover against the Hub's cold-reader rules: a
// short question with no path or run id in it.
func assertColdReadable(t *testing.T, f *fixture, facts operatordecision.Facts) {
	t.Helper()
	if problems := operatordecision.CoverProblems(facts); len(problems) > 0 {
		t.Errorf("a cold reader cannot parse the cover: %v", problems)
	}
	for name, text := range map[string]string{"question": facts.Question, "subject": facts.Subject, "big picture": facts.BigPicture} {
		if strings.Contains(text, f.home) || strings.Contains(text, "run-7") {
			t.Errorf("%s %q carries a path or run id; those belong in technical details", name, text)
		}
	}
	if facts.Explainer == "" || len(facts.Explainer) > operatordecision.MaxExplainerBytes {
		t.Fatalf("explainer page missing or too large (%d bytes)", len(facts.Explainer))
	}
}

func TestProtectedDeletionArrivesAsAPageTheOperatorCanAnswerCold(t *testing.T) {
	f := newFixture(t)
	data := filepath.Join(f.home, ".vrooli/data")
	f.decl[data] = Declaration{Disposition: Protected, Owner: "vrooli", Source: "repo-contract runtime_home/data", Root: data}
	paths := []string{
		f.folder(".vrooli/data/vrooli/treasury_cf2-1794170077", 3000, 40*24*time.Hour),
		f.folder(".vrooli/data/vrooli/treasury_cf2-1794230388", 1000, 3*24*time.Hour),
	}
	facts := f.request(paths, "Leftover treasury test books from finished runs")
	assertColdReadable(t, f, facts)
	assertContains(t, "question", facts.Question, "2 folders named “treasury_cf2-1794…”", "(3.9KiB)", "in Vrooli's runtime folder", "All 2 are protected data that does not come back.")
	assertContains(t, "big picture", facts.BigPicture, "Leftover treasury test books", "for the next 2 hours", "nothing is deleted")
	if facts.Recommended != "deny" || facts.RecommendationReason == "" {
		t.Errorf("protected data must recommend keeping it, with a reason: %q %q", facts.Recommended, facts.RecommendationReason)
	}
	assertContains(t, "approve label", facts.Label("approve"), "exactly these 2 items", "2 hours")
	assertContains(t, "page", facts.Explainer,
		"~/.vrooli/data/vrooli",                            // where, written from home
		"treasury_cf2-1794170077", "2.9KiB", "5 weeks ago", // each item: name, size, age
		"treasury_cf2-1794230388", "1000B", "3 days ago",
		"vrooli", "No, protected data", "is-danger",
		`data-od-answer="approve"`, `data-od-answer="deny"`, "If you do not answer:", "closes by itself after 7 days")
	if strings.Index(facts.Explainer, "treasury_cf2-1794170077") > strings.Index(facts.Explainer, "treasury_cf2-1794230388") {
		t.Error("the page must list the largest item first")
	}
	assertContains(t, "technical details", facts.TechnicalDetails, paths[0], "PROTECTED: inside "+data, "run-7")
}

func TestMixedDeletionGroupsItemsByOwnerAndWhetherTheyComeBack(t *testing.T) {
	f := newFixture(t)
	cache := filepath.Join(f.home, ".cache/agent-manager")
	f.decl[cache] = Declaration{Disposition: Regenerable, Owner: "agent-manager", Source: "scenario/agent-manager storage/cache", Root: cache}
	paths := []string{
		f.folder(".cache/agent-manager/build", 5000, time.Hour),
		f.folder("vrooli-git-recovery-20260908", 9000, 60*24*time.Hour),
	}
	facts := f.request(paths, "Free disk space")
	assertColdReadable(t, f, facts)
	assertContains(t, "question", facts.Question, "2 folders", "in your home folder", "No owner declares 1 of the 2")
	if facts.Recommended != "" {
		t.Errorf("only the operator knows whether undeclared data is wanted; got recommendation %q", facts.Recommended)
	}
	assertContains(t, "page", facts.Explainer, "By owner", "agent-manager", "Yes, it rebuilds itself", "Nobody", "No, nothing rebuilds it",
		">Folder<", "~/.cache/agent-manager", "2 months ago")
}

func TestRebuildableDeletionRecommendsApproval(t *testing.T) {
	f := newFixture(t)
	cache := filepath.Join(f.home, ".cache")
	f.decl[cache] = Declaration{Disposition: Regenerable, Owner: "go", Root: cache}
	facts := f.request([]string{f.folder(".cache/go-build", 2048, 24*time.Hour)}, "Disk is 95% full")
	assertColdReadable(t, f, facts)
	assertContains(t, "question", facts.Question, "the folder “go-build” (2KiB)", "Its owner says it rebuilds itself.")
	if facts.Recommended != "approve" {
		t.Errorf("data its owner declares rebuildable should recommend approval, got %q", facts.Recommended)
	}
	assertContains(t, "page", facts.Explainer, "is-success", "yesterday")
}

func TestRequestRefusesAProtectedRootItself(t *testing.T) {
	f := newFixture(t)
	data := filepath.Join(f.home, ".vrooli/data")
	f.decl[data] = Declaration{Disposition: Protected, Source: "repo-contract runtime_home/data", Root: data}
	raiser := &fakeRaiser{store: newStore(t)}
	if _, raised, err := Request(context.Background(), raiser, f.declare, RequestInput{Paths: []string{data}, Reason: "r", TTL: time.Hour}); err == nil || raised {
		t.Fatalf("raised=%v err=%v, want a refusal no grant could satisfy", raised, err)
	}
	if raiser.facts.Subject != "" {
		t.Fatal("a refused request must not reach the operator's inbox")
	}
}

func TestMeasuringStopsAtTheBudgetAndSaysSo(t *testing.T) {
	f := newFixture(t)
	dir := f.folder("big", 100, time.Hour)
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprint(i)), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	items := inspect([]string{dir}, nil, &measureBudget{entries: 2, deadline: f.now.Add(time.Hour * 1e5)})
	if !items[0].Partial || items[0].Files > 2 {
		t.Fatalf("measure = %+v, want a partial floor after 2 entries", items[0])
	}
	facts := buildFacts("id", items, RequestInput{Paths: []string{dir}, TTL: time.Hour, now: f.now, home: f.home})
	assertContains(t, "question", facts.Question, "(at least ")
	assertContains(t, "page", facts.Explainer, "Sizes are a floor.")
}

func TestLargeRequestStaysReviewableAndValid(t *testing.T) {
	f := newFixture(t)
	root := filepath.Join(f.home, ".vrooli/state/sessions/codex")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 4700)
	for i := range paths {
		paths[i] = filepath.Join(root, fmt.Sprintf("session-%05d.jsonl", i))
		if err := os.WriteFile(paths[i], make([]byte, i%50), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	facts := f.request(paths, "fixture cleanup")
	assertColdReadable(t, f, facts)
	assertContains(t, "question", facts.Question, "4700 files named “session-0…”", "in Vrooli's runtime folder")
	assertContains(t, "technical details", facts.TechnicalDetails, "4700 path(s) in 1 directory(ies)", root+"/ — 4700", "more not shown")
	assertContains(t, "page", facts.Explainer, "The 80 largest of 4700", "every path is in the technical details")
	if facts.Subject != "Delete 4700 files (112.5KiB)?" {
		t.Errorf("subject %q is not the short count title", facts.Subject)
	}
}

func TestRequestHandlerReadsDeclarationsOncePerRequest(t *testing.T) {
	raiser := &fakeRaiser{store: newStore(t)}
	loads := 0
	source := func() (func(string) Declaration, error) {
		loads++
		return func(string) Declaration { return Declaration{Disposition: Undeclared} }, nil
	}
	body := `{"paths":["/home/u/a","/home/u/b","/home/u/c"],"reason":"r"}`
	rr := httptest.NewRecorder()
	RequestHandler(raiser, source).ServeHTTP(rr, httptest.NewRequest(http.MethodPost, RequestPath, strings.NewReader(body)))
	if rr.Code != http.StatusAccepted || loads != 1 {
		t.Fatalf("status=%d loads=%d, want 202 and one declaration read", rr.Code, loads)
	}
}

func TestRequestHandlerRefusesWhenDeclarationsAreUnreadable(t *testing.T) {
	raiser := &fakeRaiser{store: newStore(t)}
	source := func() (func(string) Declaration, error) { return nil, errors.New("declarations unreadable") }
	rr := httptest.NewRecorder()
	RequestHandler(raiser, source).ServeHTTP(rr, httptest.NewRequest(http.MethodPost, RequestPath, strings.NewReader(`{"paths":["/home/u/x"],"reason":"r"}`)))
	if rr.Code != http.StatusServiceUnavailable || raiser.facts.Subject != "" {
		t.Fatalf("status=%d, want 503 and no ask without knowing what the paths hold", rr.Code)
	}
}
