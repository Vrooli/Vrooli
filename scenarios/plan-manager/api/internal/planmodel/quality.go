package planmodel

import (
	"fmt"
	"strings"
)

type QualitySeverity string

const (
	QualitySeverityWarning QualitySeverity = "warning"
	QualitySeverityFail    QualitySeverity = "fail"
)

type QualityFinding struct {
	Severity QualitySeverity
	Code     string
	Location string
	Message  string
	Guidance string
}

type QualityReport struct {
	Status   string
	Findings []QualityFinding
}

const (
	QualityStatusPass        = "pass"
	QualityStatusNeedsReview = "needs_review"
	QualityStatusFail        = "fail"
)

func (r QualityReport) HasFindings() bool {
	return len(r.Findings) > 0
}

func (r QualityReport) HasFailures() bool {
	for _, finding := range r.Findings {
		if finding.Severity == QualitySeverityFail {
			return true
		}
	}
	return false
}

func (r QualityReport) ExecutionReady() bool {
	return !r.HasFailures()
}

// AssessPlanQuality checks whether a structured plan is execution-grade. It is
// deliberately separate from command execution/reference validation so render
// surfaces, authoring repair prompts, and validation can share one judgment.
func AssessPlanQuality(p Plan, phaseID string) QualityReport {
	var findings []QualityFinding
	if phaseID == "" {
		if p.ImportProvenance != nil || len(p.PreservedLegacySections) > 0 {
			findings = append(findings, qualityWarning("plan.import", "legacy_import_requires_review", "plan was imported from legacy markdown and should be validated/repaired before execution"))
		}
		findings = append(findings, assessPlanPhasePresence(p)...)
		for i, phase := range p.Phases {
			if phase.Order <= 0 {
				phase.Order = i + 1
			}
			findings = append(findings, assessPhaseQuality(phase)...)
		}
		findings = append(findings, assessPlanStructureQuality(p)...)
		findings = append(findings, assessContextQuality("plan.relevant_context", p.RelevantContext)...)
		findings = append(findings, assessSingleHomeQuality(p)...)
	} else {
		found := false
		for _, phase := range p.Phases {
			if phase.ID != phaseID {
				continue
			}
			found = true
			findings = append(findings, assessPhaseQuality(phase)...)
			break
		}
		if !found {
			findings = append(findings, QualityFinding{
				Severity: QualitySeverityWarning,
				Code:     "phase_not_found",
				Location: "phase." + phaseID,
				Message:  fmt.Sprintf("phase %q was not found for plan quality validation", phaseID),
			})
		}
	}
	status := QualityStatusPass
	if len(findings) > 0 {
		status = QualityStatusNeedsReview
		for _, finding := range findings {
			if finding.Severity == QualitySeverityFail {
				status = QualityStatusFail
				break
			}
		}
	}
	return QualityReport{Status: status, Findings: findings}
}

func assessPlanStructureQuality(p Plan) []QualityFinding {
	var findings []QualityFinding
	requiredText := []struct {
		location string
		code     string
		message  string
		value    string
	}{
		{"plan.purpose", "plan_missing_purpose", "plan has no purpose", p.Purpose},
		{"plan.problem_statement", "plan_missing_problem", "plan has no problem/need statement", p.ProblemStatement},
		{"plan.target_outcome", "plan_missing_target_outcome", "plan has no target outcome", p.TargetOutcome},
		{"plan.scope", "plan_missing_scope", "plan has no scope", p.Scope},
		{"plan.technical_approach", "plan_missing_technical_approach", "plan has no technical approach", p.TechnicalApproach},
		{"plan.validation_strategy", "plan_missing_validation_strategy", "plan has no validation strategy", p.ValidationStrategy},
		{"plan.definition_of_done", "plan_missing_definition_of_done", "plan has no definition of done", p.DefinitionOfDone},
	}
	for _, field := range requiredText {
		if strings.TrimSpace(field.value) == "" {
			findings = append(findings, qualityFailure(field.location, field.code, field.message))
		}
	}
	if p.ChangeBoundary.IsZero() || (len(p.ChangeBoundary.AcceptanceAllow) == 0 && strings.TrimSpace(p.ChangeBoundary.OperatorOnlyReason) == "") {
		findings = append(findings, qualityFailure("plan.change_boundary", "plan_missing_change_boundary", "plan has no change boundary acceptance_allow paths or operator-only reason"))
	}
	if !anchorExecutionGrade(p.RegressionAnchor) {
		findings = append(findings, qualityFailure("plan.regression_anchor", "plan_missing_regression_anchor", "plan has no execution-grade regression anchor intent"))
	}
	if !HasPlanReferenceOrNoCodeReason(p) {
		findings = append(findings, qualityFailure("plan.references", "plan_missing_references", "plan has no connected references and no NO_CODE_REFS/operator-only reason"))
	}
	if !HasGlobalContextOrNoContextReason(p.RelevantContext) {
		findings = append(findings, qualityFailure("plan.relevant_context", "plan_missing_global_context", "plan has no global setup context and no NO_CONTEXT reason"))
	}
	if !HasGlobalSkillContextOrNoSkillReason(p.RelevantContext) {
		findings = append(findings, qualityFailure("plan.relevant_context", "plan_missing_skill_context", "plan carries no evidence of a skill sweep: no global skill setup item and no NO_SKILL_CONTEXT/NO_CONTEXT skip reason"))
	}
	return findings
}

