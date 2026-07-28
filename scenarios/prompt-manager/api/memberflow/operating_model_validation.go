package memberflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ValidateOperatingModels(models []OperatingModelDocument, runtime OperatingGraphRuntime, teamFilter, idFilter string) OperatingGraphValidationResult {
	filtered := filterOperatingModelDocuments(models, teamFilter, idFilter)
	result := ValidateOperatingGraphs(operatingGraphBlocksFromModels(filtered), runtime, "", "")
	registry, err := DefaultRuleRegistry()
	if err != nil {
		addOperatingFindings(&result, []OperatingGraphFinding{{
			Rule:     "rule_registry_invalid",
			Severity: string(SeverityError),
			Detail:   err.Error(),
		}})
		sortOperatingFindings(result.Findings)
		return result
	}
	for _, model := range filtered {
		modelContext := NewOperatingModelRuleContext(model, runtime)
		// Plan-of-record validation is a corpus operation with filesystem I/O.
		// Compute it once per model and let the granular registry entries select
		// their findings from that shared result. Individual rule identities must
		// not turn one validation pass into N repeated disk walks.
		planOfRecordFindings := ValidatePlanOfRecordManifestsForModels([]OperatingModelDocument{model}, runtime)
		ctx := RuleContext{ModelContext: &modelContext, PlanOfRecordFindings: planOfRecordFindings}
		for _, rule := range registry.RulesForPass(RulePassModel) {
			if rule.AppliesTo(ctx) {
				addOperatingFindings(&result, rule.Check(ctx))
			}
		}
	}
	sortOperatingFindings(result.Findings)
	return result
}

type requiredOperatingModelSection struct {
	heading string
	section OperatingMarkdownSection
}

func requiredOperatingModelSections(model OperatingModelDocument) []requiredOperatingModelSection {
	graph := OperatingMarkdownSection{Heading: "Operating Graph", Present: model.Sections.Graph.Present && model.Sections.Graph.Heading == "Operating Graph"}
	return []requiredOperatingModelSection{
		{heading: "Mission", section: model.Sections.Mission},
		{heading: "Scope", section: model.Sections.Scope},
		{heading: "Operating Loops", section: model.Sections.OperatingLoops},
		{heading: "Operating Graph", section: graph},
		{heading: "Topic Catalog", section: tableBackedMarkdownSection("Topic Catalog", model.Sections.TopicCatalog.Present, model.Sections.TopicCatalog.HeaderLine)},
		{heading: "Decisions", section: tableBackedMarkdownSection("Decisions", model.Sections.Decisions.Present, model.Sections.Decisions.HeaderLine)},
		{heading: "External Inputs / Triggers", section: model.Sections.ExternalInputs.OperatingMarkdownSection},
		{heading: "Outputs / Downstream Consumers", section: model.Sections.Outputs.OperatingMarkdownSection},
		{heading: "Feedback / Capability Improvement Loop", section: model.Sections.FeedbackLoop.OperatingMarkdownSection},
		{heading: "Current Implementation Gaps", section: model.Sections.Gaps.OperatingMarkdownSection},
		{heading: "Adoption / Validation", section: model.Sections.Adoption.OperatingMarkdownSection},
	}
}

func checkOperatingModelRequiredSectionMissing(ctx OperatingModelRuleContext) []OperatingGraphFinding {
	var findings []OperatingGraphFinding
	for _, required := range requiredOperatingModelSections(ctx.Model) {
		if required.section.Present {
			continue
		}
		findings = append(findings, OperatingGraphFinding{
			Rule:       "operating_model_required_section_missing",
			Severity:   string(SeverityError),
			GraphID:    ctx.Model.ID,
			Team:       ctx.Model.Team,
			SourcePath: ctx.Model.Source.Path,
			Line:       ctx.Model.Source.Line,
			Detail:     fmt.Sprintf("operating model is missing required section ## %s", required.heading),
		})
	}
	return findings
}

