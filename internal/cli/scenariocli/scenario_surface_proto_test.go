package scenariocli

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/resources"
	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/templatevalidation"
)

// renderToMap runs a render func at JSON format and decodes to a generic map.
func renderToMap(t *testing.T, render func(*bytes.Buffer) error) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	if err := render(&buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	return got
}

// isJSONNumber asserts the value is a JSON number (float64), not a string —
// the int32 vs int64-as-string contract guard.
func isJSONNumber(t *testing.T, v any, label string) {
	t.Helper()
	if _, ok := v.(float64); !ok {
		t.Errorf("%s: want JSON number (float64), got %T (%v)", label, v, v)
	}
}

func TestScenarioStatusListJSONContract(t *testing.T) {
	started := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	items := []StatusItemOutput{
		{
			Name:        "alpha",
			DisplayName: "Alpha",
			Status:      "running",
			Processes:   2,
			Runtime:     "process",
			StartedAt:   &started,
			Tags:        []string{"x"},
			Ports:       map[string]int{"API_PORT": 5329},
			PortBindings: []ListPortOutput{
				{Key: "API_PORT", Step: "develop", Port: 5329, ListenerStatus: "listening"},
			},
			Health:      "healthy",
			HealthError: "",
		},
		// sparse: nil StartedAt, nil Health, no ports.
		{Name: "beta", Status: "stopped"},
	}
	failures := []discovery.Failure{{Kind: "scenario", Name: "broken", Error: "boom"}}

	got := renderToMap(t, func(w *bytes.Buffer) error {
		return RenderStatusResponse(w, cliout.FormatJSON, StatusResponse{List: items, Failures: failures})
	})

	if got["success"] != true {
		t.Errorf("success: %v", got["success"])
	}
	summary := got["summary"].(map[string]any)
	isJSONNumber(t, summary["total_scenarios"], "total_scenarios")
	isJSONNumber(t, summary["running"], "running")
	isJSONNumber(t, summary["stopped"], "stopped")
	if summary["total_scenarios"].(float64) != 2 || summary["running"].(float64) != 1 || summary["stopped"].(float64) != 1 {
		t.Errorf("summary counts wrong: %v", summary)
	}
	scenarios := got["scenarios"].([]any)
	if len(scenarios) != 2 {
		t.Fatalf("scenarios len: %d", len(scenarios))
	}
	first := scenarios[0].(map[string]any)
	if first["name"] != "alpha" || first["status"] != "running" {
		t.Errorf("first item: %v", first)
	}
	if first["started_at"] != started.Format(time.RFC3339Nano) {
		t.Errorf("started_at: %v", first["started_at"])
	}
	if first["health_status"] != "healthy" {
		t.Errorf("health_status: %v", first["health_status"])
	}
	ports := first["ports"].(map[string]any)
	isJSONNumber(t, ports["API_PORT"], "ports.API_PORT")
	bindings := first["port_bindings"].([]any)
	pb := bindings[0].(map[string]any)
	if pb["key"] != "API_PORT" {
		t.Errorf("port binding key: %v", pb)
	}
	isJSONNumber(t, pb["port"], "port_bindings[0].port")
	// sparse item: started_at empty, health_status null.
	second := scenarios[1].(map[string]any)
	if second["started_at"] != "" {
		t.Errorf("sparse started_at should be empty: %v", second["started_at"])
	}
	if second["health_status"] != nil {
		t.Errorf("sparse health_status should be null: %v", second["health_status"])
	}
	if _, ok := got["discovery_failures"].([]any); !ok {
		t.Errorf("discovery_failures missing/wrong: %v", got["discovery_failures"])
	}
}

