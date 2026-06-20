// Package reconcile pairs a scenario's declared service.json intent with its
// code-facts-detected actuality into a per-surface model. The mismatch between
// the two halves is structure-health's core finding signal.
package reconcile

import (
	"sort"
	"strings"

	"structure-health/internal/intent"
	"structure-health/internal/profile"
)

// SurfaceState is the declared-vs-actual state of one surface.
type SurfaceState struct {
	Surface        string
	Kind           string
	Declared       bool
	Actual         bool
	DeclaredDetail string
	ActualDetail   string
	// Surf is the detected surface facts when Actual is true.
	Surf profile.Surface
}

// Model is the reconciled view consumed by the validation rules.
type Model struct {
	Scenario string
	RootPath string
	Profile  profile.Profile
	Intent   intent.Intent
	Surfaces []SurfaceState
}

// Build reconciles intent against the detected profile.
func Build(scenario, rootPath string, in intent.Intent, p profile.Profile) Model {
	states := map[string]*SurfaceState{}
	order := []string{}
	get := func(id, kind string) *SurfaceState {
		key := strings.ToLower(id)
		if s, ok := states[key]; ok {
			return s
		}
		s := &SurfaceState{Surface: id, Kind: kind}
		states[key] = s
		order = append(order, key)
		return s
	}

	// Declared surfaces: port bindings, the cli command, and health endpoints.
	for name := range in.Ports {
		s := get(name, kindForSurfaceID(name))
		s.Declared = true
		s.DeclaredDetail = "declared via ports." + name
	}
	for name := range in.Lifecycle.Health.Endpoints {
		s := get(name, kindForSurfaceID(name))
		s.Declared = true
		if s.DeclaredDetail == "" {
			s.DeclaredDetail = "declared via health endpoint"
		}
	}
	if in.CLIEnabled {
		s := get("cli", "cli")
		s.Declared = true
		s.DeclaredDetail = "declared via cli.enabled"
	}

	// Actual surfaces from code facts.
	for _, surf := range p.Surfaces {
		s := get(surf.ID, surf.Kind)
		s.Actual = true
		s.Surf = surf
		detail := surf.Language
		if surf.Framework != "" {
			detail += "/" + surf.Framework
		}
		s.ActualDetail = strings.TrimPrefix(detail, "/")
		if s.Kind == "" {
			s.Kind = surf.Kind
		}
	}

	sort.Strings(order)
	out := Model{Scenario: scenario, RootPath: rootPath, Profile: p, Intent: in}
	for _, key := range order {
		out.Surfaces = append(out.Surfaces, *states[key])
	}
	return out
}

func kindForSurfaceID(id string) string {
	switch strings.ToLower(id) {
	case "api":
		return "api"
	case "ui":
		return "ui"
	case "cli":
		return "cli"
	default:
		return id
	}
}
