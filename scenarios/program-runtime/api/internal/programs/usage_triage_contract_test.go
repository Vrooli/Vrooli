package programs

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestUsageTriageContractDeclaresBoundedSingleBatch [REQ:PRT-P1-014]
// protects the usage-triage contract's bounded attribution and inference
// shape. Live fixtures do not inject failures or candidates; deterministic
// source tests below cover those paths. This checks the declared live fixture
// and prevents the one-batch budget becoming an unbounded classifier loop.
func TestUsageTriageContractDeclaresBoundedSingleBatch(t *testing.T) {
	root := filepath.Join("..", "..", "..", ".vrooli", "program-runtime")
	contractBytes, err := os.ReadFile(filepath.Join(root, "usage-triage.json"))
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Name     string `json:"name"`
		Bindings []struct {
			ID     string `json:"id"`
			Via    string `json:"via"`
			Effect string `json:"effect"`
		} `json:"bindings"`
		Invariants []string `json:"invariants"`
		Fixtures   []struct {
			ID string `json:"id"`
		} `json:"fixtures"`
		Budget struct {
			InferenceCalls   int `json:"inference_calls"`
			MaterializeLimit int `json:"materialize_limit"`
		} `json:"budget"`
	}
	if err := json.Unmarshal(contractBytes, &contract); err != nil {
		t.Fatal(err)
	}
	if contract.Name != "program-runtime.usage-triage" {
		t.Fatalf("contract name = %q", contract.Name)
	}
	if contract.Budget.InferenceCalls != 1 || contract.Budget.MaterializeLimit != 32 {
		t.Fatalf("budget = %+v, want one inference and 32 materialized rows", contract.Budget)
	}

	wantBindings := map[string]bool{
		"program-runtime/programs/portfolio": false,
		"program-runtime/programs/list":      false,
		"agent-manager/run/list":             false,
		"agent-manager/conversation/search":  false,
		"ai-gateway/inference/run-batch":     false,
	}
	for _, binding := range contract.Bindings {
		if _, ok := wantBindings[binding.ID]; ok {
			wantBindings[binding.ID] = true
		}
		if binding.ID == "ai-gateway/inference/run-batch" && binding.Via != "ai.classify" {
			t.Errorf("classifier binding via = %q, want ai.classify", binding.Via)
		}
		if binding.Effect != "read" {
			t.Errorf("binding %s has effect %q, want read", binding.ID, binding.Effect)
		}
	}
	for id, present := range wantBindings {
		if !present {
			t.Errorf("required binding %q is missing", id)
		}
	}

	wantFixtures := map[string]bool{"no-candidates": false}
	for _, fixture := range contract.Fixtures {
		if _, ok := wantFixtures[fixture.ID]; ok {
			wantFixtures[fixture.ID] = true
		}
	}
	for id, present := range wantFixtures {
		if !present {
			t.Errorf("required fixture %q is missing", id)
		}
	}

	joined := strings.Join(contract.Invariants, "\n")
	for _, phrase := range []string{"deterministic", "At most one", "No transcript body"} {
		if !strings.Contains(joined, phrase) {
			t.Errorf("invariants do not state %q: %s", phrase, joined)
		}
	}
	source, err := os.ReadFile(filepath.Join(root, "usage-triage.py"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(source), "ai.batch("); got != 1 {
		t.Fatalf("ai.batch call count = %d, want exactly one", got)
	}
}

// TestProgramSourceDegradationFixtures exercises real program source with
// controlled binding responses; programs-phase live runs cannot inject outages.
func TestProgramSourceDegradationFixtures(t *testing.T) {
	path := filepath.Join("..", "..", "..", "kernel", "tests", "usage_triage_fixture_test.py")
	cmd := exec.Command("python3", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("deterministic program source fixtures: %v\n%s", err, output)
	}
}
