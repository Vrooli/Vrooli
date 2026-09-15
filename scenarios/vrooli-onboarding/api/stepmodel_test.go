package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/operatorstate"
)

func TestOnboardingStepModelIsOrderedAndEmptyStateIsUnsatisfied(t *testing.T) {
	if len(onboardingSteps) != 10 {
		t.Fatalf("step count = %d, want 10", len(onboardingSteps))
	}
	empty := OperatorState{}
	for index, step := range onboardingSteps {
		if step.Ordinal != index || step.ID == "" || step.Route == "" {
			t.Fatalf("invalid step %d: %#v", index, step)
		}
		if step.Satisfied(empty) {
			t.Errorf("empty state satisfies step %s", step.ID)
		}
	}
	if got := firstUnsatisfiedStep(empty); got != 0 {
		t.Fatalf("first empty step = %d, want 0", got)
	}
	if got := firstUnsatisfiedStep(OperatorState{Version: "1", Session: &operatorstate.Session{Step: 1}, Core: &operatorstate.CoreSet{Seed: []string{"demo"}, TrustedBase: []string{"demo"}}, Scenarios: map[string]ScenarioChoice{"demo": {Enabled: boolPtr(true)}}, Resources: map[string]EnabledChoice{}, HostTools: map[string]OptInChoice{}}); got != 8 {
		t.Fatalf("first configured step = %d, want apply step 8", got)
	}
}

func TestPublicStepModelMatchesCheckedInContract(t *testing.T) {
	update := flag.Lookup("update")
	if update == nil {
		flag.Bool("update", false, "rewrite generated onboarding contract artifacts")
	}
	contractPath := filepath.Join("testdata", "step-model.json")
	contract, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read %s: %v", contractPath, err)
	}
	generated, err := json.MarshalIndent(map[string]any{"steps": publicStepModel()}, "", "  ")
	if err != nil {
		t.Fatalf("marshal public step model: %v", err)
	}
	generated = append(generated, '\n')
	var want, got any
	if err := json.Unmarshal(contract, &want); err != nil {
		t.Fatalf("decode %s: %v", contractPath, err)
	}
	if err := json.Unmarshal(generated, &got); err != nil {
		t.Fatalf("decode generated model: %v", err)
	}
	if !bytes.Equal(mustCanonicalJSON(t, want), mustCanonicalJSON(t, got)) {
		if flag.Lookup("update").Value.String() == "true" {
			if err := os.WriteFile(contractPath, generated, 0o644); err != nil {
				t.Fatalf("write %s: %v", contractPath, err)
			}
			return
		}
		t.Fatalf("step model drifted; regenerate with: go test ./... -run TestPublicStepModelMatchesCheckedInContract -update")
	}
}

func mustCanonicalJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal canonical JSON: %v", err)
	}
	return data
}

func boolPtr(value bool) *bool { return &value }
