package capabilities

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
)

func TestRegistryDoesNotDescribeManifestDependencies(t *testing.T) {
	registry := capabilityRegistryForTest()
	data, err := registry.Describe(context.Background())
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	var payload struct {
		Definitions []Def   `json:"definitions"`
		States      []State `json:"states"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Definitions) != 0 || len(payload.States) != 0 {
		t.Fatalf("description duplicates manifest dependencies: %s", data)
	}
}

func capabilityRegistryForTest() *Registry {
	checkers := map[string]Checker{}
	for _, def := range Known {
		checkers[def.ID] = ScenarioChecker{Slug: def.DependencySlug, Run: func(context.Context, string, ...string) ([]byte, error) {
			return []byte(`{"status":"healthy"}`), nil
		}}
	}
	return capabilityregistry.New(Known, checkers, time.Minute)
}
