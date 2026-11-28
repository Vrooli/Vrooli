package outcomeledger

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/vrooli/cli-core/usagemode"
)

// synth describes one arm of a synthetic comparison with a known truth.
type synth struct {
	n        int
	cost     float64 // mean points per unit
	cv       float64
	accepted float64 // share accepted
	rework   float64 // share of accepted units that took more than one attempt
}

func lognormal(rng *rand.Rand, mean, cv float64) float64 {
	sigma2 := math.Log(1 + cv*cv)
	return mean * math.Exp(-sigma2/2+math.Sqrt(sigma2)*rng.NormFloat64())
}

var experimentStart = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

// units builds one arm's units in a mode, one an hour from the start. Outcomes
// are known; a share of the accepted ones are reworked (more than one attempt).
func (s synth) units(rng *rand.Rand, mode, arm string) []Unit {
	out := make([]Unit, s.n)
	for i := range out {
		u := Unit{ID: fmt.Sprintf("%s-%s-%d", mode, arm, i), Key: "x", Runner: "codex", Path: PathHeadless, Source: SourceValidation, Attempts: 1, Start: experimentStart.Add(time.Duration(i) * time.Hour),
			Points: lognormal(rng, s.cost, s.cv), Arm: arm, ArmMode: mode, ArmBasis: ArmBasisCanary, Outcome: OutcomeFailed}
		if rng.Float64() < s.accepted {
			u.Outcome = OutcomeAccepted
		}
		if rng.Float64() < s.rework {
			u.Attempts = 2
		}
		out[i] = u
	}
	return out
}

func compare(rng *rand.Rand, inc, chal synth) []Unit {
	return append(inc.units(rng, "um-t", usagemode.ArmIncumbent), chal.units(rng, "um-t", usagemode.ArmChallenger)...)
}

func analyze(units []Unit) ExperimentReport {
	return AnalyzeExperiment("um-t", units, nil, nil, DefaultExperimentConfig())
}

var baseArm = synth{n: 220, cost: 10, cv: 0.8, accepted: 0.85, rework: 0.2}

func TestSyntheticCheaperChallengerIsRecovered(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	chal := baseArm
	chal.cost = 5 // truth: 50% cheaper per unit, same acceptance and rework
	report := analyze(compare(rng, baseArm, chal))
	if report.Verdict != ExperimentCheaperBetter {
		t.Fatalf("verdict = %s (%s)", report.Verdict, report.Reason)
	}
	if !report.Cost.Available || report.Cost.Low > 0.5 || report.Cost.High < 0.5 {
		t.Fatalf("the 90%% range %.2f to %.2f must contain the true ratio 0.50 (estimate %.2f)", report.Cost.Low, report.Cost.High, report.Cost.Value)
	}
	if report.CostP > 0.01 || report.Design != DesignRandomized {
		t.Fatalf("p = %.4f design %q", report.CostP, report.Design)
	}
}

func TestSyntheticDearerChallengerReadsHigherLooksBetter(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	chal := baseArm
	chal.cost = 17 // truth: the lower model burns 70% more per unit (retries, longer runs)
	report := analyze(compare(rng, baseArm, chal))
	if report.Verdict != ExperimentHigherBetter || report.Cost.Low <= 1 {
		t.Fatalf("verdict = %s ratio %.2f..%.2f (%s)", report.Verdict, report.Cost.Low, report.Cost.High, report.Reason)
	}
}

func TestSyntheticReworkRiseReadsHigherLooksBetterEvenWhenCheaper(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	chal := baseArm
	chal.cost, chal.rework = 8, 0.55 // 20% cheaper per unit but 35 points more rework
	report := analyze(compare(rng, baseArm, chal))
	if report.Verdict != ExperimentHigherBetter || report.ReworkDiff.Value < 0.2 {
		t.Fatalf("verdict = %s rework diff %.2f (%s)", report.Verdict, report.ReworkDiff.Value, report.Reason)
	}
}

func TestSyntheticNullWithEnoughDataReadsAboutTheSame(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	same := baseArm
	same.cv = 0.3 // the look is fixed, so only a quiet cost can be shown the same there
	report := analyze(compare(rng, same, same))
	if report.Verdict != ExperimentAboutSame {
		t.Fatalf("verdict = %s ratio %.2f..%.2f p=%.3f (%s)", report.Verdict, report.Cost.Low, report.Cost.High, report.CostP, report.Reason)
	}
}

