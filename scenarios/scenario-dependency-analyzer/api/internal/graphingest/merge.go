// Package graphingest builds the unified, evidence-tagged cross-scenario
// dependency graph from every available source — proto_import ∪ go_import ∪
// declared ∪ vrooli_cli ∪ resource — deduplicating by (from, to), keeping the
// highest-confidence source and the union of evidence. It is the writer of the
// persisted graph_edges store that powers graph generation and centrality.
package graphingest

import (
	"sort"
	"time"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

// Evidence-source identifiers. Confidence ranking is
// proto_import/go_import > resource > declared > vrooli_cli, reflecting that
// import-level evidence proves a real code edge while declared/CLI signals are
// intent or heuristics retained until the AST-facts follow-up lands.
const (
	SourceProtoImport = "proto_import"
	SourceGoImport    = "go_import"
	SourceDeclared    = "declared"
	SourceVrooliCLI   = "vrooli_cli"
	SourceResource    = "resource"

	KindScenario = "scenario"
	KindResource = "resource"
)

// ConfidenceFor returns the ranking weight for an evidence source. An edge's
// confidence is the maximum across the sources that attest it.
func ConfidenceFor(source string) float64 {
	switch source {
	case SourceProtoImport:
		return 1.0
	case SourceGoImport:
		return 0.9
	case SourceResource:
		return 0.8
	case SourceDeclared:
		return 0.7
	case SourceVrooliCLI:
		return 0.5
	default:
		return 0.3
	}
}

// Contribution is one raw attestation of an edge before merge.
type Contribution struct {
	From     string
	To       string
	Kind     string
	Source   string
	Required bool
	Evidence types.UnifiedEdgeEvidence
}

// Merge collapses raw contributions into deduped unified edges. Edges are keyed
// by (from, to): the highest-confidence source wins for Source/Confidence,
// required-ness is OR-ed, and evidence is unioned. Self-edges and empty
// endpoints are dropped. The result is deterministically ordered.
func Merge(contribs []Contribution, now time.Time) []types.UnifiedGraphEdge {
	grouped := map[string]*types.UnifiedGraphEdge{}
	order := make([]string, 0, len(contribs))

	for _, c := range contribs {
		if c.From == "" || c.To == "" || c.From == c.To {
			continue
		}
		key := c.From + "\x00" + c.To
		edge := grouped[key]
		if edge == nil {
			edge = &types.UnifiedGraphEdge{
				From:         c.From,
				To:           c.To,
				Kind:         c.Kind,
				LastVerified: now,
			}
			grouped[key] = edge
			order = append(order, key)
		}
		// Scenario kind wins over resource if the same target is attested both
		// ways (should not happen for distinct namespaces, but be deterministic).
		if edge.Kind == KindResource && c.Kind == KindScenario {
			edge.Kind = KindScenario
		}
		if conf := ConfidenceFor(c.Source); conf > edge.Confidence {
			edge.Confidence = conf
			edge.Source = c.Source
		}
		if c.Required {
			edge.Required = true
		}
		if c.Evidence.Source != "" && !hasEvidence(edge.Evidence, c.Evidence) {
			edge.Evidence = append(edge.Evidence, c.Evidence)
		}
	}

	edges := make([]types.UnifiedGraphEdge, 0, len(order))
	for _, key := range order {
		edge := grouped[key]
		sort.Slice(edge.Evidence, func(i, j int) bool {
			if edge.Evidence[i].Source != edge.Evidence[j].Source {
				return edge.Evidence[i].Source < edge.Evidence[j].Source
			}
			return edge.Evidence[i].ImportPath < edge.Evidence[j].ImportPath
		})
		edges = append(edges, *edge)
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		return edges[i].To < edges[j].To
	})
	return edges
}

func hasEvidence(existing []types.UnifiedEdgeEvidence, next types.UnifiedEdgeEvidence) bool {
	for _, ev := range existing {
		if ev == next {
			return true
		}
	}
	return false
}
