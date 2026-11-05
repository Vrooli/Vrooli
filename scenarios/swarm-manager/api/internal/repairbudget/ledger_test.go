package repairbudget

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"swarm-manager/internal/identity"
)

func testLimits() identity.RepairLimits {
	return identity.RepairLimits{
		PerFingerprint:         2,
		PerComponent:           2,
		PerEffort:              3,
		ComponentActiveMinutes: 90,
	}
}

func newLedger(t *testing.T, effortID string) *Ledger {
	t.Helper()
	ledger := New(effortID, filepath.Join(t.TempDir(), "repair-ledger.json"))
	ledger.SetClock(func() time.Time { return time.Date(2026, 9, 13, 18, 0, 0, 0, time.UTC) })
	return ledger
}

func repairEvent(attemptID, fingerprint, component string, newEvidence bool) Event {
	return Event{
		AttemptID:   attemptID,
		Fingerprint: fingerprint,
		Component:   component,
		Kind:        KindRepair,
		NewEvidence: newEvidence,
	}
}

func TestBeginChargesOnceAndSurvivesRestart(t *testing.T) {
	ledger := newLedger(t, "aquila")
	admission, err := ledger.Begin(testLimits(), repairEvent("a1", "fp-1", "swarm-manager", false))
	if err != nil {
		t.Fatalf("first begin: %v", err)
	}
	if !admission.Charged || admission.Totals.Effort != 1 {
		t.Fatalf("expected one charge, got %+v", admission)
	}

	// Same attempt reported again is idempotent: no double charge.
	repeat, err := ledger.Begin(testLimits(), repairEvent("a1", "fp-1", "swarm-manager", false))
	if err != nil {
		t.Fatalf("repeat begin: %v", err)
	}
	if repeat.Charged || !repeat.AlreadyRecorded || repeat.Totals.Effort != 1 {
		t.Fatalf("expected idempotent replay, got %+v", repeat)
	}

	// A new ledger over the same path recovers cumulative totals.
	restarted := New("aquila", ledger.path)
	restarted.SetClock(ledger.now)
	totals, err := restarted.Totals()
	if err != nil {
		t.Fatalf("totals after restart: %v", err)
	}
	if totals.Effort != 1 || totals.ByFingerprint["fp-1"] != 1 || totals.ByComponent["swarm-manager"] != 1 {
		t.Fatalf("totals did not survive restart: %+v", totals)
	}
}

func TestFingerprintLimitOpensCircuit(t *testing.T) {
	ledger := newLedger(t, "aquila")
	if _, err := ledger.Begin(testLimits(), repairEvent("a1", "fp-1", "swarm-manager", false)); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := ledger.Begin(testLimits(), repairEvent("a2", "fp-1", "swarm-manager", true)); err != nil {
		t.Fatalf("second: %v", err)
	}
	_, err := ledger.Begin(testLimits(), repairEvent("a3", "fp-1", "swarm-manager", true))
	if !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("expected exhausted fingerprint budget, got %v", err)
	}
}

func TestIndependentComponentStillEligible(t *testing.T) {
	limits := testLimits()
	limits.PerComponent = 1
	limits.PerFingerprint = 1
	limits.PerEffort = 5
	ledger := newLedger(t, "aquila")
	if _, err := ledger.Begin(limits, repairEvent("a1", "fp-1", "swarm-manager", false)); err != nil {
		t.Fatalf("component A: %v", err)
	}
	if _, err := ledger.Begin(limits, repairEvent("a2", "fp-2", "swarm-manager", false)); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("expected component A exhausted, got %v", err)
	}
	if _, err := ledger.Begin(limits, repairEvent("a3", "fp-3", "prompt-manager", false)); err != nil {
		t.Fatalf("independent component must remain eligible: %v", err)
	}
}

