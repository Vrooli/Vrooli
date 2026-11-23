package conformance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePolicyFixture(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasPolicyCode(report ScanReport, code string) bool {
	for _, finding := range report.Findings {
		if finding.GetRuleId() == code {
			return true
		}
	}
	return false
}

func TestScanProjectPolicyPositiveFixture(t *testing.T) {
	root := t.TempDir()
	writePolicyFixture(t, root, "scenarios/demo/api/main.go", "package main\nfunc main() {}\n")
	report, err := NewScanner().ScanProject(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("clean policy findings = %+v", report.Findings)
	}
}

func TestScanProjectPolicyNegativeFixtures(t *testing.T) {
	tests := []struct{ name, code, body string }{
		{"ollama gateway", "AI_OLLAMA_GATEWAY_ONLY", "package main\nconst endpoint = \"/api/chat\"\n"},
		{"ollama facts", "AI_OLLAMA_POLICY_FACTS", "package main\nconst vectorSize = 768\n"},
		{"openrouter facts", "AI_OPENROUTER_POLICY_FACTS", "package main\nconst model = \"openai/gpt-4o\"\n"},
		{"ollama runner option key", "AI_OLLAMA_RUNTIME_OPTIONS", "package main\nvar options = map[string]any{\"num_ctx\": 32768}\n"},
		{"ollama runner gateway flag", "AI_OLLAMA_RUNTIME_OPTIONS", "package main\nvar args = []string{\"gateway\", \"generate\", \"--num-thread\", \"8\"}\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writePolicyFixture(t, root, "scenarios/demo/api/main.go", tc.body)
			report, err := NewScanner().ScanProject(t.Context(), root)
			if err != nil {
				t.Fatal(err)
			}
			if !hasPolicyCode(report, tc.code) {
				t.Fatalf("missing %s in %+v", tc.code, report.Findings)
			}
		})
	}
}

func TestScanProjectPolicyToleratesLargeSourceLines(t *testing.T) {
	root := t.TempDir()
	writePolicyFixture(t, root, "scenarios/demo/api/generated.json", `{"payload":"`+strings.Repeat("x", 128*1024)+`"}`)
	if _, err := NewScanner().ScanProject(t.Context(), root); err != nil {
		t.Fatalf("large source line should be scannable: %v", err)
	}
}

func TestScanProjectSkipsLargeNonRuntimeArtifactsButScansRuntimeSelectors(t *testing.T) {
	root := t.TempDir()
	writePolicyFixture(t, root, "docs/reference/large.json", `{"payload":"`+strings.Repeat("x", 5*1024*1024)+`"}`)
	writePolicyFixture(t, root, "scenarios/demo/api/testdata/large.json", `{"payload":"`+strings.Repeat("x", 5*1024*1024)+`"}`)
	writePolicyFixture(t, root, "scenarios/demo/api/main.go", "package main\nconst endpoint = \"/api/chat\"\n")
	report, err := NewScanner().ScanProject(t.Context(), root)
	if err != nil {
		t.Fatalf("non-runtime artifacts should not block policy scan: %v", err)
	}
	if !hasPolicyCode(report, "AI_OLLAMA_GATEWAY_ONLY") {
		t.Fatalf("runtime selector was not scanned: %+v", report.Findings)
	}
}

func TestScanProjectExcludesOwnerPolicyAndCatalogSurfaces(t *testing.T) {
	root := t.TempDir()
	writePolicyFixture(t, root, "resources/orcarouter/model-policy.json", `{"model":"openai/gpt-4o"}`)
	writePolicyFixture(t, root, "resources/claude-code/model-policy.json", `{"model":"anthropic/claude-3.5-sonnet"}`)
	writePolicyFixture(t, root, "scenarios/image-tools/api/internal/models/registry.seed.json", `{"id":"qwen2.5-vl-3b"}`)
	writePolicyFixture(t, root, "scenarios/vrooli-autoheal/api/vrooli-autoheal-api.freshness.json", `{"size":768}`)
	report, err := NewScanner().ScanProject(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("owner policy/catalog artifacts should not be reported as callers: %+v", report.Findings)
	}
}

func TestScanProjectFindsOllamaURLWithProtocolDelimiter(t *testing.T) {
	root := t.TempDir()
	writePolicyFixture(t, root, "scenarios/demo/api/main.go", `package main
const endpoint = "http://localhost:11434/api/chat"
`)
	report, err := NewScanner().ScanProject(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasPolicyCode(report, "AI_OLLAMA_GATEWAY_ONLY") {
		t.Fatalf("raw Ollama URL escaped policy scan: %+v", report.Findings)
	}
}

func TestScanProjectKeepsScenarioHostedProviderSelectionVisible(t *testing.T) {
	root := t.TempDir()
	writePolicyFixture(t, root, "scenarios/landing-page-business-suite/api/internal/intelligence/model_policy.go", `const model = "openai/gpt-4o"`)
	report, err := NewScanner().ScanProject(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasPolicyCode(report, "AI_OPENROUTER_POLICY_FACTS") {
		t.Fatalf("scenario-owned hosted model selection must stay visible: %+v", report.Findings)
	}
}

func TestScanProjectIgnoresCommentsAndNonOllamaCatalogIdentifiers(t *testing.T) {
	root := t.TempDir()
	writePolicyFixture(t, root, "scenarios/demo/api/main.go", strings.Join([]string{
		"package main",
		"// Previous approach used qwen3:4b and mistral:7b.",
		`// Previous endpoint was "http://localhost:11434/api/chat".`,
		`const pdfEngine = "mistral-ocr"`,
		`const provider = "mistral/"`,
	}, "\n"))
	report, err := NewScanner().ScanProject(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("comments and non-Ollama catalog names are not runtime model selection: %+v", report.Findings)
	}
}

func TestScanProjectFlagsOllamaRuntimeOptionsInOtherResourcesButNotSimilarKeys(t *testing.T) {
	root := t.TempDir()
	writePolicyFixture(t, root, "resources/opencode/cli/internal/config/config.go", "package config\nvar inner = map[string]any{\"num_ctx\": 16384}\n")
	writePolicyFixture(t, root, "resources/sherpa-onnx/server/asr.go", "package server\nvar opts = map[string]any{\"num_threads\": 2}\n")
	writePolicyFixture(t, root, "resources/ollama/cli/internal/runtimeprofile/profile.go", "package runtimeprofile\nvar keys = []string{\"num_ctx\"}\n")
	report, err := NewScanner().ScanProject(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, finding := range report.Findings {
		if finding.GetRuleId() == "AI_OLLAMA_RUNTIME_OPTIONS" {
			paths = append(paths, finding.GetPath())
		}
	}
	if len(paths) != 1 || !strings.HasPrefix(paths[0], "resources/opencode/") {
		t.Fatalf("AI_OLLAMA_RUNTIME_OPTIONS findings = %v, want only the opencode caller", paths)
	}
}