func checkOperatingModelDuplicateSection(ctx OperatingModelRuleContext) []OperatingGraphFinding {
	var findings []OperatingGraphFinding
	for _, required := range requiredOperatingModelSections(ctx.Model) {
		for _, duplicateLine := range required.section.Duplicates {
			findings = append(findings, OperatingGraphFinding{
				Rule:       "operating_model_duplicate_section",
				Severity:   string(SeverityError),
				GraphID:    ctx.Model.ID,
				Team:       ctx.Model.Team,
				SourcePath: ctx.Model.Source.Path,
				Line:       duplicateLine,
				Detail:     fmt.Sprintf("operating model section %q appears more than once", required.heading),
			})
		}
	}
	return findings
}

func tableBackedMarkdownSection(heading string, present bool, line int) OperatingMarkdownSection {
	return OperatingMarkdownSection{
		Heading: heading,
		Line:    line,
		Present: present,
	}
}

func operatingModelPrimaryGraphMode(model OperatingModelDocument) OperatingGraphMode {
	if len(model.Graphs) == 0 {
		return ""
	}
	return model.Graphs[0].Metadata.Mode
}

func validateOperatingModelDecisions(model OperatingModelDocument) []OperatingGraphFinding {
	return validateDecisionsTable(model)
}

func validateDecisionsTable(model OperatingModelDocument) []OperatingGraphFinding {
	table := model.Sections.Decisions
	if !table.Present {
		return nil
	}
	var findings []OperatingGraphFinding
	want := []string{"decision context", "owner", "purpose", "expected evidence / trigger", "accepted effect"}
	if !sameStringSlice(table.Headers, want) {
		findings = append(findings, operatingModelFinding(model, "operating_model_decisions_header_drift", table.HeaderLine, fmt.Sprintf("Decisions headers must be %q", strings.Join(want, " | "))))
	}
	if len(table.Rows) == 0 {
		findings = append(findings, operatingModelFinding(model, "operating_model_decisions_empty", table.HeaderLine, "Decisions must declare at least one row"))
	}
	for _, row := range table.Rows {
		missing := missingDecisionFields(row)
		if len(missing) > 0 {
			findings = append(findings, operatingModelFinding(model, "operating_model_decisions_row_incomplete", row.SourceLine, fmt.Sprintf("Decisions row %q is missing %s", decisionRowLabel(row), strings.Join(missing, ", "))))
			continue
		}
		if !acceptedEffectNamesDownstreamSurface(row.AcceptedEffect) {
			findings = append(findings, operatingModelFinding(model, "operating_model_decisions_effect_weak", row.SourceLine, fmt.Sprintf("Decisions row %q accepted effect must name a concrete downstream surface", decisionRowLabel(row))))
		}
	}
	return findings
}

func validateOperatingModelExternalInputs(ctx OperatingModelRuleContext) []OperatingGraphFinding {
	return validateExternalInputsTable(ctx)
}

func validateExternalInputsTable(ctx OperatingModelRuleContext) []OperatingGraphFinding {
	model := ctx.Model
	table := model.Sections.ExternalInputs
	if !table.Present {
		return nil
	}
	if !table.Table {
		return []OperatingGraphFinding{operatingModelFinding(model, "operating_model_external_inputs_table_missing", table.Line, "External Inputs / Triggers must contain a markdown table")}
	}
	var findings []OperatingGraphFinding
	want := []string{"producer / trigger", "entry surface", "drainer", "routing rule"}
	if !sameStringSlice(table.Headers, want) {
		findings = append(findings, operatingModelFinding(model, "operating_model_external_inputs_header_drift", table.HeaderLine, fmt.Sprintf("External Inputs / Triggers headers must be %q", strings.Join(want, " | "))))
	}
	if len(table.Rows) == 0 {
		findings = append(findings, operatingModelFinding(model, "operating_model_external_inputs_empty", table.HeaderLine, "External Inputs / Triggers must declare at least one row"))
	}
	for _, assurance := range ctx.ReferenceIndex.ExternalInputs {
		row := assurance.Row
		missing := missingExternalInputFields(row)
		if len(missing) > 0 {
			findings = append(findings, operatingModelFinding(model, "operating_model_external_inputs_row_incomplete", row.SourceLine, fmt.Sprintf("External Inputs / Triggers row is missing %s", strings.Join(missing, ", "))))
			continue
		}
		if !assurance.Producer {
			findings = append(findings, operatingModelFinding(model, "operating_model_external_inputs_producer_unbacked", row.SourceLine, fmt.Sprintf("External Inputs / Triggers row %q names a producer that is not backed by graph/runtime external producer relationships", row.ProducerTrigger)))
		}
		if !assurance.Entry {
			findings = append(findings, operatingModelFinding(model, "operating_model_external_inputs_entry_unbacked", row.SourceLine, fmt.Sprintf("External Inputs / Triggers row %q names an entry surface that is not backed by the graph, Topic Catalog, or Decisions table", row.EntrySurface)))
		}
		if !assurance.Drainer {
			findings = append(findings, operatingModelFinding(model, "operating_model_external_inputs_drainer_unbacked", row.SourceLine, fmt.Sprintf("External Inputs / Triggers row %q names a drainer that is not backed by topic/member, external/member, or cross-team relationships", row.Drainer)))
		}
	}
	return findings
}