func TestEffortLimitAcrossComponents(t *testing.T) {
	limits := testLimits()
	limits.PerFingerprint = 5
	limits.PerComponent = 5
	limits.PerEffort = 2
	ledger := newLedger(t, "aquila")
	if _, err := ledger.Begin(limits, repairEvent("a1", "fp-1", "swarm-manager", false)); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := ledger.Begin(limits, repairEvent("a2", "fp-2", "prompt-manager", false)); err != nil {
		t.Fatalf("second: %v", err)
	}
	if _, err := ledger.Begin(limits, repairEvent("a3", "fp-3", "plan-manager", false)); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("expected effort budget opened, got %v", err)
	}
}

func TestStaleEvidenceRepeatRefused(t *testing.T) {
	ledger := newLedger(t, "aquila")
	if _, err := ledger.Begin(testLimits(), repairEvent("a1", "fp-1", "swarm-manager", false)); err != nil {
		t.Fatalf("first: %v", err)
	}
	_, err := ledger.Begin(testLimits(), repairEvent("a2", "fp-1", "swarm-manager", false))
	if !errors.Is(err, ErrStaleEvidence) {
		t.Fatalf("expected stale-evidence refusal, got %v", err)
	}
}

func TestIdentityConflictRefused(t *testing.T) {
	ledger := newLedger(t, "aquila")
	if _, err := ledger.Begin(testLimits(), repairEvent("a1", "fp-1", "swarm-manager", false)); err != nil {
		t.Fatalf("first: %v", err)
	}
	_, err := ledger.Begin(testLimits(), repairEvent("a1", "fp-2", "swarm-manager", false))
	if !errors.Is(err, ErrIdentityConflict) {
		t.Fatalf("expected identity conflict, got %v", err)
	}
}

func TestFinishIsIdempotentAndUnknownRefused(t *testing.T) {
	ledger := newLedger(t, "aquila")
	if _, err := ledger.Begin(testLimits(), repairEvent("a1", "fp-1", "swarm-manager", false)); err != nil {
		t.Fatalf("begin: %v", err)
	}
	first, err := ledger.Finish("a1", "failed", "evidence/one.md")
	if err != nil {
		t.Fatalf("finish: %v", err)
	}
	second, err := ledger.Finish("a1", "failed", "evidence/one.md")
	if err != nil {
		t.Fatalf("repeat finish: %v", err)
	}
	if first.At != second.At || second.Event != EventFinished {
		t.Fatalf("repeat finish was not idempotent: %+v vs %+v", first, second)
	}
	if _, err := ledger.Finish("missing", "failed"); !errors.Is(err, ErrUnknownAttempt) {
		t.Fatalf("expected unknown attempt refusal, got %v", err)
	}
	// An interrupted (started, never finished) attempt stays charged.
	totals, err := ledger.Totals()
	if err != nil {
		t.Fatalf("totals: %v", err)
	}
	if totals.Effort != 1 {
		t.Fatalf("interrupted attempt must remain charged, got %+v", totals)
	}
}

func TestSeparateProbeAllowance(t *testing.T) {
	ledger := newLedger(t, "aquila")
	probe := func(id string) Event {
		return Event{AttemptID: id, Fingerprint: "fp-probe", Component: "swarm-manager", Kind: KindProbe}
	}
	if _, err := ledger.Begin(testLimits(), probe("p1")); err != nil {
		t.Fatalf("probe 1: %v", err)
	}
	if _, err := ledger.Begin(testLimits(), probe("p2")); err != nil {
		t.Fatalf("probe 2: %v", err)
	}
	if _, err := ledger.Begin(testLimits(), probe("p3")); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("expected default probe allowance exhausted, got %v", err)
	}
	totals, _ := ledger.Totals()
	if totals.Effort != 0 || totals.ByKind[KindProbe] != 2 {
		t.Fatalf("probes must not consume repair effort: %+v", totals)
	}
}

