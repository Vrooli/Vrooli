// Validation of the objective join.
//
// The rules here read three surfaces and check them against each other:
//
//	OBJECTIVES.md § The objectives  — the operator's table (downward end)
//	team.json::objectivesServed     — the team's declaration (upward end)
//	OPERATING_MODEL.md § Mission    — the team's prose restatement
//
// Both coverage directions from OBJECTIVES.md § "The coverage rule" are
// mechanical here. What stays judgment is whether the objective set is the
// right one, which is the operator's call and is not checkable by anything.
//
// DOC: docs/director-swarm/strategy/OBJECTIVES.md § The coverage rule
package memberflow

import (
	"fmt"
	"sort"
	"strings"
)

// OperatingRuleGroupObjective groups every rule that reads the objective join.
const OperatingRuleGroupObjective RuleGroup = "objective"

// ObjectiveRuleCatalog is the operator-facing contract for objective
// validation. Severities are deliberately split: a broken *declaration* is an
// error because it is always a mistake, while an *unserved objective* is a
// warning because it can be a legitimate, declared operator choice — T2 is
// stated at full weight and staffed by nobody on purpose.
func ObjectiveRuleCatalog() (RuleCatalog, error) {
	entry := func(id string, severity Severity, description, actuator string) RuleCatalogEntry {
		return RuleCatalogEntry{ID: id, Group: OperatingRuleGroupObjective, Severity: severity, Kind: KindDeclaration, Description: description, Actuator: actuator}
	}
	const fixDeclaration = "Correct team.json::objectivesServed, or route an objective-set change through director-swarm's vision-update work item type"
	const fixCoverage = "Raise outcome-direction or capability work in director-swarm"
	return NewRuleCatalog(
		entry("objective_unknown_id", SeverityError,
			"A team declares an objective id that the objective table does not define.", fixDeclaration),
		entry("objective_role_invalid", SeverityError,
			"A team declares an objective role or coverage outside the permitted vocabulary.", fixDeclaration),
		entry("objective_duplicate_declaration", SeverityError,
			"A team declares the same objective id more than once.", fixDeclaration),
		entry("objective_link_missing_upward", SeverityError,
			"A team declares an objective whose table row does not name that team.", fixDeclaration),
		entry("objective_link_missing_downward", SeverityError,
			"The objective table names a team that does not declare that objective.", fixDeclaration),
		entry("objective_role_drift", SeverityWarning,
			"A team's declared role or coverage disagrees with the objective table.", fixDeclaration),
		entry("objective_prose_drift", SeverityWarning,
			"An operating model's Objective served paragraph names ids that differ from its team.json declaration.", fixDeclaration),
		entry("objective_team_unattached", SeverityWarning,
			"A team declares no objectivesServed, so its work traces to no stated intent.", fixCoverage),
		entry("objective_unserved", SeverityWarning,
			"An objective names no serving team and carries no dated gap marker.", fixCoverage),
		entry("objective_unmeasurable", SeverityWarning,
			"An objective names no evidence source, so progress toward it cannot be scored.", fixCoverage),
	)
}

// ObjectiveValidationInput carries every surface the rules read. Each field is
// optional; an absent surface skips the rules that need it rather than failing,
// so a partial checkout still validates what it has.
type ObjectiveValidationInput struct {
	Registry ObjectiveRegistry
	// Declared maps team id to its objectivesServed block.
	Declared map[string][]TeamObjectiveDeclaration
	// TeamSourcePaths maps team id to its team.json path, for finding
	// attribution.
	TeamSourcePaths map[string]string
	// Models supplies the prose half. Only the Mission section is read.
	Models []OperatingModelDocument
}

// objectiveCheck is the single registry entry for the objective family.
//
// The ten objective_* rules were catalogued on 2026-07-30 but never registered:
// ObjectiveRuleCatalog had zero production callers and DefaultRuleCatalog
// composed only the graph, model, topic, and plan-of-record families. Ten rule
// ids reached output with no catalog enforcement behind them.
type objectiveCheck struct{}

