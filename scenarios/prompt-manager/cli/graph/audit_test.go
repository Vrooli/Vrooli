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
		"Prose/declaration coherence",
		"Cross-team coupling visibility",
		"Canon coherence",
		"Objective coverage",
		"Skill reachability",
		"Skill conditioning quality",
		"Skill-experiment loop liveness",
		"PoR entropy",
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
	// Two targets are open-loop in FRAMEWORK_HEALTH today.
	if report.Unsensored != 2 {
		t.Fatalf("unsensored = %d, want 2", report.Unsensored)
	}
	// An unsensored target must name which kind of open loop it is, using the
	// honesty vocabulary from RELIABILITY_TARGETS §"Honesty flags":
	// `pending-telemetry` when no instrument exists, `pending-baseline` when the
	// instrument exists but nothing sweeps the corpus with it. Either is honest;
	// an unsensored target carrying neither is not.
	for _, tgt := range report.Targets {
		if tgt.Status != auditStatusNoSensor {
			continue
		}
		if !strings.Contains(tgt.Observed, "pending-telemetry") && !strings.Contains(tgt.Observed, "pending-baseline") {
			t.Fatalf("unsensored target must stay honest about it: %+v", tgt)
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
