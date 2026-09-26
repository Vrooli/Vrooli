package proposals

import (
	"fmt"

	"swarm-manager/internal/graph"
)

// FromMaterializedGraph lifts a graph.MaterializedGraph into the
// CurrentState shape proposals validate/apply expects.
//
// `knownMilestones` is the set of milestones that exist on disk (used by
// OpMoveMilestone to reject phantom destinations). `inProgressRefs` is the
// subset of graph nodes whose items are currently StatusInProgress (used
// by OpInterruptInProgress). Either may be nil — in which case those checks
// degrade to "accept anything", which is fine for test contexts but the
// HTTP layer is expected to populate both.
func FromMaterializedGraph(mg *graph.MaterializedGraph, knownMilestones, inProgressRefs []string) (CurrentState, error) {
	if mg == nil {
		return CurrentState{}, fmt.Errorf("proposals: materialized graph is nil")
	}
	state := CurrentState{
		MilestoneName: mg.Goal,
		Nodes:         make(map[string]GraphNode, len(mg.Nodes)),
		Edges:         make([]GraphEdge, 0, len(mg.Edges)),
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
	if len(knownMilestones) > 0 {
		state.KnownMilestones = make(map[string]struct{}, len(knownMilestones))
		for _, n := range knownMilestones {
			state.KnownMilestones[n] = struct{}{}
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
