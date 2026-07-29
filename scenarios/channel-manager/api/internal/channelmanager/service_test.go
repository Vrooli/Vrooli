package channelmanager

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeBrowser struct {
	profile, workflow, action string
	id                        string
	err                       error
}

type fakeEnvironmentProbe struct {
	region string
	err    error
	ref    string
}

func (f *fakeEnvironmentProbe) Probe(_ context.Context, ref string) (string, error) {
	f.ref = ref
	return f.region, f.err
}

func (f *fakeBrowser) Dispatch(_ context.Context, profile, workflow, action string) (string, []string, error) {
	f.profile, f.workflow, f.action = profile, workflow, action
	return f.id, nil, f.err
}

func fixture(t *testing.T) *Service {
	t.Helper()
	s, e := New([]Platform{{ID: "x", DailyCeiling: 2, ActionKinds: []string{"engage", "publish"}, Formats: testFormats()}}, []Program{{ID: "x-warm", PlatformID: "x", Preconditions: []string{"region"}, Phases: []Phase{{ID: "one", Allowed: []string{"engage"}, Forbidden: []string{"publish"}, CountMax: 2}}, Provenance: Provenance{"operator", "speculative", "2026-07-28", "five runs", []string{"operator note"}}}})
	if e != nil {
		t.Fatal(e)
	}
	return s
}

