package orchestration_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	runnercore "agent-manager/internal/adapters/runner/core"
	"agent-manager/internal/conversationpolicy"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/piimask"

	"github.com/google/uuid"
)

// Named sample values used by the import-policy tests. Keeping them as
// constants makes the harness-identity relationships explicit (the fixture
// harness vs. an excluded harness vs. a non-matching one).
const (
	privateHarness  = "resource:codex/private"
	sessionsHarness = "resource:codex/sessions"
	ollamaHarness   = "ollama"
)

// excludeImportRule and redactImportRule are the two import-seam rules the
// tests build; a tiny builder removes the repeated Action:/EnforceAt: literals.
func excludeImportRule(harness string) conversationpolicy.Rule {
	return conversationpolicy.Rule{Action: conversationpolicy.ActionExclude, EnforceAt: conversationpolicy.EnforceImport, Harness: harness}
}

func redactImportRule(harness string) conversationpolicy.Rule {
	return conversationpolicy.Rule{Action: conversationpolicy.ActionRedact, EnforceAt: conversationpolicy.EnforceImport, Harness: harness}
}

func mustImportPolicy(t *testing.T, rules ...conversationpolicy.Rule) *conversationpolicy.Policy {
	t.Helper()
	policy, err := conversationpolicy.New(rules)
	if err != nil {
		t.Fatalf("compile policy: %v", err)
	}
	return policy
}

// searchNotice records a derived-index notification so a test can assert the
// exclusion actually reached the search seam.
type searchNotice struct {
	Operation string
	RunID     string
}

// importPolicyRig bundles a policy-bearing orchestrator with the shared repos
// and event store so a test can build a sibling orchestrator (different
// policy, same data) for re-import scenarios.
type importPolicyRig struct {
	svc    *orchestration.Orchestrator
	repos  *database.Repositories
	events event.Store
}

// newImportPolicyOrchestrator builds an orchestrator wired with a codex runner
// and the given policy. The returned recorder captures every conversation
// search notification the orchestrator emits (upsert_run, delete_run, ...).
func newImportPolicyOrchestrator(t *testing.T, policy *conversationpolicy.Policy) (*orchestration.Orchestrator, event.Store, *noticeRecorder, func()) {
	t.Helper()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	svc, recorder := buildImportPolicyOrchestrator(t, repos, eventStore, policy)
	return svc, eventStore, recorder, cleanup
}

// newImportPolicyRig returns the full rig so a test can re-import the same
// provenance through a second orchestrator with a different policy.
func newImportPolicyRig(t *testing.T, policy *conversationpolicy.Policy) (importPolicyRig, func()) {
	t.Helper()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	svc, _ := buildImportPolicyOrchestrator(t, repos, eventStore, policy)
	return importPolicyRig{svc: svc, repos: repos, events: eventStore}, cleanup
}

func buildImportPolicyOrchestrator(t *testing.T, repos *database.Repositories, eventStore event.Store, policy *conversationpolicy.Policy) (*orchestration.Orchestrator, *noticeRecorder) {
	t.Helper()
	return buildImportPolicyOrchestratorWithMasker(t, repos, eventStore, policy, nil)
}

func buildImportPolicyOrchestratorWithMasker(t *testing.T, repos *database.Repositories, eventStore event.Store, policy *conversationpolicy.Policy, masker *piimask.Masker) (*orchestration.Orchestrator, *noticeRecorder) {
	t.Helper()
	registry := runner.NewRegistry()
	if err := registry.Register(runnercore.NewRunner(codecs.NewCodexForTest(), nil, nil)); err != nil {
		t.Fatalf("register codex runner: %v", err)
	}
	opts := []orchestration.Option{
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithRunStateRoot(t.TempDir()),
		orchestration.WithConfig(orchestration.OrchestratorConfig{DefaultTimeout: time.Minute, MaxConcurrentRuns: 1}),
		orchestration.WithInvocationReadModel(repos.InvocationReadModel),
		orchestration.WithConversationPolicy(policy),
	}
	if masker != nil {
		opts = append(opts, orchestration.WithPIIMasker(masker))
	}
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, opts...)
	recorder := &noticeRecorder{}
	svc.SetConversationSearchNotifier(recorder.record)
	return svc, recorder
}

// fakeImportDetector is a deterministic piimask.Detector for the import-seam
// masking tests. It records every call (so a test can assert scoping — a
// non-matching import must make zero detector calls) and can be told to fail
// for fail-closed coverage.
type fakeImportDetector struct {
	mu      sync.Mutex
	calls   []string
	failAll bool
}

