package memberflow

import (
	"testing"

	"prompt-manager/teamcontract"
)

// Behavioral tests for the two prompt topic-contract rules.
//
// These are the only rules in the catalog that read an assembled prompt rather
// than a checked-in declaration. That makes them the closest existing relatives
// of the prompt structure invariant added in Phase 1, and neither was named by
// a test — so the one family that watches the output had no coverage at all.

func promptContext(t *testing.T, member string, sections []OperatingGraphPromptSection) RuleContext {
	t.Helper()
	const team = "team-a"
	memberNode := node("M", OperatingGraphNodeKindMember, member)
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{
			ID: team + "-operating-model", Team: team, Scope: "team", Mode: OperatingGraphModeContract,
		},
		Source: OperatingGraphSource{Path: "docs/" + team + "/operating/OPERATING_MODEL.md", FenceLine: 1},
		Graph:  OperatingGraph{Nodes: []OperatingGraphNode{memberNode}},
	}
	runtime := OperatingGraphRuntime{
		Members: []MemberTopics{{Ref: MemberRef{Team: team, Member: member}, Exists: true}},
		Contracts: TeamContractRegistry{
			team: &LoadedTeamContract{TeamID: team, Contract: &teamcontract.OperatingContract{
				SchemaVersion: teamcontract.SchemaVersion,
				Members:       map[string]teamcontract.MemberContract{member: {}},
			}},
		},
		PromptSections: map[MemberRef][]OperatingGraphPromptSection{
			{Team: team, Member: member}: sections,
		},
	}
	return RuleContext{OperatingGraphRuleContext: OperatingGraphRuleContext{
		Block: block, Runtime: runtime,
		Index:   NewOperatingGraphContractIndex(block, runtime),
		Matcher: NewOperatingRelationshipMatcher(),
	}}
}

func TestPromptTopicContractSourceMismatchFiresWhenTheSectionCitesTheWrongFile(t *testing.T) {
	const member = "member-a"
	ctx := promptContext(t, member, []OperatingGraphPromptSection{{
		Team:       "team-a",
		Member:     member,
		Kind:       operatingGraphPromptSectionKindTopicContract,
		SourcePath: "teams/team-a/members/member-a/SOMETHING_ELSE.json",
		Content:    "rendered topic contract",
	}})

	findings := (graphPromptTopicContractSourceMismatchRule{}).Check(ctx)
	if len(findings) == 0 {
		t.Fatal("graph_prompt_topic_contract_source_mismatch did not fire for a section citing the wrong source")
	}
	if findings[0].Rule != "graph_prompt_topic_contract_source_mismatch" {
		t.Errorf("rule = %q", findings[0].Rule)
	}
	if findings[0].Member != member {
		t.Errorf("finding does not name the member: %+v", findings[0])
	}

	// The section citing the declared source must stay silent.
	ok := promptContext(t, member, []OperatingGraphPromptSection{{
		Team:       "team-a",
		Member:     member,
		Kind:       operatingGraphPromptSectionKindTopicContract,
		SourcePath: expectedTopicContractSourcePath("team-a", member),
		Content:    "rendered topic contract",
	}})
	if got := (graphPromptTopicContractSourceMismatchRule{}).Check(ok); len(got) != 0 {
		t.Errorf("graph_prompt_topic_contract_source_mismatch fired on a correctly-sourced section: %+v", got)
	}
}

func TestPromptTopicContractContentMismatchFiresWhenTheSectionDiffersFromTheDeclaration(t *testing.T) {
	const member = "member-a"
	// Content that is not what topics.json renders. This is the rule that keeps
	// a member's prompt from asserting a contract its declaration does not hold.
	ctx := promptContext(t, member, []OperatingGraphPromptSection{{
		Team:       "team-a",
		Member:     member,
		Kind:       operatingGraphPromptSectionKindTopicContract,
		SourcePath: expectedTopicContractSourcePath("team-a", member),
		Content:    "# Topic Contract\n\nsomething the declaration never said\n",
	}})

	findings := (graphPromptTopicContractContentMismatchRule{}).Check(ctx)
	if len(findings) == 0 {
		t.Fatal("graph_prompt_topic_contract_content_mismatch did not fire for content differing from the declaration render")
	}
	if findings[0].Rule != "graph_prompt_topic_contract_content_mismatch" {
		t.Errorf("rule = %q", findings[0].Rule)
	}
	if findings[0].Severity != SeverityError {
		t.Errorf("severity = %q, want error", findings[0].Severity)
	}

	// A member with no topic-contract prompt section has nothing to compare.
	none := promptContext(t, member, nil)
	if got := (graphPromptTopicContractContentMismatchRule{}).Check(none); len(got) != 0 {
		t.Errorf("graph_prompt_topic_contract_content_mismatch fired with no prompt section: %+v", got)
	}
}
