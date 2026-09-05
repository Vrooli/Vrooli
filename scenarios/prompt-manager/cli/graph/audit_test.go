package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	clitest "prompt-manager/cli/internal/testutil"
)

// cleanAuditContext wires every endpoint the sweep reads with in-band values.
func cleanAuditContext(t *testing.T) *clitest.Context {
	t.Helper()
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/operating-models/validate", operatingModelValidationResponse{
		Validation: operatingGraphValidation{Errors: 0, Warnings: 0},
	})
	ctx.Respond("GET", "/operating-models", operatingModelListResponse{Models: []operatingModelDocument{
		{Team: "alpha", Graphs: []operatingGraphBlock{{Metadata: operatingGraphMetadata{Mode: "contract"}}}},
	}})
	ctx.Respond("GET", "/operating-models/diff", operatingModelDiffResponse{})
	ctx.Respond("GET", "/topics/graph", topicsGraphResponse{
		Validation: topicValidation{Errors: 0, Warnings: 0},
	})
	ctx.Respond("GET", "/operating-models/map", operatingMap{
		Teams: []operatingMapTeam{{ID: "alpha"}},
		Edges: []operatingMapEdge{{From: "alpha", To: "inbox/*"}},
	})
	ctx.Respond("GET", "/graph/orphans", []node{})
	ctx.Respond("GET", "/discovery-metrics", discoveryMetricsResponse{})
	// An objective that is unserved but carries a gap marker is in band: a
	// declared hole is a known disposition, not a defect.
	ctx.Respond("GET", "/objectives", objectiveResponse{
		Rows:     []objectiveRow{{ID: "T1", Served: true}, {ID: "T2", GapMarker: "pending-capability"}},
		Unserved: 1,
	})
	ctx.Respond("GET", "/orientation-cost", orientationCostReport{
		Teams: []orientationCost{{TeamID: "alpha", Composite: 42, ScenarioCoverage: 1}},
	})
	ctx.Respond("GET", "/operating-models/coverage", operatingModelCoverageResponse{
		Coverage: []operatingGraphCoverage{{Team: "alpha", Relationships: []operatingRelationshipCoverage{
			{Relationship: "topic_read", RuntimeDeclared: 2, GraphShown: 2, Matched: 2},
		}}},
	})
	concluded := "2026-08-01T00:00:00Z"
	ctx.Respond("GET", "/experiments", []experimentLivenessRow{
		{ID: "exp-1", Status: "concluded", ConcludedAt: &concluded},
	})
	stubCanonScript(t, "PASS one\nPassed: 9\nFailed: 0\n", nil)
	return ctx
}

// stubCanonScript replaces the shell-out seam for one test. Without it the
// sweep suite would run the real canon script and inherit its verdict, making
// an unrelated canon regression fail every audit test.
func stubCanonScript(t *testing.T, out string, err error) {
	t.Helper()
	prev := canonScriptRunner
	canonScriptRunner = func(string, string) ([]byte, error) { return []byte(out), err }
	t.Cleanup(func() { canonScriptRunner = prev })
}

func TestCmdAuditCoversEveryFrameworkHealthTarget(t *testing.T) {
	ctx := cleanAuditContext(t)
	stdout, _, err := clitest.Output(t, func() error {
		return cmdAudit(ctx, []string{"--json"})
	})
	if err != nil {
		t.Fatalf("cmdAudit: %v", err)
	}

	var report auditReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("audit output is not valid JSON: %v\n%s", err, stdout)
	}

	// The sweep is only trustworthy if it reports every canon target,
	// including the ones it cannot observe itself. A shorter list would let a
	// clean run read as full coverage.
	want := []string{
		"Operating-model contract validity",
		"Contract-mode coverage",
		"Graph/runtime drift",
		"Topic-flow declaration integrity",
		"Member-document conformance",
		"Prose/declaration coherence",
		"Cross-team coupling visibility",
		"Canon coherence",
		"Objective coverage",
		"Statically unreferenced skills",
		"Skill conditioning quality",
		"Skill-experiment loop liveness",
		"PoR entropy",
		"Instrument coverage",
		"Team orientation cost",
		"Discovery budget pressure",
		"Prompt structure invariant",
		"Catalogued rules with no finding",
		"Open-loop target count",
	}
	got := strings.Join(auditTargetTitles(report), "|")
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Fatalf("audit missing target %q; got %s", w, got)
		}
	}
	if len(report.Targets) != len(want) {
		t.Fatalf("target count = %d, want %d", len(report.Targets), len(want))
	}
}

