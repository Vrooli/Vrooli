package lifecycleinference

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPythonJavaLifecycleInferenceUsesFiniteDeclarationEvidenceOnly(t *testing.T) {
	root := t.TempDir()
	scenario := filepath.Join(root, "scenarios", "sample")
	for _, dir := range []string{"api/domain", "cli", "ui/src"} {
		if err := os.MkdirAll(filepath.Join(scenario, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	py := "from enum import Enum\nclass JobState(Enum):\n    QUEUED = 'queued'\n    RUNNING = 'running'\nclass Job:\n    status: JobState\n    phase: str\n    def run(self):\n        status: JobState\n"
	java := "enum OrderState { NEW, PAID, SHIPPED }\nclass Order {\n  private OrderState status;\n  private String state;\n  void run() {\n    OrderState status;\n  }\n}\n"
	if err := os.WriteFile(filepath.Join(scenario, "api/domain/job.py"), []byte(py), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenario, "api/domain/Order.java"), []byte(java), 0o644); err != nil {
		t.Fatal(err)
	}
	candidates, files, errors := analyzePythonJavaScenario(scenario, "sample")
	if files != 2 || errors != 0 {
		t.Fatalf("files=%d errors=%d", files, errors)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected two finite-enum lifecycle leads, got %#v", candidates)
	}
	for _, candidate := range candidates {
		if candidate.Persistence.Class != ClassUnknown || !candidate.Persistence.ScanComplete {
			t.Fatalf("declarations must not imply persistence: %#v", candidate)
		}
	}
}
