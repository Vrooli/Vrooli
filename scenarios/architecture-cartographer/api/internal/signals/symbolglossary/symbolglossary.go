// Package symbolglossary is the symbol-glossary signal. Scores domains
// whose derived-map glossary terms appear among the chunk's
// file-level exported symbols.
//
// Default weight 0.9 per SIGNAL_LADDER.md.
package symbolglossary

import (
	"context"
	"fmt"
	"strings"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
)

const name = "symbol-glossary"

// Signal is the production symbol-glossary signal.
type Signal struct{}

// New returns the production signal.
func New() *Signal { return &Signal{} }

func (Signal) Name() string                               { return name }
func (Signal) DefaultWeight() float64                     { return 0.9 }
func (Signal) IsAvailable(context.Context) (bool, string) { return true, "" }

func (Signal) Score(_ context.Context, gctx signals.GraphContext, chunk graph.Chunk) signals.ScoreResult {
	if chunk.FileID == "" {
		return signals.Abstain(name, "chunk has no file id", chunk.Path)
	}
	symbols := symbolsForFile(chunk.FileID, gctx.Snapshot)
	if len(symbols) == 0 {
		return signals.Abstain(name, "file has no exported symbols in snapshot", chunk.Path)
	}
	lowered := make(map[string]struct{}, len(symbols))
	for _, s := range symbols {
		lowered[strings.ToLower(s)] = struct{}{}
	}

	var out []signals.Score
	for _, d := range gctx.DomainMap.Domains {
		matched := 0
		var hits []string
		for _, term := range d.Glossary {
			if _, ok := lowered[strings.ToLower(term)]; ok {
				matched++
				hits = append(hits, term)
			}
		}
		if matched == 0 {
			continue
		}
		value := float64(matched) / float64(len(symbols))
		if value > 1 {
			value = 1
		}
		out = append(out, signals.Score{
			Signal: name,
			Domain: d.Name,
			Value:  value,
			Reason: fmt.Sprintf("%d glossary term(s) matched in file symbols", matched),
			Evidence: []signals.Evidence{{
				Kind:    "symbol_glossary",
				Summary: fmt.Sprintf("matched %v", hits),
				Locator: chunk.Path,
				Weight:  value,
			}},
		})
	}
	if len(out) == 0 {
		return signals.Abstain(name, "no glossary terms matched any domain", chunk.Path)
	}
	return signals.ScoreResult{Scores: out}
}

func symbolsForFile(fileID string, snap graph.GraphSnapshot) []string {
	var out []string
	for _, s := range snap.Symbols {
		if s.FileID == fileID && s.Exported {
			out = append(out, s.Name)
		}
	}
	return out
}