func validateOperatingModelOutputs(ctx OperatingModelRuleContext) []OperatingGraphFinding {
	return validateOutputsTable(ctx)
}

func validateOutputsTable(ctx OperatingModelRuleContext) []OperatingGraphFinding {
	model := ctx.Model
	table := model.Sections.Outputs
	if !table.Present {
		return nil
	}
	if !table.Table {
		return []OperatingGraphFinding{operatingModelFinding(model, "operating_model_outputs_table_missing", table.Line, "Outputs / Downstream Consumers must contain a markdown table")}
	}
	var findings []OperatingGraphFinding
	want := []string{"output", "surface", "consumer", "purpose"}
	if !sameStringSlice(table.Headers, want) {
		findings = append(findings, operatingModelFinding(model, "operating_model_outputs_header_drift", table.HeaderLine, fmt.Sprintf("Outputs / Downstream Consumers headers must be %q", strings.Join(want, " | "))))
	}
	if len(table.Rows) == 0 {
		findings = append(findings, operatingModelFinding(model, "operating_model_outputs_empty", table.HeaderLine, "Outputs / Downstream Consumers must declare at least one row"))
	}
	for _, assurance := range ctx.ReferenceIndex.Outputs {
		row := assurance.Row
		missing := missingOutputFields(row)
		if len(missing) > 0 {
			findings = append(findings, operatingModelFinding(model, "operating_model_outputs_row_incomplete", row.SourceLine, fmt.Sprintf("Outputs / Downstream Consumers row is missing %s", strings.Join(missing, ", "))))
			continue
		}
		if !assurance.Surface {
			findings = append(findings, operatingModelFinding(model, "operating_model_outputs_surface_unbacked", row.SourceLine, fmt.Sprintf("Outputs / Downstream Consumers row %q names a surface that is not backed by the graph, Topic Catalog, Decisions table, runtime output, or PoR path", row.Output)))
		}
		if !assurance.Consumer {
			findings = append(findings, operatingModelFinding(model, "operating_model_outputs_consumer_unbacked", row.SourceLine, fmt.Sprintf("Outputs / Downstream Consumers row %q names a consumer that is not backed by topic/member, decision/member, or cross-team output relationships", row.Output)))
		}
	}
	return findings
}

func validateOperatingModelFeedback(ctx OperatingModelRuleContext) []OperatingGraphFinding {
	return validateOperatingModelFeedbackLoop(ctx)
}

func validateOperatingModelFeedbackLoop(ctx OperatingModelRuleContext) []OperatingGraphFinding {
	model := ctx.Model
	section := model.Sections.FeedbackLoop
	if !section.Present {
		return nil
	}
	if len(section.Steps) == 0 {
		return []OperatingGraphFinding{operatingModelFinding(model, "operating_model_feedback_steps_missing", section.Line, "Feedback / Capability Improvement Loop must declare ordered steps")}
	}
	var findings []OperatingGraphFinding
	for _, assurance := range ctx.ReferenceIndex.Feedback {
		step := assurance.Step
		if len(step.References) == 0 {
			findings = append(findings, operatingModelFinding(model, "operating_model_feedback_step_unanchored", step.SourceLine, "Feedback loop step must name at least one concrete topic, decision, member, command, output, or downstream surface"))
			continue
		}
		for _, ref := range assurance.References {
			if ref.Backed {
				continue
			}
			findings = append(findings, operatingModelFinding(model, "operating_model_feedback_reference_unbacked", step.SourceLine, fmt.Sprintf("Feedback loop reference %q is not represented by the graph, topic catalog, decision catalog, external inputs, outputs, or team members", ref.Reference)))
		}
		if !assurance.Anchored {
			findings = append(findings, operatingModelFinding(model, "operating_model_feedback_step_unanchored", step.SourceLine, "Feedback loop step must include at least one backed operating-model surface"))
		}
	}
	return findings
}

