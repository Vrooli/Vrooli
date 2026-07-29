package channelmanager

import (
	"errors"
	"testing"
	"time"
)

func fixture(t *testing.T) *Service {
	t.Helper()
	s, e := New([]Platform{{ID: "x", DailyCeiling: 2, ActionKinds: []string{"engage", "publish"}, Formats: testFormats()}}, []Program{{ID: "x-warm", PlatformID: "x", Preconditions: []string{"region"}, Phases: []Phase{{ID: "one", Allowed: []string{"engage"}, Forbidden: []string{"publish"}, CountMax: 2}}, Provenance: Provenance{"operator", "speculative", "2026-07-28", "five runs", []string{"operator note"}}}})
	if e != nil {
		t.Fatal(e)
	}
	return s
}

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
	id, e := s.Release("i", "main", "release-1", now.AddDate(0, 0, 1))
	if e != nil {
		t.Fatal(e)
	}
	again, e := s.Release("i", "main", "release-1", now.AddDate(0, 0, 1))
	if e != nil || id != again {
		t.Fatal("release must be idempotent")
	}
}

// [REQ:CHANMGR-P0-011] A gate waits, resolves deterministically, and turns a
// bounded inconclusive series into terminal quarantine with pending work gone.
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
}

// [REQ:CHANMGR-P0-012] Elapsed time cannot grant eligibility; every declared
// criterion must be explicitly true before the identity becomes active.
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
