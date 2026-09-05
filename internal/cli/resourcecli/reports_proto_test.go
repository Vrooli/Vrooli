package resourcecli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/testenv"
)

// TestWriteStatusesJSONContract pins the fleet `resource status --json` shape.
func TestWriteStatusesJSONContract(t *testing.T) {
	healthy := true
	items := []resources.Status{
		{
			Resource:   resources.Resource{Name: "redis", Enabled: true, ControlMode: "manifest-native"},
			Installed:  true,
			Running:    true,
			Healthy:    &healthy,
			Health:     "ok",
			StatusCode: "ok",
			Message:    "running",
			Raw:        json.RawMessage(`{"pid":42}`),
		},
		// sparse case: no health probe, minimal fields.
		{Resource: resources.Resource{Name: "ollama"}},
	}
	failures := []discovery.Failure{{Kind: "resource", Name: "broken", Error: "boom"}}

	var buf bytes.Buffer
	if err := WriteStatuses(&buf, cliout.FormatJSON, items, failures); err != nil {
		t.Fatalf("WriteStatuses: %v", err)
	}
	got := testenv.DecodeJSON[map[string]any](t, buf.Bytes())
	if got["success"] != true {
		t.Errorf("success: want true, got %v", got["success"])
	}
	resources_, ok := got["resources"].([]any)
	if !ok || len(resources_) != 2 {
		t.Fatalf("resources: want 2, got %v", got["resources"])
	}
	first := resources_[0].(map[string]any)
	res := first["resource"].(map[string]any)
	if res["name"] != "redis" { //nolint:goconst // fixture resource name
		t.Errorf("resource.name: %v", res["name"])
	}
	if res["control_mode"] != "manifest-native" {
		t.Errorf("control_mode snake_case missing: %v", res)
	}
	if first["healthy"] != true {
		t.Errorf("healthy: want true, got %v", first["healthy"])
	}
	if first["status_code"] != "ok" {
		t.Errorf("status_code snake_case missing: %v", first)
	}
	// sparse case: healthy should be null (nil *bool -> structpb null).
	second := resources_[1].(map[string]any)
	if second["healthy"] != nil {
		t.Errorf("sparse healthy: want null, got %v", second["healthy"])
	}
	if _, ok := got["discovery_failures"].([]any); !ok {
		t.Errorf("discovery_failures missing/wrong (snake_case?): %v", got["discovery_failures"])
	}
}

// TestWriteStatusJSONContract pins the single `resource status --json` shape.
func TestWriteStatusJSONContract(t *testing.T) {
	healthy := false
	item := resources.Status{
		Resource:  resources.Resource{Name: "qdrant"},
		Installed: true,
		Running:   false,
		Healthy:   &healthy,
		Message:   "stopped",
	}
	var buf bytes.Buffer
	if err := WriteStatus(&buf, cliout.FormatJSON, item); err != nil {
		t.Fatalf("WriteStatus: %v", err)
	}
	got := testenv.DecodeJSON[map[string]any](t, buf.Bytes())
	if got["success"] != true {
		t.Errorf("success: %v", got["success"])
	}
	if got["name"] != "qdrant" {
		t.Errorf("name: %v", got["name"])
	}
	if got["installed"] != true || got["running"] != false {
		t.Errorf("installed/running: %v %v", got["installed"], got["running"])
	}
	if got["healthy"] != false {
		t.Errorf("healthy: want false, got %v", got["healthy"])
	}
	if got["status"] != "stopped" {
		t.Errorf("status: %v", got["status"])
	}
	if _, ok := got["resource"].(map[string]any); !ok {
		t.Errorf("resource object missing: %v", got["resource"])
	}
}

// TestWriteInfoJSONContract pins `resource info --json`.
func TestWriteInfoJSONContract(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteInfo(&buf, cliout.FormatJSON, resources.Status{Resource: resources.Resource{Name: "vault"}}); err != nil {
		t.Fatalf("WriteInfo: %v", err)
	}
	got := testenv.DecodeJSON[map[string]any](t, buf.Bytes())
	if got["success"] != true {
		t.Errorf("success: %v", got["success"])
	}
	res := got["resource"].(map[string]any)
	inner := res["resource"].(map[string]any)
	if inner["name"] != "vault" {
		t.Errorf("resource.resource.name: %v", inner["name"])
	}
}

// TestWriteControlReportJSONContract pins `resource start-all --json`.
func TestWriteControlReportJSONContract(t *testing.T) {
	report := control.StartReport{
		Started: []control.ResultItem{{Name: "redis", Message: "up"}},
		Failed:  []control.ResultItem{{Name: "ollama", Error: "nope"}},
		Message: "Started 1 targets, 1 failed",
	}
	var buf bytes.Buffer
	if err := WriteControlReport(&buf, cliout.FormatJSON, "report", "Started", &report, report.Started, report.Failed); err != nil {
		t.Fatalf("WriteControlReport: %v", err)
	}
	got := testenv.DecodeJSON[map[string]any](t, buf.Bytes())
	if got["success"] != true {
		t.Errorf("success: %v", got["success"])
	}
	rep := got["report"].(map[string]any)
	started := rep["started"].([]any)
	if len(started) != 1 || started[0].(map[string]any)["name"] != "redis" {
		t.Errorf("started: %v", rep["started"])
	}
	if rep["message"] != "Started 1 targets, 1 failed" {
		t.Errorf("message: %v", rep["message"])
	}
}