// A null at the planned look with a noisy cost has low power to show anything: it
// reads inconclusive, not "about the same". A false "looks better" stays at or
// below the family-wise level across both tests.
func TestSyntheticNullWithLowPowerIsInconclusiveAndRarelyADifference(t *testing.T) {
	falseDirectional, inconclusive, same := 0, 0, 0
	for seed := int64(0); seed < 200; seed++ {
		rng := rand.New(rand.NewSource(100 + seed))
		arm := baseArm
		arm.cv = 1.5
		cfg := DefaultExperimentConfig()
		cfg.Permutations = 499
		r := AnalyzeExperiment("um-t", compare(rng, arm, arm), nil, nil, cfg)
		switch r.Verdict {
		case ExperimentCheaperBetter, ExperimentHigherBetter:
			falseDirectional++
		case ExperimentInconclusive:
			inconclusive++
		case ExperimentAboutSame:
			same++
		}
	}
	if rate := float64(falseDirectional) / 200; rate > 0.05 {
		t.Fatalf("false directional verdicts = %d of 200 (%.1f%%), want at most 5%%", falseDirectional, rate*100)
	}
	if inconclusive < 150 || same > 20 {
		t.Fatalf("a noisy null must read inconclusive: inconclusive %d, about the same %d of 200", inconclusive, same)
	}
}

func TestLowDataIsNotEnoughDataAndSaysHowMuchMore(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	chal := baseArm
	chal.n, chal.cost = 20, 2 // a huge effect, far too few units
	small := baseArm
	small.n = 20
	report := analyze(compare(rng, small, chal))
	if report.LookReached {
		t.Fatal("twenty units per arm cannot reach the planned look")
	}
	if report.Verdict != ExperimentNotEnough || report.NeedAccepted <= 0 || report.NeedAccepted > DefaultExperimentConfig().MinAcceptedPerArm {
		t.Fatalf("low data: verdict %s need %d (%s)", report.Verdict, report.NeedAccepted, report.Reason)
	}
	empty := analyze(nil)
	if empty.Verdict != ExperimentNotEnough || empty.NeedAccepted != DefaultExperimentConfig().MinAcceptedPerArm {
		t.Fatalf("no data: %+v", empty)
	}
}

func TestUnknownOutcomesStayOutOfTheArms(t *testing.T) {
	rng := rand.New(rand.NewSource(6))
	arm := baseArm
	arm.n = 400
	units := compare(rng, arm, arm)
	for i := range units {
		if i%3 == 0 {
			units[i].Outcome = OutcomeUnknown
		}
	}
	report := analyze(units)
	if !report.LookReached || report.Incumbent.Known >= report.Incumbent.Units || report.Challenger.Known >= report.Challenger.Units || report.Incumbent.Unknown == 0 {
		t.Fatalf("known %d of %d and %d of %d units", report.Incumbent.Known, report.Incumbent.Units, report.Challenger.Known, report.Challenger.Units)
	}
	if report.Verdict == ExperimentWithheld {
		t.Fatalf("equal unknown rates must not withhold: gap %.2f", report.UnknownRateGap)
	}
}

func TestOtherModesAndUnassignedWorkAreIgnored(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	units := compare(rng, baseArm, baseArm)
	other := baseArm.units(rng, "um-other", usagemode.ArmChallenger)
	for i := range other {
		other[i].Points *= 100
	}
	stray := baseArm.units(rng, "um-t", "")
	withNoise := analyze(append(append(append([]Unit(nil), units...), other...), stray...))
	if !reflect.DeepEqual(withNoise.Cost, analyze(units).Cost) {
		t.Fatal("work from another comparison or with no arm must not move this comparison")
	}
}

func TestAnalysisIsDeterministic(t *testing.T) {
	build := func() ExperimentReport {
		rng := rand.New(rand.NewSource(8))
		chal := baseArm
		chal.cost = 7
		return analyze(compare(rng, baseArm, chal))
	}
	if a, b := build(), build(); a.CostP != b.CostP || a.ReworkP != b.ReworkP || !reflect.DeepEqual(a.Cost, b.Cost) || a.Verdict != b.Verdict {
		t.Fatalf("the fixed seed must give identical results: %+v vs %+v", a, b)
	}
}

