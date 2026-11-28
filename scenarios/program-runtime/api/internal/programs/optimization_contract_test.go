package programs

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestOptimizeProgramUsesOnlyDurableSubmitAndSingleWait(t *testing.T) {
	root := filepath.Join("..", "..", "..", ".vrooli", "program-runtime")
	contractBytes, err := os.ReadFile(filepath.Join(root, "optimize.json"))
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Name   string `json:"name"`
		Inputs map[string]struct {
			Description string `json:"description"`
		} `json:"inputs"`
		Bindings []struct {
			ID       string `json:"id"`
			Effect   string `json:"effect"`
			Optional bool   `json:"optional"`
		} `json:"bindings"`
		Invariants []string `json:"invariants"`
		Outputs    struct {
			Signals map[string]string `json:"signals"`
		} `json:"outputs"`
	}
	if err := json.Unmarshal(contractBytes, &contract); err != nil {
		t.Fatal(err)
	}
	if contract.Name != "program-runtime.optimize" {
		t.Fatalf("contract name=%q", contract.Name)
	}
	if !strings.Contains(contract.Inputs["request"].Description, "optimization.Request") {
		t.Errorf("request description does not name the canonical optimization.Request contract")
	}
	for _, field := range []string{"candidate_artifact_digests", "candidate_results", "promoted_artifact_digest", "removal_receipts"} {
		if contract.Outputs.Signals[field] == "" {
			t.Errorf("output signals do not document owner receipt field %q", field)
		}
	}
	want := map[string]string{
		"program-runtime/optimization/submit": "destructive",
		"program-runtime/optimization/wait":   "read",
	}
	for _, binding := range contract.Bindings {
		if effect, ok := want[binding.ID]; ok {
			if binding.Effect != effect {
				t.Errorf("binding %s effect=%q, want %q", binding.ID, binding.Effect, effect)
			}
			if binding.ID == "program-runtime/optimization/wait" && !binding.Optional {
				t.Errorf("wait binding must be optional because a missing observation leaves the durable run pending")
			}
			delete(want, binding.ID)
		}
	}
	for id := range want {
		t.Errorf("required binding %q is missing", id)
	}
	joined := strings.ToLower(strings.Join(contract.Invariants, "\n"))
	for _, phrase := range []string{"submit is called at most once", "wait is called at most once", "never polls", "pending result", "valid optimization.request", "unknown owner targets fail closed"} {
		if !strings.Contains(joined, phrase) {
			t.Errorf("invariants do not state %q", phrase)
		}
	}

	source, err := os.ReadFile(filepath.Join(root, "optimize.py"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	if got := strings.Count(text, "program_runtime.optimization.submit("); got != 1 {
		t.Errorf("submit binding call count=%d, want 1", got)
	}
	if got := strings.Count(text, "program_runtime.optimization.wait("); got != 1 {
		t.Errorf("wait binding call count=%d, want 1", got)
	}
	for _, forbidden := range []string{"subprocess", "os.system", "resource-ollama optimize", "shell=True"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("program source contains forbidden local execution marker %q", forbidden)
		}
	}
}

func TestOptimizeProgramSourceFixtures(t *testing.T) {
	path := filepath.Join("..", "..", "..", "kernel", "tests", "optimize_entrypoint_test.py")
	cmd := exec.Command("python3", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("deterministic optimizer source fixtures: %v\n%s", err, output)
	}
}
