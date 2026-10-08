package capacity

import "testing"

func TestCapacityPosturesHaveDistinctReserveAndTierAssignments(t *testing.T) {
	postures := []string{CapacityPostureResponsive, CapacityPostureBalanced, CapacityPostureThroughput, CapacityPostureMinimal}
	seen := map[int64]bool{}
	signatures := map[string]bool{}
	for _, posture := range postures {
		defaults := ResolvePosture(posture)
		if seen[defaults.ReserveBytes] {
			t.Fatalf("posture %s repeats reserve %d", posture, defaults.ReserveBytes)
		}
		seen[defaults.ReserveBytes] = true
		signature := defaults.Resources["ollama"].Priority + "/" + defaults.Resources["whisper"].Priority
		if signatures[signature] {
			t.Fatalf("posture %s repeats tier assignment %s", posture, signature)
		}
		signatures[signature] = true
	}
	if got := ResolvePosture(CapacityPostureResponsive).Resources["whisper"].Priority; got != "interactive" {
		t.Fatalf("responsive whisper priority = %q", got)
	}
	if got := ResolvePosture(CapacityPostureThroughput).Resources["ollama"].Priority; got != "interactive" {
		t.Fatalf("throughput ollama priority = %q", got)
	}
	if got := ResolvePosture(CapacityPostureMinimal).Resources["ollama"].Rung; got != floorRungs["ollama"] {
		t.Fatalf("minimal ollama rung = %q", got)
	}
}

func TestFitStatePrecedenceOverrideThenPostureThenManifest(t *testing.T) {
	state := FitStateForPosture(CapacityPostureMinimal, nil, map[string]FitResourceChoice{
		"ollama": {Rung: "qwen3.5:4b", Priority: "interactive"},
	})
	if state.Resources["ollama"].Rung != "qwen3.5:4b" || state.Resources["ollama"].Priority != "interactive" {
		t.Fatalf("operator override did not win: %#v", state.Resources["ollama"])
	}
	if state.Resources["whisper"].Rung != "cpu" || state.Resources["whisper"].Priority != "batch" {
		t.Fatalf("posture default did not win: %#v", state.Resources["whisper"])
	}
	if _, ok := state.Resources["future-resource"]; ok {
		t.Fatal("unknown resource received a posture choice instead of falling back to its manifest")
	}
}

func TestValidateResourceCapacityChoicesRejectsRungAndTunable(t *testing.T) {
	minimum, maximum := int64(1), int64(8)
	resource := fitResource("ollama", true, false, DegradeStep{Label: "large", AmountBytes: fitGiB}, DegradeStep{Label: "small", AmountBytes: fitGiB / 2})
	resource.Tunables = []DeclaredCapacityTunable{{Name: "num_parallel", Type: "integer", Minimum: &minimum, Maximum: &maximum}}
	if err := ValidateResourceCapacityChoices([]DeclaredResource{resource}, map[string]FitResourceChoice{"ollama": {Rung: "missing"}}); err == nil {
		t.Fatal("missing rung was accepted")
	}
	if err := ValidateResourceCapacityChoices([]DeclaredResource{resource}, map[string]FitResourceChoice{"ollama": {Rung: "small", Tunables: map[string]any{"num_parallel": float64(9)}}}); err == nil {
		t.Fatal("out-of-range tunable was accepted")
	}
	if err := ValidateResourceCapacityChoices([]DeclaredResource{resource}, map[string]FitResourceChoice{"ollama": {Rung: "small", Tunables: map[string]any{"num_parallel": float64(2)}}}); err != nil {
		t.Fatalf("valid capacity choice rejected: %v", err)
	}
}
