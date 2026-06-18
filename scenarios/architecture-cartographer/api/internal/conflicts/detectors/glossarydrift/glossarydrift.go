// Package glossarydrift detects exported symbols that use vocabulary curated
// for a different domain.
package glossarydrift

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

var ignoredTokens = map[string]struct{}{
	"api": {}, "client": {}, "config": {}, "context": {}, "data": {},
	"handler": {}, "model": {}, "node": {}, "request": {}, "response": {},
	"service": {}, "store": {}, "type": {}, "value": {},
}

// Detector flags symbols whose product vocabulary belongs to another domain's
// curated glossary. Missing or low-quality glossaries simply produce no
// findings; the detector does not invent vocabulary.
type Detector struct{}

func New() *Detector { return &Detector{} }

func (Detector) Name() string { return "glossary_drift" }

func (Detector) Description() string {
	return "Flags exported symbols whose names use another domain's curated glossary vocabulary."
}

func (Detector) EmitsTypes() []string { return []string{"glossary_drift"} }

func (Detector) Detect(_ context.Context, in conflicts.DetectInput) ([]conflicts.Conflict, error) {
	ownerByToken := glossaryTokenOwners(in.DomainMap.Domains)
	if len(ownerByToken) == 0 {
		return nil, nil
	}
	files := filesByID(in.Snapshot.Files)
	packages := packagesByID(in.Snapshot.Packages)
	var out []conflicts.Conflict
	for _, sym := range in.Snapshot.Symbols {
		if !sym.Exported || strings.TrimSpace(sym.Name) == "" {
			continue
		}
		filePath := symbolPath(sym, files, packages)
		if filePath == "" {
			continue
		}
		currentDomain := in.DomainMap.DomainFor(filePath)
		if currentDomain == "" {
			continue
		}
		if token, owners := firstForeignGlossaryToken(sym.Name, currentDomain, ownerByToken); token != "" {
			out = append(out, conflictFor(in.Scenario, currentDomain, sym, filePath, token, owners))
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Locations[0] != out[j].Locations[0] {
			return out[i].Locations[0] < out[j].Locations[0]
		}
		return out[i].Evidence[0].Summary < out[j].Evidence[0].Summary
	})
	return out, nil
}

func glossaryTokenOwners(domainList []domains.DerivedDomain) map[string][]string {
	out := map[string][]string{}
	for _, dom := range domainList {
		for _, phrase := range append([]string{dom.Name}, dom.Glossary...) {
			for _, token := range tokens(phrase) {
				out[token] = appendUnique(out[token], dom.Name)
			}
		}
	}
	for token, owners := range out {
		if len(owners) < 1 {
			delete(out, token)
			continue
		}
		sort.Strings(owners)
		out[token] = owners
	}
	return out
}

func firstForeignGlossaryToken(symbolName, currentDomain string, ownerByToken map[string][]string) (string, []string) {
	for _, token := range tokens(symbolName) {
		owners := ownerByToken[token]
		if len(owners) == 0 || contains(owners, currentDomain) {
			continue
		}
		return token, owners
	}
	return "", nil
}

func conflictFor(scenario, currentDomain string, sym graph.SymbolNode, filePath, token string, owners []string) conflicts.Conflict {
	ownerText := strings.Join(owners, ", ")
	return conflicts.Conflict{
		Scenario:  scenario,
		Detector:  "glossary_drift",
		Type:      "glossary_drift",
		Subtype:   "foreign_domain_vocabulary",
		Severity:  conflicts.SeverityWarn,
		Locations: []string{filePath},
		Domains:   appendUnique([]string{currentDomain}, owners...),
		Evidence: []conflicts.Evidence{{
			Kind:    "foreign_glossary_token",
			Summary: fmt.Sprintf("%s uses glossary token %q owned by %s while living in domain %q", sym.Name, token, ownerText, currentDomain),
			Locator: filePath,
		}},
		SuggestedFixes: []conflicts.Fix{{
			Kind:       conflicts.FixKindReassignDomain,
			Summary:    fmt.Sprintf("Move %s near the %s domain or rename it with %s vocabulary.", sym.Name, ownerText, currentDomain),
			Confidence: 0.6,
		}},
	}
}

func filesByID(files []graph.FileNode) map[string]graph.FileNode {
	out := make(map[string]graph.FileNode, len(files))
	for _, f := range files {
		out[f.ID] = f
	}
	return out
}

func packagesByID(packages []graph.PackageNode) map[string]graph.PackageNode {
	out := make(map[string]graph.PackageNode, len(packages))
	for _, p := range packages {
		out[p.ID] = p
	}
	return out
}

func symbolPath(sym graph.SymbolNode, files map[string]graph.FileNode, packages map[string]graph.PackageNode) string {
	if f, ok := files[sym.FileID]; ok {
		return f.Path
	}
	if p, ok := packages[sym.PackageID]; ok {
		return p.RepoPath
	}
	return ""
}

func tokens(s string) []string {
	var words []string
	var current []rune
	flush := func() {
		if len(current) == 0 {
			return
		}
		word := strings.ToLower(string(current))
		current = current[:0]
		if len(word) < 3 {
			return
		}
		if _, ignored := ignoredTokens[word]; ignored {
			return
		}
		words = append(words, word)
	}
	for i, r := range s {
		switch {
		case r == '_' || r == '-' || r == '/' || r == '.' || unicode.IsSpace(r):
			flush()
		case unicode.IsUpper(r) && i > 0 && len(current) > 0 && !unicode.IsUpper(current[len(current)-1]):
			flush()
			current = append(current, unicode.ToLower(r))
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			current = append(current, unicode.ToLower(r))
		default:
			flush()
		}
	}
	flush()
	return uniqueSorted(words)
}

func appendUnique(in []string, values ...string) []string {
	seen := make(map[string]struct{}, len(in)+len(values))
	out := make([]string, 0, len(in)+len(values))
	for _, value := range append(in, values...) {
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
	return out
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func uniqueSorted(values []string) []string {
	out := appendUnique(nil, values...)
	sort.Strings(out)
	return out
}

var _ conflicts.Detector = (*Detector)(nil)
