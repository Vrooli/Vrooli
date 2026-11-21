package wizard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestStepHandlersMatchTheDeclaredStepModel(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "api", "testdata", "step-model.json"))
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Steps []struct {
			ID string `json:"id"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatal(err)
	}
	want := make([]string, 0, len(contract.Steps))
	for _, step := range contract.Steps {
		want = append(want, step.ID)
	}
	got := make([]string, 0, len(stepHandlers))
	for id := range stepHandlers {
		got = append(got, id)
	}
	sort.Strings(want)
	sort.Strings(got)
	if len(want) != len(got) {
		t.Fatalf("handler ids = %v, contract ids = %v", got, want)
	}
	for index := range want {
		if want[index] != got[index] {
			t.Fatalf("handler ids = %v, contract ids = %v", got, want)
		}
	}
}
