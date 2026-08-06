package memberflow

import "testing"

// Behavioral tests for the `## Decisions` table docs rules.
//
// The table is the human-readable counterpart to the decision nodes drawn in a
// contract graph. These rules are what keep the two from describing different
// decision sets; none was named by a test at plan start, so nothing verified
// that a graph decision missing from the table would be reported.

func decisionsContext(nodes []OperatingGraphNode, table OperatingDecisionTable) RuleContext {
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
		Docs:   OperatingGraphDocs{Decisions: table},
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

func TestDecisionsTableMissingFiresWhenTheTableIsAbsent(t *testing.T) {
	ctx := decisionsContext(nil, OperatingDecisionTable{Present: false})
	findings := (graphDecisionsTableMissingRule{}).Check(ctx)
	if len(findings) != 1 {
		t.Fatalf("graph_decisions_table_missing produced %d findings, want 1", len(findings))
	}
	if findings[0].Rule != "graph_decisions_table_missing" {
		t.Errorf("rule = %q", findings[0].Rule)
	}
	if findings[0].Severity != SeverityError {
		t.Errorf("severity = %q, want error", findings[0].Severity)
	}

	present := decisionsContext(nil, OperatingDecisionTable{Present: true})
	if got := (graphDecisionsTableMissingRule{}).Check(present); len(got) != 0 {
		t.Errorf("graph_decisions_table_missing fired on a present table: %+v", got)
	}
}

func TestDecisionsTableDriftFiresWhenTheGraphDrawsADecisionTheTableOmits(t *testing.T) {
	ctx := decisionsContext(
		[]OperatingGraphNode{node("D", "decision", "undocumented-decision")},
		OperatingDecisionTable{Present: true, Rows: []OperatingDecisionRow{
			{Decision: "some-other-decision", SourceLine: 30},
		}},
	)
	findings := (graphDecisionsTableDriftRule{}).Check(ctx)
	if len(findings) == 0 {
		t.Fatal("graph_decisions_table_drift did not fire for a graph decision absent from the table")
	}
	if findings[0].Rule != "graph_decisions_table_drift" {
		t.Errorf("rule = %q", findings[0].Rule)
	}
	if findings[0].Decision != "undocumented-decision" {
		t.Errorf("finding does not name the drifting decision: %+v", findings[0])
	}

	// When the table documents the drawn decision the rule must stay silent,
	// or every graph with a Decisions table reports drift.
	agreed := decisionsContext(
		[]OperatingGraphNode{node("D", "decision", "documented-decision")},
		OperatingDecisionTable{Present: true, Rows: []OperatingDecisionRow{
			{Decision: "documented-decision", SourceLine: 30},
		}},
	)
	if got := (graphDecisionsTableDriftRule{}).Check(agreed); len(got) != 0 {
		t.Errorf("graph_decisions_table_drift fired on an agreeing table: %+v", got)
	}
}

// graph_decisions_table_owner_drift reports a Decisions row naming an owner the
// graph does not show owning that decision. The table would be assigning
// responsibility the diagram beside it contradicts.
func TestDecisionsTableOwnerDriftFiresOnAnOwnerTheGraphDoesNotShow(t *testing.T) {
	ctx := decisionsContext(
		[]OperatingGraphNode{node("D", "decision", "some-decision")},
		OperatingDecisionTable{Present: true, Rows: []OperatingDecisionRow{{
			Decision:   "some-decision",
			Owners:     []OperatingActorReference{{Kind: OperatingActorKindMember, Value: "some-member", Raw: "Some Member"}},
			SourceLine: 40,
		}}},
	)
	findings := (graphDecisionsTableOwnerDriftRule{}).Check(ctx)
	if len(findings) == 0 {
		t.Fatal("graph_decisions_table_owner_drift did not fire for an owner the graph does not show")
	}
	if findings[0].Rule != "graph_decisions_table_owner_drift" {
		t.Errorf("rule = %q", findings[0].Rule)
	}
	if findings[0].Decision != "some-decision" {
		t.Errorf("finding does not name the decision: %+v", findings[0])
	}

	// A row naming no owners has nothing to contradict.
	quiet := decisionsContext(
		[]OperatingGraphNode{node("D", "decision", "some-decision")},
		OperatingDecisionTable{Present: true, Rows: []OperatingDecisionRow{{
			Decision: "some-decision", SourceLine: 41,
		}}},
	)
	if got := (graphDecisionsTableOwnerDriftRule{}).Check(quiet); len(got) != 0 {
		t.Errorf("graph_decisions_table_owner_drift fired on a row naming no owners: %+v", got)
	}
}
