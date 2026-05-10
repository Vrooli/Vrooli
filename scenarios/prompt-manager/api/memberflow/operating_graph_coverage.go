package memberflow

import "sort"

type OperatingGraphCoverage struct {
	GraphID       string                          `json:"graph_id"`
	Team          string                          `json:"team"`
	Source        OperatingGraphSource            `json:"source"`
	Relationships []OperatingRelationshipCoverage `json:"relationships"`
	Prompts       OperatingPromptCoverage         `json:"prompts"`
	Docs          OperatingDocsCoverage           `json:"docs"`
	Exclusions    []OperatingCoverageExclusion    `json:"exclusions"`
}

type OperatingRelationshipCoverage struct {
	Relationship       string                                 `json:"relationship"`
	RuntimeDeclared    int                                    `json:"runtime_declared"`
	GraphShown         int                                    `json:"graph_shown"`
	Matched            int                                    `json:"matched"`
	GraphOnly          int                                    `json:"graph_only"`
	RuntimeOnly        int                                    `json:"runtime_only"`
	RuntimeSubtypes    []OperatingRelationshipSubtypeCoverage `json:"runtime_subtypes,omitempty"`
	ValidationRule     string                                 `json:"validation_rule,omitempty"`
	ValidationSeverity string                                 `json:"validation_severity,omitempty"`
	DiffRelationship   string                                 `json:"diff_relationship,omitempty"`
}

type OperatingRelationshipSubtypeCoverage struct {
	Relationship    string `json:"relationship"`
	RuntimeDeclared int    `json:"runtime_declared"`
	Covered         int    `json:"covered"`
	RuntimeOnly     int    `json:"runtime_only"`
}

type OperatingPromptCoverage struct {
	GraphMembers               int                     `json:"graph_members"`
	TopicContractPresent       int                     `json:"topic_contract_present"`
	TopicContractSourceMatched int                     `json:"topic_contract_source_matched"`
	TopicContractContentParity OperatingCoverageStatus `json:"topic_contract_content_parity"`
	TopicContractSourceKind    string                  `json:"topic_contract_source_kind,omitempty"`
}

type OperatingDocsCoverage struct {
	MermaidGraph                      OperatingCoverageStatus `json:"mermaid_graph"`
	RequiredSectionsPresent           int                     `json:"required_sections_present"`
	RequiredSectionsTotal             int                     `json:"required_sections_total"`
	TopicCatalogTable                 OperatingCoverageStatus `json:"topic_catalog_table"`
	TopicCatalogRows                  int                     `json:"topic_catalog_rows"`
	TopicCatalogMatched               int                     `json:"topic_catalog_matched"`
	TopicCatalogGraphOnly             int                     `json:"topic_catalog_graph_only"`
	TopicCatalogDocsOnly              int                     `json:"topic_catalog_docs_only"`
	TopicCatalogInvalid               int                     `json:"topic_catalog_invalid"`
	TopicCatalogPurposeMatched        int                     `json:"topic_catalog_purpose_matched"`
	TopicCatalogPurposeMismatch       int                     `json:"topic_catalog_purpose_mismatch"`
	TopicCatalogPurposeMissingRuntime int                     `json:"topic_catalog_purpose_missing_runtime"`
	DecisionsTable                    OperatingCoverageStatus `json:"decisions_table"`
	DecisionsRows                     int                     `json:"decisions_rows"`
	DecisionsMatched                  int                     `json:"decisions_matched"`
	DecisionsGraphOnly                int                     `json:"decisions_graph_only"`
	DecisionsDocsOnly                 int                     `json:"decisions_docs_only"`
	DecisionsInvalid                  int                     `json:"decisions_invalid"`
	DecisionsMetadataComplete         int                     `json:"decisions_metadata_complete"`
	DecisionsMetadataIncomplete       int                     `json:"decisions_metadata_incomplete"`
	DecisionsAcceptedEffectWeak       int                     `json:"decisions_accepted_effect_weak"`
	ExternalInputsTable               OperatingCoverageStatus `json:"external_inputs_table"`
	ExternalInputsRows                int                     `json:"external_inputs_rows"`
	ExternalInputsBackedRows          int                     `json:"external_inputs_backed_rows"`
	ExternalInputsUnbackedRows        int                     `json:"external_inputs_unbacked_rows"`
	OutputsTable                      OperatingCoverageStatus `json:"outputs_table"`
	OutputsRows                       int                     `json:"outputs_rows"`
	OutputsBackedRows                 int                     `json:"outputs_backed_rows"`
	OutputsUnbackedRows               int                     `json:"outputs_unbacked_rows"`
	FeedbackSteps                     int                     `json:"feedback_steps"`
	FeedbackAnchoredSteps             int                     `json:"feedback_anchored_steps"`
	FeedbackUnbackedReferences        int                     `json:"feedback_unbacked_references"`
	GapsItems                         int                     `json:"gaps_items"`
	GapsAnchoredItems                 int                     `json:"gaps_anchored_items"`
	GapsTargetStateItems              int                     `json:"gaps_target_state_items"`
	AdoptionValidationCommands        int                     `json:"adoption_validation_commands"`
	PlanOfRecordRegistration          OperatingCoverageStatus `json:"plan_of_record_registration"`
	ReadmeDiscoverability             OperatingCoverageStatus `json:"readme_discoverability"`
}

