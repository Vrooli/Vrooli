package tscodegraph_test

import (
	"context"
	"errors"
	"testing"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/graph/tscodegraph"
)

func TestClient_NameAndLanguages(t *testing.T) {
	c := tscodegraph.New("")
	if got := c.Name(); got != "typescript" {
		t.Fatalf("Name=%q", got)
	}
	langs := c.SupportedLanguages()
	if len(langs) != 1 || langs[0] != graph.LanguageTypeScript {
		t.Fatalf("SupportedLanguages=%v", langs)
	}
}

func TestClient_ExtractReturnsIntegrationError(t *testing.T) {
	c := tscodegraph.New("")
	_, err := c.Extract(context.Background(), "demo")
	var ie graph.IntegrationError
	if !errors.As(err, &ie) {
		t.Fatalf("want IntegrationError, got %v", err)
	}
	if ie.Scenario != tscodegraph.ScenarioName {
		t.Fatalf("Scenario=%q", ie.Scenario)
	}
}
