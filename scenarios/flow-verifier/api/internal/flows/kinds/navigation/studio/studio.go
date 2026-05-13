// Package studio builds the Flow Studio descriptor for a compiled
// navigation graph. The descriptor is a renderer-agnostic projection of
// routes, affordances, containers, contexts, and invariant pass/fail
// status; the UI decodes it into React Flow (or equivalent) nodes and
// edges and applies the active context to filter affordances and
// containers.
package studio

import (
	"context"
	"encoding/json"
	"fmt"

	"flow-verifier/internal/flows/kind"
	"flow-verifier/internal/flows/kinds/navigation/compile"
	"flow-verifier/internal/flows/kinds/navigation/contract"
	"flow-verifier/internal/flows/kinds/navigation/verify"
)

// Route is a single navigable destination.
type Route struct {
	ID       string
	Path     string
	Page     string
	Requires string
	Parents  []string
}

// Presentation is one rendering site of an affordance: in a specific
// container or route, with a label and test id.
type Presentation struct {
	In     string
	Label  string
	TestID string
}

// Affordance is an actionable jump from one site to a target route.
type Affordance struct {
	ID            string
	To            string
	ShowWhen      string
	SideEffect    string
	Presentations []Presentation
}

// Container is a persistent navigation surface (sidebar, mobile nav, ...).
type Container struct {
	ID         string
	Kind       string
	ShowWhen   string
	Disclosure string
	HostRoutes []string
}

// Context exposes a UI-toggleable variable (viewport, auth, ...).
type Context struct {
	Name         string
	Kind         string // "enum" | "bool"
	Values       []string
	DefaultValue string
}

// Invariant is a reachability invariant's id + pass/fail summary,
// pre-evaluated server-side so the UI can render a badge without
// re-implementing the verifier.
type Invariant struct {
	ID      string
	Passed  bool
	Message string
}

// Descriptor is the data Flow Studio needs to render a navigation flow.
type Descriptor struct {
	Renderer    string
	Routes      []Route
	Affordances []Affordance
	Containers  []Container
	Contexts    []Context
	Invariants  []Invariant
}

// Build projects a compiled navigation Graph into a Descriptor. The
// returned descriptor's Invariants slice is empty; populate it via
// BuildWithFindings if invariant status should be embedded.
func Build(g compile.Graph) Descriptor {
	return BuildWithFindings(g, nil)
}

// BuildWithFindings is Build, also stamping the descriptor's Invariants
// slice from the given verify findings (one per reachability invariant).
func BuildWithFindings(g compile.Graph, findings []kind.Finding) Descriptor {
	d := Descriptor{Renderer: "navigation-graph"}

	for _, r := range g.Contract.Routes {
		d.Routes = append(d.Routes, Route{
			ID:       r.ID,
			Path:     r.Path,
			Page:     r.Page,
			Requires: r.Requires,
			Parents:  append([]string(nil), r.Parents...),
		})
	}

	for _, a := range g.Contract.Affordances {
		af := Affordance{
			ID:         a.ID,
			To:         a.To,
			ShowWhen:   a.ShowWhen,
			SideEffect: a.SideEffect,
		}
		for _, p := range a.Presentations {
			af.Presentations = append(af.Presentations, Presentation{
				In:     p.In,
				Label:  p.Label,
				TestID: p.TestID,
			})
		}
		d.Affordances = append(d.Affordances, af)
	}

	for _, c := range g.Contract.Containers {
		d.Containers = append(d.Containers, Container{
			ID:         c.ID,
			Kind:       c.Kind,
			ShowWhen:   c.ShowWhen,
			Disclosure: c.Disclosure,
			HostRoutes: append([]string(nil), c.HostRoutes...),
		})
	}

	for name, c := range g.Contract.Contexts {
		d.Contexts = append(d.Contexts, Context{
			Name:         name,
			Kind:         c.Kind,
			Values:       append([]string(nil), c.Values...),
			DefaultValue: stringifyDefault(c.Default),
		})
	}

	if len(findings) > 0 {
		byID := make(map[string]kind.Finding, len(findings))
		for _, f := range findings {
			byID[f.ID] = f
		}
		for _, inv := range g.Contract.ReachabilityInvariants {
			if f, ok := byID[inv.ID]; ok {
				d.Invariants = append(d.Invariants, Invariant{
					ID:      f.ID,
					Passed:  f.Passed,
					Message: f.Message,
				})
			}
		}
	}

	return d
}

// VerifyAndBuild compiles, then verifies, then builds — convenience
// wrapper used by the Connect handler so the studio descriptor always
// reflects the latest invariant status without callers re-running the
// verifier themselves.
func VerifyAndBuild(ctx context.Context, g compile.Graph) (Descriptor, error) {
	findings, err := verify.CheckInvariants(g)
	if err != nil {
		return Descriptor{}, fmt.Errorf("studio: verify reachability invariants: %w", err)
	}
	_ = ctx
	return BuildWithFindings(g, findings), nil
}

// stringifyDefault renders the raw JSON default into a UI-friendly
// string. For enum contexts that's just the unquoted string; for bool
// it's "true"/"false".
func stringifyDefault(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		if b {
			return "true"
		}
		return "false"
	}
	return string(raw)
}

// EnsureKind is a defensive helper that returns g if `c` is a navigation
// contract, or zero+err otherwise. Reserved for callers that have a
// generic contract.Contract in hand.
func EnsureKind(c contract.Contract) error {
	if c.Kind != "" && c.Kind != "navigation" {
		return fmt.Errorf("studio: contract kind %q is not navigation", c.Kind)
	}
	return nil
}