type OperatingCoverageExclusion struct {
	Kind   string `json:"kind"`
	Count  int    `json:"count"`
	Detail string `json:"detail,omitempty"`
}

func BuildOperatingGraphCoverage(blocks []OperatingGraphBlock, runtime OperatingGraphRuntime, teamFilter, idFilter string) []OperatingGraphCoverage {
	coverage := []OperatingGraphCoverage{}
	for _, block := range filterOperatingGraphBlocks(blocks, teamFilter, idFilter) {
		if block.Metadata.Mode == OperatingGraphModeExplanatory {
			continue
		}
		ctx := NewOperatingGraphContractContext(block, runtime)
		coverage = append(coverage, OperatingGraphCoverage{
			GraphID:       block.Metadata.ID,
			Team:          block.Metadata.Team,
			Source:        block.Source,
			Relationships: buildOperatingRelationshipCoverage(ctx),
			Prompts:       buildOperatingPromptCoverage(ctx),
			Docs:          buildOperatingDocsCoverage(ctx),
			Exclusions:    buildOperatingCoverageExclusions(block),
		})
	}
	return coverage
}

func BuildOperatingModelCoverage(models []OperatingModelDocument, runtime OperatingGraphRuntime, teamFilter, idFilter string) []OperatingGraphCoverage {
	filtered := filterOperatingModelDocuments(models, teamFilter, idFilter)
	coverage := BuildOperatingGraphCoverage(operatingGraphBlocksFromModels(filtered), runtime, "", "")
	modelsByKey := map[string]OperatingModelDocument{}
	for _, model := range filtered {
		modelsByKey[operatingModelCoverageKey(model.Team, model.ID)] = model
	}
	for i := range coverage {
		model, ok := modelsByKey[operatingModelCoverageKey(coverage[i].Team, coverage[i].GraphID)]
		if !ok {
			continue
		}
		addOperatingModelDocsCoverage(&coverage[i].Docs, model, runtime)
	}
	return coverage
}

func buildOperatingRelationshipCoverage(ctx OperatingGraphContractContext) []OperatingRelationshipCoverage {
	registry := DefaultOperatingRelationshipRegistry()
	specs := registry.CoverageSpecs()
	out := make([]OperatingRelationshipCoverage, 0, len(specs))
	for _, spec := range specs {
		graphRels := relationshipsByGraphKind(ctx.Index.GraphRelationships.All(), spec.Kind)
		runtimeRels := spec.RuntimeCoverageRelationships(ctx)
		graphOnly := countGraphOnlyRelationships(ctx, graphRels)
		runtimeOnly := countRuntimeOnlyRelationships(ctx, runtimeRels)
		out = append(out, OperatingRelationshipCoverage{
			Relationship:       string(spec.Kind),
			RuntimeDeclared:    len(runtimeRels),
			GraphShown:         len(graphRels),
			Matched:            len(graphRels) - graphOnly,
			GraphOnly:          graphOnly,
			RuntimeOnly:        runtimeOnly,
			RuntimeSubtypes:    buildOperatingRelationshipSubtypeCoverage(ctx, spec),
			ValidationRule:     spec.ValidationRule(),
			ValidationSeverity: string(spec.ValidationSeverity),
			DiffRelationship:   coverageDiffRelationship(spec),
		})
	}
	return out
}