// Intention to treat: a session assigned to the challenger that ran at full
// power stays in the challenger arm, and Assemble reads the arm from the root run.
func TestAssembleKeepsWorkInItsAssignedArm(t *testing.T) {
	runs := []RunFacts{
		{RunID: "a", Path: PathImported, Harness: "codex", Requests: 3, WorkUnitID: "u1", UnitOutcome: OutcomeAccepted, Arm: usagemode.ArmChallenger, ArmMode: "um-t", ArmException: usagemode.ExceptionFullPower, ActualModel: "gpt-6.1-terra"},
		{RunID: "b", Path: PathImported, Harness: "codex", Requests: 2, WorkUnitID: "u2", UnitOutcome: OutcomeAccepted, Arm: usagemode.ArmIncumbent, ArmMode: "um-t", ActualModel: "gpt-6.1-terra"},
		{RunID: "c", Path: PathImported, Harness: "codex", Requests: 2, WorkUnitID: "u3", UnitOutcome: OutcomeAccepted, ActualModel: "gpt-6.1-terra"},
	}
	use := map[string]map[string]Use{"a": {"gpt-6.1-terra": {Points: 9}}, "b": {"gpt-6.1-terra": {Points: 9}}, "c": {"gpt-6.1-terra": {Points: 9}}}
	units, samples := Assemble(runs, use, ByArm)
	arms := map[string]string{}
	for _, u := range units {
		arms[u.ID] = u.Arm + "|" + u.ArmMode
	}
	if arms["u1"] != "challenger|um-t" || arms["u2"] != "incumbent|um-t" || arms["u3"] != "|" {
		t.Fatalf("unit arms = %v", arms)
	}
	for _, u := range units {
		if u.SpansArms {
			t.Fatalf("no unit here spans arms: %+v", u)
		}
	}
	keys := map[string]string{}
	for _, r := range samples {
		keys[r.ID] = r.Key
	}
	if keys["a"] != "challenger" || keys["c"] != "unassigned" {
		t.Fatalf("run keys = %v", keys)
	}
}

// A retry drew its own arm from its own run id, but it is another attempt at the
// same work and stays in the arm the unit started in. Runs joined for any other
// reason measure neither arm: the unit is flagged and excluded with a count.
func TestRetryStaysInItsArmAndAUnitSpanningArmsIsExcludedWithACount(t *testing.T) {
	at := experimentStart
	member := func(id, unit, arm, link string, parent string) RunFacts {
		return RunFacts{RunID: id, WorkUnitID: unit, Path: PathHeadless, Harness: "codex", Requests: 1, ActualModel: "m", Status: "complete", UnitOutcome: OutcomeAccepted, UnitSource: SourceValidation,
			Arm: arm, ArmMode: "um-t", ArmBasis: ArmBasisCanary, LinkKind: link, ParentRunID: parent, FirstActivity: at}
	}
	inc, chal := usagemode.ArmIncumbent, usagemode.ArmChallenger
	second := member("retry-b", "retry", inc, LinkWorkloadRetry, "")
	second.FirstActivity = at.Add(time.Hour)
	spanningChild := member("span-child", "span", inc, LinkChild, "span-root")
	spanningChild.FirstActivity = at.Add(time.Hour)
	late := member("late", "pre", chal, LinkResume, "")
	late.FirstActivity = at.Add(time.Hour)
	early := member("early", "pre", "", LinkRoot, "")
	early.ArmMode = ""
	runs := []RunFacts{member("retry-a", "retry", chal, LinkRoot, ""), second, member("span-root", "span", chal, LinkRoot, ""), spanningChild, early, late}
	use := map[string]map[string]Use{}
	for _, run := range runs {
		use[run.RunID] = map[string]Use{"m": {Points: 2, Consumption: 10}}
	}
	units, samples := Assemble(runs, use, ByArm)
	byID := map[string]Unit{}
	for _, unit := range units {
		byID[unit.ID] = unit
	}
	if retry := byID["retry"]; retry.Arm != chal || retry.SpansArms || retry.Attempts != 2 {
		t.Fatalf("a retry leaves its unit in the original arm: %+v", retry)
	}
	for _, sample := range samples {
		if sample.UnitID == "retry" && sample.Arm != chal {
			t.Fatalf("the retry's run stays in the original arm: %+v", sample)
		}
	}
	if !byID["span"].SpansArms || !byID["pre"].SpansArms {
		t.Fatalf("spanning units must be flagged: %+v %+v", byID["span"], byID["pre"])
	}
	report := AnalyzeExperiment("um-t", units, samples, nil, DefaultExperimentConfig())
	if report.SpanningUnits != 2 || report.Challenger.Units != 1 || report.Incumbent.Units != 0 {
		t.Fatalf("spanning %d, challenger %d, incumbent %d", report.SpanningUnits, report.Challenger.Units, report.Incumbent.Units)
	}
}