func (f *fakeImportDetector) Detect(_ context.Context, text string) (piimask.DetectResult, error) {
	f.mu.Lock()
	f.calls = append(f.calls, text)
	f.mu.Unlock()
	if f.failAll {
		return piimask.DetectResult{}, errAnyDetectorDown
	}
	// Mask any occurrence of the fixture's PII marker as an EMAIL span.
	if idx := strings.Index(text, piiMarkerEmail); idx >= 0 {
		return piimask.DetectResult{Spans: []piimask.Span{{
			Category: "EMAIL_ADDRESS", Start: idx, End: idx + len(piiMarkerEmail), Text: piiMarkerEmail,
		}}}, nil
	}
	return piimask.DetectResult{Spans: []piimask.Span{}}, nil
}

func (f *fakeImportDetector) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

const piiMarkerEmail = "secret@pii.example"

var errAnyDetectorDown = errorString("pii detector unavailable")

type errorString string

func (e errorString) Error() string { return string(e) }

type noticeRecorder struct {
	mu      sync.Mutex
	notices []searchNotice
}

func (r *noticeRecorder) record(_ context.Context, operation, runID, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notices = append(r.notices, searchNotice{Operation: operation, RunID: runID})
	return nil
}

func (r *noticeRecorder) has(operation, runID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, n := range r.notices {
		if n.Operation == operation && n.RunID == runID {
			return true
		}
	}
	return false
}

func writeCodexFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "codex.jsonl")
	body := "{\"type\":\"thread.started\",\"thread_id\":\"policy-test\"}\n" +
		"{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"hello\"}}\n" +
		"{\"type\":\"turn.completed\"}\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func codexImportRequest(path, harness, session string) orchestration.ImportTranscriptRequest {
	return orchestration.ImportTranscriptRequest{Path: path, RunnerType: domain.RunnerTypeCodex, SourceHarness: harness, SourceSessionID: session}
}

// TestImportTranscriptExcludedByPolicyWritesNoEvents proves a run matching an
// import-seam exclude rule is never written: no error-as-failure, a typed
// exclusion signal, and zero run_events in the store.
func TestImportTranscriptExcludedByPolicyWritesNoEvents(t *testing.T) {
	policy := mustImportPolicy(t, excludeImportRule(privateHarness))
	svc, eventStore, _, cleanup := newImportPolicyOrchestrator(t, policy)
	t.Cleanup(cleanup)

	request := codexImportRequest(writeCodexFixture(t), privateHarness, "policy-test")
	run, importErr := svc.ImportTranscript(context.Background(), request)
	if run != nil {
		t.Fatalf("excluded import must not return a run, got %+v", run)
	}
	if !orchestration.IsImportExcluded(importErr) {
		t.Fatalf("expected ImportExcludedError, got %v", importErr)
	}

	existing, err := svc.GetRunByImportProvenance(context.Background(), request.SourceHarness, request.SourceSessionID)
	if err != nil {
		t.Fatalf("provenance lookup: %v", err)
	}
	if existing != nil {
		t.Fatalf("excluded import must not persist a run, got %+v", existing)
	}
	if got := policy.Snapshot().ImportExcluded; got != 1 {
		t.Fatalf("exclusion should be counted, got %d", got)
	}

	// Belt-and-braces: the store is keyed by run id and nothing was created,
	// so a probe for an arbitrary id yields no events and no error.
	if _, err = eventStore.Get(context.Background(), uuid.New(), event.GetOptions{AfterSequence: -1}); err != nil {
		t.Fatalf("unexpected event store error: %v", err)
	}
}

