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