func TestConcurrentBeginRespectsFingerprintLimit(t *testing.T) {
	limits := testLimits()
	limits.PerFingerprint = 3
	limits.PerComponent = 100
	limits.PerEffort = 100
	ledger := newLedger(t, "aquila")

	var wg sync.WaitGroup
	var mu sync.Mutex
	charged := 0
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := "attempt-" + string(rune('a'+n%26)) + string(rune('0'+n/26))
			admission, err := ledger.Begin(limits, repairEvent(id, "fp-shared", "swarm-manager", true))
			if err == nil && admission.Charged {
				mu.Lock()
				charged++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if charged != 3 {
		t.Fatalf("expected exactly 3 concurrent charges, got %d", charged)
	}
}

func plannedEvent(attemptID, fingerprint, component string) Event {
	event := repairEvent(attemptID, fingerprint, component, false)
	event.Class = ClassPlanned
	return event
}

func TestPlannedInfrastructureDoesNotConsumeDiversionBound(t *testing.T) {
	limits := testLimits()
	limits.PerFingerprint = 1
	limits.PerComponent = 1
	limits.PerEffort = 1
	ledger := newLedger(t, "aquila")

	// Declared planned infrastructure is delivery: it is retained but must not
	// consume the diversion bound, and repeat planned work is not stale.
	first, err := ledger.Begin(limits, plannedEvent("p1", "fp-planned", "swarm-manager"))
	if err != nil {
		t.Fatalf("first planned: %v", err)
	}
	if !first.Charged || first.Totals.Planned != 1 || first.Totals.Effort != 0 {
		t.Fatalf("planned charge must be retained separately: %+v", first.Totals)
	}
	if _, err := ledger.Begin(limits, plannedEvent("p2", "fp-planned", "swarm-manager")); err != nil {
		t.Fatalf("repeat planned work is not a stale diversion: %v", err)
	}

	// Diversion on the same component still gets its full reviewed allowance.
	if _, err := ledger.Begin(limits, repairEvent("d1", "fp-1", "swarm-manager", false)); err != nil {
		t.Fatalf("diversion after planned: %v", err)
	}
	_, err = ledger.Begin(limits, repairEvent("d2", "fp-1", "swarm-manager", true))
	if !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("expected diversion bound exhausted, got %v", err)
	}

	totals, err := ledger.Totals()
	if err != nil {
		t.Fatalf("totals: %v", err)
	}
	if totals.Effort != 1 || totals.Planned != 2 {
		t.Fatalf("expected 1 diversion / 2 planned, got %+v", totals)
	}
	if totals.ByClass[ClassDiversion] != 1 || totals.ByClass[ClassPlanned] != 2 {
		t.Fatalf("class totals wrong: %+v", totals.ByClass)
	}
}

func TestReclassificationRefused(t *testing.T) {
	limits := testLimits()
	ledger := newLedger(t, "aquila")

	if _, err := ledger.Begin(limits, repairEvent("d1", "fp-diversion", "swarm-manager", false)); err != nil {
		t.Fatalf("diversion first: %v", err)
	}
	// Same fingerprint cannot later be relabelled planned to reopen its circuit.
	if _, err := ledger.Begin(limits, plannedEvent("p1", "fp-diversion", "swarm-manager")); !errors.Is(err, ErrReclassificationRefused) {
		t.Fatalf("expected reclassification refusal, got %v", err)
	}

	if _, err := ledger.Begin(limits, plannedEvent("p2", "fp-planned", "swarm-manager")); err != nil {
		t.Fatalf("planned first: %v", err)
	}
	// A planned fingerprint cannot be relabelled diversion either.
	if _, err := ledger.Begin(limits, repairEvent("d2", "fp-planned", "swarm-manager", false)); !errors.Is(err, ErrReclassificationRefused) {
		t.Fatalf("expected reverse reclassification refusal, got %v", err)
	}

	// The same attempt identity with a different class is an identity conflict.
	if _, err := ledger.Begin(limits, repairEvent("shared", "fp-shared", "swarm-manager", false)); err != nil {
		t.Fatalf("shared diversion: %v", err)
	}
	if _, err := ledger.Begin(limits, plannedEvent("shared", "fp-shared", "swarm-manager")); !errors.Is(err, ErrIdentityConflict) {
		t.Fatalf("expected identity conflict for a class change on one attempt, got %v", err)
	}
}