// sessionUnits builds whole day blocks of operator sessions with a block-level
// shock: sessions on one day resemble each other, whichever arm the day drew.
func sessionUnits(rng *rand.Rand, blocks, perBlock int, shock float64, arm func(block int) string) []Unit {
	var out []Unit
	for block := 0; block < blocks; block++ {
		day := experimentStart.Add(time.Duration(block) * 24 * time.Hour)
		level := 10 * math.Exp(shock*rng.NormFloat64())
		for i := 0; i < perBlock; i++ {
			out = append(out, Unit{ID: fmt.Sprintf("s-%d-%d", block, i), Key: "x", Path: PathImported, Source: SourceValidation, Outcome: OutcomeAccepted, Attempts: 1, Start: day.Add(time.Duration(i) * time.Minute),
				Points: lognormal(rng, level, 0.3), Arm: arm(block), ArmMode: "um-t", ArmBasis: ArmBasisCalendar})
		}
	}
	return out
}

// Sessions were randomized by the day, so the day is the unit: 8 blocks of 40
// correlated sessions are 8 observations, not 320. Permuting sessions would call
// the day-to-day swing a real difference.
func TestSessionBlocksPermuteWholeSoDayEffectsAreNotAResult(t *testing.T) {
	cfg := DefaultExperimentConfig()
	cfg.MinAcceptedPerArm = 150 // the look falls in the last block, so all 8 are read
	falsePositive, sessionLevel := 0, 0
	for seed := int64(0); seed < 60; seed++ {
		rng := rand.New(rand.NewSource(500 + seed))
		units := sessionUnits(rng, 8, 40, 0.7, func(block int) string {
			if block%2 == 0 {
				return usagemode.ArmChallenger
			}
			return usagemode.ArmIncumbent
		})
		report := AnalyzeExperiment("um-t", units, nil, nil, cfg)
		if report.Design != DesignDayBlockRandomized {
			t.Fatalf("design = %s", report.Design)
		}
		if report.Incumbent.Clusters != 4 || report.Challenger.Clusters != 4 {
			t.Fatalf("a block must count once: %d and %d clusters", report.Incumbent.Clusters, report.Challenger.Clusters)
		}
		if report.CostP <= report.AlphaPerTest {
			falsePositive++
		}
		// The same data with session-level labels, as if each session were randomized.
		var flat []Unit
		for _, unit := range units {
			unit.ArmBasis = ArmBasisCanary
			flat = append(flat, unit)
		}
		if AnalyzeExperiment("um-t", flat, nil, nil, cfg).CostP <= report.AlphaPerTest {
			sessionLevel++
		}
	}
	// 8 blocks split 4/4 give 70 labellings, so block p-values cannot go below 2/70.
	if falsePositive != 0 {
		t.Fatalf("block permutation read %d of 60 null comparisons as different", falsePositive)
	}
	if sessionLevel < 30 {
		t.Fatalf("the fixture must fool a session-level test: only %d of 60", sessionLevel)
	}
}

// A real effect still shows through block permutation when there are enough blocks.
func TestSessionBlocksRecoverAKnownEffect(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	cfg := DefaultExperimentConfig()
	cfg.MinAcceptedPerArm = 100
	units := sessionUnits(rng, 60, 6, 0.1, func(block int) string {
		if block%2 == 0 {
			return usagemode.ArmChallenger
		}
		return usagemode.ArmIncumbent
	})
	for i := range units {
		if units[i].Arm == usagemode.ArmChallenger {
			units[i].Points *= 0.5
		}
	}
	report := AnalyzeExperiment("um-t", units, nil, nil, cfg)
	if report.Verdict != ExperimentCheaperBetter || report.DecisionRange.Low > 0.5 || report.DecisionRange.High < 0.5 {
		t.Fatalf("verdict %s range %.2f..%.2f: %s", report.Verdict, report.DecisionRange.Low, report.DecisionRange.High, report.Reason)
	}
}

func withUnknown(units []Unit, arm string, every int) []Unit {
	seen := 0
	for i := range units {
		if units[i].Arm == arm {
			if seen%every == 0 {
				units[i].Outcome = OutcomeUnknown
			}
			seen++
		}
	}
	return units
}

