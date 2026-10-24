package memberflow

import "testing"

// Behavioral tests for the graph edge-truth rules.
//
// These read the arrows drawn between typed nodes. Neither rule was named by a
// test at plan start, so nothing verified that a graph could not quietly assert
// a flow the system does not support, or draw a topic that does not exist yet
// as though it already carried traffic.

func edgeContext(nodes []OperatingGraphNode, edges []OperatingGraphEdge) RuleContext {
	const team = "team-a"
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{
			ID:    team + "-operating-model",
			Team:  team,
			Scope: "team",
			Mode:  OperatingGraphModeContract,
		},
		Source: OperatingGraphSource{Path: "docs/" + team + "/operating/OPERATING_MODEL.md", FenceLine: 1},
		Graph:  OperatingGraph{Nodes: nodes, Edges: edges},
	}
	runtime := OperatingGraphRuntime{}
	return RuleContext{
		OperatingGraphRuleContext: OperatingGraphRuleContext{
			Block:   block,
			Runtime: runtime,
			Index:   NewOperatingGraphContractIndex(block, runtime),
			Matcher: NewOperatingRelationshipMatcher(),
		},
	}
}

func TestGraphFutureTopicLiveEdgeFiresWhenAFutureTopicCarriesTraffic(t *testing.T) {
	future := node("F", "topic", "planned-record/*")
	future.Qualifier = "future"
	member := node("M", OperatingGraphNodeKindMember, "known-member")

	ctx := edgeContext(
		[]OperatingGraphNode{future, member},
		[]OperatingGraphEdge{{From: "F", To: "M", SourceLine: 60}},
	)
	findings := (graphFutureTopicLiveEdgeRule{}).Check(ctx)
	if len(findings) == 0 {
		t.Fatal("graph_future_topic_live_edge did not fire for a future topic used as an active edge source")
	}
	if findings[0].Rule != "graph_future_topic_live_edge" {
		t.Errorf("rule = %q", findings[0].Rule)
	}
	if findings[0].Severity != SeverityWarning {
		t.Errorf("severity = %q, want warning", findings[0].Severity)
	}
	if findings[0].Topic != "planned-record/*" {
		t.Errorf("finding does not name the future topic: %+v", findings[0])
	}

	// The same edge from a live topic is ordinary and must stay silent.
	live := node("L", "topic", "current-record/*")
	quiet := edgeContext(
		[]OperatingGraphNode{live, member},
		[]OperatingGraphEdge{{From: "L", To: "M", SourceLine: 61}},
	)
	if got := (graphFutureTopicLiveEdgeRule{}).Check(quiet); len(got) != 0 {
		t.Errorf("graph_future_topic_live_edge fired on a live topic edge: %+v", got)
	}
}

func TestGraphUnsupportedEdgeSemanticsFiresOnAnEdgeTheRegistryCannotMap(t *testing.T) {
	// por -> por is not one of the ten relationship families. An edge like this
	// asserts a flow the runtime has no way to express, so it can never be
	// backed by a declaration.
	a := node("A", "por", "docs/team-a/strategy/ONE.md")
	b := node("B", "por", "docs/team-a/strategy/TWO.md")
	ctx := edgeContext(
		[]OperatingGraphNode{a, b},
		[]OperatingGraphEdge{{From: "A", To: "B", SourceLine: 70}},
	)
	findings := (graphUnsupportedEdgeSemanticsRule{}).Check(ctx)
	if len(findings) == 0 {
		t.Fatal("graph_unsupported_edge_semantics did not fire for an unmappable edge")
	}
	if findings[0].Rule != "graph_unsupported_edge_semantics" {
		t.Errorf("rule = %q", findings[0].Rule)
	}
	if findings[0].Severity != SeverityError {
		t.Errorf("severity = %q, want error", findings[0].Severity)
	}

	// topic -> member is the topic-read family and must map cleanly.
	topic := node("T", "topic", "current-record/*")
	member := node("M", OperatingGraphNodeKindMember, "known-member")
	supported := edgeContext(
		[]OperatingGraphNode{topic, member},
		[]OperatingGraphEdge{{From: "T", To: "M", SourceLine: 71}},
	)
	if got := (graphUnsupportedEdgeSemanticsRule{}).Check(supported); len(got) != 0 {
		t.Errorf("graph_unsupported_edge_semantics fired on a supported edge: %+v", got)
	}
}
