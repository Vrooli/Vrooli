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
	"architecture-cartographer/internal/signals/graphindex"
)

const name = "test-coupling"

// Signal is the production test-coupling signal.
type Signal struct{}

// New returns the production signal.
func New() *Signal { return &Signal{} }

func (Signal) Name() string                               { return name }
func (Signal) DefaultWeight() float64                     { return 0.7 }
func (Signal) IsAvailable(context.Context) (bool, string) { return true, "" }

func (Signal) Score(_ context.Context, gctx signals.GraphContext, chunk graph.Chunk) signals.ScoreResult {
	if chunk.FileID == "" {
		return signals.Abstain(name, "chunk has no file id", chunk.Path)
	}
	pkgID := graphindex.PackageForFile(chunk.FileID, gctx.Snapshot)
	if pkgID == "" {
		return signals.Abstain(name, "file has no package in snapshot", chunk.Path)
	}
	tests := importingTestFiles(pkgID, gctx.Snapshot)
	if len(tests) == 0 {
		return signals.Abstain(name, "no test files import this package", chunk.Path)
	}

	domainFor := graphindex.DomainPackages(gctx)
	votes := make(map[string]int)
	for _, tf := range tests {
		dom := domainFor[tf.PackageID]
		if dom == "" {
			continue
		}
		votes[dom]++
	}
	if len(votes) == 0 {
		return signals.Abstain(name, "importing test files are not mapped to any derived domain", chunk.Path)
	}

	var out []signals.Score
	for dom, count := range votes {
		value := float64(count) / float64(len(tests))
		out = append(out, signals.Score{
			Signal: name,
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
	return signals.ScoreResult{Scores: out}
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