func (objectiveCheck) ID() string                { return "objective_unknown_id" }
func (objectiveCheck) Group() RuleGroup          { return OperatingRuleGroupObjective }
func (objectiveCheck) DefaultSeverity() Severity { return SeverityError }

func (objectiveCheck) Emits() []string {
	return []string{
		"objective_unknown_id",
		"objective_role_invalid",
		"objective_duplicate_declaration",
		"objective_link_missing_upward",
		"objective_link_missing_downward",
		"objective_role_drift",
		"objective_prose_drift",
		"objective_team_unattached",
		"objective_unserved",
		"objective_unmeasurable",
	}
}

func (objectiveCheck) AppliesTo(ctx RuleContext) bool { return ctx.ObjectiveInput != nil }

func (objectiveCheck) Check(ctx RuleContext) []OperatingGraphFinding {
	if ctx.ObjectiveInput == nil {
		return nil
	}
	return objectiveFindings(*ctx.ObjectiveInput)
}

// DefaultObjectiveRules registers the objective family once.
func DefaultObjectiveRules() []Rule { return []Rule{objectiveCheck{}} }

// objectiveFindings is the check body, shared by the registered rule and by
// ValidateObjectives.
func objectiveFindings(in ObjectiveValidationInput) []OperatingGraphFinding {
	var findings []OperatingGraphFinding
	findings = append(findings, checkObjectiveDeclarations(in)...)
	findings = append(findings, checkObjectiveDownwardCoverage(in)...)
	findings = append(findings, checkObjectiveProse(in)...)
	sortObjectiveFindings(findings)
	return findings
}

// ValidateObjectives runs every objective rule and returns the combined result.
func ValidateObjectives(in ObjectiveValidationInput) OperatingGraphValidationResult {
	return tallyObjectiveFindings(objectiveFindings(in))
}

// checkObjectiveDeclarations validates each team's own block and the upward
// half of the coverage rule.
func checkObjectiveDeclarations(in ObjectiveValidationInput) []OperatingGraphFinding {
	var findings []OperatingGraphFinding
	for _, teamID := range sortedMapKeys(in.Declared) {
		decls := in.Declared[teamID]
		source := in.TeamSourcePaths[teamID]
		if len(decls) == 0 {
			findings = append(findings, objectiveFinding("objective_team_unattached", SeverityWarning, teamID, "", source,
				"team.json declares no objectivesServed; every team must trace to at least one objective in "+ObjectivesDocPath))
			continue
		}
		seen := map[string]bool{}
		for _, d := range decls {
			id := strings.TrimSpace(d.ID)
			if seen[id] {
				findings = append(findings, objectiveFinding("objective_duplicate_declaration", SeverityError, teamID, id, source,
					fmt.Sprintf("objective %q is declared more than once", id)))
				continue
			}
			seen[id] = true

			if msg := validateObjectiveVocabulary(d); msg != "" {
				findings = append(findings, objectiveFinding("objective_role_invalid", SeverityError, teamID, id, source, msg))
			}

			obj, ok := in.Registry.Get(id)
			if !ok {
				findings = append(findings, objectiveFinding("objective_unknown_id", SeverityError, teamID, id, source,
					fmt.Sprintf("objective %q is not defined in %s", id, ObjectivesDocPath)))
				continue
			}

			ref, named := objectiveRefFor(obj, teamID)
			if !named {
				findings = append(findings, objectiveFinding("objective_link_missing_upward", SeverityError, teamID, id, source,
					fmt.Sprintf("%s row %s does not name team:%s in its Served by column", ObjectivesDocPath, id, teamID)))
				continue
			}
			if msg := objectiveRoleDrift(d, ref); msg != "" {
				findings = append(findings, objectiveFinding("objective_role_drift", SeverityWarning, teamID, id, source, msg))
			}
		}
	}
	return findings
}

