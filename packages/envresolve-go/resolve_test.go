package envresolve

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResourceExportIsDerivedFromManifest(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "demo")
	scenarioDir := filepath.Join(root, "scenarios", "consumer", ".vrooli")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(resourceDir, "resource.json"), `{"environment_exports":{"static":{"DEMO_HOST":"localhost"},"derived":{"DEMO_URL":{"template":"x"}}}}`)
	writeFile(t, filepath.Join(scenarioDir, "service.json"), `{"ports":{"api":{"env_var":"API_PORT"}}}`)
	idx, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := idx.Producers("DEMO_URL"); len(got) != 1 || got[0].Resource != "demo" {
		t.Fatalf("producers = %#v", got)
	}
}

func TestUnknownVariableHasNoProducer(t *testing.T) {
	idx, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got := idx.Producers("UNKNOWN"); len(got) != 0 {
		t.Fatalf("producers = %#v", got)
	}
}

func TestScenarioPrefixIsAProducerPattern(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "scenarios", "agent-manager", ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "service.json"), `{"ports":{"api":{"env_var":"API_PORT"}}}`)
	idx, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	got := idx.Producers("AGENT_MANAGER_API_PORT")
	if len(got) != 1 || got[0].Scenario != "agent-manager" {
		t.Fatalf("producers = %#v", got)
	}
}

func TestDeadResourceIsDerivedFromSourceReference(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "scenarios", "consumer", ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "service.json"), `{}`)
	writeFile(t, filepath.Join(root, "scenarios", "consumer", "main.go"), `package main
func getResourcePort(string) string { return "" }
func main() { _ = getResourcePort("n8n") }
`)
	idx, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := idx.DeadResource("N8N_BASE_URL"); got != "n8n" {
		t.Fatalf("dead resource = %q, want n8n", got)
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
