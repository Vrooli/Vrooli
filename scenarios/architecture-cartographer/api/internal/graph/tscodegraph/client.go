// Package tscodegraph is the production CodeGraphAdapter for
// TypeScript sources. v0.1 ships a typed-error stub: the
// `typescript-code-graph` scenario does not exist yet (tracked in
// docs/internal/PROBLEMS.md) so every Extract call returns
// IntegrationError{Kind:"scenario_unreachable"}.
package tscodegraph

import (
	"context"
	"errors"

	"architecture-cartographer/internal/graph"
)

// ScenarioName is the canonical scenario identifier the discovery
// layer uses to resolve the typescript-code-graph base URL.
const ScenarioName = "typescript-code-graph"

// Client is the production CodeGraphAdapter for TypeScript.
type Client struct {
	baseURL string
}

// New returns a Client wired against baseURL.
func New(baseURL string) *Client { return &Client{baseURL: baseURL} }

var _ graph.CodeGraphAdapter = (*Client)(nil)

func (c *Client) Name() string { return "typescript" }

func (c *Client) SupportedLanguages() []graph.Language {
	return []graph.Language{graph.LanguageTypeScript}
}

func (c *Client) Extract(_ context.Context, scenario string) (graph.RawGraph, error) {
	return graph.RawGraph{}, graph.IntegrationError{
		Kind:     "scenario_unreachable",
		Scenario: ScenarioName,
		Cause: errors.New(
			"typescript-code-graph scenario not implemented; cartographer is built against the CodeGraphAdapter seam — " +
				"see docs/internal/PROBLEMS.md and docs/concepts/INTEGRATIONS.md",
		),
	}
}