func buildOperatingRelationshipSubtypeCoverage(ctx OperatingGraphContractContext, spec OperatingRelationshipSpec) []OperatingRelationshipSubtypeCoverage {
	if !spec.SubtypeCoverage {
		return nil
	}
	out := make([]OperatingRelationshipSubtypeCoverage, 0, len(spec.RuntimeKinds))
	for _, kind := range spec.RuntimeKinds {
		runtimeRels := ctx.Index.RuntimeRelationships.ByKind(kind)
		runtimeOnly := countRuntimeOnlyRelationships(ctx, runtimeRels)
		out = append(out, OperatingRelationshipSubtypeCoverage{
			Relationship:    string(kind),
			RuntimeDeclared: len(runtimeRels),
			Covered:         len(runtimeRels) - runtimeOnly,
			RuntimeOnly:     runtimeOnly,
		})
	}
	return out
}

func coverageDiffRelationship(spec OperatingRelationshipSpec) string {
	if !spec.DiffIncluded {
		return ""
	}
	return string(spec.Kind)
}

func relationshipsByGraphKind(rels []OperatingRelationship, kind OperatingRelationshipKind) []OperatingRelationship {
	var out []OperatingRelationship
	for _, rel := range rels {
		if rel.Kind == kind {
			out = append(out, rel)
		}
	}
	return out
}

func countGraphOnlyRelationships(ctx OperatingGraphContractContext, rels []OperatingRelationship) int {
	var count int
	for _, rel := range rels {
		if !ctx.Matcher.GraphBackedByRuntime(rel, ctx.Index.RuntimeRelationships) {
			count++
		}
	}
	return count
}

func countRuntimeOnlyRelationships(ctx OperatingGraphContractContext, rels []OperatingRelationship) int {
	var count int
	for _, rel := range rels {
		if !ctx.Matcher.RuntimeShownInGraph(rel, ctx.Index.GraphRelationships) {
			count++
		}
	}
	return count
}

func buildOperatingPromptCoverage(ctx OperatingGraphContractContext) OperatingPromptCoverage {
	coverage := OperatingPromptCoverage{TopicContractContentParity: OperatingCoverageStatusUnavailable}
	contentChecked := 0
	contentMismatch := 0
	liveSections := 0
	derivedSections := 0
	for _, node := range ctx.Block.Graph.Nodes {
		if node.Kind != OperatingGraphNodeKindMember {
			continue
		}
		coverage.GraphMembers++
		section, ok := topicContractPromptSection(ctx.Runtime, ctx.Block.Metadata.Team, node.Value)
		if !ok {
			continue
		}
		coverage.TopicContractPresent++
		if promptSectionIsLive(section) {
			liveSections++
		} else if section.SourceKind == OperatingGraphPromptSectionSourceDerived {
			derivedSections++
		}
		if section.SourcePath == expectedTopicContractSourcePath(ctx.Block.Metadata.Team, node.Value) {
			coverage.TopicContractSourceMatched++
		}
		if !promptSectionIsLive(section) {
			continue
		}
		expected, ok := expectedTopicContractContent(ctx.Runtime, ctx.Block.Metadata.Team, node.Value)
		if !ok || normalizePromptSectionContent(section.Content) == "" {
			continue
		}
		contentChecked++
		if normalizePromptSectionContent(section.Content) != normalizePromptSectionContent(expected) {
			contentMismatch++
		}
	}
	if contentChecked > 0 && contentChecked == coverage.GraphMembers && contentMismatch == 0 {
		coverage.TopicContractContentParity = OperatingCoverageStatusEnforced
	}
	if contentMismatch > 0 {
		coverage.TopicContractContentParity = OperatingCoverageStatusMismatch
	}
	switch {
	case liveSections > 0 && liveSections == coverage.TopicContractPresent:
		coverage.TopicContractSourceKind = string(OperatingGraphPromptSectionSourceLive)
	case derivedSections > 0 && derivedSections == coverage.TopicContractPresent:
		coverage.TopicContractSourceKind = string(OperatingGraphPromptSectionSourceDerived)
	case coverage.TopicContractPresent > 0:
		coverage.TopicContractSourceKind = "mixed"
	}
	return coverage
}

