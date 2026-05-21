package gocodegraph_test

import (
	"context"
	"errors"
	"testing"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/graph/gocodegraph"
)

func TestClient_NameAndLanguages(t *testing.T) {
	c := gocodegraph.New("http://localhost:0")
	if got := c.Name(); got != "go" {
		t.Fatalf("Name=%q want %q", got, "go")
	}
	langs := c.SupportedLanguages()
	if len(langs) != 1 || langs[0] != graph.LanguageGo {
		t.Fatalf("SupportedLanguages=%v", langs)
	}
}

func TestClient_ExtractReturnsIntegrationErrorUntilScenarioShips(t *testing.T) {
	c := gocodegraph.New("")
	_, err := c.Extract(context.Background(), "demo")
	var ie graph.IntegrationError
	if !errors.As(err, &ie) {
		t.Fatalf("want IntegrationError, got %v", err)
	}
	if ie.Kind != "scenario_unreachable" {
		t.Fatalf("Kind=%q", ie.Kind)
	}
	if ie.Scenario != gocodegraph.ScenarioName {
		t.Fatalf("Scenario=%q", ie.Scenario)
	}
}
