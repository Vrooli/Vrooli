package graph

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	return ctx
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
		"Team orientation cost",
		"Discovery budget pressure",
		"Prompt structure invariant",
		"Catalogued rules with no finding",
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
	// Six targets are open-loop in FRAMEWORK_HEALTH today. Two were added with
	// this plan: the prompt structure invariant, enforced in the heartbeat unit
	// suite rather than by a corpus-wide CLI sensor, and the rules-with-no-
	// finding trend, which cannot be banded from a single reading.
	if report.Unsensored != 6 {
		t.Fatalf("unsensored = %d, want 6", report.Unsensored)
	}
	// An unsensored target must name both the honest open-loop state and a dated
	// marker for the work that closes it. The marker prevents an intentionally
	// visible gap from becoming permanent background noise.
	for _, tgt := range report.Targets {
		if tgt.Status != auditStatusNoSensor {
			continue
		}
		if !strings.Contains(tgt.Observed, "pending-telemetry") && !strings.Contains(tgt.Observed, "pending-baseline") {
			t.Fatalf("unsensored target must stay honest about it: %+v", tgt)
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
