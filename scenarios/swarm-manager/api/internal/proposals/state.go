package proposals

import (
	"fmt"

	"swarm-manager/internal/graph"
)

// FromMaterializedGraph lifts a graph.MaterializedGraph into the
// CurrentState shape proposals validate/apply expects.
//
// `knownInitiatives` is the set of initiatives that exist on disk (used by
// OpMoveInitiative to reject phantom destinations). `inProgressRefs` is the
// subset of graph nodes whose items are currently StatusInProgress (used
// by OpInterruptInProgress). Either may be nil — in which case those checks
// degrade to "accept anything", which is fine for test contexts but the
// HTTP layer is expected to populate both.
func FromMaterializedGraph(mg *graph.MaterializedGraph, knownInitiatives, inProgressRefs []string) (CurrentState, error) {
	if mg == nil {
		return CurrentState{}, fmt.Errorf("proposals: materialized graph is nil")
	}
	state := CurrentState{
		InitiativeName: mg.Initiative,
		Nodes:          make(map[string]GraphNode, len(mg.Nodes)),
		Edges:          make([]GraphEdge, 0, len(mg.Edges)),
	}
	for _, n := range mg.Nodes {
		state.Nodes[n.ID] = GraphNode{
			ID:       n.ID,
			Kind:     n.Kind,
			Name:     n.Name,
			Title:    n.Title,
			Priority: n.Priority,
			Effort:   n.Effort,
		}
	}
	for _, e := range mg.Edges {
		state.Edges = append(state.Edges, GraphEdge{From: e.From, To: e.To, Kind: e.Kind})
	}
	if len(knownInitiatives) > 0 {
		state.KnownInitiatives = make(map[string]struct{}, len(knownInitiatives))
		for _, n := range knownInitiatives {
			state.KnownInitiatives[n] = struct{}{}
		}
	}
	if len(inProgressRefs) > 0 {
		state.InProgressRefs = make(map[string]struct{}, len(inProgressRefs))
		for _, r := range inProgressRefs {
			state.InProgressRefs[r] = struct{}{}
		}
	}
	return state, nil
}
