package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

func TestShowMainHelpIncludesContractCommand(t *testing.T) {
	var stdout bytes.Buffer
	showMainHelp(&stdout)

	if !strings.Contains(stdout.String(), "contract") {
		t.Fatalf("help missing contract command: %q", stdout.String())
	}
}

func TestRunContractShowJSON(t *testing.T) {
	root := repoRootFromCaller(t)
	t.Setenv("VROOLI_SOURCE_ROOT", root)

	app := newTestApp("/unused")
	var stdout bytes.Buffer
	code := app.Run([]string{"contract", "show", "--json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%s", code, stdout.String())
	}

	var payload struct {
		Success      bool   `json:"success"`
		Root         string `json:"root"`
		ContractPath string `json:"contract_path"`
		Version      string `json:"version"`
		Schema       string `json:"schema"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if !payload.Success || payload.Root != root {
		t.Fatalf("payload = %+v", payload)
	}
	if payload.Version != repocontractmeta.DefaultContractVersion || payload.Schema != repocontractmeta.ContractSchemaRef {
		t.Fatalf("payload = %+v", payload)
	}
	if !strings.HasSuffix(payload.ContractPath, filepath.Join(repocontractmeta.ProjectConfigDir, repocontractmeta.ContractFilename)) {
		t.Fatalf("contract path = %q", payload.ContractPath)
	}
}

func TestRunContractResolveScenarioServicePath(t *testing.T) {
	root := repoRootFromCaller(t)
	t.Setenv("VROOLI_SOURCE_ROOT", root)

	app := newTestApp("/unused")
	var stdout bytes.Buffer
	code := app.Run([]string{"contract", "resolve", "scenario", "test-genie", "--file", "service"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%s", code, stdout.String())
	}

	want := filepath.Join(root, "scenarios", "test-genie", ".vrooli", "service.json")
	if got := strings.TrimSpace(stdout.String()); got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestRunContractMatchGlobJSON(t *testing.T) {
	app := newTestApp("/unused")
	var stdout bytes.Buffer
	code := app.Run([]string{"contract", "match-glob", "scenarios/*/.vrooli/service.json", "scenarios/test-genie/.vrooli/service.json", "--json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%s", code, stdout.String())
	}

	var payload struct {
		Success bool `json:"success"`
		Matched bool `json:"matched"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if !payload.Success || !payload.Matched {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestRunContractValidateJSON(t *testing.T) {
	root := t.TempDir()
	copyRepoContractValidationFixtures(t, root)
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, filepath.Join("scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha"}}`)
	writeTestFile(t, root, filepath.Join("resources", "redis", "resource.json"), `{"name":"redis"}`)
	t.Setenv("VROOLI_SOURCE_ROOT", root)

	app := newTestApp("/unused")
	var stdout bytes.Buffer
	code := app.Run([]string{"contract", "validate", "--json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%s", code, stdout.String())
	}

	var payload struct {
		Success bool `json:"success"`
		Schema  struct {
			Passed bool `json:"passed"`
		} `json:"schema"`
		Report struct {
			Success bool `json:"success"`
			Checks  []struct {
				Name   string `json:"name"`
				Passed bool   `json:"passed"`
			} `json:"checks"`
		} `json:"report"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if !payload.Success || !payload.Schema.Passed || !payload.Report.Success {
		t.Fatalf("payload = %+v", payload)
	}
	if len(payload.Report.Checks) == 0 {
		t.Fatal("expected validation checks")
	}
}

func TestRunContractValidateReturnsSilentNonZeroOnCheckFailure(t *testing.T) {
	root := t.TempDir()
	copyRepoContractValidationFixtures(t, root)
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, "docs/repo-contract.md", "broken docs\n")
	writeTestFile(t, root, filepath.Join("scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha"}}`)
	writeTestFile(t, root, filepath.Join("resources", "redis", "resource.json"), `{"name":"redis"}`)

	t.Setenv("VROOLI_SOURCE_ROOT", root)
	app := newTestApp("/unused")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.Run([]string{"contract", "validate", "--json"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if strings.TrimSpace(stderr.String()) != "" {
		t.Fatalf("expected silent stderr, got %q", stderr.String())
	}

	var payload struct {
		Success bool `json:"success"`
		Report  struct {
			Success bool `json:"success"`
			Checks  []struct {
				Name    string `json:"name"`
				Passed  bool   `json:"passed"`
				Message string `json:"message"`
			} `json:"checks"`
		} `json:"report"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if payload.Success || payload.Report.Success {
		t.Fatalf("payload = %+v", payload)
	}
	foundDocsFailure := false
	for _, check := range payload.Report.Checks {
		if check.Name == "docs_alignment" && !check.Passed {
			foundDocsFailure = true
			break
		}
	}
	if !foundDocsFailure {
		t.Fatalf("expected docs_alignment failure, payload = %+v", payload)
	}
}

func copyRepoContractValidationFixtures(t *testing.T, dest string) {
	t.Helper()
	sourceRoot := repoRootFromCaller(t)
	for _, rel := range []string{
		filepath.Join(repocontractmeta.ProjectConfigDir, repocontractmeta.ContractFilename),
		filepath.Join(repocontractmeta.ProjectConfigDir, repocontractmeta.AdoptionExceptionsFilename),
		filepath.Join(repocontractmeta.ProjectConfigDir, repocontractmeta.SchemaDir, repocontractmeta.SchemaFilename),
		filepath.Join(repocontractmeta.ProjectConfigDir, repocontractmeta.SchemaDir, repocontractmeta.CommonSchemaFilename),
		filepath.Join(repocontractmeta.ProjectConfigDir, repocontractmeta.SchemaDir, repocontractmeta.ValidationScriptFilename),
		"AGENTS.md",
		repocontractmeta.DocsPath,
		filepath.Join("docs", "CONTRIBUTING.md"),
		filepath.Join("scenarios", "prompt-manager", "store", "skills", "packs", "core", "cross-platform-readiness", "SKILL.md"),
	} {
		data, err := os.ReadFile(filepath.Join(sourceRoot, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		writeTestFile(t, dest, rel, string(data))
	}
	for _, dir := range []string{"templates", "packages", "cmd", "internal", "docs"} {
		if err := os.MkdirAll(filepath.Join(dest, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
}