func TestUnknownImbalanceWithholdsTheVerdict(t *testing.T) {
	rng := rand.New(rand.NewSource(21))
	arm, chal := baseArm, baseArm
	arm.n, chal.n = 400, 400
	chal.cost = 5
	balanced := analyze(compare(rng, arm, chal))
	if balanced.Verdict != ExperimentCheaperBetter {
		t.Fatalf("the fixture must read cheaper before any loss: %s", balanced.Verdict)
	}
	rng = rand.New(rand.NewSource(21))
	report := analyze(withUnknown(compare(rng, arm, chal), usagemode.ArmChallenger, 2))
	if report.Verdict != ExperimentWithheld || report.UnknownRateGap < 0.4 {
		t.Fatalf("half the challenger's outcomes went missing, verdict %s gap %.2f", report.Verdict, report.UnknownRateGap)
	}
	// Acceptance-unknown counts too: completed work carries no acceptance either.
	rng = rand.New(rand.NewSource(21))
	completed := compare(rng, arm, chal)
	seen := 0
	for i := range completed {
		if completed[i].Arm == usagemode.ArmChallenger {
			if seen%2 == 0 {
				completed[i].Outcome, completed[i].Source = OutcomeCompleted, SourceHeadlessStatus
			}
			seen++
		}
	}
	if got := analyze(completed); got.Verdict != ExperimentWithheld {
		t.Fatalf("lifecycle-only outcomes in one arm must withhold too: %s (%.2f)", got.Verdict, got.UnknownRateGap)
	}
}

// Known answer for the bound and the stated total: 2 accepted of 4 units at 10
// points each on the incumbent; 4 accepted of 6 at 5 points on the challenger.
func TestWorstCaseBoundAndTotalsIncludeUnknownUnits(t *testing.T) {
	unit := func(id, arm, outcome string, points float64) Unit {
		return Unit{ID: id, Arm: arm, ArmMode: "um-t", Outcome: outcome, Points: points, ArmBasis: ArmBasisCanary, Start: experimentStart}
	}
	inc, chal := usagemode.ArmIncumbent, usagemode.ArmChallenger
	units := []Unit{unit("i1", inc, OutcomeAccepted, 10), unit("i2", inc, OutcomeAccepted, 10), unit("i3", inc, OutcomeUnknown, 10), unit("i4", inc, OutcomeCompleted, 10),
		unit("c1", chal, OutcomeAccepted, 5), unit("c2", chal, OutcomeAccepted, 5), unit("c3", chal, OutcomeAccepted, 5), unit("c4", chal, OutcomeAccepted, 5), unit("c5", chal, OutcomeUnknown, 5), unit("c6", chal, OutcomeCompleted, 5)}
	report := AnalyzeExperiment("um-t", units, nil, nil, DefaultExperimentConfig())
	if report.Incumbent.TotalPoints != 40 || report.Incumbent.UnknownPoints != 10 || report.Challenger.TotalPoints != 30 || report.Challenger.UnknownPoints != 5 {
		t.Fatalf("totals: %+v %+v", report.Incumbent, report.Challenger)
	}
	// Best for the challenger: all 6 of its units accepted, only 2 of the incumbent's.
	// (30/6)/(40/2) = 0.25. Worst: only its 4 accepted, all 4 of the incumbent's.
	// (30/4)/(40/4) = 0.75.
	near(t, "bound low", report.CostBound.Low, 0.25, 1e-9)
	near(t, "bound high", report.CostBound.High, 0.75, 1e-9)
}

func TestVerdictIsReachableOnlyAtThePlannedLookAndLaterDataCannotMoveIt(t *testing.T) {
	rng := rand.New(rand.NewSource(31))
	arm, chal := baseArm, baseArm
	arm.n, chal.n = 400, 400
	chal.cost = 4
	units := compare(rng, arm, chal)
	sort.SliceStable(units, func(i, j int) bool { return units[i].Start.Before(units[j].Start) })
	cfg := DefaultExperimentConfig()
	cfg.Permutations = 999
	var first ExperimentReport
	reached := false
	// Reading the status after every few units, as a watcher would. A huge effect
	// is visible long before the look; no read before it may say so.
	for n := 20; n <= len(units); n += 20 {
		report := AnalyzeExperiment("um-t", units[:n], nil, nil, cfg)
		if !report.LookReached {
			if report.Verdict != ExperimentNotEnough || report.Cost.Available {
				t.Fatalf("after %d units, before the look: verdict %s", n, report.Verdict)
			}
			continue
		}
		if !reached {
			reached, first = true, report
			if first.Verdict != ExperimentCheaperBetter {
				t.Fatalf("the look read %s: %s", first.Verdict, first.Reason)
			}
			continue
		}
		if report.Verdict != first.Verdict || report.CostP != first.CostP || !reflect.DeepEqual(report.Cost, first.Cost) || !report.LookAt.Equal(first.LookAt) {
			t.Fatalf("after %d units the settled look moved: %+v vs %+v", n, report.Cost, first.Cost)
		}
		if n > 600 && report.UnitsAfterLook == 0 {
			t.Fatal("units after the look must be counted")
		}
	}
	if !reached {
		t.Fatal("the look was never reached")
	}
	again := AnalyzeExperiment("um-t", units, nil, nil, cfg)
	if !reflect.DeepEqual(again.Cost, first.Cost) || again.Verdict != first.Verdict {
		t.Fatal("repeated reads of the same data must agree")
	}
}