func TestCmdAuditReportsUnsensoredTargetsRatherThanDroppingThem(t *testing.T) {
	ctx := cleanAuditContext(t)
	stdout, _, err := clitest.Output(t, func() error {
		return cmdAudit(ctx, []string{"--json"})
	})
	if err != nil {
		t.Fatalf("cmdAudit: %v", err)
	}
	var report auditReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if report.OutOfBand != 0 {
		t.Fatalf("clean fixtures should be in band, got %d out of band", report.OutOfBand)
	}
	// Seven targets are open-loop in FRAMEWORK_HEALTH today. Two were added with
	// the sensor-map plan: the prompt structure invariant, enforced in the
	// heartbeat unit suite rather than by a corpus-wide CLI sensor, and the
	// rules-with-no-finding trend, which cannot be banded from a single reading.
	// The seventh is the open-loop count itself, which is a trend target for the
	// same reason and therefore counts itself.
	if report.Unsensored != 7 {
		t.Fatalf("unsensored = %d, want 7", report.Unsensored)
	}
	// An unsensored target must name both the honest open-loop state and a dated
	// marker for the work that closes it. The marker prevents an intentionally
	// visible gap from becoming permanent background noise.
	for _, tgt := range report.Targets {
		if tgt.Status != auditStatusNoSensor {
			continue
		}
		// The flag is the routing input — pending-telemetry means build an
		// instrument, pending-baseline means run the sweep — so it is asserted
		// on the typed field, and the prose is required to agree with it.
		if tgt.HonestyFlag != auditHonestyTelemetry && tgt.HonestyFlag != auditHonestyBaseline {
			t.Fatalf("unsensored target must carry a typed honesty flag: %+v", tgt)
		}
		if !strings.HasPrefix(tgt.Observed, tgt.HonestyFlag) {
			t.Fatalf("observed prose must lead with the honesty flag %q: %q", tgt.HonestyFlag, tgt.Observed)
		}
		if !strings.HasPrefix(tgt.GapMarker, "2026-") {
			t.Fatalf("unsensored target must carry a dated gap marker: %+v", tgt)
		}
		if strings.TrimSpace(tgt.Deadband) == "" {
			t.Fatalf("unsensored target must declare its future deadband: %+v", tgt)
		}
	}
}

func TestCmdAuditFlagsOutOfBandTargetWithLeads(t *testing.T) {
	ctx := cleanAuditContext(t)
	ctx.Respond("GET", "/operating-models/validate", operatingModelValidationResponse{
		Validation: operatingGraphValidation{
			Errors: 2,
			Findings: []operatingGraphFinding{
				{Severity: "error", Rule: "graph_topic_catalog_purpose_drift", Detail: "row has no runtime entry"},
			},
		},
	})

	stdout, _, err := clitest.Output(t, func() error {
		return cmdAudit(ctx, nil)
	})
	if err != nil {
		t.Fatalf("cmdAudit: %v", err)
	}
	if !strings.Contains(stdout, "[OUT]") || !strings.Contains(stdout, "2 errors") {
		t.Fatalf("out-of-band target not surfaced: %s", stdout)
	}
	// An out-of-band row must name where the work goes, or the finding has no
	// route out of the report.
	if !strings.Contains(stdout, "actuator:") {
		t.Fatalf("out-of-band row missing actuator: %s", stdout)
	}
	if !strings.Contains(stdout, "graph_topic_catalog_purpose_drift") {
		t.Fatalf("out-of-band row missing lead detail: %s", stdout)
	}
}

func TestCmdAuditWritesArtifactToOutPath(t *testing.T) {
	ctx := cleanAuditContext(t)
	path := filepath.Join(t.TempDir(), "audit.json")

	if _, _, err := clitest.Output(t, func() error {
		return cmdAudit(ctx, []string{"--out", path})
	}); err != nil {
		t.Fatalf("cmdAudit: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	var report auditReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("artifact is not valid JSON: %v", err)
	}
	if report.SchemaVersion != auditSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", report.SchemaVersion, auditSchemaVersion)
	}
	if report.GeneratedAt == "" {
		t.Fatal("artifact missing generated_at")
	}
}

