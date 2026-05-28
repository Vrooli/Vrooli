// Package importervoting is the importer-voting signal. For each
// candidate domain D, counts how many packages that import the chunk's
// package belong to D. The candidate with the highest fraction of
// "votes" scores highest.
//
// Default weight 0.8 per SIGNAL_LADDER.md.
package importervoting

import (
	"context"
	"fmt"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
)

// Signal is the production importer-voting signal.
type Signal struct{}

// New returns the production signal.
func New() *Signal { return &Signal{} }

func (Signal) Name() string                               { return "importer-voting" }
func (Signal) DefaultWeight() float64                     { return 0.8 }
func (Signal) IsAvailable(context.Context) (bool, string) { return true, "" }

func (Signal) Score(_ context.Context, gctx signals.GraphContext, chunk graph.Chunk) []signals.Score {
	if chunk.FileID == "" {
		return nil
	}
	pkgID := packageForFile(chunk.FileID, gctx.Snapshot)
	if pkgID == "" {
		return nil
	}
	importers := importersOf(pkgID, gctx.Snapshot)
	if len(importers) == 0 {
		return nil
	}

	domainPackages := indexDomainPackages(gctx)
	votes := make(map[string]int)
	for _, imp := range importers {
		dom := domainPackages[imp]
		if dom == "" {
			continue
		}
		votes[dom]++
	}
	if len(votes) == 0 {
		return nil
	}

	var out []signals.Score
	for dom, count := range votes {
		value := float64(count) / float64(len(importers))
		out = append(out, signals.Score{
			Signal: "importer-voting",
			Domain: dom,
			Value:  value,
			Reason: fmt.Sprintf("%d of %d importers belong to %q", count, len(importers), dom),
			Evidence: []signals.Evidence{{
				Kind:    "importer_voting",
				Summary: fmt.Sprintf("%d/%d importers vote %q", count, len(importers), dom),
				Locator: chunk.Path,
				Weight:  value,
			}},
		})
	}
	return out
}

func packageForFile(fileID string, snap graph.GraphSnapshot) string {
	for _, f := range snap.Files {
		if f.ID == fileID {
			return f.PackageID
		}
	}
	return ""
}

func importersOf(pkgID string, snap graph.GraphSnapshot) []string {
	seen := make(map[string]struct{})
	for _, e := range snap.Imports {
		if e.ToPackageID != pkgID {
			continue
		}
		from := packageFor(e.From, snap)
		if from == "" || from == pkgID {
			continue
		}
		seen[from] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out
}

func packageFor(id string, snap graph.GraphSnapshot) string {
	for _, f := range snap.Files {
		if f.ID == id {
			return f.PackageID
		}
	}
	for _, p := range snap.Packages {
		if p.ID == id {
			return p.ID
		}
	}
	return ""
}

// indexDomainPackages maps package_id -> domain_name based on the
// derived map's path globs over each package's directory.
func indexDomainPackages(gctx signals.GraphContext) map[string]string {
	out := make(map[string]string, len(gctx.Snapshot.Packages))
	for _, p := range gctx.Snapshot.Packages {
		if p.Directory == "" {
			continue
		}
		for _, d := range gctx.DomainMap.Domains {
			if pathInDomain(p.Directory, d.Paths) {
				out[p.ID] = d.Name
				break
			}
		}
	}
	return out
}

func pathInDomain(path string, globs []string) bool {
	for _, g := range globs {
		if matches(path, g) {
			return true
		}
	}
	return false
}

func matches(path, glob string) bool {
	switch {
	case glob == "**":
		return true
	case hasSuffix(glob, "/**"):
		prefix := glob[:len(glob)-3]
		return path == prefix || hasPrefix(path, prefix+"/")
	default:
		return path == glob
	}
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
