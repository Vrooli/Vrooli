package outcomeledger

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

var quick = PowerOptions{Sims: 200, Permutations: 199, Resamples: 199}

func TestPowerRisesWithDataAndTheNullStaysQuiet(t *testing.T) {
	pilot := Pilot{CostCV: 1.5, AcceptedShare: 0.85, UnitsPerDay: 1}
	small, large := SimulateDetection(pilot, 20, 0.5, 0.3, quick), SimulateDetection(pilot, 280, 0.5, 0.3, quick)
	if !(small < 0.6 && large > 0.95) {
		t.Fatalf("power at a 50%% effect: %d units per arm %.2f, 280 per arm %.2f", 20, small, large)
	}
	if got := SimulateDetection(pilot, 130, 0, 0.3, quick); got > 0.04 {
		t.Fatalf("a true null read as 'cheaper looks better' in %.0f%% of simulations", got*100)
	}
	// A 30% effect sits at the reading threshold itself, so it can never be
	// read reliably: only large effects are detectable.
	if got := SimulateDetection(pilot, 700, 0.3, 0.3, quick); got > 0.7 {
		t.Fatalf("a 30%% effect at the threshold read reliably (%.2f), which would mean the gate is not binding", got)
	}
}

func TestPlannedLookGivesAboutEightyPercentPowerForAHalvingAtPooledVariance(t *testing.T) {
	pilot := Pilot{CostCV: DefaultCostCV, AcceptedShare: 0.85, UnitsPerDay: 1}
	cfg := DefaultExperimentConfig()
	perArm := int(float64(cfg.MinAcceptedPerArm)/pilot.AcceptedShare + 0.5)
	if power := SimulateDetection(pilot, perArm, 0.5, cfg.MinEffect, PowerOptions{Sims: 1000, Permutations: 399, Resamples: 399}); power < 0.77 || power > 0.9 {
		t.Fatalf("at the look of %d accepted (%d units) per arm and CV %.1f, power to detect a halving is %.2f; the look was set for about 0.8", cfg.MinAcceptedPerArm, perArm, pilot.CostCV, power)
	}
}

func TestEquivalenceNeedsFarMoreDataThanADifference(t *testing.T) {
	pilot := Pilot{CostCV: 1.0, AcceptedShare: 0.85, UnitsPerDay: 1}
	if detect, same := SimulateDetection(pilot, 90, 0.5, 0.3, quick), SimulateEquivalence(pilot, 90, 0.3, quick); detect < 0.8 || same > 0.3 {
		t.Fatalf("at 90 per arm a halving reads at %.2f and 'about the same' only at %.2f", detect, same)
	}
	if same := SimulateEquivalence(pilot, 500, 0.3, quick); same < 0.8 {
		t.Fatalf("at 500 per arm 'about the same' reads at %.2f", same)
	}
}

// A challenger accepted less often pays for it in cost per accepted outcome: a
// halving of cost per unit with 0.6x the acceptance is truly a 0.83x cost ratio,
// not 0.5x, and the simulation recovers that it is rarely readable as cheaper.
func TestPowerAllowsUnequalAcceptance(t *testing.T) {
	pilot := Pilot{CostCV: 1.0, AcceptedShare: 0.9, UnitsPerDay: 1}
	equal := SimulateDetection(pilot, 210, 0.5, 0.3, quick)
	lower := quick
	lower.ChallengerAcceptance = 0.6
	unequal := SimulateDetection(pilot, 210, 0.5, 0.3, lower)
	if equal < 0.9 || unequal > 0.15 {
		t.Fatalf("equal acceptance %.2f, 0.6x acceptance %.2f", equal, unequal)
	}
}

func TestPilotFromUnitsMeasuresPaceVarianceAndFlagsAThinPilot(t *testing.T) {
	var units []Unit
	for i := 0; i < 30; i++ {
		units = append(units, Unit{Outcome: OutcomeAccepted, Points: float64(1 + i%3)}) // mean 2, sd ~0.83
	}
	units = append(units, Unit{Outcome: OutcomeUnknown, Points: 99})
	pilot := PilotFromUnits(units, 30)
	if pilot.Assumed || pilot.Units != 30 || pilot.UnitsPerDay != 1 || pilot.CostCV < 0.35 || pilot.CostCV > 0.5 || pilot.AcceptedShare != 1 {
		t.Fatalf("pilot = %+v", pilot)
	}
	thin := PilotFromUnits(units[:4], 30)
	if !thin.Assumed || thin.CostCV != DefaultCostCV {
		t.Fatalf("a thin pilot must fall back to the stated variance and say so: %+v", thin)
	}
	if empty := PilotFromUnits(nil, 30); empty.UnitsPerDay != 0 || !empty.Assumed {
		t.Fatalf("empty = %+v", empty)
	}
	// Lifecycle completion is not acceptance: the shares stay apart.
	mixed := []Unit{{Outcome: OutcomeAccepted, Points: 1}, {Outcome: OutcomeCompleted, Points: 1}, {Outcome: OutcomeCompleted, Points: 1}, {Outcome: OutcomeFailed, Points: 1}}
	if got := PilotFromUnits(mixed, 30); got.AcceptedShare != 0.25 || got.CompletedShare != 0.75 {
		t.Fatalf("shares = %+v", got)
	}
}