func TestAppendCappedLimitsLeadsToThree(t *testing.T) {
	var list []string
	for i := 0; i < 10; i++ {
		list = appendCapped(list, "lead")
	}
	if len(list) != 3 {
		t.Fatalf("len = %d, want 3", len(list))
	}
}

// An undeclared hole must take the sweep out of band. This is the check that
// makes editing OBJECTIVES.md consequential: adding an objective that no team
// declares now moves a sensor rather than waiting to be noticed by hand.
func TestCmdAuditFlagsAnUndeclaredObjectiveHole(t *testing.T) {
	ctx := cleanAuditContext(t)
	ctx.Respond("GET", "/objectives", objectiveResponse{
		Rows:       []objectiveRow{{ID: "T4"}},
		Unserved:   1,
		Undeclared: 1,
	})
	stdout, _, err := clitest.Output(t, func() error { return cmdAudit(ctx, []string{"--json"}) })
	if err != nil {
		t.Fatalf("cmdAudit: %v", err)
	}
	var report auditReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, tgt := range report.Targets {
		if tgt.Target != "Objective coverage" {
			continue
		}
		if tgt.Status != auditStatusOutOfBand {
			t.Fatalf("status = %q, want out-of-band", tgt.Status)
		}
		if len(tgt.Detail) == 0 {
			t.Fatal("an unserved objective must reach the reader as a detail line")
		}
		return
	}
	t.Fatal("audit did not report Objective coverage")
}

// A team that declares no objective is the upward half of the coverage rule:
// effort that traces to no stated intent.
func TestCmdAuditFlagsAnUnattachedTeam(t *testing.T) {
	ctx := cleanAuditContext(t)
	ctx.Respond("GET", "/objectives", objectiveResponse{
		Rows:            []objectiveRow{{ID: "T1", Served: true}},
		UnattachedTeams: []string{"orphan-team"},
	})
	stdout, _, err := clitest.Output(t, func() error { return cmdAudit(ctx, []string{"--json"}) })
	if err != nil {
		t.Fatalf("cmdAudit: %v", err)
	}
	var report auditReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, tgt := range report.Targets {
		if tgt.Target == "Objective coverage" && tgt.Status != auditStatusOutOfBand {
			t.Fatalf("status = %q, want out-of-band", tgt.Status)
		}
	}
}

// Orientation cost must report every team, not the usual three-lead cap: the
// record is what the next cycle diffs against, and a truncated one silently
// drops teams out of the trend.
func TestCmdAuditRecordsEveryTeamsOrientationCost(t *testing.T) {
	ctx := cleanAuditContext(t)
	ctx.Respond("GET", "/orientation-cost", orientationCostReport{Teams: []orientationCost{
		{TeamID: "a"}, {TeamID: "b"}, {TeamID: "c"}, {TeamID: "d"}, {TeamID: "e"}, {TeamID: "f"},
	}})
	stdout, _, err := clitest.Output(t, func() error { return cmdAudit(ctx, []string{"--json"}) })
	if err != nil {
		t.Fatalf("cmdAudit: %v", err)
	}
	var report auditReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, tgt := range report.Targets {
		if tgt.Target != "Team orientation cost" {
			continue
		}
		if len(tgt.Detail) != 6 {
			t.Fatalf("detail lines = %d, want 6 (one per team)", len(tgt.Detail))
		}
		return
	}
	t.Fatal("audit did not report Team orientation cost")
}

// writeBaseline runs one sweep and persists it, returning the artifact path the
// next sweep bands against. Round-tripping through --out is deliberate: it
// proves the two flags agree on a shape rather than testing a hand-built one.
func writeBaseline(t *testing.T, ctx *clitest.Context) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "baseline.json")
	if _, _, err := clitest.Output(t, func() error {
		return cmdAudit(ctx, []string{"--json", "--out", path})
	}); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	return path
}

func auditWithBaseline(t *testing.T, ctx *clitest.Context, baseline string) auditReport {
	t.Helper()
	stdout, _, err := clitest.Output(t, func() error {
		return cmdAudit(ctx, []string{"--json", "--baseline", baseline})
	})
	if err != nil {
		t.Fatalf("cmdAudit: %v", err)
	}
	var report auditReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return report
}