func TestPreClassLedgerEventRemainsDiversion(t *testing.T) {
	ledger := newLedger(t, "aquila")
	// A durable event written before charge classes existed carries no class.
	preExisting := state{EffortID: "aquila", Events: []Event{{
		AttemptID:   "legacy",
		Fingerprint: "fp-legacy",
		Component:   "swarm-manager",
		Kind:        KindRepair,
		Event:       EventStarted,
		At:          "2026-09-13T00:00:00Z",
	}}}
	if err := ledger.save(preExisting); err != nil {
		t.Fatalf("save legacy ledger: %v", err)
	}
	totals, err := ledger.Totals()
	if err != nil {
		t.Fatalf("totals: %v", err)
	}
	if totals.Effort != 1 || totals.Planned != 0 {
		t.Fatalf("legacy event must stay diversion, got %+v", totals)
	}
	// Reclassifying the legacy fingerprint to planned is refused, so the
	// circuit cannot be reopened by upgrading the durable record.
	if _, err := ledger.Begin(testLimits(), plannedEvent("new", "fp-legacy", "swarm-manager")); !errors.Is(err, ErrReclassificationRefused) {
		t.Fatalf("expected refusal to reclassify legacy diversion, got %v", err)
	}
}

// newTimedLedger roots a ledger on a mutable clock so a test can advance active
// time without sleeping.
func newTimedLedger(t *testing.T, effortID string) (*Ledger, *time.Time) {
	t.Helper()
	now := time.Date(2026, 9, 13, 18, 0, 0, 0, time.UTC)
	ledger := New(effortID, filepath.Join(t.TempDir(), "repair-ledger.json"))
	ledger.SetClock(func() time.Time { return now })
	return ledger, &now
}

func TestComponentActiveMinutesOpensCircuit(t *testing.T) {
	ledger, now := newTimedLedger(t, "aquila")
	limits := testLimits()
	limits.ComponentActiveMinutes = 30
	limits.PerFingerprint = 5
	limits.PerComponent = 5
	limits.PerEffort = 5

	if _, err := ledger.Begin(limits, repairEvent("a1", "fp-1", "swarm-manager", false)); err != nil {
		t.Fatalf("first: %v", err)
	}
	// The open attempt accrues elapsed time; reaching the cap stops new repair.
	*now = now.Add(31 * time.Minute)
	if _, err := ledger.Begin(limits, repairEvent("a2", "fp-2", "swarm-manager", false)); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("expected component active-time bound, got %v", err)
	}
	// An independent component still gets its own full allowance.
	if _, err := ledger.Begin(limits, repairEvent("b1", "fp-3", "prompt-manager", false)); err != nil {
		t.Fatalf("independent component must remain eligible: %v", err)
	}
}

func TestFinishedRepairIntervalCountsTowardTimeBound(t *testing.T) {
	ledger, now := newTimedLedger(t, "aquila")
	limits := testLimits()
	limits.ComponentActiveMinutes = 60
	limits.PerFingerprint = 5
	limits.PerComponent = 5
	limits.PerEffort = 5

	if _, err := ledger.Begin(limits, repairEvent("a1", "fp-1", "swarm-manager", false)); err != nil {
		t.Fatalf("first: %v", err)
	}
	*now = now.Add(45 * time.Minute)
	if _, err := ledger.Finish("a1", "failed"); err != nil {
		t.Fatalf("finish: %v", err)
	}
	// 45 accrued minutes are under the 60-minute cap.
	if _, err := ledger.Begin(limits, repairEvent("a2", "fp-1", "swarm-manager", true)); err != nil {
		t.Fatalf("under the cap: %v", err)
	}
	// 45 finished + 20 open = 65 minutes reaches the cap.
	*now = now.Add(20 * time.Minute)
	if _, err := ledger.Begin(limits, repairEvent("a3", "fp-2", "swarm-manager", false)); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("expected finished-interval time bound, got %v", err)
	}
}

