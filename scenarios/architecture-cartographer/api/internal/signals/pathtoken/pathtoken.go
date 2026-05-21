// Package pathtoken is the path-token signal. Splits the chunk's path
// into segments + tokens and scores domains whose Name (or paths
// glob's last segment) appears in the chunk's path.
//
// Default weight 1.5 per SIGNAL_LADDER.md. Highest-confidence
// deterministic signal — path tokens are the cheapest, most obvious
// evidence that a file belongs to a domain.
package pathtoken

import (
	"context"
	"fmt"
	"strings"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
)

// Signal is the production path-token signal.
type Signal struct{}

// New returns the production signal.
func New() *Signal { return &Signal{} }

func (Signal) Name() string             { return "path-token" }
func (Signal) DefaultWeight() float64   { return 1.5 }
func (Signal) IsAvailable(context.Context) (bool, string) {
	return true, ""
}

// Score returns one Score per domain whose name token appears in the
// chunk's path. Value is the fraction of path tokens that matched, in
// [0, 1].
func (Signal) Score(_ context.Context, gctx signals.GraphContext, chunk graph.Chunk) []signals.Score {
	tokens := pathTokens(chunk.Path)
	if len(tokens) == 0 {
		return nil
	}
	tokenSet := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		tokenSet[strings.ToLower(t)] = struct{}{}
	}

	var out []signals.Score
	for _, d := range gctx.Manifest.Domains {
		name := strings.ToLower(d.Name)
		if _, ok := tokenSet[name]; !ok {
			continue
		}
		// Match strength: presence of the domain name token, plus any
		// additional path-segment matches against the domain's declared
		// paths' final segments.
		matched := 1
		for _, glob := range d.Paths {
			for _, seg := range strings.Split(glob, "/") {
				seg = strings.TrimSuffix(seg, "**")
				seg = strings.TrimSuffix(seg, "*")
				if seg == "" || seg == name {
					continue
				}
				if _, ok := tokenSet[strings.ToLower(seg)]; ok {
					matched++
				}
			}
		}
		value := float64(matched) / float64(len(tokens))
		if value > 1 {
			value = 1
		}
		out = append(out, signals.Score{
			Signal: "path-token",
			Domain: d.Name,
			Value:  value,
			Reason: fmt.Sprintf("path contains domain token %q", d.Name),
			Evidence: []signals.Evidence{{
				Kind:    "path_token",
				Summary: fmt.Sprintf("token %q present in %s", d.Name, chunk.Path),
				Locator: chunk.Path,
				Weight:  value,
			}},
		})
	}
	return out
}

// pathTokens splits the path into lowercased segment + identifier tokens.
func pathTokens(path string) []string {
	if path == "" {
		return nil
	}
	parts := strings.Split(path, "/")
	out := make([]string, 0, len(parts)*2)
	for _, p := range parts {
		// Strip file extensions.
		if idx := strings.LastIndex(p, "."); idx > 0 {
			p = p[:idx]
		}
		if p == "" {
			continue
		}
		// Split snake_case + kebab-case.
		for _, t := range splitCase(p) {
			if t == "" {
				continue
			}
			out = append(out, strings.ToLower(t))
		}
	}
	return out
}

func splitCase(s string) []string {
	// Treat '_' and '-' as separators; collapse to lowercase tokens.
	out := []string{}
	cur := strings.Builder{}
	for _, r := range s {
		if r == '_' || r == '-' {
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}