// checkObjectiveDownwardCoverage runs the downward direction: every objective
// traces to a team, or carries a dated gap marker.
func checkObjectiveDownwardCoverage(in ObjectiveValidationInput) []OperatingGraphFinding {
	var findings []OperatingGraphFinding
	for _, obj := range in.Registry.Objectives {
		if obj.Unserved() {
			// A declared hole is reported by the audit every cycle, but it is
			// not a validation error: deferring work is a legitimate operator
			// decision as long as it is visible.
			if obj.GapMarker == "" {
				findings = append(findings, objectiveFinding("objective_unserved", SeverityWarning, "", obj.ID, ObjectivesDocPath,
					fmt.Sprintf("objective %s names no serving team and carries no gap marker", obj.ID)))
			}
		}
		if !obj.HasEvidence {
			findings = append(findings, objectiveFinding("objective_unmeasurable", SeverityWarning, "", obj.ID, ObjectivesDocPath,
				fmt.Sprintf("objective %s names no evidence source; route it to the capability ladder", obj.ID)))
		}
		for _, ref := range obj.ServedBy {
			decls, present := in.Declared[ref.TeamID]
			if !present {
				// The team named by the table does not exist in the store at
				// all. That is a dangling reference, reported downward.
				findings = append(findings, objectiveFinding("objective_link_missing_downward", SeverityError, ref.TeamID, obj.ID, ObjectivesDocPath,
					fmt.Sprintf("%s names team:%s for %s but no such team.json exists", ObjectivesDocPath, ref.TeamID, obj.ID)))
				continue
			}
			if !declaresObjective(decls, obj.ID) {
				findings = append(findings, objectiveFinding("objective_link_missing_downward", SeverityError, ref.TeamID, obj.ID, in.TeamSourcePaths[ref.TeamID],
					fmt.Sprintf("%s names team:%s for %s but that team.json does not declare it in objectivesServed", ObjectivesDocPath, ref.TeamID, obj.ID)))
			}
		}
	}
	return findings
}

// checkObjectiveProse compares each operating model's Mission paragraph against
// the team's declaration. Prose is allowed to say more than the declaration —
// it carries the reasoning — but it must not name a different id set.
func checkObjectiveProse(in ObjectiveValidationInput) []OperatingGraphFinding {
	var findings []OperatingGraphFinding
	for _, model := range in.Models {
		teamID := strings.TrimSpace(model.Team)
		if teamID == "" {
			continue
		}
		decls, present := in.Declared[teamID]
		if !present {
			continue
		}
		proseIDs, found := ProseObjectiveIDs(model.Sections.Mission.Body)
		if !found {
			continue
		}
		declaredIDs := make([]string, 0, len(decls))
		for _, d := range decls {
			declaredIDs = append(declaredIDs, strings.TrimSpace(d.ID))
		}
		missing := setDifference(declaredIDs, proseIDs)
		extra := setDifference(proseIDs, declaredIDs)
		if len(missing) == 0 && len(extra) == 0 {
			continue
		}
		var parts []string
		if len(extra) > 0 {
			parts = append(parts, "prose names "+strings.Join(extra, ", ")+" which team.json does not declare")
		}
		if len(missing) > 0 {
			parts = append(parts, "team.json declares "+strings.Join(missing, ", ")+" which the prose does not name")
		}
		findings = append(findings, OperatingGraphFinding{
			Rule:       "objective_prose_drift",
			Severity:   SeverityWarning,
			GraphID:    model.ID,
			Team:       teamID,
			SourcePath: model.Source.Path,
			Line:       model.Sections.Mission.Line,
			Detail:     strings.Join(parts, "; "),
		})
	}
	return findings
}