func findTarget(t *testing.T, r auditReport, name string) auditTarget {
	t.Helper()
	for _, tgt := range r.Targets {
		if tgt.Target == name {
			return tgt
		}
	}
	t.Fatalf("audit did not report %q", name)
	return auditTarget{}
}

// A trend target is open-loop until a baseline arrives, then bands. This is the
// whole point of --baseline: three targets reported pending-baseline forever
// while the readings they needed were being written to a record nobody read.
func TestCmdAuditBandsTrendTargetsAgainstBaseline(t *testing.T) {
	ctx := cleanAuditContext(t)
	baseline := writeBaseline(t, ctx)

	report := auditWithBaseline(t, ctx, baseline)
	tgt := findTarget(t, report, "Team orientation cost")
	if tgt.Status != auditStatusInBand {
		t.Fatalf("flat orientation cost should band in-band, got %q (%s)", tgt.Status, tgt.Observed)
	}
	if report.BaselineFrom == "" {
		t.Fatal("a banded sweep must name the baseline it compared against")
	}
	if tgt.GapMarker != "" {
		t.Fatalf("a banded target must drop the gap marker that asked for the baseline: %q", tgt.GapMarker)
	}
}

// A rise only counts where the guard series also rose. Orientation cost that
// grew while scenario coverage held flat is expected growth, and banding it
// out would make the deadband fire on the thing it explicitly permits.
func TestCmdAuditGuardSuppressesRiseWithoutCoverageGrowth(t *testing.T) {
	ctx := cleanAuditContext(t)
	baseline := writeBaseline(t, ctx)

	// Composite climbs 42 -> 99; scenario coverage stays at 1.
	ctx.Respond("GET", "/orientation-cost", orientationCostReport{
		Teams: []orientationCost{{TeamID: "alpha", Composite: 99, ScenarioCoverage: 1}},
	})
	tgt := findTarget(t, auditWithBaseline(t, ctx, baseline), "Team orientation cost")
	if tgt.Status != auditStatusInBand {
		t.Fatalf("rise without coverage growth must stay in band, got %q (%s)", tgt.Status, tgt.Observed)
	}

	// Same rise, but coverage grew with it: now it is the defect the band names.
	ctx.Respond("GET", "/orientation-cost", orientationCostReport{
		Teams: []orientationCost{{TeamID: "alpha", Composite: 99, ScenarioCoverage: 5}},
	})
	tgt = findTarget(t, auditWithBaseline(t, ctx, baseline), "Team orientation cost")
	if tgt.Status != auditStatusOutOfBand {
		t.Fatalf("rise with coverage growth must be out of band, got %q (%s)", tgt.Status, tgt.Observed)
	}
	if len(tgt.Detail) == 0 || !strings.Contains(tgt.Detail[0], "42 → 99") {
		t.Fatalf("out-of-band trend must name the movement, got %v", tgt.Detail)
	}
}

// A team absent from the baseline is new, not a regression. Treating a first
// reading as a rise would penalise declaring a team at all.
func TestCmdAuditTreatsUnseenSeriesAsNewRatherThanRisen(t *testing.T) {
	ctx := cleanAuditContext(t)
	baseline := writeBaseline(t, ctx)
	ctx.Respond("GET", "/orientation-cost", orientationCostReport{Teams: []orientationCost{
		{TeamID: "alpha", Composite: 42, ScenarioCoverage: 1},
		{TeamID: "brand-new", Composite: 9000, ScenarioCoverage: 99},
	}})
	tgt := findTarget(t, auditWithBaseline(t, ctx, baseline), "Team orientation cost")
	if tgt.Status != auditStatusInBand {
		t.Fatalf("a new series must not read as a regression, got %q (%s)", tgt.Status, tgt.Observed)
	}
}