// TestWriteDeprecationReportJSONContract pins `resource deprecate --json`.
func TestWriteDeprecationReportJSONContract(t *testing.T) {
	report := resources.DeprecationReport{
		Resource:   resources.DeprecatedResource{Name: "old", RetentionPolicyDays: 30, RestoreSupported: true},
		Archived:   true,
		ArchiveDir: "/archives/old",
	}
	var buf bytes.Buffer
	if err := WriteDeprecationReport(&buf, cliout.FormatJSON, report); err != nil {
		t.Fatalf("WriteDeprecationReport: %v", err)
	}
	got := testenv.DecodeJSON[map[string]any](t, buf.Bytes())
	rep := got["report"].(map[string]any)
	res := rep["resource"].(map[string]any)
	// integer field must be a JSON number, not a string.
	days, ok := res["retention_policy_days"].(float64)
	if !ok || days != 30 {
		t.Errorf("retention_policy_days: want number 30, got %T %v", res["retention_policy_days"], res["retention_policy_days"])
	}
	if rep["archive_dir"] != "/archives/old" {
		t.Errorf("archive_dir snake_case: %v", rep["archive_dir"])
	}
}

// TestWriteArchiveGCReportJSONContract pins `resource archive gc --json`.
func TestWriteArchiveGCReportJSONContract(t *testing.T) {
	report := resources.ArchiveGCReport{
		Removed: []resources.ArchiveGCItem{{Name: "old", Removed: true}},
		Skipped: nil,
	}
	var buf bytes.Buffer
	if err := WriteArchiveGCReport(&buf, cliout.FormatJSON, report, "deprecated resource"); err != nil {
		t.Fatalf("WriteArchiveGCReport: %v", err)
	}
	got := testenv.DecodeJSON[map[string]any](t, buf.Bytes())
	rep := got["report"].(map[string]any)
	if _, ok := rep["removed"].([]any); !ok {
		t.Errorf("removed missing: %v", rep)
	}
	if _, ok := rep["skipped"].([]any); !ok {
		t.Errorf("skipped not emitted as []: %v", rep["skipped"])
	}
}

// TestWriteSchemaValidationReportJSONContract pins `resource schema validate --json`.
func TestWriteSchemaValidationReportJSONContract(t *testing.T) {
	report := resources.ResourceSchemaValidationReport{
		Passed:        false,
		ResourceCount: 7,
		ArtifactIssues: []resources.SchemaArtifactIssue{
			{Path: "p", Message: "stale"},
		},
		MissingReferences: []resources.ScenarioResourceReference{
			{Scenario: "s", Resource: "r", ManifestPath: "m"},
		},
	}
	var buf bytes.Buffer
	if err := WriteSchemaValidationReport(&buf, cliout.FormatJSON, report); err != nil {
		t.Fatalf("WriteSchemaValidationReport: %v", err)
	}
	got := testenv.DecodeJSON[map[string]any](t, buf.Bytes())
	// success mirrors report.passed (WriteFieldsWithSuccess semantics).
	if got["success"] != false {
		t.Errorf("success should mirror passed=false, got %v", got["success"])
	}
	rep := got["report"].(map[string]any)
	count, ok := rep["resource_count"].(float64)
	if !ok || count != 7 {
		t.Errorf("resource_count: want number 7, got %T %v", rep["resource_count"], rep["resource_count"])
	}
	if _, ok := rep["missing_references"].([]any); !ok {
		t.Errorf("missing_references snake_case missing: %v", rep)
	}
}

// TestWriteBlueprintListJSONContract pins `resource blueprint list --json`.
func TestWriteBlueprintListJSONContract(t *testing.T) {
	items := []resources.Blueprint{
		{
			Name:        "redis-cache",
			DisplayName: "Redis Cache",
			Category:    "storage",
			Status:      "validated",
			PlatformSupport: resources.BlueprintPlatformSupport{
				Linux: "supported",
			},
			References: []resources.BlueprintReference{{Kind: "doc", Value: "u"}},
		},
	}
	var buf bytes.Buffer
	if err := WriteBlueprintList(&buf, cliout.FormatJSON, items); err != nil {
		t.Fatalf("WriteBlueprintList: %v", err)
	}
	got := testenv.DecodeJSON[map[string]any](t, buf.Bytes())
	if got["success"] != true {
		t.Errorf("success: %v", got["success"])
	}
	bps := got["blueprints"].([]any)
	if len(bps) != 1 {
		t.Fatalf("blueprints: want 1, got %v", got["blueprints"])
	}
	bp := bps[0].(map[string]any)
	if bp["display_name"] != "Redis Cache" {
		t.Errorf("display_name snake_case: %v", bp["display_name"])
	}
}

// TestWriteBlueprintSearchJSONContract pins `resource blueprint search --json`.
func TestWriteBlueprintSearchJSONContract(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteBlueprintSearch(&buf, cliout.FormatJSON, "cache", []resources.Blueprint{{Name: "redis-cache"}}); err != nil {
		t.Fatalf("WriteBlueprintSearch: %v", err)
	}
	got := testenv.DecodeJSON[map[string]any](t, buf.Bytes())
	if got["query"] != "cache" {
		t.Errorf("query: %v", got["query"])
	}
	if _, ok := got["blueprints"].([]any); !ok {
		t.Errorf("blueprints missing: %v", got["blueprints"])
	}
}
