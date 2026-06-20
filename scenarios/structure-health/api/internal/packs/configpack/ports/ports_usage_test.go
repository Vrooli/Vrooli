package ports

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServicePortEnvUsageFlagsUnusedDeclaredPort(t *testing.T) {
	root := t.TempDir()
	servicePath := writePortUsageScenarioFile(t, root, ".vrooli/service.json", `{
  "ports": {
    "metrics": {
      "env_var": "METRICS_PORT",
      "range": "30000-30010"
    }
  }
}`)
	writePortUsageScenarioFile(t, root, "api/main.go", `package main

import "net/http"

func main() {
	_ = http.ListenAndServe(":0", nil)
}
`)

	violations := CheckServicePortConfiguration(mustReadFile(t, servicePath), servicePath, "demo")

	if !hasPortUsageMessage(violations, "METRICS_PORT") {
		t.Fatalf("expected unused METRICS_PORT usage violation, got %#v", violations)
	}
}

func TestServicePortEnvUsageAcceptsLikelyListenerReference(t *testing.T) {
	root := t.TempDir()
	servicePath := writePortUsageScenarioFile(t, root, ".vrooli/service.json", `{
  "ports": {
    "metrics": {
      "env_var": "METRICS_PORT",
      "range": "30000-30010"
    }
  }
}`)
	writePortUsageScenarioFile(t, root, "api/main.go", `package main

import (
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("METRICS_PORT")
	_ = http.ListenAndServe(":"+port, nil)
}
`)

	violations := CheckServicePortConfiguration(mustReadFile(t, servicePath), servicePath, "demo")

	if hasPortUsageMessage(violations, "METRICS_PORT") {
		t.Fatalf("expected runtime METRICS_PORT reference to satisfy static usage, got %#v", violations)
	}
}

func TestServicePortEnvUsageRuntimeReferenceWithoutListenerIsAmbiguous(t *testing.T) {
	root := t.TempDir()
	servicePath := writePortUsageScenarioFile(t, root, ".vrooli/service.json", `{
  "ports": {
    "metrics": {
      "env_var": "METRICS_PORT",
      "range": "30000-30010"
    }
  }
}`)
	writePortUsageScenarioFile(t, root, "api/main.go", `package main

import "os"

func main() {
	_ = os.Getenv("METRICS_PORT")
}
`)

	violations := CheckServicePortConfiguration(mustReadFile(t, servicePath), servicePath, "demo")

	if !hasPortUsageMessage(violations, "referenced by runtime source but not near recognized listener startup code") {
		t.Fatalf("expected ambiguous runtime METRICS_PORT reference warning, got %#v", violations)
	}
}

func TestServicePortEnvUsageDocsOnlyIsEvidence(t *testing.T) {
	root := t.TempDir()
	servicePath := writePortUsageScenarioFile(t, root, ".vrooli/service.json", `{
  "ports": {
    "metrics": {
      "env_var": "METRICS_PORT",
      "range": "30000-30010"
    }
  }
}`)
	writePortUsageScenarioFile(t, root, "docs/ports.md", "METRICS_PORT is mentioned here.\n")

	violations := CheckServicePortConfiguration(mustReadFile(t, servicePath), servicePath, "demo")

	if !hasPortUsageMessage(violations, "only referenced") {
		t.Fatalf("expected docs-only METRICS_PORT usage evidence, got %#v", violations)
	}
}

func TestServicePortEnvUsageCorrelatesOptionalRuntimeEvidenceArtifact(t *testing.T) {
	root := t.TempDir()
	servicePath := writePortUsageScenarioFile(t, root, ".vrooli/service.json", `{
  "ports": {
    "metrics": {
      "env_var": "METRICS_PORT",
      "range": "30000-30010"
    }
  }
}`)
	evidencePath := writePortUsageScenarioFile(t, root, "evidence.json", `{
  "registry_claims": [
    {
      "scenario": "demo",
      "port_name": "metrics",
      "env_var": "METRICS_PORT",
      "listener_status": "not_listening",
      "consecutive_listener_misses": 4,
      "recommendation_code": "port-unbound-likely-manifest-drift",
      "recommendation_confidence": "medium"
    }
  ]
}`)
	t.Setenv(runtimePortEvidencePathEnv, evidencePath)

	violations := CheckServicePortConfiguration(mustReadFile(t, servicePath), servicePath, "demo")

	if !hasPortUsageMessage(violations, "historical runtime evidence reports runtime recommendation port-unbound-likely-manifest-drift") {
		t.Fatalf("expected runtime evidence to be included in METRICS_PORT finding, got %#v", violations)
	}
}

func writePortUsageScenarioFile(t *testing.T, root, rel, content string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return content
}

func hasPortUsageMessage(violations []Violation, want string) bool {
	for _, violation := range violations {
		if violation.Type == "config_service_port_usage" && strings.Contains(violation.Description, want) {
			return true
		}
	}
	return false
}