// The open-loop count is one of the rows it counts. Whether it counts itself
// depends on whether a baseline bands it, and the reported number must match a
// tally of the printed list in both states.
func TestCmdAuditOpenLoopCountMatchesTheListInBothStates(t *testing.T) {
	ctx := cleanAuditContext(t)
	baseline := writeBaseline(t, ctx)

	for _, tc := range []struct{ name, arg string }{
		{"without baseline", ""},
		{"with baseline", baseline},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"--json"}
			if tc.arg != "" {
				args = append(args, "--baseline", tc.arg)
			}
			stdout, _, err := clitest.Output(t, func() error { return cmdAudit(ctx, args) })
			if err != nil {
				t.Fatalf("cmdAudit: %v", err)
			}
			var report auditReport
			if err := json.Unmarshal([]byte(stdout), &report); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			tgt := findTarget(t, report, "Open-loop target count")
			if tgt.Trend == nil {
				t.Fatal("the open-loop count must carry its reading as a trend")
			}
			// The invariant in both states: the number this target reports is
			// the number of no-sensor rows a reader can tally from the list.
			// Unbanded it counts itself (7 of 18); banded it does not (6 of 18).
			if got := int(tgt.Trend.Values[""]); got != report.Unsensored {
				t.Fatalf("open-loop count reported %d, list holds %d unsensored", got, report.Unsensored)
			}
			if !strings.Contains(tgt.Observed, fmt.Sprintf("%d of %d", report.Unsensored, len(report.Targets))) &&
				tgt.Status == auditStatusNoSensor {
				t.Fatalf("unbanded observed text must state the tally, got %q", tgt.Observed)
			}
		})
	}
}

// A baseline the operator asked for but that cannot be read is an error, not a
// silent fallback. Degrading quietly would report every trend as
// pending-baseline while the operator believed a comparison had happened.
func TestCmdAuditRejectsUnreadableBaseline(t *testing.T) {
	ctx := cleanAuditContext(t)
	_, _, err := clitest.Output(t, func() error {
		return cmdAudit(ctx, []string{"--json", "--baseline", filepath.Join(t.TempDir(), "missing.json")})
	})
	if err == nil {
		t.Fatal("an unreadable --baseline must fail the sweep, not fall back to no baseline")
	}
}

func TestCmdAuditRejectsBaselineFromAnotherSchemaVersion(t *testing.T) {
	ctx := cleanAuditContext(t)
	path := filepath.Join(t.TempDir(), "old.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":0,"targets":[]}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, _, err := clitest.Output(t, func() error {
		return cmdAudit(ctx, []string{"--json", "--baseline", path})
	})
	if err == nil {
		t.Fatal("a baseline from a different schema version must be rejected, not compared")
	}
}

// Canon coherence is collected now, so a failing script must reach the band.
// It previously reported "not collected" while failing, which reads as a pass.
func TestCmdAuditCollectsCanonCoherenceVerdict(t *testing.T) {
	ctx := cleanAuditContext(t)
	stubCanonScript(t, "\x1b[31mFAIL  technique pairing\x1b[0m\nPassed: 8\nFailed: 1\n", nil)
	stdout, _, err := clitest.Output(t, func() error { return cmdAudit(ctx, []string{"--json"}) })
	if err != nil {
		t.Fatalf("cmdAudit: %v", err)
	}
	var report auditReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	tgt := findTarget(t, report, "Canon coherence")
	if tgt.Status != auditStatusOutOfBand {
		t.Fatalf("a failing canon run must be out of band, got %q (%s)", tgt.Status, tgt.Observed)
	}
	if tgt.Observed != "8 pass, 1 fail" {
		t.Fatalf("observed = %q, want the tally", tgt.Observed)
	}
	for _, d := range tgt.Detail {
		if strings.Contains(d, "\x1b[") {
			t.Fatalf("detail carried terminal escapes into the artifact: %q", d)
		}
	}
}

// A script whose output shape changed must surface as uncollected, never as
// zero failures — the same dead-sensor shape the deadband rule forbids.
func TestCmdAuditTreatsUnparseableCanonOutputAsUncollected(t *testing.T) {
	ctx := cleanAuditContext(t)
	stubCanonScript(t, "something went very wrong\n", nil)
	stdout, _, err := clitest.Output(t, func() error { return cmdAudit(ctx, []string{"--json"}) })
	if err != nil {
		t.Fatalf("cmdAudit: %v", err)
	}
	var report auditReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if tgt := findTarget(t, report, "Canon coherence"); tgt.Status != auditStatusExternal {
		t.Fatalf("unparseable output must read as uncollected, got %q", tgt.Status)
	}
	if report.External != 1 {
		t.Fatalf("external = %d, want 1 — uncollected targets must reach the summary", report.External)
	}
}

