package components

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// DependencyReader is the narrow catalog seam used to expand manifest-pinned
// asset dependencies. Components Service and its test fakes satisfy it.
type DependencyReader interface {
	Get(context.Context, string) (Component, error)
	GetByLibraryID(context.Context, string) (Component, error)
	GetVersion(context.Context, string, string) (ComponentVersion, error)
}

// ResolvedAsset is one immutable asset version in a dependency closure.
// Assets are ordered dependencies-first, then the requested root, so callers
// can materialize imports without a second sort or per-edge lookup.
type ResolvedAsset struct {
	Asset   Component
	Version ComponentVersion
}

// PortCandidate describes one possible satisfier for a composition port.
// Template providers always win. For adopted providers, the lowest rank wins;
// equal-rank candidates are rejected instead of selected nondeterministically.
type PortCandidate struct {
	Port     string
	Source   string
	Rank     int
	Template bool
}

type PortResolution struct {
	Satisfied []string
	Sources   map[string]string
}

// ResolvePortPrecedence applies the port rule independently of filesystem
// placement. This makes the precedence contract testable without a scenario
// tree and prevents adoption from silently copying a foundation that the host
// already provides.
func ResolvePortPrecedence(expects, templateProvides []string, adopted []PortCandidate) (PortResolution, error) {
	provided := make(map[string]struct{}, len(templateProvides))
	for _, port := range templateProvides {
		provided[strings.TrimSpace(port)] = struct{}{}
	}
	byPort := map[string][]PortCandidate{}
	for _, candidate := range adopted {
		port := strings.TrimSpace(candidate.Port)
		if port != "" {
			byPort[port] = append(byPort[port], candidate)
		}
	}
	out := PortResolution{Sources: map[string]string{}}
	for _, raw := range expects {
		port := strings.TrimSpace(raw)
		if port == "" {
			continue
		}
		if _, ok := provided[port]; ok {
			out.Satisfied = append(out.Satisfied, port)
			out.Sources[port] = "template"
			continue
		}
		candidates := byPort[port]
		if len(candidates) == 0 {
			continue
		}
		sort.SliceStable(candidates, func(i, j int) bool {
			if candidates[i].Rank != candidates[j].Rank {
				return candidates[i].Rank < candidates[j].Rank
			}
			return candidates[i].Source < candidates[j].Source
		})
		if len(candidates) > 1 && candidates[0].Rank == candidates[1].Rank {
			return PortResolution{}, fmt.Errorf("port %q has an unresolved satisfier tie at dependency rank %d", port, candidates[0].Rank)
		}
		out.Satisfied = append(out.Satisfied, port)
		out.Sources[port] = candidates[0].Source
	}
	sort.Strings(out.Satisfied)
	return out, nil
}

type ClosureReport struct {
	Assets               []ResolvedAsset
	SatisfiedPorts       []string
	AvailableSuggestions []string
}

// ErrAssetDependency is returned when a manifest pin cannot be resolved.
type ErrAssetDependency struct {
	FromLibraryID, LibraryID, Version string
	Cause                             error
}

func (e ErrAssetDependency) Error() string {
	return fmt.Sprintf("asset dependency %s@%s required by %s cannot be resolved: %v", e.LibraryID, e.Version, e.FromLibraryID, e.Cause)
}

func (e ErrAssetDependency) Unwrap() error { return e.Cause }

// ErrAssetDependencyCycle preserves the complete dependency path so callers
// can tell an authored cycle from a generic version-resolution failure.
type ErrAssetDependencyCycle struct{ Path []string }

func (e ErrAssetDependencyCycle) Error() string {
	return "asset dependency cycle: " + strings.Join(e.Path, " -> ")
}

// ResolveDependencyClosure resolves rootID at rootVersion (or the asset's
// latest version when omitted) and every manifest-declared, version-pinned
// dependency. The result is stable across index order: dependency edges are
// traversed by library id then version, duplicate pins are emitted once, and
// cycles are rejected before a caller can write partial output.
func ResolveDependencyClosure(ctx context.Context, reader DependencyReader, rootID, rootVersion string) ([]ResolvedAsset, error) {
	return ResolveDependencyClosureWithOptions(ctx, reader, rootID, rootVersion, nil)
}