func TestPlanStatesATimeFromThePace(t *testing.T) {
	fast := PlanExperiment(Pilot{CostCV: 1.0, AcceptedShare: 0.85, CompletedShare: 0.85, UnitsPerDay: 2.3, Units: 68}, 0.5, 0.5, 0.3, 0.8, 75, quick)
	slow := PlanExperiment(Pilot{CostCV: 1.0, AcceptedShare: 0.85, CompletedShare: 0.85, UnitsPerDay: 0.23, Units: 7}, 0.5, 0.5, 0.3, 0.8, 75, quick)
	if !fast.Detect.Reached || fast.Detect.Days <= 0 || fast.Detect.LowDays > fast.Detect.Days || fast.PowerAtLook < 0.7 {
		t.Fatalf("fast plan = %+v", fast)
	}
	if ratio := slow.Detect.Days / fast.Detect.Days; ratio < 9 || ratio > 11 {
		t.Fatalf("ten times slower work should take ten times longer: %.1f", ratio)
	}
	none := PlanExperiment(Pilot{CostCV: 1.0, AcceptedShare: 0.85}, 0.5, 0.5, 0.3, 0.8, 75, quick)
	if none.Detect.Reached {
		t.Fatal("a role with no finished work has no time to a verdict")
	}
}

// Known answer: the look needs 75 accepted per arm; at 2 units a day split 50/50
// and half of them accepted, that is 150 units per arm, 150 days.
func TestPlanCountsOnlyAcceptedOutcomesTowardTheLook(t *testing.T) {
	pilot := Pilot{CostCV: 0.5, AcceptedShare: 0.5, CompletedShare: 1, UnitsPerDay: 2, Units: 60}
	plan := PlanExperiment(pilot, 0.5, 0.6, 0.3, 0.8, 75, quick)
	if plan.Detect.PerArm != 150 || plan.Detect.Days != 150 {
		t.Fatalf("verified look: %d per arm, %.1f days", plan.Detect.PerArm, plan.Detect.Days)
	}
	if plan.CompletedOnly.PerArm != 75 || plan.CompletedOnly.Days != 75 {
		t.Fatalf("completed-only look: %d per arm, %.1f days", plan.CompletedOnly.PerArm, plan.CompletedOnly.Days)
	}
}

// A role that only ever finishes, with nothing checking the result, can never
// reach a verdict, and the plan says that rather than a long number.
func TestPlanSaysNoVerdictIsReachableWithoutAcceptedOutcomes(t *testing.T) {
	plan := PlanExperiment(Pilot{CostCV: 1, AcceptedShare: 0, CompletedShare: 1, UnitsPerDay: 2.3, Units: 68}, 0.5, 0.5, 0.3, 0.8, 75, quick)
	if plan.Detect.Reached || plan.PowerAtLook != 0 || len(plan.Notes) == 0 || !strings.Contains(plan.Notes[0], "no verdict is reachable") {
		t.Fatalf("plan = %+v", plan)
	}
	if !plan.CompletedOnly.Reached || plan.CompletedPower <= 0 {
		t.Fatalf("the weaker reading is still stated: %+v", plan)
	}
}

// TestPowerReport prints the tables in docs/reference/operational-stats.md.
// LEARN_POWER_REPORT=1 runs it.
func TestPowerReport(t *testing.T) {
	if os.Getenv("LEARN_POWER_REPORT") != "1" {
		t.Skip("set LEARN_POWER_REPORT=1 to print the power tables")
	}
	opts := PowerOptions{Sims: 1000, Permutations: 499, Resamples: 499}
	cfg := DefaultExperimentConfig()
	var table strings.Builder
	for _, cv := range []float64{1.0, 1.5, 1.84} {
		pilot := Pilot{CostCV: cv, AcceptedShare: 0.85, UnitsPerDay: 1}
		for _, effect := range []float64{0.4, 0.5, 0.6} {
			fmt.Fprintf(&table, "CV %.2f, reduction %.0f%%:", cv, effect*100)
			for _, n := range []int{40, 70, 100, 130, 180, 280} {
				fmt.Fprintf(&table, "  n=%d %.2f", n, SimulateDetection(pilot, n, effect, cfg.MinEffect, opts))
			}
			table.WriteByte('\n')
		}
		fmt.Fprintf(&table, "CV %.2f, about the same:", cv)
		for _, n := range []int{130, 210, 380, 700} {
			fmt.Fprintf(&table, "  n=%d %.2f", n, SimulateEquivalence(pilot, n, cfg.MinEffect, opts))
		}
		fmt.Fprintf(&table, "\nCV %.2f, null, false cheaper: n=130 %.3f\n", cv, SimulateDetection(pilot, 130, 0, cfg.MinEffect, opts))
		lower := opts
		lower.ChallengerAcceptance = 0.8
		fmt.Fprintf(&table, "CV %.2f, 50%% per unit, 0.8x acceptance: n=130 %.2f\n", cv, SimulateDetection(pilot, 130, 0.5, cfg.MinEffect, lower))
	}
	two := opts
	two.Alpha = 0.0125
	fmt.Fprintf(&table, "two roles, CV 1.5, halving: n=130 %.2f\n", SimulateDetection(Pilot{CostCV: 1.5, AcceptedShare: 0.85}, 130, 0.5, cfg.MinEffect, two))
	t.Log("\n" + table.String())
}
