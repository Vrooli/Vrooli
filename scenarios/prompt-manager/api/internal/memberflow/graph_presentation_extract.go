package memberflow

import "sort"

// ExtractGraphPresentation lifts the readability layer out of a hand-drawn
// contract block so it survives generation.
//
// This is a one-time transcription, not a permanent code path: the drawn graph
// is the specification for what a team wants its diagram to look like, and this
// reads that intent out rather than asking anyone to retype it. Only
// presentation is extracted — short names, display labels, node order, and the
// process/future nodes and their edges, which OPERATING_GRAPHS.md states
// satisfy no completeness rule and therefore assert nothing about runtime.
func ExtractGraphPresentation(block OperatingGraphBlock) GraphPresentation {
	presentation := GraphPresentation{
		ShortNames: map[string]string{},
		Displays:   map[string]string{},
	}

	readability := map[string]bool{}
	for _, node := range block.Graph.Nodes {
		if node.Kind == "" || node.Value == "" {
			continue
		}
		value := string(node.Kind) + ":" + node.Value
		if node.ID != "" {
			presentation.ShortNames[value] = node.ID
		}
		if node.Display != "" && node.Display != node.Value {
			presentation.Displays[value] = node.Display
		}
		presentation.NodeOrder = append(presentation.NodeOrder, value)
		if node.Kind == OperatingGraphNodeKindProcess || node.Kind == OperatingGraphNodeKindFuture {
			readability[node.ID] = true
		}
	}

	byID := map[string]string{}
	for _, node := range block.Graph.Nodes {
		if node.Kind != "" && node.Value != "" {
			byID[node.ID] = string(node.Kind) + ":" + node.Value
		}
	}
	for _, edge := range block.Graph.Edges {
		if !readability[edge.From] && !readability[edge.To] {
			continue
		}
		from, fok := byID[edge.From]
		to, tok := byID[edge.To]
		if !fok || !tok {
			continue
		}
		presentation.ReadabilityEdges = append(presentation.ReadabilityEdges, GraphPresentationEdge{From: from, To: to})
	}
	sort.Slice(presentation.ReadabilityEdges, func(i, j int) bool {
		if presentation.ReadabilityEdges[i].From != presentation.ReadabilityEdges[j].From {
			return presentation.ReadabilityEdges[i].From < presentation.ReadabilityEdges[j].From
		}
		return presentation.ReadabilityEdges[i].To < presentation.ReadabilityEdges[j].To
	})
	return presentation
}
