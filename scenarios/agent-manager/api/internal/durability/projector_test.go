package durability

import (
	"context"
	"testing"
	"time"
)

func TestProjectReportsPushbackAndKeepsLanesSeparate(t *testing.T) {
	epoch := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	got := Project(Boundary{Epoch: epoch, Reason: "capture began"}, Work{ID: "run-1", Subject: []string{"scenario-a"}, StartedAt: epoch.Add(time.Hour)}, []Evidence{
		{Kind: "pushback", Reference: "event-1", At: epoch.Add(time.Minute), Lane: LaneVerified},
		{Kind: "rework", Reference: "record-1", At: epoch.Add(2 * time.Minute), Lane: LaneObserved},
		{Kind: "friction", Reference: "episode-1", At: epoch.Add(3 * time.Minute), Lane: ""},
	})
	if got.Verdict != VerdictSignalPresent || got.SampleSize != 3 {
		t.Fatalf("verdict = %#v", got)
	}
	if got.Coverage != (Coverage{Verified: 1, Observed: 1, Unlinked: 1}) {
		t.Fatalf("coverage = %#v", got.Coverage)
	}
}

func TestProjectReportsDurableWithoutObservedSignals(t *testing.T) {
	got := Project(Boundary{Epoch: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)}, Work{ID: "run-2", Subject: []string{"path:scenarios/agent-manager"}, StartedAt: time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC), Lane: LaneVerified}, nil)
	if got.Verdict != VerdictDurable || got.SampleSize != 0 || got.NoVerdict {
		t.Fatalf("verdict = %#v", got)
	}
}

// Durable is a claim about observed work. Work we could not attribute to a run
// was never observed, so an empty evidence set says nothing about it and must
// not be reported as the best outcome.
func TestProjectDoesNotGradeUnattributedWorkAsDurable(t *testing.T) {
	got := Project(Boundary{Epoch: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)}, Work{ID: "run-3", StartedAt: time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC), Lane: LaneUnlinked}, nil)
	if got.Verdict != VerdictUnknown || got.UnknownReason != ReasonUnattributedWork {
		t.Fatalf("verdict = %#v", got)
	}
}

// An unset lane is unattributed, not a default-clean one.
func TestProjectTreatsMissingLaneAsUnlinked(t *testing.T) {
	got := Project(Boundary{Epoch: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)}, Work{ID: "run-4", StartedAt: time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)}, nil)
	if got.Lane != LaneUnlinked || got.Verdict != VerdictUnknown {
		t.Fatalf("verdict = %#v", got)
	}
}

// A failed evidence read is not a finding and not a clean bill of health. It
// must neither inflate the sample nor let an unread lane read as durable.
func TestProjectTreatsDegradedReadAsUnknownNotSignal(t *testing.T) {
	epoch := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	got := Project(Boundary{Epoch: epoch}, Work{ID: "run-5", Subject: []string{"path:scenarios/agent-manager"}, StartedAt: epoch.Add(time.Hour), Lane: LaneVerified}, []Evidence{
		{Kind: "swarm-evidence-unavailable", Reference: "swarm-manager://durability/evidence", At: epoch, Lane: LaneUnlinked, Degraded: true},
	})
	if got.Verdict != VerdictUnknown || got.UnknownReason != ReasonEvidenceDegraded {
		t.Fatalf("verdict = %#v", got)
	}
	if got.SampleSize != 0 || len(got.Degradations) != 1 {
		t.Fatalf("degraded read leaked into evidence: %#v", got)
	}
	if got.Coverage != (Coverage{}) {
		t.Fatalf("coverage = %#v", got.Coverage)
	}
}

// Adverse evidence is reportable even when another lane failed to read: the
// finding carries its own reference and does not depend on the missing lane.
func TestProjectReportsSignalDespiteDegradedLane(t *testing.T) {
	epoch := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	got := Project(Boundary{Epoch: epoch}, Work{ID: "run-6", StartedAt: epoch.Add(time.Hour), Lane: LaneVerified}, []Evidence{
		{Kind: "friction", Reference: "episode-1", At: epoch.Add(time.Minute), Lane: LaneVerified},
		{Kind: "swarm-evidence-unavailable", Reference: "swarm-manager://durability/evidence", At: epoch, Lane: LaneUnlinked, Degraded: true},
	})
	if got.Verdict != VerdictSignalPresent || got.SampleSize != 1 || len(got.Degradations) != 1 {
		t.Fatalf("verdict = %#v", got)
	}
}