func buildOperatingDocsCoverage(ctx OperatingGraphContractContext) OperatingDocsCoverage {
	block := ctx.Block
	docs := OperatingDocsCoverage{
		MermaidGraph:      OperatingCoverageStatusReferenceOnly,
		TopicCatalogTable: OperatingCoverageStatusNotImplemented,
		DecisionsTable:    OperatingCoverageStatusNotImplemented,
	}
	if block.Metadata.Mode == OperatingGraphModeContract || block.Metadata.Mode == OperatingGraphModeCheckable {
		docs.MermaidGraph = OperatingCoverageStatusEnforced
	}
	docs.TopicCatalogTable = docsTableStatus(block.Docs.TopicCatalog.Present)
	docs.DecisionsTable = docsTableStatus(block.Docs.Decisions.Present)
	docs.TopicCatalogRows, docs.TopicCatalogMatched, docs.TopicCatalogGraphOnly, docs.TopicCatalogDocsOnly, docs.TopicCatalogInvalid = topicCatalogCoverageCounts(block)
	docs.TopicCatalogPurposeMatched, docs.TopicCatalogPurposeMismatch, docs.TopicCatalogPurposeMissingRuntime = topicCatalogPurposeCoverageCounts(block, ctx.Runtime.Contracts[block.Metadata.Team])
	docs.DecisionsRows, docs.DecisionsMatched, docs.DecisionsGraphOnly, docs.DecisionsDocsOnly, docs.DecisionsInvalid = decisionTableCoverageCounts(block)
	docs.DecisionsMetadataComplete, docs.DecisionsMetadataIncomplete, docs.DecisionsAcceptedEffectWeak = decisionMetadataCoverageCounts(block)
	return docs
}

func addOperatingModelDocsCoverage(docs *OperatingDocsCoverage, model OperatingModelDocument, runtime OperatingGraphRuntime) {
	references := NewOperatingModelReferenceIndex(model, runtime)
	required := requiredOperatingModelSections(model)
	docs.RequiredSectionsTotal = len(required)
	for _, section := range required {
		if section.section.Present {
			docs.RequiredSectionsPresent++
		}
	}
	docs.ExternalInputsTable = docsTableStatus(model.Sections.ExternalInputs.Table)
	docs.ExternalInputsRows = len(model.Sections.ExternalInputs.Rows)
	docs.ExternalInputsBackedRows, docs.ExternalInputsUnbackedRows = externalInputsCoverageCounts(references)
	docs.OutputsTable = docsTableStatus(model.Sections.Outputs.Table)
	docs.OutputsRows = len(model.Sections.Outputs.Rows)
	docs.OutputsBackedRows, docs.OutputsUnbackedRows = outputsCoverageCounts(references)
	docs.FeedbackSteps, docs.FeedbackAnchoredSteps, docs.FeedbackUnbackedReferences = feedbackLoopCoverageCounts(references)
	docs.GapsItems, docs.GapsAnchoredItems, docs.GapsTargetStateItems = gapsCoverageCounts(model)
	docs.AdoptionValidationCommands = countOperatingModelValidationCommands(model)
	if runtime.Contracts.HasPlanOfRecordPath(model.Team, model.Source.Path) {
		docs.PlanOfRecordRegistration = OperatingCoverageStatusEnforced
	} else {
		docs.PlanOfRecordRegistration = OperatingCoverageStatusMissing
	}
	readmePath := operatingModelTeamReadmePath(model.Source.Path)
	if readmePath == "" {
		docs.ReadmeDiscoverability = OperatingCoverageStatusUnavailable
	} else if operatingModelReadmeLinksModel(runtime.RepoRoot, readmePath, model.Source.Path) {
		docs.ReadmeDiscoverability = OperatingCoverageStatusEnforced
	} else {
		docs.ReadmeDiscoverability = OperatingCoverageStatusMissing
	}
}

