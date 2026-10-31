package closure

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	hostreq "github.com/vrooli/vrooli/packages/hostreq"
)

// HostRequirement is one resolved host tool or safeguard for the platform.
type HostRequirement struct {
	Name       string
	Kind       string // tool | safeguard
	Required   bool
	Privilege  string
	Bundling   string
	Reasons    []string
	DeclaredBy []string // "scenario:<id>" / "resource:<id>"
}

// HostRequirementsQuery scopes a host requirement resolution to the closure's
// included components and platform.
type HostRequirementsQuery struct {
	RepoRoot      string
	Environment   string
	Platform      Platform
	Scenarios     []*ScenarioDeclaration
	Resources     []*ResourceDeclaration
	ResourcePaths []string
}

// HostRequirementsResolver resolves host tools and safeguards. Production uses
// ControlPlaneHostRequirements (packages/hostreq); fixtures use
// DeclaredHostRequirements, which reads the same declarations without the
// control-plane catalog.
type HostRequirementsResolver interface {
	Resolve(ctx context.Context, query HostRequirementsQuery) ([]HostRequirement, error)
	Name() string
}

// ControlPlaneHostRequirements resolves through packages/hostreq so cloud never
// carries a private copy of tool/safeguard policy (privilege, bundling,
// platform applicability, operator state).
type ControlPlaneHostRequirements struct {
	Home string
}

// Name implements HostRequirementsResolver.
func (ControlPlaneHostRequirements) Name() string { return "packages/hostreq" }

// Resolve implements HostRequirementsResolver.
func (r ControlPlaneHostRequirements) Resolve(_ context.Context, query HostRequirementsQuery) ([]HostRequirement, error) {
	home := r.Home
	if home == "" {
		resolved, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve home: %w", err)
		}
		home = resolved
	}
	paths := make([]string, 0, len(query.Scenarios))
	for _, scenario := range query.Scenarios {
		paths = append(paths, scenario.Path)
	}
	resourceIDs := make([]string, 0, len(query.Resources))
	for _, resource := range query.Resources {
		resourceIDs = append(resourceIDs, resource.ID)
	}
	resources := "none"
	if len(resourceIDs) > 0 {
		resources = strings.Join(resourceIDs, ",")
	}
	resolution, err := hostreq.Resolve(query.RepoRoot, home, hostreq.ResolveOptions{
		ExcludeRoot:   true,
		Environment:   query.Environment,
		Resources:     resources,
		ScenarioPaths: paths,
		Platform:      query.Platform.OS,
		Architecture:  query.Platform.Arch,
	})
	if err != nil {
		return nil, err
	}
	out := make([]HostRequirement, 0, len(resolution.Tools)+len(resolution.Safeguards))
	convert := func(items []hostreq.ResolvedRequirement) {
		for _, item := range items {
			if len(item.Platforms) > 0 && !containsFold(item.Platforms, query.Platform.OS) {
				continue
			}
			requirement := HostRequirement{
				Name:      item.Name,
				Kind:      string(item.Kind),
				Required:  item.Required,
				Privilege: string(item.Privilege),
				Bundling:  string(item.Bundling),
				Reasons:   append([]string(nil), item.Reasons...),
			}
			for _, provenance := range item.Provenance {
				requirement.DeclaredBy = append(requirement.DeclaredBy, provenance.Kind+":"+provenance.Name)
			}
			out = append(out, requirement)
		}
	}
	convert(resolution.Tools)
	convert(resolution.Safeguards)
	return normalizeHostRequirements(out), nil
}

// DeclaredHostRequirements resolves hostTools/hostSafeguards straight from
// the included declarations. It carries exactly the declared fields; privilege
// and bundling are those the declaration states.
type DeclaredHostRequirements struct{}

// Name implements HostRequirementsResolver.
func (DeclaredHostRequirements) Name() string { return "declarations" }

// Resolve implements HostRequirementsResolver.
func (DeclaredHostRequirements) Resolve(_ context.Context, query HostRequirementsQuery) ([]HostRequirement, error) {
	var out []HostRequirement
	add := func(owner string, kind string, declarations []HostRequirementDeclaration) {
		for _, declaration := range declarations {
			if len(declaration.Platforms) > 0 && !containsFold(declaration.Platforms, query.Platform.OS) {
				continue
			}
			out = append(out, HostRequirement{
				Name:       declaration.Name,
				Kind:       kind,
				Required:   declaration.Required,
				Privilege:  declaration.Privilege,
				Bundling:   declaration.Bundling,
				Reasons:    []string{declaration.Reason},
				DeclaredBy: []string{owner},
			})
		}
	}
	for _, scenario := range query.Scenarios {
		add("scenario:"+scenario.ID, "tool", scenario.HostTools)
		add("scenario:"+scenario.ID, "safeguard", scenario.HostSafeguards)
	}
	for _, resource := range query.Resources {
		add("resource:"+resource.ID, "tool", resource.HostTools)
		add("resource:"+resource.ID, "safeguard", resource.HostSafeguards)
	}
	return normalizeHostRequirements(out), nil
}

// normalizeHostRequirements merges duplicate (kind, name) entries, ORs
// required, unions reasons and provenance, and sorts deterministically.
func normalizeHostRequirements(items []HostRequirement) []HostRequirement {
	merged := map[string]*HostRequirement{}
	for _, item := range items {
		key := item.Kind + ":" + item.Name
		existing, ok := merged[key]
		if !ok {
			copied := item
			merged[key] = &copied
			continue
		}
		existing.Required = existing.Required || item.Required
		existing.Reasons = append(existing.Reasons, item.Reasons...)
		existing.DeclaredBy = append(existing.DeclaredBy, item.DeclaredBy...)
		if existing.Privilege == "" {
			existing.Privilege = item.Privilege
		}
		if existing.Bundling == "" {
			existing.Bundling = item.Bundling
		}
	}
	out := make([]HostRequirement, 0, len(merged))
	for _, item := range merged {
		item.Reasons = sortedUnique(item.Reasons)
		item.DeclaredBy = sortedUnique(item.DeclaredBy)
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}

func sortedUnique(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