func TestPlannedWorkNotBoundedByActiveTime(t *testing.T) {
	ledger, now := newTimedLedger(t, "aquila")
	limits := testLimits()
	limits.ComponentActiveMinutes = 5
	limits.PerFingerprint = 5
	limits.PerComponent = 5
	limits.PerEffort = 5

	if _, err := ledger.Begin(limits, plannedEvent("p1", "fp-planned", "swarm-manager")); err != nil {
		t.Fatalf("planned first: %v", err)
	}
	*now = now.Add(60 * time.Minute)
	// Planned delivery is not diversion, so its elapsed time cannot open the
	// diversion time circuit.
	if _, err := ledger.Begin(limits, plannedEvent("p2", "fp-planned", "swarm-manager")); err != nil {
		t.Fatalf("planned repeat must not be time-bounded: %v", err)
	}
	// Diversion on the same component is still bounded once it accrues.
	if _, err := ledger.Begin(limits, repairEvent("d1", "fp-1", "swarm-manager", false)); err != nil {
		t.Fatalf("diversion start: %v", err)
	}
	*now = now.Add(6 * time.Minute)
	if _, err := ledger.Begin(limits, repairEvent("d2", "fp-2", "swarm-manager", false)); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("expected diversion time bound, got %v", err)
	}
}

func TestEffortFractionBoundsDiversionAcrossComponents(t *testing.T) {
	ledger, now := newTimedLedger(t, "aquila")
	limits := testLimits()
	limits.ComponentActiveMinutes = 1000
	limits.ApprovedActiveMinutes = 100
	limits.EffortFraction = 0.2 // 20-minute effort-wide diversion bound
	limits.PerFingerprint = 5
	limits.PerComponent = 5
	limits.PerEffort = 5

	if _, err := ledger.Begin(limits, repairEvent("a1", "fp-1", "swarm-manager", false)); err != nil {
		t.Fatalf("a1: %v", err)
	}
	*now = now.Add(12 * time.Minute)
	if _, err := ledger.Begin(limits, repairEvent("a2", "fp-2", "prompt-manager", false)); err != nil {
		t.Fatalf("a2 under effort bound: %v", err)
	}
	*now = now.Add(9 * time.Minute) // a1 21m + a2 9m = 30m >= 20m
	if _, err := ledger.Begin(limits, repairEvent("a3", "fp-3", "plan-manager", false)); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("expected effort-fraction bound, got %v", err)
	}

	totals, err := ledger.Totals()
	if err != nil {
		t.Fatalf("totals: %v", err)
	}
	if totals.EffortActiveMinutes < 30 {
		t.Fatalf("expected >= 30 accrued diversion minutes, got %.1f", totals.EffortActiveMinutes)
	}
	if totals.ComponentActiveMinutes["swarm-manager"] < 21 {
		t.Fatalf("expected swarm-manager minutes recorded, got %+v", totals.ComponentActiveMinutes)
	}
}

func TestUnboundedEffortLeavesFractionUnenforceable(t *testing.T) {
	ledger, now := newTimedLedger(t, "aquila")
	limits := testLimits()
	limits.ComponentActiveMinutes = 1000
	limits.EffortFraction = 0.2 // no approved denominator: unbounded
	limits.PerFingerprint = 5
	limits.PerComponent = 5
	limits.PerEffort = 5

	if _, err := ledger.Begin(limits, repairEvent("a1", "fp-1", "swarm-manager", false)); err != nil {
		t.Fatalf("a1: %v", err)
	}
	*now = now.Add(500 * time.Minute)
	if _, err := ledger.Begin(limits, repairEvent("a2", "fp-2", "prompt-manager", false)); err != nil {
		t.Fatalf("unbounded effort must not enforce the fraction: %v", err)
	}
}