func operatingModelCoverageKey(team, id string) string {
	return team + "\x00" + id
}

func countOperatingModelValidationCommands(model OperatingModelDocument) int {
	seen := map[string]bool{}
	for _, command := range model.Sections.Adoption.Commands {
		verb, ok := operatingModelValidationCommandVerb(command.Command, model.Team, model.ID)
		if ok {
			seen[verb] = true
		}
	}
	return len(seen)
}

func feedbackLoopCoverageCounts(references OperatingModelReferenceIndex) (steps, anchored, unbacked int) {
	for _, assurance := range references.Feedback {
		steps++
		if assurance.Anchored {
			anchored++
		}
		for _, ref := range assurance.References {
			if !ref.Backed {
				unbacked++
			}
		}
	}
	return
}

func externalInputsCoverageCounts(references OperatingModelReferenceIndex) (backed, unbacked int) {
	for _, assurance := range references.ExternalInputs {
		if assurance.Backed() {
			backed++
		} else {
			unbacked++
		}
	}
	return
}

func outputsCoverageCounts(references OperatingModelReferenceIndex) (backed, unbacked int) {
	for _, assurance := range references.Outputs {
		if assurance.Backed() {
			backed++
		} else {
			unbacked++
		}
	}
	return
}

func gapsCoverageCounts(model OperatingModelDocument) (items, anchored, targetState int) {
	for _, item := range model.Sections.Gaps.Items {
		items++
		if len(item.References) > 0 {
			anchored++
		}
		if item.TargetState {
			targetState++
		}
	}
	return
}

func docsTableStatus(present bool) OperatingCoverageStatus {
	if present {
		return OperatingCoverageStatusEnforced
	}
	return OperatingCoverageStatusMissing
}

func topicCatalogCoverageCounts(block OperatingGraphBlock) (rows, matched, graphOnly, docsOnly, invalid int) {
	graphTopics := map[string]bool{}
	for _, node := range block.Graph.Nodes {
		if node.Kind != OperatingGraphNodeKindTopic || node.Qualifier == OperatingGraphQualifierOld || node.Qualifier == OperatingGraphQualifierExternal {
			continue
		}
		graphTopics[qualifiedTopicKey(string(node.Qualifier), node.Value)] = true
	}
	docTopics := map[string]bool{}
	for _, row := range block.Docs.TopicCatalog.Rows {
		rows++
		if row.Topic == "" {
			invalid++
			continue
		}
		docTopics[qualifiedTopicKey(row.Qualifier, row.Topic)] = true
	}
	for key := range graphTopics {
		if docTopics[key] {
			matched++
		} else {
			graphOnly++
		}
	}
	for key := range docTopics {
		if !graphTopics[key] {
			docsOnly++
		}
	}
	return
}

func topicCatalogPurposeCoverageCounts(block OperatingGraphBlock, contract *LoadedTeamContract) (matched, mismatch, missingRuntime int) {
	catalog := topicCatalogByQualifiedTopic(contract)
	for _, row := range block.Docs.TopicCatalog.Rows {
		if row.Topic == "" {
			continue
		}
		entry, ok := catalog[qualifiedTopicKey(row.Qualifier, row.Topic)]
		if !ok {
			if operatingTopicCatalogStatusIsCurrent(row.StatusKind) {
				missingRuntime++
			}
			continue
		}
		if normalizeCatalogPurpose(row.Purpose) == normalizeCatalogPurpose(entry.Purpose) {
			matched++
		} else {
			mismatch++
		}
	}
	return
}