// ResolveDependencyClosureWithOptions expands requires edges and only the
// explicitly opted-in suggests library ids. Suggestions never enter the
// closure by accident.
func ResolveDependencyClosureWithOptions(ctx context.Context, reader DependencyReader, rootID, rootVersion string, includeSuggestions []string) ([]ResolvedAsset, error) {
	report, err := ResolveDependencyClosureReport(ctx, reader, rootID, rootVersion, includeSuggestions, nil, nil)
	return report.Assets, err
}

// ResolveDependencyClosureReport returns the copied closure plus the two
// operator-visible sets that are otherwise lost during materialization.
func ResolveDependencyClosureReport(ctx context.Context, reader DependencyReader, rootID, rootVersion string, includeSuggestions, templateProvides []string, adopted []PortCandidate) (ClosureReport, error) {
	root, err := reader.Get(ctx, strings.TrimSpace(rootID))
	if err != nil {
		return ClosureReport{}, err
	}
	rootVersion = strings.TrimSpace(rootVersion)
	if rootVersion == "" {
		rootVersion = firstAssetVersion(root.LatestVersion, root.Version)
	}
	if rootVersion == "" {
		return ClosureReport{}, ErrAssetDependency{FromLibraryID: root.LibraryID, LibraryID: root.LibraryID, Cause: fmt.Errorf("asset has no indexed version")}
	}

	state := make(map[string]uint8)
	resolved := make(map[string]ResolvedAsset)
	order := make([]string, 0)
	path := make([]string, 0)
	var visit func(Component, string, string) error
	visit = func(asset Component, version, from string) error {
		key := asset.LibraryID + "@" + version
		switch state[key] {
		case 1:
			cycle := append(append([]string(nil), path...), key)
			return ErrAssetDependencyCycle{Path: cycle}
		case 2:
			return nil
		}
		state[key] = 1
		path = append(path, key)
		immutable, err := reader.GetVersion(ctx, asset.ID, version)
		if err != nil {
			return ErrAssetDependency{FromLibraryID: from, LibraryID: asset.LibraryID, Version: version, Cause: err}
		}
		declarations := append([]AssetDependency(nil), immutable.Dependencies...)
		if !immutable.DependencyLockPresent {
			declarations = append([]AssetDependency(nil), asset.Dependencies...)
		}
		sort.Slice(declarations, func(i, j int) bool {
			if declarations[i].LibraryID == declarations[j].LibraryID {
				return declarations[i].Version < declarations[j].Version
			}
			return declarations[i].LibraryID < declarations[j].LibraryID
		})
		for _, dep := range declarations {
			if dep.Kind.normalized() == DependencySuggests && !containsString(includeSuggestions, dep.LibraryID) {
				continue
			}
			dependency, err := reader.GetByLibraryID(ctx, dep.LibraryID)
			if err != nil {
				return ErrAssetDependency{FromLibraryID: asset.LibraryID, LibraryID: dep.LibraryID, Version: dep.Version, Cause: err}
			}
			if err := visit(dependency, dep.Version, asset.LibraryID); err != nil {
				return err
			}
		}
		resolved[key] = ResolvedAsset{Asset: asset, Version: immutable}
		order = append(order, key)
		state[key] = 2
		path = path[:len(path)-1]
		return nil
	}
	if err := visit(root, rootVersion, root.LibraryID); err != nil {
		return ClosureReport{}, err
	}
	out := make([]ResolvedAsset, 0, len(order))
	for _, key := range order {
		out = append(out, resolved[key])
	}
	resolution, err := ResolvePortPrecedence(root.Expects, templateProvides, adopted)
	if err != nil {
		return ClosureReport{}, err
	}
	available := make(map[string]struct{})
	for _, resolved := range out {
		for _, dep := range resolved.Asset.Dependencies {
			if dep.Kind.normalized() == DependencySuggests && !containsString(includeSuggestions, dep.LibraryID) {
				available[dep.LibraryID] = struct{}{}
			}
		}
	}
	suggestions := make([]string, 0, len(available))
	for id := range available {
		suggestions = append(suggestions, id)
	}
	sort.Strings(suggestions)
	return ClosureReport{Assets: out, SatisfiedPorts: resolution.Satisfied, AvailableSuggestions: suggestions}, nil
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func firstAssetVersion(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