func validateOperatingModelGapItems(model OperatingModelDocument) []OperatingGraphFinding {
	return validateOperatingModelGaps(model)
}

func validateOperatingModelGaps(model OperatingModelDocument) []OperatingGraphFinding {
	section := model.Sections.Gaps
	if !section.Present {
		return nil
	}
	if len(section.Items) == 0 {
		return []OperatingGraphFinding{operatingModelFinding(model, "operating_model_gaps_items_missing", section.Line, "Current Implementation Gaps must declare explicit list items")}
	}
	var findings []OperatingGraphFinding
	for _, item := range section.Items {
		if len(item.References) == 0 {
			findings = append(findings, operatingModelFinding(model, "operating_model_gap_item_unanchored", item.SourceLine, "Current Implementation Gaps item must name at least one concrete surface in inline code"))
		}
		if !item.TargetState {
			findings = append(findings, operatingModelFinding(model, "operating_model_gap_item_target_state_missing", item.SourceLine, "Current Implementation Gaps item must state its target-state disposition"))
		}
	}
	return findings
}

func validateOperatingModelAdoptionCommands(model OperatingModelDocument) []OperatingGraphFinding {
	return validateOperatingModelAdoption(model)
}

func validateOperatingModelAdoption(model OperatingModelDocument) []OperatingGraphFinding {
	section := model.Sections.Adoption
	if !section.Present {
		return nil
	}
	required := map[string]bool{"validate": false, "diff": false, "coverage": false}
	var findings []OperatingGraphFinding
	for _, command := range section.Commands {
		verb, ok := operatingModelValidationCommandVerb(command.Command, model.Team, model.ID)
		if !ok {
			continue
		}
		required[verb] = true
	}
	for verb, seen := range required {
		if seen {
			continue
		}
		findings = append(findings, operatingModelFinding(model, "operating_model_adoption_command_missing", section.Line, fmt.Sprintf("Adoption / Validation must include `prompt-manager graph operating-model %s --team %s --id %s`", verb, model.Team, model.ID)))
	}
	return findings
}

func validateOperatingModelDiscoverability(model OperatingModelDocument, runtime OperatingGraphRuntime) []OperatingGraphFinding {
	if len(runtime.Contracts) == 0 && strings.TrimSpace(runtime.RepoRoot) == "" {
		return nil
	}
	var findings []OperatingGraphFinding
	if !runtime.Contracts.HasPlanOfRecordPath(model.Team, model.Source.Path) {
		findings = append(findings, operatingModelFinding(model, "operating_model_plan_of_record_missing", model.Source.Line, fmt.Sprintf("team.json::operatingContract.documents.planOfRecord must register %s", model.Source.Path)))
	}
	readmePath := operatingModelTeamReadmePath(model.Source.Path)
	if readmePath == "" {
		return findings
	}
	if !operatingModelReadmeLinksModel(runtime.RepoRoot, readmePath, model.Source.Path) {
		findings = append(findings, operatingModelFinding(model, "operating_model_readme_link_missing", model.Source.Line, fmt.Sprintf("%s must link to %s", readmePath, model.Source.Path)))
	}
	return findings
}

func operatingModelTeamReadmePath(modelPath string) string {
	if !strings.HasSuffix(modelPath, "/OPERATING_MODEL.md") {
		return ""
	}
	dir := filepath.ToSlash(filepath.Dir(modelPath))
	if filepath.Base(dir) == "operating" {
		parent := filepath.ToSlash(filepath.Dir(dir))
		return parent + "/README.md"
	}
	return strings.TrimSuffix(modelPath, "OPERATING_MODEL.md") + "README.md"
}