// purposeWordTarget is the soft length target for the Purpose section: an
// abstract, not a second problem statement. Exceeding it is a warning only
// (single-home rule D9 — discipline is advisory, never a hard gate).
const purposeWordTarget = 120

// singleHomeDuplicateMinLength guards the near-duplicate warnings against
// trivially short strings ("go test ./...") that legitimately repeat.
const singleHomeDuplicateMinLength = 40

// assessSingleHomeQuality emits the soft single-home warnings (D9): every fact
// should live in exactly one section. Warnings only — never failures.
func assessSingleHomeQuality(p Plan) []QualityFinding {
	var findings []QualityFinding
	if len(strings.Fields(p.Purpose)) > purposeWordTarget {
		findings = append(findings, qualityWarning("plan.purpose", "purpose_over_length_target",
			fmt.Sprintf("purpose is %d words (target <= %d); keep it an abstract — if it restates Problem or Outcome, delete it here", len(strings.Fields(p.Purpose)), purposeWordTarget)))
	}
	strategy := normalizeForNearDuplicate(p.ValidationStrategy)
	for _, phase := range p.Phases {
		validation := normalizeForNearDuplicate(phase.Validation)
		if nearDuplicate(validation, strategy) {
			findings = append(findings, qualityWarning(PhaseQualityLocation(phase)+".validation", "phase_validation_duplicates_strategy",
				"phase validation restates the global validation strategy; phases should state only their delta"))
		}
	}
	dodLines := normalizedDoDLines(p.DefinitionOfDone)
	for _, phase := range p.Phases {
		acceptance := normalizeForNearDuplicate(phase.Acceptance)
		if len(acceptance) < singleHomeDuplicateMinLength/2 {
			continue
		}
		for _, line := range dodLines {
			if line == acceptance {
				findings = append(findings, qualityWarning("plan.definition_of_done", "dod_restates_phase_acceptance",
					"a definition-of-done item restates the acceptance of "+PhaseQualityLocation(phase)+"; DoD carries plan-level gates only"))
				break
			}
		}
	}
	return findings
}

// normalizeForNearDuplicate lowercases, strips list/checkbox markers and
// punctuation-adjacent whitespace so cosmetic differences don't hide a
// restated fact.
func normalizeForNearDuplicate(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "- ")
	s = strings.TrimPrefix(s, "[ ] ")
	s = strings.TrimPrefix(s, "[x] ")
	return strings.Join(strings.Fields(s), " ")
}

// nearDuplicate reports whether two normalized strings restate each other:
// equal, or one contains the other (both long enough to be meaningful).
func nearDuplicate(a, b string) bool {
	if len(a) < singleHomeDuplicateMinLength || len(b) < singleHomeDuplicateMinLength {
		return false
	}
	return a == b || strings.Contains(a, b) || strings.Contains(b, a)
}

func normalizedDoDLines(dod string) []string {
	var out []string
	for _, line := range strings.Split(dod, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "[ ]")
		line = strings.TrimPrefix(line, "[x]")
		if normalized := normalizeForNearDuplicate(line); normalized != "" {
			out = append(out, normalized)
		}
	}
	return out
}

func assessPlanPhasePresence(p Plan) []QualityFinding {
	if len(p.Phases) > 0 {
		return nil
	}
	return []QualityFinding{qualityFailure("plan.phases", "plan_missing_phases", "plan has no phases to execute")}
}

func assessPhaseQuality(phase Phase) []QualityFinding {
	location := PhaseQualityLocation(phase)
	var findings []QualityFinding
	if strings.TrimSpace(phase.Title) == "" {
		findings = append(findings, qualityFailure(location+".title", "phase_missing_title", "phase has no title"))
	}
	if strings.TrimSpace(phase.Intent) == "" {
		findings = append(findings, qualityFailure(location+".intent", "phase_missing_intent", "phase has no intent"))
	}
	if len(phase.Steps) == 0 {
		findings = append(findings, qualityFailure(location+".steps", "phase_missing_steps", "phase has no ordered implementation steps"))
	}
	if strings.TrimSpace(phase.Validation) == "" {
		findings = append(findings, qualityFailure(location+".validation", "phase_missing_validation", "phase has no validation method"))
	}
	if strings.TrimSpace(phase.Acceptance) == "" {
		findings = append(findings, qualityFailure(location+".acceptance", "phase_missing_acceptance", "phase has no objective acceptance outcome"))
	}
	if !HasPhaseReferenceOrNoCodeReason(phase) {
		findings = append(findings, qualityFailure(location+".references", "phase_missing_references", "phase has no connected references and no NO_CODE_REFS reason"))
	}
	if !HasPhaseContextOrNoContextReason(phase) {
		findings = append(findings, qualityFailure(location+".relevant_context", "phase_missing_context", "phase has no relevant context and no NO_CONTEXT reason"))
	}
	findings = append(findings, assessContextQuality(location+".relevant_context", phase.RelevantContext)...)
	return findings
}