func TestProjectDoesNotGradeBeforeEpoch(t *testing.T) {
	epoch := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	got := Project(Boundary{Epoch: epoch, Reason: "capture began"}, Work{ID: "old", StartedAt: epoch.Add(-time.Hour)}, []Evidence{{Kind: "pushback", Reference: "old-event", Lane: LaneVerified}})
	if !got.NoVerdict || got.Verdict != "" || got.BoundaryState != "before_analysis_epoch" {
		t.Fatalf("verdict = %#v", got)
	}
	if got.Boundary.Reason != "capture began" {
		t.Fatalf("boundary = %#v", got.Boundary)
	}
}

type stubBoundaryStore struct {
	stored *Boundary
	calls  int
}

func (s *stubBoundaryStore) Boundary(_ context.Context, seed Boundary) (Boundary, error) {
	s.calls++
	if s.stored == nil {
		copied := seed
		s.stored = &copied
	}
	return *s.stored, nil
}

// The epoch must not drift with wall-clock time or process restarts. Once a
// deployment establishes it, every later read returns the same value, so work
// cannot move in or out of scope without someone deciding it.
func TestResolveBoundaryIsStableAcrossReads(t *testing.T) {
	store := &stubBoundaryStore{}
	first, err := ResolveBoundary(context.Background(), store, time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	second, err := ResolveBoundary(context.Background(), store, time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !first.Epoch.Equal(second.Epoch) {
		t.Fatalf("epoch drifted: %s then %s", first.Epoch, second.Epoch)
	}
	if first.Epoch != time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("seed epoch = %s", first.Epoch)
	}
	if first.Reason == "" {
		t.Fatal("boundary reason must explain why the epoch exists")
	}
}

// Without a store there is no epoch, and inventing one would silently change
// which work is graded. Refusing is the only honest option.
func TestResolveBoundaryRefusesWithoutStore(t *testing.T) {
	if _, err := ResolveBoundary(context.Background(), nil, time.Now()); err == nil {
		t.Fatal("expected an error when no boundary store is configured")
	}
}

// A source that was never consulted is not a clean source. Reporting durable
// here would mean "we found no pushback" when the truth is "we never looked".
func TestProjectTreatsUnconsultedSourceAsUnknown(t *testing.T) {
	epoch := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	got := Project(Boundary{Epoch: epoch}, Work{ID: "run-7", Subject: []string{"path:scenarios/agent-manager"}, StartedAt: epoch.Add(time.Hour), Lane: LaneVerified}, []Evidence{
		{Kind: "swarm-evidence-not-configured", Reference: "swarm-manager://durability/evidence", At: epoch, Lane: LaneUnlinked, Degraded: true},
	})
	if got.Verdict != VerdictUnknown || got.UnknownReason != ReasonEvidenceDegraded {
		t.Fatalf("verdict = %#v", got)
	}
}

// The rework lane searches by subject, so subjectless work leaves it blind.
// An empty evidence set then means "nothing was searched for", which reads
// identically to a clean scan unless the verdict says otherwise.
func TestProjectDoesNotGradeSubjectlessWorkAsDurable(t *testing.T) {
	epoch := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	for name, subject := range map[string][]string{
		"nil":    nil,
		"empty":  {},
		"blank":  {"   "},
		"blanks": {"", "\t"},
	} {
		t.Run(name, func(t *testing.T) {
			got := Project(Boundary{Epoch: epoch}, Work{ID: "run-8", Subject: subject, StartedAt: epoch.Add(time.Hour), Lane: LaneVerified}, nil)
			if got.Verdict != VerdictUnknown || got.UnknownReason != ReasonNoSubject {
				t.Fatalf("verdict = %#v", got)
			}
		})
	}
}

// Unattributed work is the more fundamental gap: reporting a missing subject
// for work we could not attribute at all would send a consumer to fix the
// wrong thing.
func TestProjectPrefersUnattributedOverNoSubject(t *testing.T) {
	epoch := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	got := Project(Boundary{Epoch: epoch}, Work{ID: "run-9", StartedAt: epoch.Add(time.Hour), Lane: LaneUnlinked}, nil)
	if got.Verdict != VerdictUnknown || got.UnknownReason != ReasonUnattributedWork {
		t.Fatalf("verdict = %#v", got)
	}
}
