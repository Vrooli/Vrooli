// Package naming detects generic vocabulary that hides domain intent.
package naming

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/graph"
)

var defaultBannedVocabulary = []string{"common", "helpers", "manager", "misc", "stuff", "utils"}

// Detector flags domain and package names that are too generic to carry
// screaming-architecture intent.
type Detector struct {
	banned map[string]struct{}
}

// New returns the production detector with the default banned vocabulary.
func New() *Detector { return NewWithBannedVocabulary(defaultBannedVocabulary) }

// NewWithBannedVocabulary returns a detector with a caller-provided banned list.
func NewWithBannedVocabulary(words []string) *Detector {
	banned := make(map[string]struct{}, len(words))
	for _, w := range words {
		w = normalizeWord(w)
		if w != "" {
			banned[w] = struct{}{}
		}
	}
	if len(banned) == 0 {
		for _, w := range defaultBannedVocabulary {
			banned[w] = struct{}{}
		}
	}
	return &Detector{banned: banned}
}

func (Detector) Name() string { return "naming" }

func (Detector) Description() string {
	return "Flags generic package or domain names that obscure product intent."
}

func (Detector) EmitsTypes() []string { return []string{"naming"} }

func (Detector) Class() conflicts.FindingClass {
	return conflicts.FindingClassHeuristic
}

func (d Detector) Detect(_ context.Context, in conflicts.DetectInput) ([]conflicts.Conflict, error) {
	var out []conflicts.Conflict
	seenDomains := make(map[string]struct{}, len(in.DomainMap.Domains))
	for _, dom := range in.DomainMap.Domains {
		if !d.isBanned(dom.Name) {
			continue
		}
		seenDomains[dom.Name] = struct{}{}
		out = append(out, conflictForDomain(in.Scenario, dom.Name, dom.Paths))
	}
	for _, pkg := range in.Snapshot.Packages {
		segment := lastSegment(pkg.RepoPath)
		if !d.isBanned(segment) {
			continue
		}
		domain := in.DomainMap.DomainFor(pkg.RepoPath)
		if _, already := seenDomains[domain]; already && domain == segment {
			continue
		}
		out = append(out, conflictForPackage(in.Scenario, pkg, segment, domain))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Subtype != out[j].Subtype {
			return out[i].Subtype < out[j].Subtype
		}
		return strings.Join(out[i].Locations, "\x00") < strings.Join(out[j].Locations, "\x00")
	})
	return out, nil
}

func (d Detector) isBanned(word string) bool {
	_, ok := d.banned[normalizeWord(word)]
	return ok
}

func conflictForDomain(scenario, domain string, paths []string) conflicts.Conflict {
	locs := append([]string(nil), paths...)
	if len(locs) == 0 {
		locs = []string{domain}
	}
	sort.Strings(locs)
	return conflicts.Conflict{
		Scenario:  scenario,
		Detector:  "naming",
		Type:      "naming",
		Subtype:   "generic-domain-name",
		Severity:  conflicts.SeverityWarn,
		Locations: locs,
		Domains:   []string{domain},
		Evidence: []conflicts.Evidence{{
			Kind:    "banned_vocabulary",
			Summary: fmt.Sprintf("domain name %q is generic and hides product intent", domain),
			Locator: domain,
		}},
		SuggestedFixes: []conflicts.Fix{{
			Kind:       conflicts.FixKindReassignDomain,
			Summary:    fmt.Sprintf("Rename domain %q to a product capability name from DOMAINS.md/PRD vocabulary.", domain),
			Confidence: 0.7,
		}},
	}
}

func conflictForPackage(scenario string, pkg graph.PackageNode, segment, domain string) conflicts.Conflict {
	return conflicts.Conflict{
		Scenario:  scenario,
		Detector:  "naming",
		Type:      "naming",
		Subtype:   "generic-package-name",
		Severity:  conflicts.SeverityWarn,
		Locations: []string{pkg.RepoPath},
		Domains:   domainList(domain),
		Evidence: []conflicts.Evidence{{
			Kind:    "banned_vocabulary",
			Summary: fmt.Sprintf("package path segment %q is generic and hides ownership", segment),
			Locator: pkg.RepoPath,
		}},
		SuggestedFixes: []conflicts.Fix{{
			Kind:       conflicts.FixKindReassignDomain,
			Summary:    fmt.Sprintf("Rename %s to a capability or substrate name with concrete responsibility.", pkg.RepoPath),
			Confidence: 0.65,
		}},
	}
}

func normalizeWord(s string) string {
	return strings.ToLower(strings.TrimSpace(strings.Trim(s, "`\"'")))
}

func lastSegment(path string) string {
	path = strings.Trim(strings.TrimSpace(path), "/")
	if path == "" {
		return ""
	}
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[i+1:]
	}
	return path
}

func domainList(parts ...string) []string {
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
