package proposals

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// Normalize converts a Proposal into its canonical FormMutationList shape.
//
// For FormMutationList proposals, Normalize is a pass-through (copy with
// whitespace trimming and ID defaulting).
//
// For FormFullGraph proposals, Normalize diffs the target Graph against
// `current` and emits the minimal mutation_list that would converge the
// former to the latter:
//
//   - Nodes in target but not current → OpAddItem
//   - Nodes in current but not target → OpArchiveItem
//     (remove_item is intentionally excluded; "gone" means archived)
//   - Nodes in both whose metadata differs → OpUpdateItem and/or
//     OpChangePriority (whichever subset of fields changed)
//   - Edges in target but not current → OpAddEdge
//   - Edges in current but not target → OpRemoveEdge
//
// Status changes are never emitted from diff — status is lifecycle-owned.
// The agent must emit OpChangeStatus explicitly against a mutation_list to
// propose a non-terminal status change.
//
// The returned proposal preserves the source Rationale and assigns
// deterministic IDs (m1, m2, …) so the UI can render a stable checklist.
func Normalize(p Proposal, current CurrentState) (Proposal, error) {
	switch p.Form {
	case FormMutationList:
		return normalizeMutationList(p), nil
	case FormFullGraph:
		return normalizeFullGraph(p, current)
	}
	return Proposal{}, fmt.Errorf("%w: unknown form %q", ErrInvalidProposal, p.Form)
}

func normalizeMutationList(p Proposal) Proposal {
	out := Proposal{
		Form:      FormMutationList,
		Rationale: p.Rationale,
		Mutations: make([]Mutation, 0, len(p.Mutations)),
	}
	for i, m := range p.Mutations {
		m.Target = strings.TrimSpace(m.Target)
		m.From = strings.TrimSpace(m.From)
		m.To = strings.TrimSpace(m.To)
		m.Initiative = strings.TrimSpace(m.Initiative)
		m.Status = strings.ToLower(strings.TrimSpace(m.Status))
		if strings.TrimSpace(m.ID) == "" {
			m.ID = fmt.Sprintf("m%d", i+1)
		}
		out.Mutations = append(out.Mutations, m)
	}
	return out
}

func normalizeFullGraph(p Proposal, current CurrentState) (Proposal, error) {
	if p.Graph == nil {
		return Proposal{}, fmt.Errorf("%w: form=%s without graph payload", ErrInvalidProposal, FormFullGraph)
	}

	target := indexTargetNodes(p.Graph.Nodes)
	targetEdges := dedupeEdges(p.Graph.Edges)

	out := Proposal{
		Form:      FormMutationList,
		Rationale: p.Rationale,
	}

	// Deterministic order: sorted ref for ops that reference a single node,
	// then sorted (from,to) for edge ops. This keeps mutation IDs stable
	// across equivalent inputs — important for UI checklist state and
	// round-trip tests.
	addRefs := sortedRefDiff(target, current.Nodes)
	removeRefs := sortedRefDiff(current.Nodes, target)
	commonRefs := sortedRefIntersect(target, current.Nodes)

	for _, ref := range addRefs {
		node := target[ref]
		spec := nodeToItemSpec(node)
		out.Mutations = append(out.Mutations, Mutation{
			Op:        OpAddItem,
			Item:      &spec,
			Rationale: "proposed in full-graph target",
		})
	}

	for _, ref := range commonRefs {
		targetNode := target[ref]
		currentNode := current.Nodes[ref]
		patch, priorityChange := diffNodeMetadata(currentNode, targetNode)
		if patch != nil {
			out.Mutations = append(out.Mutations, Mutation{
				Op:        OpUpdateItem,
				Target:    ref,
				Patch:     patch,
				Rationale: "metadata differs from full-graph target",
			})
		}
		if priorityChange != nil {
			out.Mutations = append(out.Mutations, Mutation{
				Op:        OpChangePriority,
				Target:    ref,
				Priority:  priorityChange,
				Rationale: "priority differs from full-graph target",
			})
		}
	}

	for _, ref := range removeRefs {
		out.Mutations = append(out.Mutations, Mutation{
			Op:        OpArchiveItem,
			Target:    ref,
			Rationale: "absent from full-graph target",
		})
	}

	// Edge diff — canonicalize edges to "from|to" keys.
	currentEdgeSet := edgeSet(current.Edges)
	targetEdgeSet := edgeSet(targetEdges)

	addEdges := sortedEdgeDiff(targetEdgeSet, currentEdgeSet)
	for _, key := range addEdges {
		from, to := splitEdgeKey(key)
		out.Mutations = append(out.Mutations, Mutation{
			Op:        OpAddEdge,
			From:      from,
			To:        to,
			Rationale: "edge added in full-graph target",
		})
	}

	removeEdges := sortedEdgeDiff(currentEdgeSet, targetEdgeSet)
	for _, key := range removeEdges {
		from, to := splitEdgeKey(key)
		out.Mutations = append(out.Mutations, Mutation{
			Op:        OpRemoveEdge,
			From:      from,
			To:        to,
			Rationale: "edge removed in full-graph target",
		})
	}

	// Assign deterministic IDs last so reordering above is captured.
	for i := range out.Mutations {
		out.Mutations[i].ID = fmt.Sprintf("m%d", i+1)
	}
	return out, nil
}

