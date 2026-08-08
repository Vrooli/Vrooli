package memberflow

import (
	"testing"

	"prompt-manager/teamcontract"
)

// Behavioral tests for the graph entity rules.
//
// These read the nodes drawn in a contract graph and check each against what
// the runtime declares. None was named by a test at plan start, so nothing
// verified that an unknown member, decision, team, or plan-of-record path in a
// hand-drawn graph would actually be reported.

// entityContext wires a contract block whose graph holds the given nodes,
// against a team contract declaring exactly one member.
func entityContext(nodes []OperatingGraphNode) RuleContext {
	const team = "team-a"
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{
			ID:    team + "-operating-model",
			Team:  team,
			Scope: "team",
			Mode:  OperatingGraphModeContract,
		},
		Source: OperatingGraphSource{Path: "docs/" + team + "/operating/OPERATING_MODEL.md", FenceLine: 1},
		Graph:  OperatingGraph{Nodes: nodes},
	}
	runtime := OperatingGraphRuntime{
		Contracts: TeamContractRegistry{
			team: &LoadedTeamContract{TeamID: team, Contract: &teamcontract.OperatingContract{
				SchemaVersion: teamcontract.SchemaVersion,
				Members:       map[string]teamcontract.MemberContract{"known-member": {}},
			}},
		},
	}
	return RuleContext{
		OperatingGraphRuleContext: OperatingGraphRuleContext{
			Block:   block,
			Runtime: runtime,
			Index:   NewOperatingGraphContractIndex(block, runtime),
			Matcher: NewOperatingRelationshipMatcher(),
		},
	}
}

func node(id string, kind OperatingGraphNodeKind, value string) OperatingGraphNode {
	return OperatingGraphNode{ID: id, Kind: kind, Value: value, Display: value, RawLabel: value, SourceLine: 10}
}

func TestGraphEntityRulesFireOnNodesTheRuntimeDoesNotBack(t *testing.T) {
	for _, tc := range []struct {
		name  string
		rule  string
		nodes []OperatingGraphNode
		check func(RuleContext) []OperatingGraphFinding
	}{
		{
			name:  "node carries no typed machine label",
			rule:  "graph_untyped_node",
			nodes: []OperatingGraphNode{{ID: "X", RawLabel: "Some Box", SourceLine: 10}},
			check: func(c RuleContext) []OperatingGraphFinding { return (graphUntypedNodeRule{}).Check(c) },
		},
		{
			name:  "node uses a type outside the vocabulary",
			rule:  "graph_unknown_node_kind",
			nodes: []OperatingGraphNode{node("X", OperatingGraphNodeKind("banana"), "whatever")},
			check: func(c RuleContext) []OperatingGraphFinding { return (graphUnknownNodeKindRule{}).Check(c) },
		},
		{
			name:  "member node names nobody in the team contract",
			rule:  "graph_unknown_member",
			nodes: []OperatingGraphNode{node("X", OperatingGraphNodeKindMember, "ghost-member")},
			check: func(c RuleContext) []OperatingGraphFinding { return (graphUnknownMemberRule{}).Check(c) },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			findings := tc.check(entityContext(tc.nodes))
			if len(findings) == 0 {
				t.Fatalf("%s did not fire", tc.rule)
			}
			got := findings[0]
			if got.Rule != tc.rule {
				t.Errorf("rule = %q, want %q", got.Rule, tc.rule)
			}
			if got.Severity == "" {
				t.Errorf("%s fired with no severity", tc.rule)
			}
			if got.Detail == "" {
				t.Errorf("%s fired with no detail naming what to change", tc.rule)
			}
		})
	}
}

// The same rules must stay silent on nodes the runtime does back, or they would
// report every graph in the fleet.
func TestGraphEntityRulesStaySilentOnBackedNodes(t *testing.T) {
	ctx := entityContext([]OperatingGraphNode{
		node("M", OperatingGraphNodeKindMember, "known-member"),
	})
	for name, findings := range map[string][]OperatingGraphFinding{
		"graph_untyped_node":      (graphUntypedNodeRule{}).Check(ctx),
		"graph_unknown_node_kind": (graphUnknownNodeKindRule{}).Check(ctx),
		"graph_unknown_member":    (graphUnknownMemberRule{}).Check(ctx),
	} {
		if len(findings) != 0 {
			t.Errorf("%s fired on a backed graph: %+v", name, findings)
		}
	}
}

// Three more entity rules that read node values against the wider runtime:
// the team registry, the filesystem, and the shape convention.
func TestGraphEntityRulesFireOnUnbackedTeamPORAndShape(t *testing.T) {
	t.Run("team node names no registered team", func(t *testing.T) {
		ctx := entityContext([]OperatingGraphNode{node("T", "team", "no-such-team")})
		findings := (graphUnknownTeamRule{}).Check(ctx)
		if len(findings) == 0 {
			t.Fatal("graph_unknown_team did not fire")
		}
		if findings[0].Rule != "graph_unknown_team" {
			t.Errorf("rule = %q", findings[0].Rule)
		}
		if findings[0].Severity != SeverityWarning {
			t.Errorf("severity = %q, want warning", findings[0].Severity)
		}
		// A registered team must not fire it.
		backed := entityContext([]OperatingGraphNode{node("T", "team", "team-a")})
		if got := (graphUnknownTeamRule{}).Check(backed); len(got) != 0 {
			t.Errorf("graph_unknown_team fired on a registered team: %+v", got)
		}
	})

	t.Run("plan-of-record node points at a path that does not exist", func(t *testing.T) {
		ctx := entityContext([]OperatingGraphNode{node("P", "por", "docs/team-a/strategy/ABSENT.md")})
		ctx.Runtime.RepoRoot = t.TempDir()
		findings := (graphUnknownPORRule{}).Check(ctx)
		if len(findings) == 0 {
			t.Fatal("graph_unknown_por did not fire for a path that does not exist")
		}
		if findings[0].Rule != "graph_unknown_por" {
			t.Errorf("rule = %q", findings[0].Rule)
		}
		if findings[0].Severity != SeverityError {
			t.Errorf("severity = %q, want error", findings[0].Severity)
		}
	})

	t.Run("node shape contradicts its declared kind", func(t *testing.T) {
		// A member is drawn as a rectangle by convention; a cylinder is the
		// topic shape, so this node claims one thing and reads as another.
		bad := node("M", OperatingGraphNodeKindMember, "known-member")
		bad.Shape = "cylinder"
		findings := (graphNodeShapeConventionDriftRule{}).Check(entityContext([]OperatingGraphNode{bad}))
		if len(findings) == 0 {
			t.Fatal("graph_node_shape_convention_drift did not fire")
		}
		if findings[0].Rule != "graph_node_shape_convention_drift" {
			t.Errorf("rule = %q", findings[0].Rule)
		}
		if findings[0].Severity != SeverityWarning {
			t.Errorf("severity = %q, want warning", findings[0].Severity)
		}

		ok := node("M", OperatingGraphNodeKindMember, "known-member")
		ok.Shape = "rectangle"
		if got := (graphNodeShapeConventionDriftRule{}).Check(entityContext([]OperatingGraphNode{ok})); len(got) != 0 {
			t.Errorf("graph_node_shape_convention_drift fired on a conventional shape: %+v", got)
		}
	})
}