func TestScenarioStatusSingleJSONContract(t *testing.T) {
	fixed := 8080
	single := StatusSingleOutput{
		Success:  true,
		Scenario: StatusItemOutput{Name: "alpha", Status: "running", Processes: 1},
		Info: InfoScenarioData{
			Name:        "alpha",
			Path:        "/x",
			ServicePath: "/x/.vrooli",
			Ports:       []scenariomodel.PortSummary{{Name: "API", EnvVar: "API_PORT", FixedPort: &fixed}},
			Phases:      []scenariomodel.PhaseSummary{{Name: "setup", Steps: 3, Defined: true}},
			Generation: &scenariomodel.GenerationMetadata{
				Template: scenariomodel.GenerationTemplate{ID: "t", Version: "1"},
				Design:   scenariomodel.GenerationDesign{ID: "d", Adapter: "a"},
			},
		},
		Runtime: InfoRuntimeData{
			Status:      "running",
			Processes:   1,
			ProcessInfo: []process.Record{{PID: 42, Command: "serve", StartedAt: time.Date(2026, 6, 11, 9, 0, 0, 0, time.UTC)}},
		},
	}

	got := renderToMap(t, func(w *bytes.Buffer) error {
		return RenderStatusResponse(w, cliout.FormatJSON, StatusResponse{Single: &single})
	})
	if got["success"] != true {
		t.Errorf("success: %v", got["success"])
	}
	info := got["info"].(map[string]any)
	if info["service_path"] != "/x/.vrooli" {
		t.Errorf("service_path snake_case missing: %v", info)
	}
	infoPorts := info["ports"].([]any)
	isJSONNumber(t, infoPorts[0].(map[string]any)["fixed_port"], "info.ports[0].fixed_port")
	gen := info["generation"].(map[string]any)
	if gen["template"].(map[string]any)["id"] != "t" {
		t.Errorf("generation.template.id: %v", gen)
	}
	rt := got["runtime"].(map[string]any)
	procs := rt["process_records"].([]any)
	rec := procs[0].(map[string]any)
	isJSONNumber(t, rec["pid"], "process_records[0].pid")
	if rec["started_at"] != "2026-06-11T09:00:00Z" {
		t.Errorf("process started_at: %v", rec["started_at"])
	}
}

func TestScenarioInfoJSONContract(t *testing.T) {
	got := renderToMap(t, func(w *bytes.Buffer) error {
		return RenderInfoResponse(w, cliout.FormatJSON, InfoOutput{
			Success:  true,
			Scenario: InfoScenarioData{Name: "alpha", SandboxRedirect: true},
			Runtime:  InfoRuntimeData{Status: "stopped"},
		})
	})
	if got["success"] != true {
		t.Errorf("success: %v", got["success"])
	}
	scen := got["scenario"].(map[string]any)
	if scen["sandbox_redirected"] != true {
		t.Errorf("sandbox_redirected: %v", scen)
	}
}

func TestScenarioPortJSONContract(t *testing.T) {
	gotSingle := renderToMap(t, func(w *bytes.Buffer) error {
		return RenderPortResponse(w, cliout.FormatJSON, PortResponse{
			Single: &PortSingleOutput{Success: true, Scenario: "alpha", PortName: "API_PORT", Port: 5329},
		})
	})
	if gotSingle["success"] != true || gotSingle["scenario"] != "alpha" {
		t.Errorf("port single: %v", gotSingle)
	}
	if gotSingle["port_name"] != "API_PORT" {
		t.Errorf("port_name snake_case: %v", gotSingle)
	}
	isJSONNumber(t, gotSingle["port"], "single.port")

	gotList := renderToMap(t, func(w *bytes.Buffer) error {
		return RenderPortResponse(w, cliout.FormatJSON, PortResponse{
			List: &PortListOutput{
				Success:  true,
				Scenario: "alpha",
				Ports:    []ListPortOutput{{Key: "API_PORT", Port: 5329}},
				Metadata: map[string]int{"count": 1},
			},
		})
	})
	if gotList["success"] != true {
		t.Errorf("port list success: %v", gotList)
	}
	meta := gotList["metadata"].(map[string]any)
	isJSONNumber(t, meta["count"], "metadata.count")
}

