package teamcontract

import "prompt-manager/finding"

// OperatingRuleGroupContract groups every rule that reads an operating
// contract out of team.json.
const OperatingRuleGroupContract finding.RuleGroup = "contract"

// contractRuleCatalog is the operator-facing identity for team.json contract
// validation.
//
// Before this, ValidateFindings returned a shape carrying only Field and
// Message. A malformed team.json could therefore never be catalogued, ranked
// by severity, or rendered into the responsible member's `# Contract Findings`
// — the one surface where a contract defect reaches the agent who can fix it.
// Every check now names a rule id, and registration fails if a check emits an
// id this catalog does not define.
var contractRuleCatalog = mustContractCatalog()

func mustContractCatalog() finding.RuleCatalog {
	catalog, err := ContractRuleCatalog()
	if err != nil {
		panic(err)
	}
	return catalog
}

// ContractRuleCatalog returns the contract-family catalog.
func ContractRuleCatalog() (finding.RuleCatalog, error) {
	entry := func(id, description string) finding.RuleCatalogEntry {
		return finding.RuleCatalogEntry{
			ID:          id,
			Group:       OperatingRuleGroupContract,
			Severity:    finding.SeverityError,
			Kind:        finding.KindDeclaration,
			Description: description,
			Actuator:    "Correct the named field in the team's team.json operatingContract",
		}
	}
	return finding.NewRuleCatalog(
		entry("contract_missing", "A team declares no operating contract."),
		entry("contract_schema_version_invalid", "An operating contract declares an unsupported schema version."),
		entry("contract_decision_mode_missing", "An operating contract declares no governance decision mode."),
		entry("contract_decision_mode_team_mismatch", "A contract decision mode disagrees with the team's decisionMode."),
		entry("contract_decision_mode_unknown", "A contract decision mode is outside the permitted vocabulary."),
		entry("contract_team_pending_ceiling_negative", "A team pending-decision ceiling is negative."),
		entry("contract_decision_contexts_missing", "An operating contract declares no decision contexts."),
		entry("contract_knowledge_topics_missing", "An operating contract declares no knowledge topics."),
		entry("contract_members_missing", "An operating contract declares no members."),
		entry("contract_member_absent", "An active team member is absent from the operating contract."),
		entry("contract_member_id_empty", "An operating contract declares a member with an empty id."),
		entry("contract_member_decision_cap_negative", "A member's per-heartbeat decision cap is negative."),
		entry("contract_member_pending_cap_negative", "A member's pending-owned decision cap is negative."),
		entry("contract_member_context_cap_negative", "A member's per-context decision cap is negative."),
		entry("contract_member_context_cap_undeclared", "A member caps a decision context the contract does not declare."),
		entry("contract_member_owned_context_undeclared", "A member owns a decision context the contract does not declare."),
		entry("contract_member_decision_cap_contradicts_forbidden_write", "A member forbids decision writes yet declares a per-heartbeat decision cap."),
		entry("contract_member_pending_cap_contradicts_forbidden_write", "A member forbids decision writes yet declares a pending-owned decision cap."),
		entry("contract_member_context_cap_contradicts_forbidden_write", "A member forbids decision writes yet declares a per-context decision cap."),
		entry("contract_member_allowed_writes_invalid", "A member's allowedWrites declaration cannot be resolved."),
		entry("contract_member_forbidden_writes_invalid", "A member's forbiddenWrites declaration cannot be resolved."),
		entry("contract_decision_context_unowned", "A decision context names neither an owner member nor an external raiser."),
		entry("contract_decision_context_owner_unknown", "A decision context names an owner that is not a contract member."),
		entry("contract_stale_policy_after_heartbeats_invalid", "A stale-decision policy triggers after fewer than one heartbeat."),
		entry("contract_stale_policy_owner_unknown", "A stale-decision policy names an owner that is not a contract member."),
		entry("contract_stale_policy_outcomes_missing", "A stale-decision policy declares no required outcomes."),
		entry("contract_documents_invalid", "An operating contract's documents declaration cannot be resolved."),
	)
}