// With no experiments at all the loop is not live, and the deadband's "once the
// loop is live" precondition is unmet. Reporting out-of-band would flag a
// failure to conclude work nobody started.
func TestCmdAuditTreatsAbsentExperimentLoopAsOpenLoop(t *testing.T) {
	ctx := cleanAuditContext(t)
	ctx.Respond("GET", "/experiments", []experimentLivenessRow{})
	stdout, _, err := clitest.Output(t, func() error { return cmdAudit(ctx, []string{"--json"}) })
	if err != nil {
		t.Fatalf("cmdAudit: %v", err)
	}
	var report auditReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	tgt := findTarget(t, report, "Skill-experiment loop liveness")
	if tgt.Status != auditStatusNoSensor {
		t.Fatalf("an unstarted loop is open-loop, not a failure; got %q (%s)", tgt.Status, tgt.Observed)
	}
	// A live loop that concludes nothing IS the defect the band names.
	ctx.Respond("GET", "/experiments", []experimentLivenessRow{{ID: "exp-1", Status: "running"}})
	stdout, _, err = clitest.Output(t, func() error { return cmdAudit(ctx, []string{"--json"}) })
	if err != nil {
		t.Fatalf("cmdAudit: %v", err)
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if tgt := findTarget(t, report, "Skill-experiment loop liveness"); tgt.Status != auditStatusOutOfBand {
		t.Fatalf("a live loop concluding nothing must be out of band, got %q", tgt.Status)
	}
}

// The coupling deadband names two clauses; the edge count only speaks to one.
// Before this, the check was `len(edges) > 0`, which could detect nothing but
// total collapse of the map while runtime-only rows accumulated unseen.
func TestCmdAuditCouplingDetectsRuntimeOnlyRows(t *testing.T) {
	ctx := cleanAuditContext(t)
	ctx.Respond("GET", "/operating-models/coverage", operatingModelCoverageResponse{
		Coverage: []operatingGraphCoverage{{Team: "alpha", Relationships: []operatingRelationshipCoverage{
			{Relationship: "topic_read", RuntimeDeclared: 3, GraphShown: 1, Matched: 1, RuntimeOnly: 2},
		}}},
	})
	stdout, _, err := clitest.Output(t, func() error { return cmdAudit(ctx, []string{"--json"}) })
	if err != nil {
		t.Fatalf("cmdAudit: %v", err)
	}
	var report auditReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	tgt := findTarget(t, report, "Cross-team coupling visibility")
	if tgt.Status != auditStatusOutOfBand {
		t.Fatalf("runtime-only rows must be out of band, got %q (%s)", tgt.Status, tgt.Observed)
	}
	if !strings.Contains(tgt.Observed, "2 runtime-only") {
		t.Fatalf("observed must name the runtime-only count, got %q", tgt.Observed)
	}
	if len(tgt.Detail) == 0 || !strings.Contains(tgt.Detail[0], "alpha/topic_read") {
		t.Fatalf("out-of-band coupling must name the relationship, got %v", tgt.Detail)
	}
}

// A gap marker's date is parsed into a computable age. A marker frozen in a
// string literal cannot answer "how long has this been open", which is the only
// question that turns a declared hole into an overdue one.
func TestCmdAuditComputesGapAgeFromMarkerDate(t *testing.T) {
	prev := auditNow
	auditNow = func() time.Time { return time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { auditNow = prev })

	ctx := cleanAuditContext(t)
	stdout, _, err := clitest.Output(t, func() error { return cmdAudit(ctx, []string{"--json"}) })
	if err != nil {
		t.Fatalf("cmdAudit: %v", err)
	}
	var report auditReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// PoR entropy's marker is dated 2026-07-27: 14 days before the stubbed now.
	tgt := findTarget(t, report, "PoR entropy")
	if tgt.GapOpenedOn != "2026-07-27" {
		t.Fatalf("gap_opened_on = %q, want 2026-07-27", tgt.GapOpenedOn)
	}
	if tgt.GapOpenDays != 14 {
		t.Fatalf("gap_open_days = %d, want 14", tgt.GapOpenDays)
	}
	for _, other := range report.Targets {
		if other.Status == auditStatusNoSensor && other.GapOpenDays <= 0 {
			t.Fatalf("every open-loop target must carry a computable gap age: %q (%q)", other.Target, other.GapMarker)
		}
	}
}
