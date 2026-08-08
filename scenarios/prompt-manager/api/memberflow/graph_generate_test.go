package memberflow

import (
	"strings"
	"testing"

	"prompt-manager/teamcontract"
)

// The generator's contract: what it emits must parse back through the same
// Mermaid subset parser the hand-drawn graphs use, and must carry exactly the
// relationships the declarations state — no more, no fewer.
func generatorRuntime(team, member string, topics Topics) OperatingGraphRuntime {
	return OperatingGraphRuntime{
		Members: []MemberTopics{{
			Ref: MemberRef{Team: team, Member: member}, Topics: topics, Exists: true,
		}},
		Contracts: TeamContractRegistry{
			team: &LoadedTeamContract{TeamID: team, Contract: &teamcontract.OperatingContract{
				SchemaVersion: teamcontract.SchemaVersion,
				Members:       map[string]teamcontract.MemberContract{member: {}},
			}},
		},
	}
}

func TestGeneratedGraphCarriesEveryDeclaredRelationship(t *testing.T) {
	const team, member = "team-a", "member-a"
	block, err := GenerateOperatingGraphBlock(GenerateOperatingGraphInput{
		TeamID: team,
		Runtime: generatorRuntime(team, member, Topics{
			Intake:       []IntakeEntry{{Prefix: "some-inbox/*", Taxonomy: "friction"}},
			Output:       []OutputEntry{{Prefix: "some-output/*", DestinationKind: "knowledge"}},
			RequiredRead: []RequiredReadEntry{{Prefix: "some-record/*"}},
		}),
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	for _, want := range []string{
		"```mermaid",
		"flowchart LR",
		"%% @node",
		"member:" + member,
		"topic:some-inbox/*",
		"topic:some-output/*",
		"topic:some-record/*",
	} {
		if !strings.Contains(block, want) {
			t.Errorf("generated block missing %q:\n%s", want, block)
		}
	}
	// The member is drawn as a rectangle and topics as cylinders, so a generated
	// graph can never trip graph_node_shape_convention_drift.
	if !strings.Contains(block, "[(some-inbox/*)]") {
		t.Errorf("topic node does not use the cylinder shape:\n%s", block)
	}
}

// The generated block must be readable by the parser that reads hand-drawn
// blocks, or generation would replace one format with a second one.
func TestGeneratedGraphParsesThroughTheExistingParser(t *testing.T) {
	const team, member = "team-a", "member-a"
	block, err := GenerateOperatingGraphBlock(GenerateOperatingGraphInput{
		TeamID: team,
		Runtime: generatorRuntime(team, member, Topics{
			Intake: []IntakeEntry{{Prefix: "some-inbox/*", Taxonomy: "friction"}},
			Output: []OutputEntry{{Prefix: "some-output/*", DestinationKind: "knowledge"}},
		}),
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// Strip the fence; the parser takes the diagram lines.
	lines := strings.Split(strings.TrimSpace(block), "\n")
	if len(lines) < 2 {
		t.Fatalf("generated block is too short:\n%s", block)
	}
	lines = lines[1 : len(lines)-1]
	graph, err := ParseOperatingMermaid("team-a-operating-model", lines, 1)
	if err != nil {
		t.Fatalf("generated block does not parse through the Mermaid subset parser: %v\n%s", err, block)
	}
	if len(graph.Nodes) == 0 {
		t.Fatalf("parser found no nodes in the generated block:\n%s", block)
	}
	byValue := map[string]OperatingGraphNode{}
	for _, node := range graph.Nodes {
		byValue[string(node.Kind)+":"+node.Value] = node
	}
	for _, want := range []string{"member:" + member, "topic:some-inbox/*", "topic:some-output/*"} {
		if _, ok := byValue[want]; !ok {
			t.Errorf("parsed graph is missing %q; parsed: %v", want, byValue)
		}
	}
	if len(graph.Edges) == 0 {
		t.Errorf("parser found no edges in the generated block:\n%s", block)
	}
}

// Generation must be deterministic, or every regeneration produces a diff and
// the drift test becomes noise.
func TestGeneratedGraphIsDeterministic(t *testing.T) {
	const team, member = "team-a", "member-a"
	in := GenerateOperatingGraphInput{
		TeamID: team,
		Runtime: generatorRuntime(team, member, Topics{
			Intake: []IntakeEntry{{Prefix: "b-inbox/*", Taxonomy: "friction"}, {Prefix: "a-inbox/*", Taxonomy: "friction"}},
			Output: []OutputEntry{{Prefix: "z-output/*", DestinationKind: "knowledge"}},
		}),
	}
	first, err := GenerateOperatingGraphBlock(in)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for i := 0; i < 5; i++ {
		again, err := GenerateOperatingGraphBlock(in)
		if err != nil {
			t.Fatalf("regenerate: %v", err)
		}
		if again != first {
			t.Fatalf("generation is not deterministic:\n--- first ---\n%s\n--- again ---\n%s", first, again)
		}
	}
}

// The presentation layer may not smuggle in a contract relationship. This is
// what keeps the readability file from becoming a second declaration surface.
func TestGraphPresentationRejectsAnEdgeBetweenContractNodes(t *testing.T) {
	err := GraphPresentation{ReadabilityEdges: []GraphPresentationEdge{
		{From: "member:member-a", To: "topic:some-record/*"},
	}}.Validate()
	if err == nil {
		t.Fatal("presentation accepted a readability edge between two contract nodes")
	}
	if !strings.Contains(err.Error(), "derived content") {
		t.Errorf("error does not explain why: %v", err)
	}

	// An edge touching a process node is readability and is allowed.
	if err := (GraphPresentation{ReadabilityEdges: []GraphPresentationEdge{
		{From: "process:learning-synthesis", To: "member:member-a"},
	}}).Validate(); err != nil {
		t.Errorf("presentation rejected a legitimate readability edge: %v", err)
	}
}
