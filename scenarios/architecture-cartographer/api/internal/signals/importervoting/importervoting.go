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
	"architecture-cartographer/internal/signals/graphindex"
)

const name = "importer-voting"

// Signal is the production importer-voting signal.
type Signal struct{}

// New returns the production signal.
func New() *Signal { return &Signal{} }

func (Signal) Name() string                               { return name }
func (Signal) DefaultWeight() float64                     { return 0.8 }
func (Signal) IsAvailable(context.Context) (bool, string) { return true, "" }

func (Signal) Score(ctx context.Context, gctx signals.GraphContext, chunk graph.Chunk) signals.ScoreResult {
	if err := ctx.Err(); err != nil {
		return signals.Abstain(name, err.Error(), chunk.Path)
	}
	if chunk.FileID == "" {
		return signals.Abstain(name, "chunk has no file id", chunk.Path)
	}
	pkgID := graphindex.PackageForFileIn(chunk.FileID, gctx)
	if pkgID == "" {
		return signals.Abstain(name, "file has no package in snapshot", chunk.Path)
	}
	importers := graphindex.PackageImporters(pkgID, gctx)
	if len(importers) == 0 {
		return signals.Abstain(name, "no importers for this file in current snapshot", chunk.Path)
	}

	domainPackages := graphindex.DomainPackages(gctx)
	votes := make(map[string]int)
	for _, imp := range importers {
		dom := domainPackages[imp]
		if dom == "" {
			continue
		}
		votes[dom]++
	}
	if len(votes) == 0 {
		return signals.Abstain(name, "importers are not mapped to any derived domain", chunk.Path)
	}

	var out []signals.Score
	for dom, count := range votes {
		value := float64(count) / float64(len(importers))
		out = append(out, signals.Score{
			Signal: name,
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
	return signals.ScoreResult{Scores: out}
}
