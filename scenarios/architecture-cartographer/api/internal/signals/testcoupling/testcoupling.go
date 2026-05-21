// Package testcoupling is the test-coupling signal. Scores domains
// whose test files exercise the chunk's symbols. The premise is that
// the test file location is the strongest behavioral assertion about
// where the code under test belongs.
//
// Default weight 0.7 per SIGNAL_LADDER.md.
package testcoupling

import (
	"context"
	"fmt"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
)

// Signal is the production test-coupling signal.
type Signal struct{}

// New returns the production signal.
func New() *Signal { return &Signal{} }

func (Signal) Name() string                            { return "test-coupling" }
func (Signal) DefaultWeight() float64                  { return 0.7 }
func (Signal) IsAvailable(context.Context) (bool, string) { return true, "" }

func (Signal) Score(_ context.Context, gctx signals.GraphContext, chunk graph.Chunk) []signals.Score {
	if chunk.FileID == "" {
		return nil
	}
	pkgID := packageForFile(chunk.FileID, gctx.Snapshot)
	if pkgID == "" {
		return nil
	}
	tests := importingTestFiles(pkgID, gctx.Snapshot)
	if len(tests) == 0 {
		return nil
	}

	domainFor := indexDomainPackages(gctx)
	votes := make(map[string]int)
	for _, tf := range tests {
		dom := domainFor[tf.PackageID]
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
		value := float64(count) / float64(len(tests))
		out = append(out, signals.Score{
			Signal: "test-coupling",
			Domain: dom,
			Value:  value,
			Reason: fmt.Sprintf("%d/%d test file(s) live in domain %q", count, len(tests), dom),
			Evidence: []signals.Evidence{{
				Kind:    "test_coupling",
				Summary: fmt.Sprintf("test files in %s exercise this package", dom),
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

func importingTestFiles(pkgID string, snap graph.GraphSnapshot) []graph.FileNode {
	importers := make(map[string]struct{})
	for _, e := range snap.Imports {
		if e.ToPackageID != pkgID {
			continue
		}
		importers[e.From] = struct{}{}
	}
	var out []graph.FileNode
	for _, f := range snap.Files {
		if !f.IsTest {
			continue
		}
		if _, ok := importers[f.ID]; ok {
			out = append(out, f)
			continue
		}
		if _, ok := importers[f.PackageID]; ok {
			out = append(out, f)
		}
	}
	return out
}

func indexDomainPackages(gctx signals.GraphContext) map[string]string {
	out := make(map[string]string, len(gctx.Snapshot.Packages))
	for _, p := range gctx.Snapshot.Packages {
		if p.Directory == "" {
			continue
		}
		for _, d := range gctx.Manifest.Domains {
			for _, g := range d.Paths {
				if matches(p.Directory, g) {
					out[p.ID] = d.Name
					break
				}
			}
		}
	}
	return out
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