// [REQ:CHANMGR-P0-001] An identity is validated against the descriptor and
// retains its operational references through the durable action lifecycle.
func TestIdentityWarmingQueueAndManualEvidence(t *testing.T) {
	s := fixture(t)
	if e := s.CreateIdentity(Identity{ID: "i", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", VaultRef: "vault://x", Attestations: map[string]bool{"region": true}}); e != nil {
		t.Fatal(e)
	}
	if e := s.StartProgram("i", "x-warm"); e != nil {
		t.Fatal(e)
	}
	now := time.Date(2026, 1, 2, 9, 0, 0, 0, time.UTC)
	a, e := s.Enqueue("i", "engage", now, 42, "")
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Complete(a.ID, "https://evidence", now); e != nil {
		t.Fatal(e)
	}
	if a.Status != Succeeded {
		t.Fatalf("completion must traverse to succeeded, got %s", a.Status)
	}
	if a.Executor != "manual" || a.Evidence == "" || a.CompletedAt.IsZero() {
		t.Fatal("manual completion must retain durable evidence")
	}
	if _, e = s.Enqueue("i", "publish", now, 1, ""); !errors.Is(e, ErrForbiddenAction) {
		t.Fatalf("got %v", e)
	}
}

func TestActionTransitionRejectsTerminalAndMissingEvidence(t *testing.T) {
	s := fixture(t)
	if err := s.CreateIdentity(Identity{ID: "i", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", Attestations: map[string]bool{"region": true}}); err != nil {
		t.Fatal(err)
	}
	if err := s.StartProgram("i", "x-warm"); err != nil {
		t.Fatal(err)
	}
	a, err := s.Enqueue("i", "engage", time.Now().UTC(), 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Complete(a.ID, "", time.Now().UTC()); err == nil {
		t.Fatal("manual completion without evidence must be rejected")
	}
	if err := s.Complete(a.ID, "manual://evidence", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := TransitionAction(a.Status, ActionCancel); err == nil {
		t.Fatal("terminal action must reject cancellation")
	}
}

func TestCredentialCadenceSignalAndIdempotency(t *testing.T) {
	s := fixture(t)
	if e := s.CreateIdentity(Identity{ID: "bad", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", VaultRef: "token=secret"}); !errors.Is(e, ErrCredentialValue) {
		t.Fatal(e)
	}
	if e := s.CreateIdentity(Identity{ID: "i", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", LaneGrants: []string{"main"}, Status: "active"}); e != nil {
		t.Fatal(e)
	}
	now := time.Now()
	for range 2 {
		if _, e := s.Enqueue("i", "engage", now, 1, ""); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := s.Enqueue("i", "engage", now, 2, ""); !errors.Is(e, ErrCadence) {
		t.Fatal(e)
	}
	s.Observations["i"] = []Observation{{"reach", 100, now}, {"reach", 100, now}}
	f, e := s.RecordObservation("i", "reach", 20, now, 3, .5)
	if e != nil || f == nil || s.Identities["i"].Status != "paused" {
		t.Fatal("decay must flag and pause only")
	}
	s.Identities["i"].Status = "active"
	receipt, e := s.Release("i", "main", "draft-1", "release-1", now.AddDate(0, 0, 1))
	if e != nil {
		t.Fatal(e)
	}
	again, e := s.Release("i", "main", "draft-1", "release-1", now.AddDate(0, 0, 1))
	if e != nil || receipt != again {
		t.Fatal("release must be idempotent")
	}
}

// [REQ:CHANMGR-P1-002] Publish actions from different identities are deferred
// into the earliest slot that satisfies the operator's portfolio separation
// and rolling-window ceiling; engagement work is unaffected.
func TestPortfolioPolicySeparatesCrossIdentityPublishes(t *testing.T) {
	s := fixture(t)
	for _, id := range []string{"a", "b", "c"} {
		if err := s.CreateIdentity(Identity{ID: id, PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", Status: "active"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.ConfigurePortfolioPolicy(PortfolioPolicy{MinimumPostGapMinutes: 30, WindowMinutes: 60, MaxPostsPerWindow: 2}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 2, 2, 9, 0, 0, 0, time.UTC)
	first, err := s.Enqueue("a", "publish", now, 1, "a")
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Enqueue("b", "publish", now.Add(5*time.Minute), 2, "b")
	if err != nil {
		t.Fatal(err)
	}
	third, err := s.Enqueue("c", "publish", now.Add(10*time.Minute), 3, "c")
	if err != nil {
		t.Fatal(err)
	}
	if first.Deferred || !second.Deferred || second.Window.Sub(first.Window) < 30*time.Minute || !third.Deferred || third.Window.Sub(second.Window) < 30*time.Minute {
		t.Fatalf("portfolio slots=%+v %+v %+v", first, second, third)
	}
	engagement, err := s.Enqueue("b", "engage", now.Add(6*time.Minute), 4, "")
	if err != nil || engagement.Deferred || !engagement.Window.Equal(now.Add(6*time.Minute)) {
		t.Fatalf("non-publish action must not be portfolio-deferred: %+v %v", engagement, err)
	}
}

// [REQ:CHANMGR-P1-005] Platform-owned retry rules retain bounded transient
// failures for backoff while terminal codes fail once and never retry.
func TestExecutionFailureUsesPlatformRetryPolicy(t *testing.T) {
	s, err := New([]Platform{{ID: "x", DailyCeiling: 3, ActionKinds: []string{"engage"}, Retry: RetryPolicy{RetryableCodes: []string{"timeout"}, MaxAttempts: 2, BackoffMinutes: 5}, Formats: testFormats()}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.CreateIdentity(Identity{ID: "i", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 2, 3, 9, 0, 0, 0, time.UTC)
	retryable, err := s.Enqueue("i", "engage", now, 1, "retryable")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RecordExecutionFailure(retryable.ID, "timeout", now); err != nil {
		t.Fatal(err)
	}
	if retryable.Status != Scheduled || retryable.FailureClass != "retryable" || !retryable.NextAttemptAt.Equal(now.Add(5*time.Minute)) {
		t.Fatalf("retryable=%+v", retryable)
	}
	if len(s.Actions) != 1 {
		t.Fatal("retry must retain the original cadence-accounted action rather than create another slot")
	}
	terminal, err := s.Enqueue("i", "engage", now.Add(time.Hour), 2, "terminal")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RecordExecutionFailure(terminal.ID, "credential_rejected", now); err != nil {
		t.Fatal(err)
	}
	if terminal.Status != Failed || terminal.FailureClass != "terminal" || terminal.AttemptCount != 1 {
		t.Fatalf("terminal=%+v", terminal)
	}
}

// [REQ:CHANMGR-P1-010] A credential-free environment probe records healthy,
// mismatch, and unknown outcomes; uncertain or divergent region evidence
// pauses the identity queue instead of allowing silent release.
func TestEnvironmentLivenessPausesUnknownOrMismatchedIdentity(t *testing.T) {
	s := fixture(t)
	if err := s.CreateIdentity(Identity{ID: "i", PlatformID: "x", Purpose: "brand", EnvironmentRef: "environment://i", ExpectedRegion: "US", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 2, 4, 9, 0, 0, 0, time.UTC)
	probe := &fakeEnvironmentProbe{region: "US"}
	healthy, err := s.CheckEnvironment(context.Background(), "i", probe, now)
	if err != nil || healthy.Status != "healthy" || s.Identities["i"].Status != "active" {
		t.Fatalf("healthy=%+v err=%v", healthy, err)
	}
	if probe.ref != "environment://i" {
		t.Fatalf("probe received %q, want only the opaque environment reference", probe.ref)
	}
	mismatch, err := s.CheckEnvironment(context.Background(), "i", &fakeEnvironmentProbe{region: "CA"}, now)
	if err != nil || mismatch.Status != "mismatch" || s.Identities["i"].Status != "paused" {
		t.Fatalf("mismatch=%+v err=%v", mismatch, err)
	}
	s.Identities["i"].Status = "active"
	unknown, err := s.CheckEnvironment(context.Background(), "i", nil, now)
	if err != nil || unknown.Status != "unknown" || s.Identities["i"].Status != "paused" {
		t.Fatalf("unknown=%+v err=%v", unknown, err)
	}
}

// [REQ:CHANMGR-P1-003] [REQ:CHANMGR-P1-004] A completed asset cannot be
// released by another identity, and persona-actor release fails closed until
// the platform-required disclosure is explicitly visible.
func TestReleaseRejectsReusedAssetAndMissingPersonaDisclosure(t *testing.T) {
	s := fixture(t)
	platform := s.Platforms["x"]
	platform.DisclosureRequired = true
	s.Platforms["x"] = platform
	for _, identity := range []Identity{{ID: "a", PlatformID: "x", Purpose: "persona-actor", EnvironmentRef: "env", LaneGrants: []string{"main"}, Status: "active"}, {ID: "b", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", LaneGrants: []string{"main"}, Status: "active"}} {
		if err := s.CreateIdentity(identity); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 2, 5, 9, 0, 0, 0, time.UTC)
	if _, err := s.ReleaseWithOptions("a", "main", "draft-a", "missing-disclosure", ReleaseOptions{AssetIDs: []string{"asset-1"}}, now); err == nil {
		t.Fatal("persona release without disclosure must fail")
	}
	first, err := s.ReleaseWithOptions("a", "main", "draft-a", "a-release", ReleaseOptions{AssetIDs: []string{"asset-1"}, DisclosureVisible: true}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CompleteRelease(first.ActionID, "post-a", "https://example.test/a", "not_requested", "", now); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ReleaseWithOptions("b", "main", "draft-b", "b-release", ReleaseOptions{AssetIDs: []string{"asset-1"}}, now.Add(time.Hour)); err == nil {
		t.Fatal("cross-identity asset reuse must fail")
	}
}

// [REQ:CHANMGR-P0-014] [REQ:CHANMGR-P1-008] A publish receipt is durable,
// idempotent, and represents a first-comment failure as partial completion.
func TestReleaseReceiptCompletesExactlyOnceWithPartialFirstComment(t *testing.T) {
	s := fixture(t)
	if err := s.CreateIdentity(Identity{ID: "i", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", LaneGrants: []string{"main"}, Status: "active"}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)
	receipt, err := s.Release("i", "main", "draft-7", "key-7", now)
	if err != nil {
		t.Fatal(err)
	}
	completed, err := s.CompleteRelease(receipt.ActionID, "post-7", "https://platform.example/post-7", "failed", "comment validation rejected", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "partial" || completed.PlatformPostID != "post-7" || completed.PublishedURL == "" {
		t.Fatalf("unexpected receipt: %+v", completed)
	}
	again, err := s.CompleteRelease(receipt.ActionID, "different", "https://other.example", "succeeded", "", now.Add(2*time.Minute))
	if err != nil || again != completed || again.PlatformPostID != "post-7" {
		t.Fatalf("completion must be idempotent: %+v %v", again, err)
	}
}

// [REQ:CHANMGR-P1-001] BAS dispatch accepts only a durable action and profile
// reference; failures preserve manual fallback and never complete the action.
func TestBrowserDispatchUsesAssignedProfileAndPreservesManualFallback(t *testing.T) {
	s := fixture(t)
	if err := s.CreateIdentity(Identity{ID: "i", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	a, err := s.Enqueue("i", "engage", time.Now().UTC(), 1, "browser-1")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.AssignAutomation("i", "bas-profile-i", "workflow-i", []string{"engage"}, "operator accepted synthetic test"); err != nil {
		t.Fatal(err)
	}
	browser := &fakeBrowser{id: "bas-exec-1"}
	id, err := s.DispatchBrowser(context.Background(), a.ID, browser)
	if err != nil || id != "bas-exec-1" || browser.profile != "bas-profile-i" || browser.workflow != "workflow-i" || browser.action != a.ID {
		t.Fatalf("dispatch=%q err=%v fake=%+v", id, err, browser)
	}
	if a.Status != Scheduled {
		t.Fatal("dispatch must not complete action")
	}
	failing := &fakeBrowser{err: errors.New("BAS unavailable")}
	b, err := s.Enqueue("i", "engage", time.Now().AddDate(0, 0, 1), 2, "browser-2")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DispatchBrowser(context.Background(), b.ID, failing); err == nil || b.ExecutionError == "" || b.Status != Scheduled {
		t.Fatal("dispatch failure must retain manual fallback")
	}
}

// [REQ:CHANMGR-P1-007] Metric samples are append-only, release-attributed,
// idempotent, and retain delivery state until Content Desk acknowledges them.
func TestMetricSampleRequiresCompletedReceiptAndAcknowledgesOnce(t *testing.T) {
	s := fixture(t)
	if err := s.CreateIdentity(Identity{ID: "i", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", LaneGrants: []string{"main"}, Status: "active"}); err != nil {
		t.Fatal(err)
	}
	receipt, err := s.Release("i", "main", "draft-metric", "metric-release", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.RecordMetric(receipt.ID, "sample-1", "impressions", 10, time.Now().UTC()); err == nil {
		t.Fatal("queued release must not accept metrics")
	}
	if _, err = s.CompleteRelease(receipt.ActionID, "post-metric", "https://example.test/post", "not_requested", "", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	sample, err := s.RecordMetric(receipt.ID, "sample-1", "impressions", 10, time.Now().UTC())
	if err != nil || sample.DraftID != "draft-metric" || sample.DeliveryStatus != "pending" {
		t.Fatalf("sample=%+v err=%v", sample, err)
	}
	again, err := s.RecordMetric(receipt.ID, "sample-1", "impressions", 99, time.Now().UTC())
	if err != nil || again != sample || again.Value != 10 {
		t.Fatal("same sample id must be idempotent")
	}
	if err = s.AcknowledgeMetric(sample.ID); err != nil || sample.DeliveryStatus != "acknowledged" {
		t.Fatal(err)
	}
}

// [REQ:CHANMGR-P0-011] [REQ:CHANMGR-P1-006] A gate waits, resolves
// deterministically, turns a bounded inconclusive series into terminal
// quarantine with pending work gone, and appends its evidence.
func TestGateWaitsThenQuarantinesAfterBoundedInconclusiveResults(t *testing.T) {
	s, err := New([]Platform{{ID: "x", DailyCeiling: 3, ActionKinds: []string{"engage"}, Formats: testFormats()}}, []Program{{ID: "warm", PlatformID: "x", Preconditions: []string{"region"}, Phases: []Phase{{ID: "p", Allowed: []string{"engage"}}}, Gates: []Gate{{ID: "reach", Metric: "reach", Minimum: 100, WaitMinutes: 10, MaxRepeats: 2}}, Provenance: Provenance{SourceKind: "operator", Confidence: "speculative", CapturedAt: "today", RevisitTrigger: "runs", Sources: []string{"manual"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.CreateIdentity(Identity{ID: "i", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", Attestations: map[string]bool{"region": true}}); err != nil {
		t.Fatal(err)
	}
	if err = s.StartProgram("i", "warm"); err != nil {
		t.Fatal(err)
	}
	start := s.ProgramStartedAt["i"]
	if result, err := s.EvaluateGate("i", "reach", start.Add(time.Minute)); err != nil || result.Outcome != "waiting" {
		t.Fatalf("premature=%+v %v", result, err)
	}
	if _, err = s.Enqueue("i", "engage", start.Add(11*time.Minute), 1, ""); err != nil {
		t.Fatal(err)
	}
	if result, err := s.EvaluateGate("i", "reach", start.Add(11*time.Minute)); err != nil || result.Outcome != "inconclusive" {
		t.Fatalf("first=%+v %v", result, err)
	}
	result, err := s.EvaluateGate("i", "reach", start.Add(12*time.Minute))
	if err != nil || result.Outcome != "fail" {
		t.Fatalf("second=%+v %v", result, err)
	}
	if s.Identities["i"].Status != "quarantined" {
		t.Fatal("failed gate must quarantine")
	}
	for _, a := range s.Actions {
		if a.Status != Cancelled {
			t.Fatal("quarantine must cancel pending work")
		}
	}
	if len(s.ProgramOutcomes) != 1 || s.ProgramOutcomes[0].Outcome != "quarantined" || s.ProgramOutcomes[0].Gates["reach"].Outcome != "fail" {
		t.Fatalf("program outcome=%+v", s.ProgramOutcomes)
	}
}

// [REQ:CHANMGR-P0-012] [REQ:CHANMGR-P1-006] Elapsed time cannot grant
// eligibility; every declared criterion must be explicitly true before the
// identity becomes active, and graduation appends a durable program outcome.
func TestGraduationRequiresEveryCriterion(t *testing.T) {
	s, err := New([]Platform{{ID: "x", DailyCeiling: 1, ActionKinds: []string{"engage"}, Formats: testFormats()}}, []Program{{ID: "warm", PlatformID: "x", Preconditions: []string{"region"}, Phases: []Phase{{ID: "p"}}, Gates: []Gate{{ID: "reach", Metric: "reach", MaxRepeats: 1}}, GraduationCriteria: []string{"gate:reach", "attestation:review"}, Provenance: Provenance{SourceKind: "operator", Confidence: "speculative", CapturedAt: "today", RevisitTrigger: "runs", Sources: []string{"manual"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.CreateIdentity(Identity{ID: "i", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", Attestations: map[string]bool{"region": true}}); err != nil {
		t.Fatal(err)
	}
	if err = s.StartProgram("i", "warm"); err != nil {
		t.Fatal(err)
	}
	if err = s.Graduate("i", map[string]bool{"gate:reach": true}, []string{"main"}); err == nil {
		t.Fatal("partial criteria must not graduate")
	}
	if err = s.Graduate("i", map[string]bool{"gate:reach": true, "attestation:review": true}, []string{"main"}); err != nil {
		t.Fatal(err)
	}
	if s.Eligibility("i", "main") != "eligible" {
		t.Fatal("earned graduation must grant lane eligibility")
	}
	if len(s.ProgramOutcomes) != 1 || s.ProgramOutcomes[0].Outcome != "graduated" {
		t.Fatalf("program outcome=%+v", s.ProgramOutcomes)
	}
}

// [REQ:CHANMGR-P0-008] [REQ:CHANMGR-P0-009] Session placement honours a
// minimum gap while a seeded warming roll remains in the declared range.
func TestSessionSchedulingAndRecordedRoll(t *testing.T) {
	s, err := New([]Platform{{ID: "x", DailyCeiling: 4, ActionKinds: []string{"engage"}, Formats: testFormats()}}, []Program{{ID: "warm", PlatformID: "x", Preconditions: []string{"region"}, Sessions: SessionPolicy{Count: 1, MinimumGapMinutes: 30}, Phases: []Phase{{ID: "p", Allowed: []string{"engage"}, CountMin: 2, CountMax: 2}}, Provenance: Provenance{SourceKind: "operator", Confidence: "speculative", CapturedAt: "today", RevisitTrigger: "runs", Sources: []string{"manual"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.CreateIdentity(Identity{ID: "i", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", Attestations: map[string]bool{"region": true}}); err != nil {
		t.Fatal(err)
	}
	if err = s.StartProgram("i", "warm"); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	first, err := s.Enqueue("i", "engage", now, 7, "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Enqueue("i", "engage", now.Add(time.Minute), 8, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.RolledCount != 2 || second.RolledCount != 2 || first.Seed != 7 {
		t.Fatal("descriptor range and seed must be retained")
	}
	if err = s.ScheduleSessions("i"); err != nil {
		t.Fatal(err)
	}
	if second.SessionNumber != 1 || second.Window.Sub(first.Window) < 30*time.Minute {
		t.Fatalf("sessions=%+v %+v", first, second)
	}
}

func testFormats() []Format {
	return []Format{{Kind: "test", MIMETypes: []string{"application/test"}, MaxBytes: 1, MaxDurationSecs: 1, MinWidth: 1, MinHeight: 1, MaxWidth: 1, MaxHeight: 1}}
}
