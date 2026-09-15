package memberflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Presentation content for a generated operating graph.
//
// Every node and edge in a contract graph is DERIVED: some field of topics.json
// or team.json already states it. What a hand-drawn diagram adds on top is
// readability — which boxes sit together, what short name a node carries, what
// order things appear in. That is the only part a human should still author,
// and it lives here so the derived part can be generated without losing it.
//
// docs/agent-system/OPERATING_GRAPHS.md § Non-Goals already says "Do not make
// Mermaid the runtime source of truth" and "Do not use operating models or
// graph sections to bypass topics.json". Splitting presentation from derived
// content makes both structurally impossible rather than merely discouraged.

// GraphPresentation is the per-team readability layer, loaded from
// store/teams/<team>/graph-presentation.json. Every field is optional: a team
// with no presentation file still generates a correct, if plainer, graph.
type GraphPresentation struct {
	// Subgraphs group nodes under a titled box, in declaration order.
	Subgraphs []GraphPresentationSubgraph `json:"subgraphs,omitempty"`

	// ShortNames maps a node's typed value to the mermaid identifier it should
	// carry (`portfolio-manager` -> `PM`). Absent values get a generated id.
	ShortNames map[string]string `json:"shortNames,omitempty"`

	// Displays maps a node's typed value to the human label drawn in the box.
	// Absent values are drawn with the value itself.
	Displays map[string]string `json:"displays,omitempty"`

	// NodeOrder lists typed values in the order they should be emitted. Values
	// absent from the list follow, sorted, so adding a declaration never
	// silently reorders an existing diagram.
	NodeOrder []string `json:"nodeOrder,omitempty"`

	// ReadabilityEdges are edges that exist for a reader's benefit and carry no
	// contract meaning. They are restricted to process and future endpoints:
	// OPERATING_GRAPHS.md states those kinds satisfy no completeness rule, so an
	// edge touching one cannot smuggle in an unbacked runtime claim.
	ReadabilityEdges []GraphPresentationEdge `json:"readabilityEdges,omitempty"`
}

type GraphPresentationSubgraph struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Values []string `json:"values,omitempty"`
}

type GraphPresentationEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label,omitempty"`
}

// LoadGraphPresentation reads a team's presentation file. A missing file is not
// an error: it is a team that has declared no readability preferences.
func LoadGraphPresentation(storeDir, teamID string) (GraphPresentation, error) {
	path := filepath.Join(storeDir, "teams", teamID, "graph-presentation.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return GraphPresentation{}, nil
		}
		return GraphPresentation{}, fmt.Errorf("read %s: %w", path, err)
	}
	var presentation GraphPresentation
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&presentation); err != nil {
		return GraphPresentation{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := presentation.Validate(); err != nil {
		return GraphPresentation{}, fmt.Errorf("%s: %w", path, err)
	}
	return presentation, nil
}

// Validate rejects a presentation file that tries to carry contract meaning.
//
// The restriction on readability edges is the load-bearing one. Without it the
// presentation layer becomes a second place to assert a runtime relationship,
// which is exactly the duplication generation removes.
func (p GraphPresentation) Validate() error {
	seen := map[string]bool{}
	for _, subgraph := range p.Subgraphs {
		if subgraph.ID == "" {
			return fmt.Errorf("subgraph requires an id")
		}
		if seen[subgraph.ID] {
			return fmt.Errorf("duplicate subgraph id %q", subgraph.ID)
		}
		seen[subgraph.ID] = true
	}
	for _, edge := range p.ReadabilityEdges {
		if edge.From == "" || edge.To == "" {
			return fmt.Errorf("readability edge requires both endpoints")
		}
		if !isReadabilityEndpoint(edge.From) && !isReadabilityEndpoint(edge.To) {
			return fmt.Errorf(
				"readability edge %s -> %s touches no process: or future: node; "+
					"an edge between contract nodes is derived content and must come from topics.json",
				edge.From, edge.To)
		}
	}
	return nil
}

// isReadabilityEndpoint reports whether a typed value names a node kind that
// satisfies no completeness rule, and therefore cannot assert a runtime claim.
func isReadabilityEndpoint(value string) bool {
	return hasTypePrefix(value, string(OperatingGraphNodeKindProcess)) ||
		hasTypePrefix(value, string(OperatingGraphNodeKindFuture))
}

func hasTypePrefix(value, kind string) bool {
	return len(value) > len(kind)+1 && value[:len(kind)+1] == kind+":"
}