func TestScenarioLifecycleAndBatchJSONContract(t *testing.T) {
	items := []LifecycleItemOutput{{
		Name:      "alpha",
		Status:    "started",
		Health:    "healthy",
		Ports:     map[string]int{"API_PORT": 5329},
		Endpoints: []EndpointOutput{{Name: "API", Key: "API_PORT", Port: 5329, URL: "http://x"}},
	}}

	gotLC := renderToMap(t, func(w *bytes.Buffer) error {
		return WriteLifecycleItems(w, cliout.FormatJSON, items)
	})
	if gotLC["success"] != true {
		t.Errorf("lifecycle success: %v", gotLC)
	}
	lcScen := gotLC["scenarios"].([]any)[0].(map[string]any)
	ep := lcScen["endpoints"].([]any)[0].(map[string]any)
	isJSONNumber(t, ep["port"], "endpoints[0].port")
	if _, ok := lcScen["failed_dependencies"].([]any); !ok {
		t.Errorf("failed_dependencies missing/wrong (snake_case?): %v", lcScen)
	}

	gotBatch := renderToMap(t, func(w *bytes.Buffer) error {
		return WriteBatchReport(w, cliout.FormatJSON, BatchResponse{
			Verb:    "Started",
			Started: items,
			Failed:  []BatchFailure{{Name: "beta", Error: "boom"}},
		})
	})
	if gotBatch["success"] != true {
		t.Errorf("batch success: %v", gotBatch)
	}
	data := gotBatch["data"].(map[string]any)
	if _, ok := data["stopped"].([]any); !ok {
		t.Errorf("batch stopped should be [] not absent: %v", data)
	}
	failed := data["failed"].([]any)[0].(map[string]any)
	if failed["name"] != "beta" || failed["error"] != "boom" {
		t.Errorf("batch failed: %v", failed)
	}
}

func TestScenarioSetupJSONContract(t *testing.T) {
	got := renderToMap(t, func(w *bytes.Buffer) error {
		return RenderSetupResponse(w, cliout.FormatJSON, lifecycle.PhaseResult{
			Status:        lifecycle.PhaseExecutionCompleted,
			Defined:       true,
			ExecutedSteps: 3,
			SkippedSteps:  1,
		})
	})
	if got["success"] != true || got["phase"] != "setup" {
		t.Errorf("setup envelope: %v", got)
	}
	if got["status"] != "completed" {
		t.Errorf("setup status: %v", got["status"])
	}
	steps := got["steps"].(map[string]any)
	isJSONNumber(t, steps["executed"], "steps.executed")
	isJSONNumber(t, steps["skipped"], "steps.skipped")
}

func TestScenarioTemplateListJSONContract(t *testing.T) {
	required := true
	got := renderToMap(t, func(w *bytes.Buffer) error {
		return RenderTemplateListResponse(w, cliout.FormatJSON, []TemplateInfo{{
			Name: "react-vite",
			Path: "/templates/react-vite",
			Manifest: TemplateManifest{
				Name:        "react-vite",
				DisplayName: "React + Vite",
				Stack:       []string{"react"},
				Design:      TemplateDesign{Adapter: "ad", Required: true},
				Orientation: &TemplateOrientation{
					Version: "1",
					Steps:   []TemplateOrientationStep{{ID: "s1", Required: &required, Checks: []TemplateOrientationCheck{{Kind: "file", Path: "x"}}}},
				},
				RequiredVars: map[string]TemplateVar{"NAME": {Flag: "name"}},
			},
		}})
	})
	if got["success"] != true {
		t.Errorf("template list success: %v", got)
	}
	tpl := got["templates"].([]any)[0].(map[string]any)
	if tpl["name"] != "react-vite" {
		t.Errorf("template name: %v", tpl)
	}
	manifest := tpl["manifest"].(map[string]any)
	if manifest["display_name"] != "React + Vite" {
		t.Errorf("display_name snake_case: %v", manifest)
	}
	step := manifest["orientation"].(map[string]any)["steps"].([]any)[0].(map[string]any)
	if step["required"] != true {
		t.Errorf("step required: %v", step)
	}
}

