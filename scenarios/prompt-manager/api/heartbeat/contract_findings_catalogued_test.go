package heartbeat

import (
	"strings"
	"testing"

	"prompt-manager/finding"
	"prompt-manager/store"
	"prompt-manager/teamcontract"
)

// A malformed team.json must reach the member who can fix it, carrying the
// catalogued rule id and severity that let an operator rank and document it.
//
// Before the finding types were unified, teamcontract.ValidateFindings returned
// a shape with only Field and Message, and the heartbeat merge synthesized
// `team_teamcontract_invalid` for every defect regardless of what was wrong.
// One anonymous id stood for 27 distinct contract defects.
func TestMalformedTeamContractReachesTheMemberWithACataloguedRuleID(t *testing.T) {
	catalog, err := teamcontract.ContractRuleCatalog()
	if err != nil {
		t.Fatalf("build contract catalog: %v", err)
	}

	// A contract that names a decision-context owner who is not a member.
	contract := &teamcontract.OperatingContract{
		SchemaVersion: teamcontract.SchemaVersion,
		Governance: teamcontract.Governance{
			DecisionMode: teamcontract.DecisionModeYolo,
		},
		DecisionContext: map[string]teamcontract.DecisionContext{
			"some-context": {OwnerMemberIDs: []string{"not-a-member"}},
		},
		KnowledgeTopics: map[string]teamcontract.KnowledgeTopic{"t/*": {}},
		Members:         map[string]teamcontract.MemberContract{"real-member": {}},
	}

	findings := teamcontract.ValidateFindings(contract, teamcontract.ValidationInput{
		TeamID:       "fixture-team",
		DecisionMode: teamcontract.DecisionModeYolo,
	})
	if len(findings) == 0 {
		t.Fatal("a contract naming an unknown decision-context owner produced no finding")
	}

	var owner *finding.Finding
	for i := range findings {
		if findings[i].Rule == "contract_decision_context_owner_unknown" {
			owner = &findings[i]
		}
		if findings[i].Rule == "" {
			t.Errorf("finding %q carries no rule id", findings[i].Detail)
			continue
		}
		if _, ok := catalog[findings[i].Rule]; !ok {
			t.Errorf("finding emits rule id %q that the contract catalog does not define", findings[i].Rule)
		}
		if findings[i].Severity == "" {
			t.Errorf("finding %q carries no severity", findings[i].Rule)
		}
		if findings[i].Kind != finding.KindDeclaration {
			t.Errorf("contract finding %q has kind %q, want declaration", findings[i].Rule, findings[i].Kind)
		}
	}
	if owner == nil {
		t.Fatalf("no contract_decision_context_owner_unknown finding: %+v", findings)
	}
	if owner.Path == "" {
		t.Error("finding names no field to change")
	}

	// The merge into # Contract Findings must preserve that identity rather
	// than collapsing it to a synthesized per-source id.
	byMember := map[string][]ContractFinding{}
	mergeTeamValidationFindings(byMember, []store.Team{{
		ID:                "fixture-team",
		OperatingContract: contract,
		ValidationFindings: []store.TeamValidationFinding{{
			Source:  "teamcontract",
			Rule:    owner.Rule,
			Field:   owner.Path,
			Message: owner.Detail,
		}},
	}})

	merged := byMember["fixture-team/real-member"]
	if len(merged) != 1 {
		t.Fatalf("merged findings for the member = %d, want 1: %+v", len(merged), byMember)
	}
	if merged[0].Rule != "contract_decision_context_owner_unknown" {
		t.Errorf("merged rule id = %q, want the catalogued id", merged[0].Rule)
	}
	if merged[0].Severity != finding.SeverityError {
		t.Errorf("merged severity = %q, want error", merged[0].Severity)
	}

	rendered := renderContractFindings("fixture-team", merged)
	if !strings.Contains(rendered, "contract_decision_context_owner_unknown") {
		t.Errorf("# Contract Findings does not name the rule id:\n%s", rendered)
	}
	if !strings.Contains(strings.ToLower(rendered), "error") {
		t.Errorf("# Contract Findings does not name the severity:\n%s", rendered)
	}
}
