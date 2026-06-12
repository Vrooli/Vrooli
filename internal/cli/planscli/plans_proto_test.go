package planscli

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	planapp "github.com/vrooli/vrooli/internal/app/plans"
	"github.com/vrooli/vrooli/internal/cliout"
)

func samplePlanRecord() planapp.PlanRecord {
	return planapp.PlanRecord{
		ID:          "plan-1",
		Title:       "My Plan",
		Slug:        "my-plan",
		Path:        "/plans/my-plan.md",
		CreatedAt:   time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 6, 11, 11, 0, 0, 0, time.UTC),
		Archived:    true,
		ArchivedAt:  time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC),
		SourcePath:  "/src/my-plan.md",
		ContentHash: "abc123",
	}
}

func assertRecord(t *testing.T, plan map[string]any) {
	t.Helper()
	if plan["id"] != "plan-1" || plan["title"] != "My Plan" || plan["slug"] != "my-plan" {
		t.Errorf("record identity mismatch: %v", plan)
	}
	if plan["created_at"] != "2026-06-11T10:00:00Z" {
		t.Errorf("created_at: %v", plan["created_at"])
	}
	if plan["archived"] != true {
		t.Errorf("archived: %v", plan["archived"])
	}
	if plan["source_path"] != "/src/my-plan.md" || plan["content_hash"] != "abc123" {
		t.Errorf("source_path/content_hash mismatch: %v", plan)
	}
}

func TestRenderAddJSONContract(t *testing.T) {
	resp := planapp.AddOutput{Success: true, Plan: samplePlanRecord()}
	var buf bytes.Buffer
	if err := RenderAdd(&buf, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("RenderAdd: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if got["success"] != true {
		t.Errorf("success: %v", got["success"])
	}
	plan, ok := got["plan"].(map[string]any)
	if !ok {
		t.Fatalf("plan missing: %v", got["plan"])
	}
	assertRecord(t, plan)
}

func TestRenderListJSONContract(t *testing.T) {
	// Include a sparse record (zero timestamps -> "") alongside a full one.
	resp := planapp.ListOutput{
		Success: true,
		Plans: []planapp.PlanRecord{
			samplePlanRecord(),
			{ID: "plan-2", Title: "Sparse", Slug: "sparse"},
		},
	}
	var buf bytes.Buffer
	if err := RenderList(&buf, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	plans, ok := got["plans"].([]any)
	if !ok || len(plans) != 2 {
		t.Fatalf("plans: want 2, got %v", got["plans"])
	}
	assertRecord(t, plans[0].(map[string]any))
	sparse := plans[1].(map[string]any)
	if sparse["created_at"] != "" || sparse["archived_at"] != "" {
		t.Errorf("sparse zero-time should map to empty string: %v", sparse)
	}
}

func TestRenderShowJSONContract(t *testing.T) {
	resp := planapp.ShowOutput{Success: true, Plan: samplePlanRecord(), Content: "body text"}
	var buf bytes.Buffer
	if err := RenderShow(&buf, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("RenderShow: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if got["content"] != "body text" {
		t.Errorf("content: %v", got["content"])
	}
	assertRecord(t, got["plan"].(map[string]any))
}

func TestRenderPathJSONContract(t *testing.T) {
	resp := planapp.PathOutput{Success: true, ID: "plan-1", Path: "/plans/my-plan.md"}
	var buf bytes.Buffer
	if err := RenderPath(&buf, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("RenderPath: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if got["id"] != "plan-1" || got["path"] != "/plans/my-plan.md" {
		t.Errorf("path output mismatch: %v", got)
	}
}

func TestRenderImportJSONContract(t *testing.T) {
	resp := planapp.ImportOutput{Success: true, Plan: samplePlanRecord(), Deleted: true}
	var buf bytes.Buffer
	if err := RenderImport(&buf, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("RenderImport: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if got["deleted_source"] != true {
		t.Errorf("deleted_source (snake_case): %v", got)
	}
	assertRecord(t, got["plan"].(map[string]any))
}

func TestRenderArchiveJSONContract(t *testing.T) {
	resp := planapp.ArchiveOutput{Success: true, Plan: samplePlanRecord()}
	var buf bytes.Buffer
	if err := RenderArchive(&buf, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("RenderArchive: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	assertRecord(t, got["plan"].(map[string]any))
}

func TestRenderExportJSONContract(t *testing.T) {
	resp := planapp.ExportOutput{Success: true, ID: "plan-1", Path: "/out/my-plan.md"}
	var buf bytes.Buffer
	if err := RenderExport(&buf, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("RenderExport: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if got["success"] != true || got["id"] != "plan-1" || got["path"] != "/out/my-plan.md" {
		t.Errorf("export output mismatch: %v", got)
	}
}
