package capacity

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
)

type fakeHost struct {
	snap hostinventory.Snapshot
	err  error
}

func (f fakeHost) Collect(context.Context) (hostinventory.Snapshot, error) {
	return f.snap, f.err
}

type fakeOllama struct {
	tags    map[string]bool
	running []ensure.RunningModel
}

func (f fakeOllama) ListTags(context.Context) (map[string]bool, error) {
	return f.tags, nil
}

func (f fakeOllama) ListRunning(context.Context) ([]ensure.RunningModel, error) {
	return f.running, nil
}

func TestBuildReportResolvesScenarioRolesAndBudgets(t *testing.T) {
	root := testRoot(t)
	writeScenario(t, root, "search-hub", `{
	  "dependencies": {
	    "resources": {
	      "ollama": {
	        "type": "ollama",
	        "enabled": true,
	        "model_roles": ["embedding.default", {"role":"chat.default","reason":"classification"}]
	      }
	    }
	  }
	}`)

	h := testHandlers(root)
	report, err := h.BuildReport(context.Background(), PlanRequest{Scenario: "search-hub"})
	if err != nil {
		t.Fatalf("BuildReport() error = %v", err)
	}
	if len(report.Scenarios) != 1 {
		t.Fatalf("Scenarios len = %d, want 1", len(report.Scenarios))
	}
	if got, want := strings.Join(report.Scenarios[0].Models, ","), "nomic-embed-text:latest,qwen3:4b"; got != want {
		t.Fatalf("resolved models = %q, want %q", got, want)
	}
	if report.Totals.DistinctModels != 2 {
		t.Fatalf("DistinctModels = %d, want 2", report.Totals.DistinctModels)
	}
	if len(report.Failures) != 0 {
		t.Fatalf("Failures = %v, want none", report.Failures)
	}
}

func TestBuildReportFlagsDirectModelsAndBudgetFailure(t *testing.T) {
	root := testRoot(t)
	writeScenario(t, root, "agent-manager", `{
	  "dependencies": {
	    "resources": {
	      "ollama": {
	        "type": "ollama",
	        "enabled": true,
	        "models": ["qwen2.5-coder:14b"]
	      }
	    }
	  }
	}`)

	h := testHandlers(root)
	report, err := h.BuildReport(context.Background(), PlanRequest{Scenario: "agent-manager"})
	if err != nil {
		t.Fatalf("BuildReport() error = %v", err)
	}
	if len(report.Models) != 1 || !report.Models[0].Direct {
		t.Fatalf("Models = %+v, want one direct model", report.Models)
	}
	if len(report.Failures) == 0 {
		t.Fatal("Failures empty, want runtime memory budget failure")
	}
	if !containsSubstring(report.Warnings, "direct model") {
		t.Fatalf("Warnings = %v, want direct model warning", report.Warnings)
	}
}

func TestPlanJSONOutput(t *testing.T) {
	root := testRoot(t)
	writeScenario(t, root, "prompt-manager", `{
	  "dependencies": {
	    "resources": {
	      "ollama": {
	        "type": "ollama",
	        "enabled": true,
	        "model_roles": ["embedding.default"]
	      }
	    }
	  }
	}`)

	var stdout bytes.Buffer
	h := testHandlers(root)
	h.Stdout = &stdout
	if err := h.Plan([]string{"--scenario", "prompt-manager", "--json"}); err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	var report Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode JSON output: %v\n%s", err, stdout.String())
	}
	if len(report.Models) != 1 || report.Models[0].Ref != "nomic-embed-text:latest" {
		t.Fatalf("Models = %+v, want nomic-embed-text", report.Models)
	}
}

func testHandlers(root string) *Handlers {
	return &Handlers{
		SourceRoot:  root,
		PolicyPath:  filepath.Join(root, "resources", "ollama", "model-policy.json"),
		RuntimePath: filepath.Join(root, "resources", "ollama", "resource.json"),
		Host: fakeHost{snap: hostinventory.Snapshot{
			OS:   "linux",
			Arch: "amd64",
			CPU:  hostinventory.CPU{Cores: 8},
			Memory: hostinventory.Memory{
				TotalBytes:     32 * bytesPerGB,
				AvailableBytes: 24 * bytesPerGB,
			},
			GPUs: []hostinventory.GPU{{
				Index:     0,
				Name:      "Test GPU",
				VRAMBytes: 24 * bytesPerGB,
				Source:    "nvidia-smi",
			}},
			DockerGPU: hostinventory.DockerGPU{NvidiaRuntime: true},
		}},
		NewClient: func() OllamaClient {
			return fakeOllama{
				tags: map[string]bool{
					"nomic-embed-text:latest": true,
					"qwen3:4b":                true,
				},
				running: []ensure.RunningModel{{Name: "qwen3:4b", SizeVRAM: int64(5 * bytesPerGB)}},
			}
		},
		GetEnv: func(string) string { return "" },
		Stdout: bytes.NewBuffer(nil),
		Stderr: bytes.NewBuffer(nil),
	}
}

func testRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "resources", "ollama"))
	policyData, err := os.ReadFile(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatalf("read live model policy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "resources", "ollama", "model-policy.json"), policyData, 0o644); err != nil {
		t.Fatalf("write model policy: %v", err)
	}
	resourceData := []byte(`{
	  "runtime": {
	    "memory_limit": "12g",
	    "env": {
	      "OLLAMA_NUM_PARALLEL": "4",
	      "OLLAMA_MAX_LOADED_MODELS": "3"
	    }
	  }
	}`)
	if err := os.WriteFile(filepath.Join(root, "resources", "ollama", "resource.json"), resourceData, 0o644); err != nil {
		t.Fatalf("write resource manifest: %v", err)
	}
	return root
}

func writeScenario(t *testing.T, root, name, manifest string) {
	t.Helper()
	dir := filepath.Join(root, "scenarios", name, ".vrooli")
	mustMkdir(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "service.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write scenario manifest: %v", err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func containsSubstring(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
