package sketch

import (
	"context"
	"fmt"
	"strings"

	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/catalogsearch"
)

type DesignConstraints struct {
	DesignSourceHash         string `json:"designSourceHash,omitempty"`
	DesignSource             string `json:"designSource,omitempty"`
	PreserveRoutes           *bool  `json:"preserveRoutes,omitempty"`
	PreserveBusinessBehavior *bool  `json:"preserveBusinessBehavior,omitempty"`
}
type DesignIntent struct {
	Constraints          *DesignConstraints `json:"constraints,omitempty"`
	Intent               string             `json:"intent"`
	Users                []string           `json:"users"`
	PrimaryTasks         []string           `json:"primaryTasks"`
	Target               string             `json:"target"`
	Kit                  string             `json:"kit"`
	RequiredCapabilities []string           `json:"requiredCapabilities,omitempty"`
	Viewports            []string           `json:"viewports"`
}
type Proposal struct {
	Title       string
	Document    Document
	Obligations []string
	Template    catalogsearch.Result
}

// Propose retrieves bounded lexical alternatives. It never invents versions,
// infers source bindings as facts, or invokes generated code. Executable ports
// and fixtures remain explicit obligations until validated by the renderer.
func Propose(ctx context.Context, snapshot Snapshot, intent DesignIntent, limit int, search ImportSearch, catalog []catalogcoverage.Asset) ([]Proposal, error) {
	if strings.TrimSpace(intent.Intent) == "" || len(intent.Intent) > 8000 {
		return nil, fmt.Errorf("intent must contain 1 to 8000 characters")
	}
	if len(intent.PrimaryTasks) == 0 || len(intent.PrimaryTasks) > 12 || len(intent.Users) == 0 || len(intent.Users) > 12 {
		return nil, fmt.Errorf("specify one to twelve users and primary tasks")
	}
	if limit < 1 || limit > 3 {
		return nil, fmt.Errorf("candidate limit must be between one and three")
	}
	if search == nil {
		return nil, fmt.Errorf("catalog search unavailable")
	}
	if intent.Target == "" || intent.Kit == "" || len(intent.Viewports) == 0 || len(intent.Viewports) > 3 || len(intent.RequiredCapabilities) > 32 {
		return nil, fmt.Errorf("target, design kit, and viewports are required")
	}
	for _, list := range [][]string{intent.Users, intent.PrimaryTasks, intent.Viewports, intent.RequiredCapabilities} {
		for _, value := range list {
			if strings.TrimSpace(value) == "" || len(value) > 1000 {
				return nil, fmt.Errorf("intent fields must contain nonempty bounded values")
			}
		}
	}
	for _, viewport := range intent.Viewports {
		if viewport != "phone" && viewport != "desktop" && viewport != "tablet" {
			return nil, fmt.Errorf("unsupported viewport %q", viewport)
		}
	}
	constraints := DesignConstraints{}
	if intent.Constraints != nil {
		constraints = *intent.Constraints
	}
	source, err := normalizeDesignSource(constraints.DesignSource)
	if err != nil {
		return nil, err
	}
	constraints.DesignSource = source
	preserveRoutes, preserveBehavior := true, true
	if constraints.PreserveRoutes != nil {
		preserveRoutes = *constraints.PreserveRoutes
	}
	if constraints.PreserveBusinessBehavior != nil {
		preserveBehavior = *constraints.PreserveBusinessBehavior
	}
	constraints.PreserveRoutes = &preserveRoutes
	constraints.PreserveBusinessBehavior = &preserveBehavior
	intent.Constraints = &constraints
	byID := map[string]catalogcoverage.Asset{}
	for _, a := range catalog {
		byID[a.ID] = a
	}
	query := intent.Intent + " " + strings.Join(intent.PrimaryTasks, " ")
	matches := search.SearchContext(ctx, query, 40, "page-template", "", "")
	var out []Proposal
	seen := map[string]bool{}
	for _, match := range matches {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		a, ok := byID[match.CatalogID]
		if !ok || seen[a.ID] || a.Kind != "page-template" || !match.Availability.IsBuilt() || match.Availability.CatalogID != a.ID || match.Availability.Version == "" {
			continue
		}
		if !includes(a.Targets, intent.Target) || !includes(a.Kits, intent.Kit) || len(a.Regions) == 0 {
			continue
		}
		compatible := true
		for _, capability := range intent.RequiredCapabilities {
			if !includes(a.Capabilities, capability) {
				compatible = false
			}
		}
		if !compatible {
			continue
		}
		seen[a.ID] = true
		change, err := ChangeTemplate(snapshot.Document, AssetRef{Asset: a.ID, Version: match.Availability.Version}, a.Regions, nil)
		if err != nil {
			return nil, err
		}
		doc := change.Document
		authored := map[string]bool{}
		for _, list := range [][]Region{snapshot.Document.Regions, snapshot.DeclaredRegions} {
			for _, r := range list {
				authored[r.ID] = true
			}
		}
		for i, r := range doc.Regions {
			if !authored[r.ID] {
				doc.Regions[i].Origin = "template"
			}
		}
		// Preserve semantic regions even when the old sketch has not imported them.
		known := map[string]bool{}
		for _, r := range doc.Regions {
			known[r.ID] = true
		}
		for _, r := range snapshot.DeclaredRegions {
			if !known[r.ID] {
				doc.Regions = append(doc.Regions, r)
				known[r.ID] = true
			}
		}
		doc.Intent = &intent
		obligations := []string{"Map semantic regions to validated template ports.", "Supply deterministic data and inert action fixtures for every required region and state."}
		if constraints.DesignSource != "" {
			obligations = append(obligations, "Read and reconcile design source: "+constraints.DesignSource+". This reference is not proof of source conformance.")
		}
		if preserveRoutes {
			obligations = append(obligations, "Preserve existing routes and verify navigation reachability.")
		} else {
			obligations = append(obligations, "Route preservation is explicitly disabled; review route changes before acceptance.")
		}
		if preserveBehavior {
			obligations = append(obligations, "Preserve existing business behavior and verify cross-page journeys.")
		} else {
			obligations = append(obligations, "Business behavior preservation is explicitly disabled; review behavior changes before acceptance.")
		}
		out = append(out, Proposal{Title: a.Name, Document: doc, Template: match, Obligations: obligations})
		if len(out) == limit {
			break
		}
	}
	return out, ctx.Err()
}
func includes(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