func TestScenarioTemplateDriftJSONContract(t *testing.T) {
	got := renderToMap(t, func(w *bytes.Buffer) error {
		return RenderTemplateDriftResponse(w, cliout.FormatJSON, TemplateDriftReport{
			Scenarios: []TemplateDriftScenarioReport{{
				Scenario:        "alpha",
				TemplateID:      "react-vite",
				ManifestDrifted: true,
				Status:          TemplateDriftStatusDrifted,
				FileDiffs:       []TemplateDriftFileDiff{{Path: "a.go", Reason: "modified"}},
			}},
		})
	})
	if got["success"] != true {
		t.Errorf("drift success: %v", got)
	}
	scen := got["drift"].(map[string]any)["scenarios"].([]any)[0].(map[string]any)
	if scen["template_id"] != "react-vite" || scen["status"] != "drifted" {
		t.Errorf("drift scenario: %v", scen)
	}
	if scen["recorded_manifest_sha"] != "" {
		t.Errorf("recorded_manifest_sha should be empty string: %v", scen["recorded_manifest_sha"])
	}
}

func TestScenarioTemplateValidateJSONContract(t *testing.T) {
	// failing case: success mirrors len(issues)==0.
	got := renderToMap(t, func(w *bytes.Buffer) error {
		return RenderTemplateValidateResponse(w, cliout.FormatJSON, TemplateValidationReport{
			Mode:           TemplateValidationModeDeep,
			Count:          2,
			WarningSummary: TemplateValidationWarningSummary{Total: 1, Phases: []TemplateValidationPhaseWarningSummary{{Name: "build", Count: 1, Warnings: []TemplateValidationWarning{{Message: "w"}}}}},
			Issues:         []TemplateValidationIssue{{Template: "t", Message: "bad"}},
		})
	})
	if got["success"] != false {
		t.Errorf("validate success should be false with issues: %v", got["success"])
	}
	report := got["report"].(map[string]any)
	isJSONNumber(t, report["count"], "report.count")
	if report["mode"] != "deep" {
		t.Errorf("mode: %v", report["mode"])
	}
	ws := report["warning_summary"].(map[string]any)
	isJSONNumber(t, ws["total"], "warning_summary.total")
}

func TestScenarioTemplateCleanupJSONContract(t *testing.T) {
	got := renderToMap(t, func(w *bytes.Buffer) error {
		return RenderTemplateCleanupResponse(w, cliout.FormatJSON, templatevalidation.CleanupResult{
			CleanupPlan: templatevalidation.CleanupPlan{
				DryRun:    true,
				OlderThan: time.Hour,
				Eligible: []templatevalidation.Run{{
					MarkerPath: "/m",
					Marker:     templatevalidation.RunMarker{RunID: "r1", CreatorPID: 7, CreatedAt: time.Date(2026, 6, 11, 8, 0, 0, 0, time.UTC)},
					Age:        "1h",
				}},
				Skipped: []templatevalidation.SkippedRun{{Path: "/p", Reason: "retained"}},
			},
			Message: "done",
		})
	})
	if got["success"] != true {
		t.Errorf("cleanup success (no failures): %v", got["success"])
	}
	cleanup := got["cleanup"].(map[string]any)
	// older_than is int64 -> JSON STRING (documented), not a number.
	if _, ok := cleanup["older_than"].(string); !ok {
		t.Errorf("older_than (int64) should be a JSON string: %T %v", cleanup["older_than"], cleanup["older_than"])
	}
	run := cleanup["eligible"].([]any)[0].(map[string]any)
	if run["marker_path"] != "/m" {
		t.Errorf("marker_path snake_case: %v", run)
	}
	marker := run["marker"].(map[string]any)
	isJSONNumber(t, marker["creator_pid"], "marker.creator_pid")
	if marker["created_at"] != "2026-06-11T08:00:00Z" {
		t.Errorf("marker created_at: %v", marker["created_at"])
	}
}