func anchorExecutionGrade(anchor RegressionAnchor) bool {
	strategy := strings.TrimSpace(anchor.Strategy)
	if strategy == AnchorStrategyLegacyProse {
		return false
	}
	if anchor.Unavailable && strings.TrimSpace(anchor.CaptureReason) == "" && strings.TrimSpace(anchor.Fallback) == "" {
		return false
	}
	if strategy == AnchorStrategyChangeBoundary {
		return true
	}
	return strings.TrimSpace(anchor.Scenario) != "" ||
		strings.TrimSpace(anchor.BaselineName) != "" ||
		strings.TrimSpace(anchor.HeadSha) != "" ||
		len(anchor.AllowlistPaths) > 0 ||
		len(anchor.Commands) > 0
}

func assessContextQuality(location string, items []RelevantContextItem) []QualityFinding {
	var findings []QualityFinding
	for i, item := range items {
		itemLocation := fmt.Sprintf("%s[%d]", location, i)
		// A doubled `prompt-manager skill read` prefix renders an unrunnable
		// command, whatever path wrote the item — check every source, not just
		// migrated items (the historical corruption arrived via authored
		// submissions too).
		if contextHasDuplicatedSkillRead(item) {
			findings = append(findings, qualityFailure(itemLocation, "context_duplicated_skill_read", "skill setup command repeats the prompt-manager skill read prefix and is not runnable"))
		}
		if item.Source != RelevantContextSourceMigrated {
			continue
		}
		if migratedContextLooksLikeMarkdownFence(item) {
			findings = append(findings, qualityFailure(itemLocation, "migrated_context_markdown_fence", "migrated setup context contains markdown fence text instead of a usable setup item"))
		}
		if migratedContextHasDuplicatedSed(item) {
			findings = append(findings, qualityFailure(itemLocation, "migrated_context_malformed_sed", "migrated doc setup command repeats sed invocation"))
		}
	}
	return findings
}

func migratedContextLooksLikeMarkdownFence(item RelevantContextItem) bool {
	for _, value := range []string{item.Label, item.Target, item.Command, item.Instruction} {
		if strings.Contains(value, "```") {
			return true
		}
	}
	return false
}

func migratedContextHasDuplicatedSed(item RelevantContextItem) bool {
	command := strings.TrimSpace(item.Command)
	if command == "" && len(item.Argv) > 0 {
		command = strings.Join(item.Argv, " ")
	}
	return strings.Count(command, "sed -n") > 1
}

// contextHasDuplicatedSkillRead reports whether a context item would produce
// (or already carries) a doubled `prompt-manager skill read` prefix: a stored
// command with the prefix literally repeated is corruption, and a skill item
// whose Target contains the prefix at all is malformed — Target must be a bare
// skill slug, since a command-assembling consumer prefixes it.
func contextHasDuplicatedSkillRead(item RelevantContextItem) bool {
	const prefix = "prompt-manager skill read"
	command := strings.TrimSpace(item.Command)
	if command == "" && len(item.Argv) > 0 {
		command = strings.Join(item.Argv, " ")
	}
	if strings.Count(command, prefix) > 1 {
		return true
	}
	return item.Kind == RelevantContextSkill && strings.Contains(item.Target, prefix)
}

func PhaseQualityLocation(phase Phase) string {
	if strings.TrimSpace(phase.ID) != "" {
		return "phase." + phase.ID
	}
	if phase.Order > 0 {
		return fmt.Sprintf("phase.%d", phase.Order)
	}
	return "phase"
}

func qualityFailure(location, code, message string) QualityFinding {
	return QualityFinding{
		Severity: QualitySeverityFail,
		Code:     code,
		Location: location,
		Message:  message,
		Guidance: "Populate the structured phase/setup fields during authoring or repair the imported plan before execution.",
	}
}

func qualityWarning(location, code, message string) QualityFinding {
	return QualityFinding{
		Severity: QualitySeverityWarning,
		Code:     code,
		Location: location,
		Message:  message,
		Guidance: "Run plan-manager validate run and repair any execution-grade gaps before handing this plan to an implementation agent.",
	}
}
