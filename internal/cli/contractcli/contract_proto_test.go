package contractcli

import (
	"bytes"
	"encoding/json"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	"github.com/vrooli/vrooli/internal/cliout"
)

// TestRenderValidateJSONContract pins the `contract validate --json` wire shape.
func TestRenderValidateJSONContract(t *testing.T) {
	output := contractapp.ValidationOutput{
		Success: true,
		Root:    "/repo",
		Schema:  contractapp.ValidationCheck{Passed: true, Message: "ok"},
		Report: contractapp.Report{
			Root:         "/repo",
			ContractPath: "/repo/.vrooli/repo-contract.json",
			Success:      true,
			Checks: []contractapp.CheckResult{
				{Name: "phase1_semantics", Passed: true, Message: "ok"},
			},
		},
	}

	var buf bytes.Buffer
	if err := RenderValidate(&buf, cliout.FormatJSON, output); err != nil {
		t.Fatalf("RenderValidate: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if got["success"] != true {
		t.Errorf("success: want true, got %v", got["success"])
	}
	if got["root"] != "/repo" {
		t.Errorf("root: %v", got["root"])
	}
	schema, ok := got["schema"].(map[string]any)
	if !ok || schema["passed"] != true || schema["message"] != "ok" {
		t.Fatalf("schema mismatch: %v", got["schema"])
	}
	report, ok := got["report"].(map[string]any)
	if !ok {
		t.Fatalf("report missing/wrong: %v", got["report"])
	}
	if report["contract_path"] != "/repo/.vrooli/repo-contract.json" {
		t.Errorf("contract_path (snake_case?): %v", report)
	}
	checks, ok := report["checks"].([]any)
	if !ok || len(checks) != 1 {
		t.Fatalf("checks: want 1, got %v", report["checks"])
	}
	first := checks[0].(map[string]any)
	if first["name"] != "phase1_semantics" {
		t.Errorf("check name: %v", first["name"])
	}
}

// TestRenderShowJSONContract pins the `contract show --json` wire shape,
// including a sparse case (empty profiles/environment).
func TestRenderShowJSONContract(t *testing.T) {
	output := contractapp.ShowOutput{
		Success:      true,
		Root:         "/repo",
		ContractPath: "/repo/.vrooli/repo-contract.json",
		Schema:       "https://example/schema.json",
		Version:      "1.0.0",
		Platform:     repocontract.Platform{Mode: "cross_platform_go_native"},
		Markers: repocontract.RootMarkers{
			RequiredDirs:  []string{"scenarios"},
			RequiredFiles: []string{"go.mod"},
		},
		Layout: repocontract.Layout{
			ScenarioDir: "scenarios",
			InternalDir: "internal",
			DocsDir:     "docs",
		},
		Scenario: repocontract.ScenarioSpec{
			RequiredFiles:  []string{".vrooli/service.json"},
			WellKnownPaths: map[string]string{"service": ".vrooli/service.json"},
		},
		Resource: repocontract.ResourceSpec{Manifest: "resource.json"},
		Globs:    repocontract.GlobSpec{Syntax: "doublestar", RootRelative: true},
		Sandbox: contractapp.ShowSandbox{
			FullRepoScopes:      []string{"", "."},
			ScenarioScopePrefix: "scenarios/",
		},
		Profiles: map[string]repocontract.Profile{
			"bundle": {Description: "bundle profile", Parameters: []string{"scenario"}},
		},
	}

	var buf bytes.Buffer
	if err := RenderShow(&buf, cliout.FormatJSON, output); err != nil {
		t.Fatalf("RenderShow: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if got["success"] != true {
		t.Errorf("success: %v", got["success"])
	}
	if got["contract_path"] != "/repo/.vrooli/repo-contract.json" {
		t.Errorf("contract_path (snake_case?): %v", got["contract_path"])
	}
	layout, ok := got["layout"].(map[string]any)
	if !ok || layout["scenario_dir"] != "scenarios" {
		t.Fatalf("layout mismatch: %v", got["layout"])
	}
	scenario := got["scenario"].(map[string]any)
	wk, ok := scenario["well_known_paths"].(map[string]any)
	if !ok || wk["service"] != ".vrooli/service.json" {
		t.Errorf("well_known_paths (snake_case?): %v", scenario["well_known_paths"])
	}
	sandbox := got["sandbox"].(map[string]any)
	if _, ok := sandbox["full_repo_scopes"].([]any); !ok {
		t.Errorf("full_repo_scopes (snake_case?): %v", sandbox)
	}
	profiles, ok := got["profiles"].(map[string]any)
	if !ok {
		t.Fatalf("profiles missing: %v", got["profiles"])
	}
	bundle := profiles["bundle"].(map[string]any)
	if bundle["description"] != "bundle profile" {
		t.Errorf("profile description: %v", bundle)
	}
}

// TestRenderResolveAndMatchJSONContract pins the resolve-scenario and match-glob
// `--json` shapes, asserting bool fields stay bools.
func TestRenderResolveAndMatchJSONContract(t *testing.T) {
	var rbuf bytes.Buffer
	resolve := contractapp.ResolveScenarioOutput{
		Success:  true,
		Root:     "/repo",
		Scenario: "demo",
		File:     "service",
		Path:     "scenarios/demo/.vrooli/service.json",
	}
	if err := RenderResolveScenario(&rbuf, cliout.FormatJSON, resolve); err != nil {
		t.Fatalf("RenderResolveScenario: %v", err)
	}
	var rgot map[string]any
	if err := json.Unmarshal(rbuf.Bytes(), &rgot); err != nil {
		t.Fatalf("resolve not valid JSON: %v\n%s", err, rbuf.String())
	}
	if rgot["success"] != true || rgot["scenario"] != "demo" || rgot["file"] != "service" {
		t.Errorf("resolve mismatch: %v", rgot)
	}

	var mbuf bytes.Buffer
	match := contractapp.MatchGlobOutput{Success: true, Pattern: "**/*.go", Path: "a/b.go", Matched: true}
	if err := RenderMatchGlob(&mbuf, cliout.FormatJSON, match); err != nil {
		t.Fatalf("RenderMatchGlob: %v", err)
	}
	var mgot map[string]any
	if err := json.Unmarshal(mbuf.Bytes(), &mgot); err != nil {
		t.Fatalf("match not valid JSON: %v\n%s", err, mbuf.String())
	}
	if mgot["matched"] != true || mgot["pattern"] != "**/*.go" {
		t.Errorf("match mismatch: %v", mgot)
	}
}