func TestScenarioDesignJSONContract(t *testing.T) {
	kits := []DesignKitInfo{{
		ID:   "modern",
		Path: "/kits/modern",
		Manifest: DesignKitManifest{
			ID:       "modern",
			Name:     "Modern",
			Default:  true,
			Adapters: map[string]DesignKitAdapter{"react": {Path: "/a", Supports: []string{"web"}}},
		},
	}}

	gotList := renderToMap(t, func(w *bytes.Buffer) error {
		return RenderDesignListResponse(w, cliout.FormatJSON, kits)
	})
	if gotList["success"] != true {
		t.Errorf("design list success: %v", gotList)
	}
	kit := gotList["design_kits"].([]any)[0].(map[string]any)
	if kit["id"] != "modern" {
		t.Errorf("design kit id: %v", kit)
	}

	gotShow := renderToMap(t, func(w *bytes.Buffer) error {
		return RenderDesignShowResponse(w, cliout.FormatJSON, kits[0])
	})
	if gotShow["design_kit"].(map[string]any)["id"] != "modern" {
		t.Errorf("design show: %v", gotShow)
	}

	gotVal := renderToMap(t, func(w *bytes.Buffer) error {
		return RenderDesignValidateResponse(w, cliout.FormatJSON, DesignValidationReport{
			Count:  1,
			Issues: []DesignValidationIssue{{Kit: "modern", Message: "x"}},
		})
	})
	if gotVal["success"] != true {
		t.Errorf("design validate success: %v", gotVal)
	}
	dv := gotVal["design_validation"].(map[string]any)
	isJSONNumber(t, dv["count"], "design_validation.count")
}

func TestScenarioOrientationJSONContract(t *testing.T) {
	got := renderToMap(t, func(w *bytes.Buffer) error {
		return RenderOrientationResponse(w, cliout.FormatJSON, OrientationReport{
			Scenario:  "alpha",
			Completed: 1,
			Required:  3,
			Template:  GenerationTemplate{ID: "t", Version: "1"},
			Design:    GenerationDesign{ID: "d"},
			Steps:     []OrientationStepReport{{ID: "s1", Required: true, Complete: true, Checks: []OrientationCheckReport{{Kind: "file", Passed: true}}}},
			NextStep:  &OrientationStepReport{ID: "s2"},
		})
	})
	if got["success"] != true {
		t.Errorf("orientation success: %v", got)
	}
	report := got["orientation"].(map[string]any)
	isJSONNumber(t, report["completed"], "orientation.completed")
	isJSONNumber(t, report["required"], "orientation.required")
	if report["template"].(map[string]any)["id"] != "t" {
		t.Errorf("orientation template: %v", report)
	}
	if report["next_step"].(map[string]any)["id"] != "s2" {
		t.Errorf("next_step snake_case: %v", report)
	}
}

func TestScenarioValidateEnvJSONContract(t *testing.T) {
	got := renderToMap(t, func(w *bytes.Buffer) error {
		return RenderValidateEnvResponse(w, cliout.FormatJSON, ValidateEnvResponse{
			Report: resources.ScenarioEnvValidationReport{
				Scenario: "alpha",
				Values:   map[string]string{"API_PORT": "5329"},
				Issues:   []resources.ValidationIssue{{Severity: "warning", Message: "w"}},
				ResourceReports: []resourceenv.ResourceReport{
					{Name: "redis", Manifest: "/m", Values: map[string]string{"REDIS_PORT": "6379"}},
				},
				Passed: true,
			},
		})
	})
	if got["success"] != true {
		t.Errorf("validate-env success mirrors passed: %v", got["success"])
	}
	report := got["report"].(map[string]any)
	if report["passed"] != true {
		t.Errorf("report.passed: %v", report)
	}
	rr := report["resource_reports"].([]any)[0].(map[string]any)
	if rr["manifest_path"] != "/m" {
		t.Errorf("resource report manifest_path snake_case: %v", rr)
	}
}