// validateObjectiveVocabulary checks role and coverage against the permitted
// values. Both are optional; only a stated value can be wrong.
func validateObjectiveVocabulary(d TeamObjectiveDeclaration) string {
	switch strings.TrimSpace(d.Role) {
	case "", ObjectiveRolePrimary, ObjectiveRoleSupporting:
	default:
		return fmt.Sprintf("role %q is not one of %q, %q", d.Role, ObjectiveRolePrimary, ObjectiveRoleSupporting)
	}
	switch strings.TrimSpace(d.Coverage) {
	case "", ObjectiveCoverageFull, ObjectiveCoveragePartial:
	default:
		return fmt.Sprintf("coverage %q is not one of %q, %q", d.Coverage, ObjectiveCoverageFull, ObjectiveCoveragePartial)
	}
	if strings.TrimSpace(d.ID) == "" {
		return "objectivesServed entry omits its id"
	}
	return ""
}

// objectiveRoleDrift compares a team's declared role/coverage against the
// table. An unqualified table cell constrains nothing: the table qualifies
// roles for some objectives and not others, and a team is free to state a role
// the table leaves open.
func objectiveRoleDrift(d TeamObjectiveDeclaration, ref ObjectiveTeamRef) string {
	declaredRole := strings.TrimSpace(d.Role)
	if ref.Role != "" && declaredRole != "" && ref.Role != declaredRole {
		return fmt.Sprintf("declares role %q but %s says %q", declaredRole, ObjectivesDocPath, ref.Role)
	}
	declaredCoverage := strings.TrimSpace(d.Coverage)
	if declaredCoverage == "" {
		declaredCoverage = ObjectiveCoverageFull
	}
	tableCoverage := ref.Coverage
	if tableCoverage == "" {
		tableCoverage = ObjectiveCoverageFull
	}
	if declaredCoverage != tableCoverage {
		return fmt.Sprintf("declares coverage %q but %s says %q", declaredCoverage, ObjectivesDocPath, tableCoverage)
	}
	return ""
}

func objectiveRefFor(obj Objective, teamID string) (ObjectiveTeamRef, bool) {
	for _, ref := range obj.ServedBy {
		if ref.TeamID == teamID {
			return ref, true
		}
	}
	return ObjectiveTeamRef{}, false
}

func declaresObjective(decls []TeamObjectiveDeclaration, id string) bool {
	for _, d := range decls {
		if strings.EqualFold(strings.TrimSpace(d.ID), id) {
			return true
		}
	}
	return false
}

func objectiveFinding(rule string, severity Severity, teamID, objectiveID, sourcePath, detail string) OperatingGraphFinding {
	return OperatingGraphFinding{
		Rule:       rule,
		Severity:   severity,
		Team:       teamID,
		NodeID:     objectiveID,
		SourcePath: sourcePath,
		Detail:     detail,
	}
}

func tallyObjectiveFindings(findings []OperatingGraphFinding) OperatingGraphValidationResult {
	result := OperatingGraphValidationResult{Findings: findings}
	for _, f := range findings {
		if f.Severity == SeverityError {
			result.Errors++
		} else {
			result.Warnings++
		}
	}
	if result.Findings == nil {
		result.Findings = []OperatingGraphFinding{}
	}
	return result
}

func sortObjectiveFindings(findings []OperatingGraphFinding) {
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			// Errors first: severity strings happen to sort that way, but
			// state the intent rather than relying on it.
			return findings[i].Severity == SeverityError
		}
		if findings[i].Rule != findings[j].Rule {
			return findings[i].Rule < findings[j].Rule
		}
		if findings[i].Team != findings[j].Team {
			return findings[i].Team < findings[j].Team
		}
		return findings[i].NodeID < findings[j].NodeID
	})
}

func sortedMapKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// setDifference returns the members of a that are absent from b, order
// preserved and deduplicated.
func setDifference(a, b []string) []string {
	inB := map[string]bool{}
	for _, v := range b {
		inB[strings.ToUpper(strings.TrimSpace(v))] = true
	}
	var out []string
	seen := map[string]bool{}
	for _, v := range a {
		key := strings.ToUpper(strings.TrimSpace(v))
		if key == "" || inB[key] || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, strings.TrimSpace(v))
	}
	return out
}
