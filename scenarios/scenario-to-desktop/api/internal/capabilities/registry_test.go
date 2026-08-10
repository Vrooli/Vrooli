package capabilities

import (
	"context"
	"strings"
	"testing"
	"time"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
)

func TestRegistryDescribesDeclaredDependencies(t *testing.T) {
	registry := capabilityRegistryForTest()
	data, err := registry.Describe(context.Background())
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	body := string(data)
	for _, slug := range []string{"agent-manager", "deployment-manager", "vrooli-bridge"} {
		if !strings.Contains(body, `"dependencySlug":"`+slug+`"`) {
			t.Fatalf("description does not contain %q: %s", slug, body)
		}
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