func indexTargetNodes(nodes []GraphNode) map[string]GraphNode {
	out := make(map[string]GraphNode, len(nodes))
	for _, n := range nodes {
		id := n.ID
		if id == "" && n.Kind != "" && n.Name != "" {
			id = n.Kind + "/" + n.Name
		}
		if id == "" {
			continue
		}
		// Back-fill Kind/Name from ID if only ID was provided, so downstream
		// mapping works uniformly.
		if n.Kind == "" || n.Name == "" {
			if parts := strings.SplitN(id, "/", 2); len(parts) == 2 {
				n.Kind = parts[0]
				n.Name = parts[1]
			}
		}
		n.ID = id
		out[id] = n
	}
	return out
}

func nodeToItemSpec(n GraphNode) ItemSpec {
	return ItemSpec{
		Kind:        n.Kind,
		Name:        n.Name,
		Title:       n.Title,
		Description: n.Description,
		Priority:    n.Priority,
		Effort:      n.Effort,
		Tags:        append([]string(nil), n.Tags...),
	}
}

// diffNodeMetadata returns the ItemPatch (title/description/effort/tags) and
// a priority pointer describing the diff from current to target. Either or
// both may be nil if the corresponding fields match.
//
// Status is never included — lifecycle owns it. DependsOn is never included
// at the patch level either; edge diff emits add_edge/remove_edge so the UI
// can render each dependency change separately.
func diffNodeMetadata(cur, target GraphNode) (*ItemPatch, *int) {
	patch := &ItemPatch{}
	changed := false

	if target.Title != "" && target.Title != cur.Title {
		t := target.Title
		patch.Title = &t
		changed = true
	}
	if target.Description != cur.Description {
		d := target.Description
		patch.Description = &d
		changed = true
	}
	if target.Effort != cur.Effort {
		e := target.Effort
		patch.Effort = &e
		changed = true
	}
	if !slices.Equal(target.Tags, cur.Tags) {
		tags := append([]string(nil), target.Tags...)
		patch.Tags = &tags
		changed = true
	}

	var priorityChange *int
	if target.Priority != 0 && target.Priority != cur.Priority {
		p := target.Priority
		priorityChange = &p
	}

	if !changed {
		patch = nil
	}
	return patch, priorityChange
}

func sortedRefDiff(a, b map[string]GraphNode) []string {
	out := make([]string, 0)
	for ref := range a {
		if _, exists := b[ref]; !exists {
			out = append(out, ref)
		}
	}
	sort.Strings(out)
	return out
}

func sortedRefIntersect(a, b map[string]GraphNode) []string {
	out := make([]string, 0)
	for ref := range a {
		if _, exists := b[ref]; exists {
			out = append(out, ref)
		}
	}
	sort.Strings(out)
	return out
}

func edgeSet(edges []GraphEdge) map[string]struct{} {
	out := make(map[string]struct{}, len(edges))
	for _, e := range edges {
		if e.From == "" || e.To == "" {
			continue
		}
		out[e.From+"|"+e.To] = struct{}{}
	}
	return out
}

func dedupeEdges(edges []GraphEdge) []GraphEdge {
	seen := make(map[string]struct{}, len(edges))
	out := make([]GraphEdge, 0, len(edges))
	for _, e := range edges {
		key := e.From + "|" + e.To
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, e)
	}
	return out
}

func sortedEdgeDiff(a, b map[string]struct{}) []string {
	out := make([]string, 0)
	for key := range a {
		if _, exists := b[key]; !exists {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

func splitEdgeKey(key string) (string, string) {
	parts := strings.SplitN(key, "|", 2)
	if len(parts) != 2 {
		return key, ""
	}
	return parts[0], parts[1]
}