func decisionTableCoverageCounts(block OperatingGraphBlock) (rows, matched, graphOnly, docsOnly, invalid int) {
	graphDecisions := map[string]bool{}
	for _, node := range block.Graph.Nodes {
		if node.Kind == OperatingGraphNodeKindDecision {
			graphDecisions[node.Value] = true
		}
	}
	docDecisions := map[string]bool{}
	for _, row := range block.Docs.Decisions.Rows {
		rows++
		if row.Decision == "" {
			invalid++
			continue
		}
		docDecisions[row.Decision] = true
	}
	for decision := range graphDecisions {
		if docDecisions[decision] {
			matched++
		} else {
			graphOnly++
		}
	}
	for decision := range docDecisions {
		if !graphDecisions[decision] {
			docsOnly++
		}
	}
	return
}

func decisionMetadataCoverageCounts(block OperatingGraphBlock) (complete, incomplete, effectWeak int) {
	for _, row := range block.Docs.Decisions.Rows {
		if len(missingDecisionFields(row)) > 0 {
			incomplete++
			continue
		}
		complete++
		if !acceptedEffectNamesDownstreamSurface(row.AcceptedEffect) {
			effectWeak++
		}
	}
	return
}

func buildOperatingCoverageExclusions(block OperatingGraphBlock) []OperatingCoverageExclusion {
	counts := map[string]int{}
	for _, node := range block.Graph.Nodes {
		kind := operatingCoverageExclusionKind(node)
		if kind == "" {
			continue
		}
		counts[kind]++
	}
	for _, edge := range block.Graph.Edges {
		from, fok := operatingGraphNodeByID(block.Graph.Nodes, edge.From)
		to, tok := operatingGraphNodeByID(block.Graph.Nodes, edge.To)
		if !fok || !tok || operatingGraphEdgeActionable(from, to) {
			continue
		}
		counts["non_actionable_edges"]++
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]OperatingCoverageExclusion, 0, len(keys))
	for _, key := range keys {
		out = append(out, OperatingCoverageExclusion{Kind: key, Count: counts[key], Detail: operatingCoverageExclusionDetail(key)})
	}
	return out
}

func operatingCoverageExclusionKind(node OperatingGraphNode) string {
	switch {
	case node.Kind == OperatingGraphNodeKindProcess:
		return "process_nodes"
	case node.Kind == OperatingGraphNodeKindFuture:
		return "future_nodes"
	case node.Kind == OperatingGraphNodeKindTopic && node.Qualifier == OperatingGraphQualifierFuture:
		return "future_topic_nodes"
	case node.Kind == OperatingGraphNodeKindTopic && node.Qualifier == OperatingGraphQualifierOld:
		return "old_topic_nodes"
	case node.Kind == OperatingGraphNodeKindTopic && node.Qualifier == OperatingGraphQualifierExternal:
		return "external_topic_nodes"
	default:
		return ""
	}
}

func operatingCoverageExclusionDetail(kind string) string {
	switch kind {
	case "process_nodes":
		return "process nodes explain workflow shape but do not map to runtime declarations"
	case "future_nodes":
		return "future nodes are target-state placeholders and are excluded from runtime completeness"
	case "future_topic_nodes":
		return "future topic nodes are target-state placeholders and are excluded from runtime completeness"
	case "old_topic_nodes":
		return "old topic nodes document transitional surfaces and are excluded from runtime completeness"
	case "external_topic_nodes":
		return "external topic nodes document outside surfaces and are excluded from runtime completeness"
	case "non_actionable_edges":
		return "edges touching non-actionable nodes are excluded from runtime relationship matching"
	default:
		return ""
	}
}

func operatingGraphNodeByID(nodes []OperatingGraphNode, id string) (OperatingGraphNode, bool) {
	for _, node := range nodes {
		if node.ID == id {
			return node, true
		}
	}
	return OperatingGraphNode{}, false
}