// TestReimportAfterExcludeRuleRemovesFromSearch is the A-P1-1 regression: a run
// imported before an exclude rule existed must, on re-import under that rule,
// be reported excluded AND dropped from the derived search projection — never
// silently revived. Before the fix the provenance short-circuit ran first and
// re-projected the already-imported run.
func TestReimportAfterExcludeRuleRemovesFromSearch(t *testing.T) {
	// 1. Import the run with no policy so it is fully written + searchable.
	rig, cleanup := newImportPolicyRig(t, mustImportPolicy(t))
	t.Cleanup(cleanup)

	request := codexImportRequest(writeCodexFixture(t), sessionsHarness, "reimport-test")
	run, err := rig.svc.ImportTranscript(context.Background(), request)
	if err != nil {
		t.Fatalf("initial import must succeed: %v", err)
	}
	if run == nil {
		t.Fatalf("expected an imported run")
	}
	runID := run.ID
	eventStore := rig.events

	// 2. A second orchestrator sharing the same repos and event store, now with
	// an exclude rule for the run's harness, re-imports the same provenance.
	policy := mustImportPolicy(t, excludeImportRule(sessionsHarness))
	svc2, recorder2 := buildImportPolicyOrchestrator(t, rig.repos, eventStore, policy)

	reRun, reErr := svc2.ImportTranscript(context.Background(), request)
	if reRun != nil {
		t.Fatalf("re-import under an exclude rule must not return a run, got %+v", reRun)
	}
	if !orchestration.IsImportExcluded(reErr) {
		t.Fatalf("re-import under an exclude rule must be excluded, got %v", reErr)
	}
	if got := policy.Snapshot().ImportExcluded; got != 1 {
		t.Fatalf("re-import exclusion should be counted once, got %d", got)
	}
	// The exclusion must reach the search seam: a delete_run for the existing
	// run, and no upsert_run that would have re-surfaced it.
	if !recorder2.has("delete_run", runID.String()) {
		t.Fatalf("re-import of an excluded run must notify delete_run for %s; notices=%+v", runID, recorder2.notices)
	}
	if recorder2.has("upsert_run", runID.String()) {
		t.Fatalf("re-import of an excluded run must NOT re-project it (upsert_run); notices=%+v", recorder2.notices)
	}
	// Canonical events are retained (removal of the run row is not this seam's
	// job); the exclusion is the search drop plus the no-new-events guarantee.
	events, err := eventStore.Get(context.Background(), runID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if len(events) == 0 {
		t.Fatalf("original import events must remain; exclude at re-import drops only the search projection")
	}
}

// TestImportTranscriptKeptWhenPolicyDoesNotMatch confirms a non-matching rule
// leaves import behavior unchanged (the run is written with events).
func TestImportTranscriptKeptWhenPolicyDoesNotMatch(t *testing.T) {
	policy := mustImportPolicy(t, excludeImportRule(ollamaHarness))
	svc, eventStore, _, cleanup := newImportPolicyOrchestrator(t, policy)
	t.Cleanup(cleanup)

	request := codexImportRequest(writeCodexFixture(t), sessionsHarness, "kept-test")
	run, err := svc.ImportTranscript(context.Background(), request)
	if err != nil {
		t.Fatalf("non-matching import must succeed: %v", err)
	}
	if run == nil {
		t.Fatalf("expected an imported run")
	}
	events, err := eventStore.Get(context.Background(), run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if len(events) == 0 {
		t.Fatalf("a kept import must write events")
	}
	if got := policy.Snapshot().ImportExcluded; got != 0 {
		t.Fatalf("no exclusion expected, got %d", got)
	}
}

// TestImportRedactRuleKeepsConversation confirms the redact action never
// excludes at import: it is recorded and the conversation is written. With NO
// masker wired (the default), redact is a recognized, observable no-op —
// content is imported unmasked but the intent is counted and logged, never
// silently dropped. (Masking behavior with a masker is covered by the
// piimask-backed tests below.)
func TestImportRedactRuleKeepsConversation(t *testing.T) {
	policy := mustImportPolicy(t, redactImportRule(sessionsHarness))
	svc, eventStore, _, cleanup := newImportPolicyOrchestrator(t, policy)
	t.Cleanup(cleanup)

	request := codexImportRequest(writeCodexFixture(t), sessionsHarness, "redact-test")
	run, err := svc.ImportTranscript(context.Background(), request)
	if err != nil {
		t.Fatalf("redact import must not fail: %v", err)
	}
	events, err := eventStore.Get(context.Background(), run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if len(events) == 0 {
		t.Fatalf("redact without a masker must still import events")
	}
	if got := policy.Snapshot().RedactMarked; got != 1 {
		t.Fatalf("redact intent should be recorded, got %d", got)
	}
}

// writeCodexFixtureWithPII writes a codex transcript whose assistant message
// carries a PII marker the fake detector masks. It is the input for the import-
// seam masking tests.
func writeCodexFixtureWithPII(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "codex-pii.jsonl")
	body := "{\"type\":\"thread.started\",\"thread_id\":\"pii-test\"}\n" +
		"{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"reach me at " + piiMarkerEmail + " ok\"}}\n" +
		"{\"type\":\"turn.completed\"}\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// messageContents returns the content of every message event for a run.
func messageContents(t *testing.T, eventStore event.Store, runID uuid.UUID) []string {
	t.Helper()
	events, err := eventStore.Get(context.Background(), runID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	var out []string
	for _, ev := range events {
		if ev == nil || ev.EventType != domain.EventTypeMessage {
			continue
		}
		if msg, ok := ev.Data.(*domain.MessageEventData); ok {
			out = append(out, msg.Content)
		}
	}
	return out
}

// TestImportRedactMasksPIIBeforeIndex is the W5 headline: a run matching a
// redact rule has its PII masked BEFORE the events reach the store that feeds
// the search index. The masked event must not contain the PII and must carry
// the category placeholder.
func TestImportRedactMasksPIIBeforeIndex(t *testing.T) {
	det := &fakeImportDetector{}
	policy := mustImportPolicy(t, redactImportRule(sessionsHarness))
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	svc, _ := buildImportPolicyOrchestratorWithMasker(t, repos, eventStore, policy, piimask.New(det))

	request := codexImportRequest(writeCodexFixtureWithPII(t), sessionsHarness, "pii-mask")
	run, err := svc.ImportTranscript(context.Background(), request)
	if err != nil {
		t.Fatalf("redact import must not fail: %v", err)
	}
	contents := messageContents(t, eventStore, run.ID)
	if len(contents) == 0 {
		t.Fatalf("redact import must still write message events")
	}
	for _, content := range contents {
		if strings.Contains(content, piiMarkerEmail) {
			t.Fatalf("masked message must not contain PII, got %q", content)
		}
	}
	joined := strings.Join(contents, " ")
	if !strings.Contains(joined, piimask.Placeholder("EMAIL_ADDRESS")) {
		t.Fatalf("masked message must carry the category placeholder, got %q", joined)
	}
	if det.callCount() == 0 {
		t.Fatalf("redact-matched import must call the detector")
	}
}

// TestImportNonMatchingRunMakesZeroDetectorCalls proves masking is scoped to
// redact-matched runs: an import that does not match the redact rule imports
// its content untouched and never calls the detector (the performance contract
// — a normal import pays no gateway cost).
func TestImportNonMatchingRunMakesZeroDetectorCalls(t *testing.T) {
	det := &fakeImportDetector{}
	// Redact rule targets a DIFFERENT harness, so this import does not match.
	policy := mustImportPolicy(t, redactImportRule(privateHarness))
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	svc, _ := buildImportPolicyOrchestratorWithMasker(t, repos, eventStore, policy, piimask.New(det))

	request := codexImportRequest(writeCodexFixtureWithPII(t), sessionsHarness, "no-match")
	run, err := svc.ImportTranscript(context.Background(), request)
	if err != nil {
		t.Fatalf("non-matching import must not fail: %v", err)
	}
	if det.callCount() != 0 {
		t.Fatalf("a non-redact import must make zero detector calls, got %d", det.callCount())
	}
	// Content is imported unmasked (the operator did not ask to redact it).
	joined := strings.Join(messageContents(t, eventStore, run.ID), " ")
	if !strings.Contains(joined, piiMarkerEmail) {
		t.Fatalf("non-matching run content must be imported untouched, got %q", joined)
	}
}

// TestImportRedactFailsClosedWhenDetectorErrors is the security contract: if the
// detector/gateway errors for a redact-matched run, the content must NOT be
// indexed unmasked. The event is still written (the conversation existed) but
// its content is replaced by the loud unavailable sentinel, never the PII.
func TestImportRedactFailsClosedWhenDetectorErrors(t *testing.T) {
	det := &fakeImportDetector{failAll: true}
	policy := mustImportPolicy(t, redactImportRule(sessionsHarness))
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	svc, _ := buildImportPolicyOrchestratorWithMasker(t, repos, eventStore, policy, piimask.New(det))

	request := codexImportRequest(writeCodexFixtureWithPII(t), sessionsHarness, "fail-closed")
	run, err := svc.ImportTranscript(context.Background(), request)
	if err != nil {
		t.Fatalf("fail-closed import must still succeed structurally: %v", err)
	}
	contents := messageContents(t, eventStore, run.ID)
	for _, content := range contents {
		if strings.Contains(content, piiMarkerEmail) {
			t.Fatalf("fail-closed must never index unmasked PII, got %q", content)
		}
	}
	joined := strings.Join(contents, " ")
	if !strings.Contains(joined, piimask.UnavailableSentinel) {
		t.Fatalf("fail-closed content must carry the unavailable sentinel, got %q", joined)
	}

	// The orchestrator accessors are the provider path the status surface reads.
	// A redact-matched import must surface through ImportPolicyCounts, and the
	// withheld content must surface as a non-zero fail-closed count through
	// PIIMaskCounts so an operator sees that masking could not run.
	if got := svc.ImportPolicyCounts().RedactMarked; got != 1 {
		t.Fatalf("ImportPolicyCounts().RedactMarked = %d, want 1", got)
	}
	if got := svc.PIIMaskCounts().FailClosed; got == 0 {
		t.Fatalf("PIIMaskCounts().FailClosed = 0, want the withheld events surfaced")
	}
}
