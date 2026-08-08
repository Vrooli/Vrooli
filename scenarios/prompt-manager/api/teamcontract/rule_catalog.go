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
		entry("contract_knowledge_topics_missing", "An operating contract declares no knowledge topics."),
		entry("contract_members_missing", "An operating contract declares no members."),
		entry("contract_member_absent", "An active team member is absent from the operating contract."),
		entry("contract_member_id_empty", "An operating contract declares a member with an empty id."),
		entry("contract_member_allowed_writes_invalid", "A member's allowedWrites declaration cannot be resolved."),
		entry("contract_member_forbidden_writes_invalid", "A member's forbiddenWrites declaration cannot be resolved."),
		entry("contract_documents_invalid", "An operating contract's documents declaration cannot be resolved."),
	)
}