func operatingModelReadmeLinksModel(repoRoot, readmePath, modelPath string) bool {
	if strings.TrimSpace(repoRoot) == "" {
		return false
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(readmePath)))
	if err != nil {
		return false
	}
	text := string(data)
	modelBase := filepath.Base(modelPath)
	return strings.Contains(text, modelBase) || strings.Contains(text, modelPath)
}

func operatingModelValidationCommandVerb(command, team, id string) (string, bool) {
	fields := strings.Fields(command)
	if len(fields) < 8 {
		return "", false
	}
	if fields[0] != "prompt-manager" || fields[1] != "graph" || fields[2] != "operating-model" {
		return "", false
	}
	verb := fields[3]
	if verb != "validate" && verb != "diff" && verb != "coverage" {
		return "", false
	}
	if commandFlagValue(fields, "--team") != team || commandFlagValue(fields, "--id") != id {
		return "", false
	}
	return verb, true
}

func commandFlagValue(fields []string, flag string) string {
	for i := 0; i+1 < len(fields); i++ {
		if fields[i] == flag {
			return fields[i+1]
		}
	}
	return ""
}

func missingExternalInputFields(row OperatingExternalInputRow) []string {
	var missing []string
	if strings.TrimSpace(row.ProducerTrigger) == "" {
		missing = append(missing, "producer / trigger")
	}
	if strings.TrimSpace(row.EntrySurface) == "" {
		missing = append(missing, "entry surface")
	}
	if strings.TrimSpace(row.Drainer) == "" {
		missing = append(missing, "drainer")
	}
	if strings.TrimSpace(row.RoutingRule) == "" {
		missing = append(missing, "routing rule")
	}
	return missing
}

func missingOutputFields(row OperatingOutputRow) []string {
	var missing []string
	if strings.TrimSpace(row.Output) == "" {
		missing = append(missing, "output")
	}
	if strings.TrimSpace(row.Surface) == "" {
		missing = append(missing, "surface")
	}
	if strings.TrimSpace(row.Consumer) == "" {
		missing = append(missing, "consumer")
	}
	if strings.TrimSpace(row.Purpose) == "" {
		missing = append(missing, "purpose")
	}
	return missing
}

func missingDecisionFields(row OperatingDecisionRow) []string {
	var missing []string
	if strings.TrimSpace(row.Decision) == "" {
		missing = append(missing, "decision context")
	}
	if len(row.Owners) == 0 {
		missing = append(missing, "owner")
	}
	if strings.TrimSpace(row.Purpose) == "" {
		missing = append(missing, "purpose")
	}
	if strings.TrimSpace(row.ExpectedEvidenceTrigger) == "" {
		missing = append(missing, "expected evidence / trigger")
	}
	if strings.TrimSpace(row.AcceptedEffect) == "" {
		missing = append(missing, "accepted effect")
	}
	return missing
}

func decisionRowLabel(row OperatingDecisionRow) string {
	if strings.TrimSpace(row.Decision) != "" {
		return row.Decision
	}
	if strings.TrimSpace(row.RawDecision) != "" {
		return row.RawDecision
	}
	return "<empty>"
}

func acceptedEffectNamesDownstreamSurface(effect string) bool {
	normalized := strings.ToLower(effect)
	for _, token := range []string{
		"`topic:",
		"`path:",
		"docs/",
		"backlog",
		"team",
		"member",
		"skill",
		"action",
		"scenario",
		"config",
		"canon",
		"plan-of-record",
		"document",
		"documentation",
		"surface",
		"artifact",
		"campaign",
		"channel",
		"gap",
		"framework",
		"decision",
		"publisher",
	} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}

func sameStringSlice(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func operatingModelFinding(model OperatingModelDocument, rule string, line int, detail string) OperatingGraphFinding {
	return OperatingGraphFinding{
		Rule:       rule,
		Severity:   string(SeverityError),
		GraphID:    model.ID,
		Team:       model.Team,
		SourcePath: model.Source.Path,
		Line:       line,
		Detail:     detail,
	}
}
