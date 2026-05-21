// Package gocodegraph is the production CodeGraphAdapter for Go
// sources. v0.1 ships a typed-error stub: the `go-code-graph` scenario
// does not exist yet (tracked in docs/internal/PROBLEMS.md) so every
// Extract call returns IntegrationError{Kind:"scenario_unreachable"}.
//
// Once go-code-graph ships its Connect-RPC contract, this package
// becomes a thin Connect client per docs/concepts/INTEGRATIONS.md. The
// interface is fixed; the body is the only thing that changes.
package gocodegraph

import (
	"context"
	"errors"

	"architecture-cartographer/internal/graph"
)

// ScenarioName is the canonical scenario identifier the discovery
// layer uses to resolve the go-code-graph base URL.
const ScenarioName = "go-code-graph"

// Client is the production CodeGraphAdapter for Go.
type Client struct {
	baseURL string
}

// New returns a Client wired against baseURL (resolved by the
// scenario-discovery layer). An empty baseURL is allowed at
// construction; Extract returns IntegrationError on first use.
func New(baseURL string) *Client { return &Client{baseURL: baseURL} }

var _ graph.CodeGraphAdapter = (*Client)(nil)

// Name returns the adapter identifier.
func (c *Client) Name() string { return "go" }

// SupportedLanguages reports that this adapter returns Go-language
// nodes only.
func (c *Client) SupportedLanguages() []graph.Language {
	return []graph.Language{graph.LanguageGo}
}

// Extract performs an ExtractGraph call against the go-code-graph
// scenario. v0.1 always returns IntegrationError until the dependency
// scenario ships.
func (c *Client) Extract(_ context.Context, scenario string) (graph.RawGraph, error) {
	return graph.RawGraph{}, graph.IntegrationError{
		Kind:     "scenario_unreachable",
		Scenario: ScenarioName,
		Cause: errors.New(
			"go-code-graph scenario not implemented; cartographer is built against the CodeGraphAdapter seam — " +
				"see docs/internal/PROBLEMS.md and docs/concepts/INTEGRATIONS.md",
		),
	}
}
