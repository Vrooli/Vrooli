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
	addOperatingFindings(&result, validateOperatingModelDocumentStructure(filtered, runtime))
	sortOperatingFindings(result.Findings)
	return result
}

func validateOperatingModelDocumentStructure(models []OperatingModelDocument, runtime OperatingGraphRuntime) []OperatingGraphFinding {
	var findings []OperatingGraphFinding
	for _, model := range models {
		if operatingModelPrimaryGraphMode(model) != OperatingGraphModeContract {
			continue
		}
		for _, required := range requiredOperatingModelSections(model) {
			if required.section.Present {
				for _, duplicateLine := range required.section.Duplicates {
					findings = append(findings, OperatingGraphFinding{
						Rule:       "operating_model_duplicate_section",
						Severity:   string(SeverityError),
						GraphID:    model.ID,
						Team:       model.Team,
						SourcePath: model.Source.Path,
						Line:       duplicateLine,
						Detail:     fmt.Sprintf("operating model section %q appears more than once", required.heading),
					})
				}
				continue
			}
			findings = append(findings, OperatingGraphFinding{
				Rule:       "operating_model_required_section_missing",
				Severity:   string(SeverityError),
				GraphID:    model.ID,
				Team:       model.Team,
				SourcePath: model.Source.Path,
				Line:       model.Source.Line,
				Detail:     fmt.Sprintf("operating model is missing required section ## %s", required.heading),
			})
		}
		findings = append(findings, validateOperatingModelSectionTables(model)...)
		findings = append(findings, validateOperatingModelFeedbackLoop(model)...)
		findings = append(findings, validateOperatingModelGaps(model)...)
		findings = append(findings, validateOperatingModelAdoption(model)...)
		findings = append(findings, validateOperatingModelDiscoverability(model, runtime)...)
	}
	return findings
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

func validateOperatingModelSectionTables(model OperatingModelDocument) []OperatingGraphFinding {
	var findings []OperatingGraphFinding
	findings = append(findings, validateDecisionsTable(model)...)
	findings = append(findings, validateExternalInputsTable(model)...)
	findings = append(findings, validateOutputsTable(model)...)
	return findings
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

func validateExternalInputsTable(model OperatingModelDocument) []OperatingGraphFinding {
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
	for _, row := range table.Rows {
		missing := missingExternalInputFields(row)
		if len(missing) > 0 {
			findings = append(findings, operatingModelFinding(model, "operating_model_external_inputs_row_incomplete", row.SourceLine, fmt.Sprintf("External Inputs / Triggers row is missing %s", strings.Join(missing, ", "))))
			continue
		}
		check := operatingExternalInputBacking(model, row)
		if !check.Producer {
			findings = append(findings, operatingModelFinding(model, "operating_model_external_inputs_producer_unbacked", row.SourceLine, fmt.Sprintf("External Inputs / Triggers row %q names a producer that is not backed by graph/runtime external producer relationships", row.ProducerTrigger)))
		}
		if !check.Entry {
			findings = append(findings, operatingModelFinding(model, "operating_model_external_inputs_entry_unbacked", row.SourceLine, fmt.Sprintf("External Inputs / Triggers row %q names an entry surface that is not backed by the graph, Topic Catalog, or Decisions table", row.EntrySurface)))
		}
		if !check.Drainer {
			findings = append(findings, operatingModelFinding(model, "operating_model_external_inputs_drainer_unbacked", row.SourceLine, fmt.Sprintf("External Inputs / Triggers row %q names a drainer that is not backed by topic/member, external/member, or cross-team relationships", row.Drainer)))
		}
	}
	return findings
}

func validateOutputsTable(model OperatingModelDocument) []OperatingGraphFinding {
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
	for _, row := range table.Rows {
		missing := missingOutputFields(row)
		if len(missing) > 0 {
			findings = append(findings, operatingModelFinding(model, "operating_model_outputs_row_incomplete", row.SourceLine, fmt.Sprintf("Outputs / Downstream Consumers row is missing %s", strings.Join(missing, ", "))))
			continue
		}
		check := operatingOutputBacking(model, row)
		if !check.Surface {
			findings = append(findings, operatingModelFinding(model, "operating_model_outputs_surface_unbacked", row.SourceLine, fmt.Sprintf("Outputs / Downstream Consumers row %q names a surface that is not backed by the graph, Topic Catalog, Decisions table, runtime output, or PoR path", row.Output)))
		}
		if !check.Consumer {
			findings = append(findings, operatingModelFinding(model, "operating_model_outputs_consumer_unbacked", row.SourceLine, fmt.Sprintf("Outputs / Downstream Consumers row %q names a consumer that is not backed by topic/member, decision/member, or cross-team output relationships", row.Output)))
		}
	}
	return findings
}

func validateOperatingModelFeedbackLoop(model OperatingModelDocument) []OperatingGraphFinding {
	section := model.Sections.FeedbackLoop
	if !section.Present {
		return nil
	}
	if len(section.Steps) == 0 {
		return []OperatingGraphFinding{operatingModelFinding(model, "operating_model_feedback_steps_missing", section.Line, "Feedback / Capability Improvement Loop must declare ordered steps")}
	}
	var findings []OperatingGraphFinding
	for _, step := range section.Steps {
		if len(step.References) == 0 {
			findings = append(findings, operatingModelFinding(model, "operating_model_feedback_step_unanchored", step.SourceLine, "Feedback loop step must name at least one concrete topic, decision, member, command, output, or downstream surface"))
			continue
		}
		var backed bool
		for _, ref := range step.References {
			if operatingFeedbackReferenceBacked(model, ref) {
				backed = true
				continue
			}
			findings = append(findings, operatingModelFinding(model, "operating_model_feedback_reference_unbacked", step.SourceLine, fmt.Sprintf("Feedback loop reference %q is not represented by the graph, topic catalog, decision catalog, external inputs, outputs, or team members", ref)))
		}
		if !backed {
			findings = append(findings, operatingModelFinding(model, "operating_model_feedback_step_unanchored", step.SourceLine, "Feedback loop step must include at least one backed operating-model surface"))
		}
	}
	return findings
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

func operatingFeedbackReferenceBacked(model OperatingModelDocument, ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false
	}
	normalized := strings.TrimPrefix(ref, "topic:")
	normalized = strings.TrimPrefix(normalized, "path:")
	if operatingFeedbackGraphNodeBacked(model, ref, normalized) {
		return true
	}
	if operatingFeedbackTopicCatalogBacked(model, normalized) {
		return true
	}
	if operatingFeedbackDecisionBacked(model, normalized) {
		return true
	}
	if operatingFeedbackExternalInputBacked(model, ref, normalized) {
		return true
	}
	if operatingFeedbackOutputBacked(model, ref, normalized) {
		return true
	}
	return false
}

func operatingFeedbackGraphNodeBacked(model OperatingModelDocument, raw, normalized string) bool {
	for _, block := range model.Graphs {
		for _, node := range block.Graph.Nodes {
			if node.Value != normalized && string(node.Kind)+":"+node.Value != raw {
				continue
			}
			switch node.Kind {
			case OperatingGraphNodeKindTopic, OperatingGraphNodeKindDecision, OperatingGraphNodeKindMember, OperatingGraphNodeKindTeam, OperatingGraphNodeKindPOR, OperatingGraphNodeKindExternal:
				return true
			}
		}
	}
	return false
}

func operatingFeedbackTopicCatalogBacked(model OperatingModelDocument, ref string) bool {
	for _, row := range model.Sections.TopicCatalog.Rows {
		if row.Topic == ref {
			return true
		}
	}
	return false
}

func operatingFeedbackDecisionBacked(model OperatingModelDocument, ref string) bool {
	for _, row := range model.Sections.Decisions.Rows {
		if row.Decision == ref {
			return true
		}
	}
	return false
}

func operatingFeedbackExternalInputBacked(model OperatingModelDocument, raw, normalized string) bool {
	for _, row := range model.Sections.ExternalInputs.Rows {
		if operatingFeedbackTextMentions(row.ProducerTrigger, raw, normalized) ||
			operatingFeedbackTextMentions(row.EntrySurface, raw, normalized) ||
			operatingFeedbackTextMentions(row.Drainer, raw, normalized) ||
			operatingFeedbackTextMentions(row.RoutingRule, raw, normalized) {
			return true
		}
	}
	return false
}

func operatingFeedbackOutputBacked(model OperatingModelDocument, raw, normalized string) bool {
	for _, row := range model.Sections.Outputs.Rows {
		if operatingFeedbackTextMentions(row.Output, raw, normalized) ||
			operatingFeedbackTextMentions(row.Surface, raw, normalized) ||
			operatingFeedbackTextMentions(row.Consumer, raw, normalized) ||
			operatingFeedbackTextMentions(row.Purpose, raw, normalized) {
			return true
		}
	}
	return false
}

func operatingFeedbackTextMentions(text, raw, normalized string) bool {
	text = strings.ToLower(text)
	raw = strings.ToLower(raw)
	normalized = strings.ToLower(normalized)
	return strings.Contains(text, raw) || strings.Contains(text, normalized)
}

type operatingExternalInputBackingResult struct {
	Producer bool
	Entry    bool
	Drainer  bool
}

type operatingOutputBackingResult struct {
	Surface  bool
	Consumer bool
}

type operatingSurfaceReference struct {
	Kind      OperatingGraphNodeKind
	Qualifier OperatingGraphQualifier
	Value     string
}

func operatingExternalInputBacking(model OperatingModelDocument, row OperatingExternalInputRow) operatingExternalInputBackingResult {
	if operatingModelRowIsTargetState(row.ProducerTrigger, row.EntrySurface, row.Drainer, row.RoutingRule) {
		return operatingExternalInputBackingResult{Producer: true, Entry: true, Drainer: true}
	}
	producers := operatingModelActorRefs(model, row.ProducerTrigger)
	surfaces := operatingSurfaceReferences(row.EntrySurface)
	return operatingExternalInputBackingResult{
		Producer: operatingExternalInputProducerBacked(model, producers),
		Entry:    operatingEntrySurfaceBacked(model, row.EntrySurface, surfaces),
		Drainer:  operatingExternalInputDrainerBacked(model, producers, surfaces, operatingModelActorRefs(model, row.Drainer), row.Drainer),
	}
}

func operatingOutputBacking(model OperatingModelDocument, row OperatingOutputRow) operatingOutputBackingResult {
	if operatingModelRowIsTargetState(row.Output, row.Surface, row.Consumer, row.Purpose) {
		return operatingOutputBackingResult{Surface: true, Consumer: true}
	}
	surfaces := operatingSurfaceReferences(row.Output + " " + row.Surface)
	surfaceBacked := operatingOutputSurfaceBacked(model, row.Surface, surfaces)
	return operatingOutputBackingResult{
		Surface:  surfaceBacked,
		Consumer: operatingOutputConsumerBacked(model, surfaces, operatingModelActorRefs(model, row.Consumer), row.Consumer) || surfaceBacked && operatingOutputConsumerTextIsDownstream(row.Consumer),
	}
}

func operatingModelActorRefs(model OperatingModelDocument, raw string) []OperatingActorReference {
	if len(model.Graphs) == 0 {
		return nil
	}
	resolver := NewOperatingActorResolver(model.Graphs[0].Metadata, model.Graphs[0].Graph)
	var refs []OperatingActorReference
	for _, token := range extractInlineCodeTokens(raw) {
		refs = append(refs, resolver.Resolve(model.Team, OperatingGraphRuntime{}, token)...)
	}
	refs = append(refs, resolver.Resolve(model.Team, OperatingGraphRuntime{}, raw)...)
	if strings.Contains(strings.ToLower(raw), "operator") {
		refs = append(refs, resolver.Resolve(model.Team, OperatingGraphRuntime{}, "operator")...)
	}
	return dedupeOperatingActorRefs(refs)
}

func dedupeOperatingActorRefs(refs []OperatingActorReference) []OperatingActorReference {
	seen := map[string]bool{}
	var out []OperatingActorReference
	for _, ref := range refs {
		if ref.Kind == "" || ref.Value == "" {
			continue
		}
		key := string(ref.Kind) + "\x00" + ref.Value
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, ref)
	}
	return out
}

func operatingExternalInputProducerBacked(model OperatingModelDocument, producers []OperatingActorReference) bool {
	var externalSeen bool
	for _, producer := range producers {
		if producer.Kind != OperatingActorKindExternal {
			continue
		}
		externalSeen = true
		if operatingGraphHasNode(model, OperatingGraphNodeKindExternal, producer.Value) ||
			operatingGraphHasRelationship(model, func(rel OperatingRelationship) bool {
				return (rel.Kind == operatingRelExternalProducer || rel.Kind == operatingRelExternalProducerIntake) && rel.External == producer.Value
			}) {
			return true
		}
	}
	return !externalSeen
}

func operatingEntrySurfaceBacked(model OperatingModelDocument, raw string, surfaces []operatingSurfaceReference) bool {
	if len(surfaces) == 0 {
		normalized := strings.ToLower(raw)
		return strings.Contains(normalized, "direct member context") || strings.Contains(normalized, "heartbeat trigger")
	}
	for _, surface := range surfaces {
		if operatingSurfaceReferenceIsTargetState(surface) {
			continue
		}
		if operatingSurfaceReferenceBacked(model, surface) {
			return true
		}
	}
	return false
}

func operatingOutputSurfaceBacked(model OperatingModelDocument, raw string, surfaces []operatingSurfaceReference) bool {
	if len(surfaces) == 0 {
		return operatingSurfaceTextMentionsBackedDecision(model, raw) || operatingSurfaceTextMentionsBackedPath(model, raw)
	}
	for _, surface := range surfaces {
		if operatingSurfaceReferenceIsTargetState(surface) {
			continue
		}
		if operatingSurfaceReferenceBacked(model, surface) {
			return true
		}
	}
	return false
}

func operatingExternalInputDrainerBacked(model OperatingModelDocument, producers []OperatingActorReference, surfaces []operatingSurfaceReference, drainers []OperatingActorReference, raw string) bool {
	if len(drainers) == 0 {
		return false
	}
	if strings.Contains(strings.ToLower(raw), "decision owner") && operatingSurfacesContainBackedTopicOrDecision(model, surfaces) {
		return true
	}
	for _, drainer := range drainers {
		if operatingActorExistsInGraph(model, drainer) {
			if len(surfaces) == 0 && strings.Contains(strings.ToLower(raw), "direct member context") {
				return true
			}
		}
		switch drainer.Kind {
		case OperatingActorKindMember:
			if operatingMemberDrainerBacked(model, producers, surfaces, drainer.Value) {
				return true
			}
		case OperatingActorKindTeam:
			if operatingTeamConsumerBacked(model, surfaces, drainer.Value) {
				return true
			}
		case OperatingActorKindExternal:
			if operatingGraphHasNode(model, OperatingGraphNodeKindExternal, drainer.Value) {
				return true
			}
		}
	}
	return false
}

func operatingOutputConsumerBacked(model OperatingModelDocument, surfaces []operatingSurfaceReference, consumers []OperatingActorReference, raw string) bool {
	if len(consumers) == 0 {
		return false
	}
	for _, consumer := range consumers {
		if operatingActorExistsInGraph(model, consumer) {
			return true
		}
		switch consumer.Kind {
		case OperatingActorKindMember:
			if operatingMemberSurfaceConsumerBacked(model, surfaces, consumer.Value) {
				return true
			}
		case OperatingActorKindTeam:
			if operatingTeamConsumerBacked(model, surfaces, consumer.Value) {
				return true
			}
		case OperatingActorKindExternal:
			if operatingGraphHasNode(model, OperatingGraphNodeKindExternal, consumer.Value) {
				return true
			}
		}
	}
	return strings.Contains(strings.ToLower(raw), "decision owner") && operatingSurfacesContainBackedDecision(model, surfaces)
}

func operatingMemberDrainerBacked(model OperatingModelDocument, producers []OperatingActorReference, surfaces []operatingSurfaceReference, member string) bool {
	for _, producer := range producers {
		if producer.Kind == OperatingActorKindExternal && operatingGraphHasRelationship(model, func(rel OperatingRelationship) bool {
			return rel.Kind == operatingRelExternalProducer && rel.External == producer.Value && rel.Member == member
		}) {
			return true
		}
	}
	return operatingMemberSurfaceConsumerBacked(model, surfaces, member)
}

func operatingMemberSurfaceConsumerBacked(model OperatingModelDocument, surfaces []operatingSurfaceReference, member string) bool {
	for _, surface := range surfaces {
		switch surface.Kind {
		case OperatingGraphNodeKindTopic:
			if operatingGraphHasRelationship(model, func(rel OperatingRelationship) bool {
				return rel.Kind == operatingRelTopicRead && rel.Member == member && topicsOverlap(rel.Topic, surface.Value)
			}) {
				return true
			}
		case OperatingGraphNodeKindDecision:
			if operatingGraphHasRelationship(model, func(rel OperatingRelationship) bool {
				return (rel.Kind == operatingRelDecisionConsumed || rel.Kind == operatingRelDecisionOwned || rel.Kind == operatingRelCapabilityGapRaised) && rel.Member == member && operatingDecisionRefsOverlap(rel.Decision, surface.Value)
			}) {
				return true
			}
		}
	}
	return false
}

func operatingTeamConsumerBacked(model OperatingModelDocument, surfaces []operatingSurfaceReference, team string) bool {
	if operatingGraphHasNode(model, OperatingGraphNodeKindTeam, team) {
		return true
	}
	for _, surface := range surfaces {
		if surface.Kind != OperatingGraphNodeKindTopic {
			continue
		}
		if operatingGraphHasRelationship(model, func(rel OperatingRelationship) bool {
			return rel.Kind == operatingRelCrossTeamOutput && rel.TargetTeam == team && topicsOverlap(rel.Topic, surface.Value)
		}) {
			return true
		}
	}
	return false
}

func operatingSurfaceReferenceBacked(model OperatingModelDocument, ref operatingSurfaceReference) bool {
	switch ref.Kind {
	case OperatingGraphNodeKindTopic:
		return operatingGraphHasNode(model, OperatingGraphNodeKindTopic, ref.Value) &&
			operatingTopicCatalogHasTopic(model, ref.Value)
	case OperatingGraphNodeKindDecision:
		return operatingGraphHasNode(model, OperatingGraphNodeKindDecision, ref.Value) &&
			operatingDecisionCatalogHasDecision(model, ref.Value)
	case OperatingGraphNodeKindPOR:
		return operatingGraphHasNode(model, OperatingGraphNodeKindPOR, ref.Value)
	case OperatingGraphNodeKindMember, OperatingGraphNodeKindTeam, OperatingGraphNodeKindExternal:
		return operatingGraphHasNode(model, ref.Kind, ref.Value)
	default:
		return false
	}
}

func operatingSurfaceTextMentionsBackedDecision(model OperatingModelDocument, raw string) bool {
	normalized := strings.ToLower(raw)
	if !strings.Contains(normalized, "decision") {
		return false
	}
	for _, row := range model.Sections.Decisions.Rows {
		if row.Decision != "" && strings.Contains(normalized, strings.ToLower(row.Decision)) {
			return true
		}
	}
	return false
}

func operatingSurfaceTextMentionsBackedPath(model OperatingModelDocument, raw string) bool {
	for _, token := range extractInlineCodeTokens(raw) {
		if strings.HasPrefix(token, "path:") || strings.HasPrefix(token, "docs/") {
			if operatingGraphHasNode(model, OperatingGraphNodeKindPOR, strings.TrimPrefix(token, "path:")) {
				return true
			}
		}
	}
	return false
}

func operatingSurfacesContainBackedDecision(model OperatingModelDocument, surfaces []operatingSurfaceReference) bool {
	for _, surface := range surfaces {
		if surface.Kind == OperatingGraphNodeKindDecision && operatingSurfaceReferenceBacked(model, surface) {
			return true
		}
	}
	return false
}

func operatingSurfacesContainBackedTopicOrDecision(model OperatingModelDocument, surfaces []operatingSurfaceReference) bool {
	for _, surface := range surfaces {
		switch surface.Kind {
		case OperatingGraphNodeKindTopic, OperatingGraphNodeKindDecision:
			if operatingSurfaceReferenceBacked(model, surface) {
				return true
			}
		}
	}
	return false
}

func operatingActorExistsInGraph(model OperatingModelDocument, actor OperatingActorReference) bool {
	switch actor.Kind {
	case OperatingActorKindMember:
		return operatingGraphHasNode(model, OperatingGraphNodeKindMember, actor.Value)
	case OperatingActorKindTeam:
		return operatingGraphHasNode(model, OperatingGraphNodeKindTeam, actor.Value)
	case OperatingActorKindExternal:
		return operatingGraphHasNode(model, OperatingGraphNodeKindExternal, actor.Value)
	case OperatingActorKindProcess:
		return operatingGraphHasNode(model, OperatingGraphNodeKindProcess, actor.Value)
	default:
		return false
	}
}

func operatingGraphHasNode(model OperatingModelDocument, kind OperatingGraphNodeKind, value string) bool {
	for _, block := range model.Graphs {
		for _, node := range block.Graph.Nodes {
			if node.Kind == kind && operatingModelSurfaceValuesOverlap(node.Value, value) {
				return true
			}
		}
	}
	return false
}

func operatingGraphHasRelationship(model OperatingModelDocument, match func(OperatingRelationship) bool) bool {
	for _, block := range model.Graphs {
		for _, rel := range BuildGraphOperatingRelationships(block) {
			if match(rel) {
				return true
			}
		}
	}
	return false
}

func operatingTopicCatalogHasTopic(model OperatingModelDocument, topic string) bool {
	for _, row := range model.Sections.TopicCatalog.Rows {
		if operatingModelSurfaceValuesOverlap(row.Topic, topic) {
			return true
		}
	}
	return false
}

func operatingDecisionCatalogHasDecision(model OperatingModelDocument, decision string) bool {
	for _, row := range model.Sections.Decisions.Rows {
		if operatingDecisionRefsOverlap(row.Decision, decision) {
			return true
		}
	}
	return false
}

func operatingSurfaceReferences(raw string) []operatingSurfaceReference {
	var refs []operatingSurfaceReference
	seen := map[string]bool{}
	add := func(ref operatingSurfaceReference) {
		if ref.Kind == "" || ref.Value == "" {
			return
		}
		key := string(ref.Kind) + "\x00" + string(ref.Qualifier) + "\x00" + ref.Value
		if seen[key] {
			return
		}
		seen[key] = true
		refs = append(refs, ref)
	}
	for _, token := range extractInlineCodeTokens(raw) {
		add(operatingSurfaceReferenceFromToken(token, raw))
	}
	if len(refs) == 0 {
		for _, token := range operatingSurfaceLooseTokens(raw) {
			add(operatingSurfaceReferenceFromToken(token, raw))
		}
	}
	return refs
}

func operatingSurfaceLooseTokens(raw string) []string {
	raw = strings.ReplaceAll(raw, " and ", ",")
	raw = strings.ReplaceAll(raw, " or ", ",")
	parts := strings.Split(raw, ",")
	var out []string
	for _, part := range parts {
		token := strings.TrimSpace(part)
		token = strings.TrimSuffix(token, " topics")
		token = strings.TrimSuffix(token, " topic")
		token = strings.TrimSuffix(token, " decisions")
		token = strings.TrimSuffix(token, " decision")
		if token != "" {
			out = append(out, token)
		}
	}
	return out
}

func operatingSurfaceReferenceFromToken(token, context string) operatingSurfaceReference {
	token = parseInlineCodeToken(token)
	if kind, qualifier, value, ok := parseOperatingGraphTypedToken(token); ok {
		return operatingSurfaceReference{Kind: kind, Qualifier: qualifier, Value: value}
	}
	switch {
	case strings.HasPrefix(token, "docs/"):
		return operatingSurfaceReference{Kind: OperatingGraphNodeKindPOR, Value: token}
	case strings.Contains(strings.ToLower(context), "decision") && !strings.Contains(token, "/"):
		return operatingSurfaceReference{Kind: OperatingGraphNodeKindDecision, Value: token}
	case strings.Contains(token, "/") || strings.Contains(token, "*"):
		return operatingSurfaceReference{Kind: OperatingGraphNodeKindTopic, Value: strings.TrimPrefix(token, "topic:")}
	default:
		return operatingSurfaceReference{}
	}
}

func operatingSurfaceReferenceIsTargetState(ref operatingSurfaceReference) bool {
	return ref.Qualifier == OperatingGraphQualifierFuture || ref.Kind == OperatingGraphNodeKindFuture
}

func operatingModelRowIsTargetState(parts ...string) bool {
	normalized := strings.ToLower(strings.Join(parts, " "))
	return strings.Contains(normalized, "target-state") ||
		strings.Contains(normalized, "target state") ||
		strings.Contains(normalized, "future") ||
		strings.Contains(normalized, "not yet") ||
		strings.Contains(normalized, "until ")
}

func operatingDecisionRefsOverlap(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	if strings.HasSuffix(a, "*") && strings.HasPrefix(b, strings.TrimSuffix(a, "*")) {
		return true
	}
	if strings.HasSuffix(b, "*") && strings.HasPrefix(a, strings.TrimSuffix(b, "*")) {
		return true
	}
	return false
}

func operatingModelSurfaceValuesOverlap(a, b string) bool {
	if a == b {
		return true
	}
	if strings.Contains(a, "/") || strings.Contains(b, "/") {
		return topicsOverlap(a, b)
	}
	return operatingDecisionRefsOverlap(a, b)
}

func operatingOutputConsumerTextIsDownstream(raw string) bool {
	normalized := strings.ToLower(raw)
	for _, token := range []string{"operator", "downstream", "cross-team", "implementation", "owner", "team", "member", "validator", "consumer"} {
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
