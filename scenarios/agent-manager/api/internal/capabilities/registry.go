// Package capabilities exposes Agent Manager's declared scenario integrations
// to the dependency analyzer and other control-plane consumers.
package capabilities

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	DependencyScenario      = "scenario"
	ActionKindScenarioStart = "scenario_start"
	StatusUnknown           = "unknown"
)

type PlatformVerdict struct {
	Support string `json:"support,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// RegistryMetadata describes Agent Manager itself. It is separate from
// Known, because service.json is the sole source of declared dependencies.
type RegistryMetadata struct {
	Platform PlatformVerdict `json:"platform"`
}

var Metadata = RegistryMetadata{Platform: PlatformVerdict{Support: "supported", Reason: "Agent Manager's orchestration API is host-neutral; runner adapters declare their own platform boundaries."}}

type Definition struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	DependencyKind  string          `json:"dependencyKind"`
	DependencySlug  string          `json:"dependencySlug"`
	Features        []string        `json:"features,omitempty"`
	ActionKind      string          `json:"actionKind,omitempty"`
	ActionLabel     string          `json:"actionLabel,omitempty"`
	OperatorCommand string          `json:"operatorCommand,omitempty"`
	Platform        PlatformVerdict `json:"platform,omitempty"`
}

type State struct {
	Definition
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type Registry struct {
	definitions []Definition
}

// knownDefinitions contains optional capabilities only. service.json is the
// source of truth for declared dependencies.
var knownDefinitions = []Definition{}

func NewRegistry() *Registry {
	definitions := make([]Definition, len(knownDefinitions))
	copy(definitions, knownDefinitions)
	return &Registry{definitions: definitions}
}

// Check is intentionally conservative: this static descriptor does not claim
// that an optional dependency is live. Runtime health remains the authority
// for availability, while this method gives source-based conformance checks a
// typed reachability seam to inspect.
func (r *Registry) Check(_ context.Context, dependency string) (string, string) {
	for _, definition := range r.definitions {
		if definition.DependencySlug == dependency {
			return StatusUnknown, fmt.Sprintf("%s availability is reported by dependency health", dependency)
		}
	}
	return StatusUnknown, "dependency is not declared"
}

func (r *Registry) Describe(ctx context.Context) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("capability registry is not configured")
	}
	states := make([]State, 0, len(r.definitions))
	for _, definition := range r.definitions {
		status, message := r.Check(ctx, definition.DependencySlug)
		states = append(states, State{Definition: definition, Status: status, Message: message})
	}
	return json.Marshal(struct {
		Definitions []Definition `json:"definitions"`
		States      []State      `json:"states"`
	}{Definitions: r.definitions, States: states})
}
