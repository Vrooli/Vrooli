package hygienecli

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	hygieneapp "github.com/vrooli/vrooli/internal/app/hygiene"
	planapp "github.com/vrooli/vrooli/internal/app/plans"
	shareddriftapp "github.com/vrooli/vrooli/internal/app/shareddrift"
	"github.com/vrooli/vrooli/internal/cliout"
)

// TestRenderHygieneJSONContract pins the `vrooli hygiene --json` wire shape with
// a fully populated report (findings, actions, plan candidates, fixes, embedded
// contract + shared-drift).
func TestRenderHygieneJSONContract(t *testing.T) {
	created := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	report := hygieneapp.Report{
		Success: true,
		Root:    "/repo",
		Checks: []hygieneapp.Check{
			{Name: "plan_lifecycle", Passed: true, Severity: hygieneapp.SeverityInfo, Message: "ok"},
		},
		Findings: []hygieneapp.Finding{
			{
				Severity:   hygieneapp.SeverityWarning,
				Code:       "untracked_plan",
				Path:       "plans/foo.md",
				Locations:  []string{"plans/foo.md"},
				Message:    "untracked plan file",
				Why:        "plans should be imported",
				Fixability: hygieneapp.FixabilityGuided,
				NextActions: []hygieneapp.Action{
					{Code: "import_plan", Message: "import it", Command: "vrooli plans import plans/foo.md"},
				},
			},
		},
		Actions: []hygieneapp.Action{
			{Code: "rerun", Message: "rerun hygiene", Command: "vrooli hygiene"},
		},
		PlanCandidates: []hygieneapp.PlanCandidate{
			{Path: "plans/foo.md", Status: "untracked", Reason: "looks like a plan"},
		},
		FixesApplied: []hygieneapp.PlanFix{
			{Source: "plans/foo.md", Plan: planapp.PlanRecord{ID: "p1", Title: "Foo", Slug: "foo", CreatedAt: created}},
		},
		ConfigFixes: []string{"fixed pnpm workspace"},
		Contract: contractapp.ValidationOutput{
			Success: true,
			Root:    "/repo",
			Schema:  contractapp.ValidationCheck{Passed: true, Message: "ok"},
		},
		SharedDrift: &shareddriftapp.Report{
			Clean: true,
			Root:  "/repo",
			Scenarios: []shareddriftapp.ScenarioReport{
				{Path: "scenarios/demo", APIDir: "scenarios/demo/api", Status: shareddriftapp.StatusClean},
			},
			ElapsedMs: 1234,
		},
		BlockingFailures: 0,
		Warnings:         1,
	}

	var buf bytes.Buffer
	if err := Render(&buf, cliout.FormatJSON, report, OutputModeDefault); err != nil {
		t.Fatalf("Render: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if got["success"] != true || got["root"] != "/repo" {
		t.Errorf("top-level mismatch: success=%v root=%v", got["success"], got["root"])
	}
	// blocking_failures / warnings must be JSON numbers (float64), not strings.
	if bf, ok := got["blocking_failures"].(float64); !ok || bf != 0 {
		t.Errorf("blocking_failures want number 0, got %T %v", got["blocking_failures"], got["blocking_failures"])
	}
	if wn, ok := got["warnings"].(float64); !ok || wn != 1 {
		t.Errorf("warnings want number 1, got %T %v", got["warnings"], got["warnings"])
	}

	findings := got["findings"].([]any)
	if len(findings) != 1 {
		t.Fatalf("findings: want 1, got %v", got["findings"])
	}
	finding := findings[0].(map[string]any)
	if finding["code"] != "untracked_plan" || finding["fixability"] != "guided" {
		t.Errorf("finding mismatch: %v", finding)
	}
	na, ok := finding["next_actions"].([]any)
	if !ok || len(na) != 1 {
		t.Fatalf("next_actions (snake_case?): %v", finding["next_actions"])
	}

	candidates := got["plan_candidates"].([]any)
	if len(candidates) != 1 || candidates[0].(map[string]any)["status"] != "untracked" {
		t.Errorf("plan_candidates mismatch: %v", got["plan_candidates"])
	}

	fixes := got["fixes_applied"].([]any)
	plan := fixes[0].(map[string]any)["plan"].(map[string]any)
	if plan["id"] != "p1" {
		t.Errorf("plan id (snake_case 'id'?): %v", plan)
	}
	if plan["created_at"] != "2026-06-11T10:00:00Z" {
		t.Errorf("created_at RFC3339: %v", plan["created_at"])
	}
	if plan["archived_at"] != "" {
		t.Errorf("zero archived_at should be empty: %v", plan["archived_at"])
	}

	contract := got["contract"].(map[string]any)
	if contract["success"] != true {
		t.Errorf("embedded contract mismatch: %v", contract)
	}

	drift := got["shared_drift"].(map[string]any)
	if drift["clean"] != true {
		t.Errorf("shared_drift mismatch: %v", drift)
	}
	if em, ok := drift["elapsed_ms"].(float64); !ok || em != 1234 {
		t.Errorf("elapsed_ms want number 1234, got %T %v", drift["elapsed_ms"], drift["elapsed_ms"])
	}
	sc := drift["scenarios"].([]any)[0].(map[string]any)
	if sc["api_dir"] != "scenarios/demo/api" {
		t.Errorf("api_dir (snake_case?): %v", sc)
	}
}

// TestRenderHygieneJSONSparse pins the sparse case: nil shared-drift maps to an
// absent shared_drift field.
func TestRenderHygieneJSONSparse(t *testing.T) {
	report := hygieneapp.Report{
		Success:     false,
		Root:        "/repo",
		SharedDrift: nil,
	}

	var buf bytes.Buffer
	if err := Render(&buf, cliout.FormatJSON, report, OutputModeDefault); err != nil {
		t.Fatalf("Render: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if got["success"] != false {
		t.Errorf("success: want false, got %v", got["success"])
	}
	// EmitUnpopulated emits a nil message as JSON null (key present, value null).
	if v, present := got["shared_drift"]; !present || v != nil {
		t.Errorf("shared_drift should be JSON null when nil, got present=%v value=%v", present, v)
	}
	// contract is always emitted (struct value, not pointer).
	if _, ok := got["contract"].(map[string]any); !ok {
		t.Errorf("contract should always be present: %v", got["contract"])
	}
}