// The level divides over the cost and rework tests and over the comparisons
// started by the look. One started after the look cannot unsettle a verdict.
func TestLevelIsCorrectedAcrossTestsAndRolesAndLaterComparisonsDoNotMoveIt(t *testing.T) {
	rng := rand.New(rand.NewSource(41))
	units := compare(rng, baseArm, baseArm)
	cfg := DefaultExperimentConfig()
	one := AnalyzeExperiment("um-t", units, nil, nil, cfg)
	near(t, "one comparison", one.AlphaPerTest, 0.025, 1e-12)
	near(t, "its level", one.DecisionLevel, 0.95, 1e-12)
	cfg.ComparisonStarts = []time.Time{experimentStart, experimentStart.Add(time.Hour), experimentStart.Add(5000 * time.Hour)}
	two := AnalyzeExperiment("um-t", units, nil, nil, cfg)
	if two.Comparisons != 2 {
		t.Fatalf("family = %d, want the 2 started by the look", two.Comparisons)
	}
	near(t, "two comparisons", two.AlphaPerTest, 0.0125, 1e-12)
	near(t, "their level", two.DecisionLevel, 0.975, 1e-12)
	if width := func(r Range) float64 { return r.High - r.Low }; width(two.DecisionRange) <= width(one.DecisionRange) {
		t.Fatalf("a stricter level must widen the range: %+v vs %+v", two.DecisionRange, one.DecisionRange)
	}
	// A p between the corrected level and the uncorrected one reads inconclusive.
	report := ExperimentReport{Config: cfg, AlphaPerTest: 0.0125, DecisionLevel: 0.975, CostP: 0.03, Cost: Range{Available: true, Value: 0.6, Low: 0.45, High: 0.8}, DecisionRange: Range{Available: true, Value: 0.6, Low: 0.4, High: 0.9},
		ReworkDiff: Range{Available: true}, reworkDecision: Range{Available: true, High: 0.1}, ReworkP: 1}
	if got := decide(report); got.Verdict != ExperimentInconclusive {
		t.Fatalf("p=0.03 against 0.0125 read %s", got.Verdict)
	}
}

func TestDesignIsReadFromTheAssignmentBasis(t *testing.T) {
	for _, tc := range []struct {
		bases []string
		want  string
	}{
		{nil, DesignObservational},
		{[]string{ArmBasisCanary, ArmBasisCanary}, DesignRandomized},
		{[]string{ArmBasisCanary, ArmBasisCalendar}, DesignDayBlockRandomized},
		{[]string{ArmBasisMarker}, DesignDayBlockRandomized},
		{[]string{ArmBasisCanary, ""}, DesignObservational},
	} {
		var units []Unit
		for _, basis := range tc.bases {
			units = append(units, Unit{ArmBasis: basis})
		}
		if got := DesignOf(units); got != tc.want {
			t.Fatalf("%v: %s, want %s", tc.bases, got, tc.want)
		}
	}
}

// Acceptance that rests only on the agent's own verdict is counted apart, so a
// reader can see how much of it is assumed.
func TestArmReportsHowMuchAcceptanceRestsOnTheAgentsOwnVerdict(t *testing.T) {
	small := synth{n: 6, cost: 1, cv: 0.5, accepted: 1}
	units := compare(rand.New(rand.NewSource(9)), small, small)
	for i := range units {
		if units[i].Arm == usagemode.ArmChallenger && i%2 == 0 {
			units[i].Source = SourceAgentVerdict
		}
	}
	report := analyze(units)
	if report.Incumbent.Accepted != 6 || report.Incumbent.AssumedAccepted != 0 {
		t.Fatalf("incumbent: %+v", report.Incumbent)
	}
	if report.Challenger.Accepted != 6 || report.Challenger.AssumedAccepted != 3 {
		t.Fatalf("challenger: %+v", report.Challenger)
	}
}
