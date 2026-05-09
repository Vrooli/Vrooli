package graph

type operatingModelDocument struct {
	ID       string                 `json:"id"`
	Team     string                 `json:"team"`
	Status   string                 `json:"status,omitempty"`
	Source   operatingModelSource   `json:"source"`
	Sections operatingModelSections `json:"sections"`
	Graphs   []operatingGraphBlock  `json:"graphs,omitempty"`
}

type operatingModelSource struct {
	Path string `json:"path"`
	Line int    `json:"line"`
}

type operatingModelSections struct {
	Graph        operatingGraphSection      `json:"graph,omitempty"`
	TopicCatalog operatingTopicCatalogTable `json:"topic_catalog,omitempty"`
	Decisions    operatingDecisionTable     `json:"decisions,omitempty"`
}

type operatingGraphSection struct {
	operatingGraphBlock
	Heading string `json:"heading,omitempty"`
	Present bool   `json:"present,omitempty"`
}

type operatingGraphBlock struct {
	Metadata operatingGraphMetadata `json:"metadata"`
	Graph    operatingGraph         `json:"graph"`
	Docs     operatingGraphDocs     `json:"docs,omitempty"`
	Source   operatingGraphSource   `json:"source"`
}

type operatingGraphMetadata struct {
	ID     string            `json:"id"`
	Scope  string            `json:"scope"`
	Team   string            `json:"team"`
	Mode   string            `json:"mode"`
	Status string            `json:"status,omitempty"`
	Extra  map[string]string `json:"extra,omitempty"`
}

type operatingGraph struct {
	ID        string                `json:"id"`
	Direction string                `json:"direction"`
	Nodes     []operatingGraphNode  `json:"nodes"`
	Edges     []operatingGraphEdge  `json:"edges"`
	Groups    []operatingGraphGroup `json:"groups,omitempty"`
}

type operatingGraphSource struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	FenceLine int    `json:"fence_line"`
}

type operatingGraphNode struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Value      string `json:"value"`
	Qualifier  string `json:"qualifier,omitempty"`
	Display    string `json:"display,omitempty"`
	RawLabel   string `json:"raw_label"`
	Shape      string `json:"shape,omitempty"`
	SourceLine int    `json:"source_line"`
	Implicit   bool   `json:"implicit,omitempty"`
}

type operatingGraphGroup struct {
	ID         string   `json:"id"`
	Display    string   `json:"display"`
	SourceLine int      `json:"source_line"`
	NodeIDs    []string `json:"node_ids,omitempty"`
}

type operatingGraphEdge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Label      string `json:"label,omitempty"`
	SourceLine int    `json:"source_line"`
}

type operatingGraphDocs struct {
	TopicCatalog operatingTopicCatalogTable `json:"topic_catalog,omitempty"`
	Decisions    operatingDecisionTable     `json:"decisions,omitempty"`
}

type operatingTopicCatalogTable struct {
	HeaderLine int                        `json:"header_line,omitempty"`
	Rows       []operatingTopicCatalogRow `json:"rows,omitempty"`
	Present    bool                       `json:"present,omitempty"`
}

type operatingTopicCatalogRow struct {
	Topic      string                    `json:"topic"`
	Qualifier  string                    `json:"qualifier,omitempty"`
	Status     string                    `json:"status"`
	Writers    []operatingActorReference `json:"writers,omitempty"`
	Readers    []operatingActorReference `json:"readers,omitempty"`
	Purpose    string                    `json:"purpose"`
	SourceLine int                       `json:"source_line"`
	RawTopic   string                    `json:"raw_topic"`
}

type operatingDecisionTable struct {
	HeaderLine int                    `json:"header_line,omitempty"`
	Rows       []operatingDecisionRow `json:"rows,omitempty"`
	Present    bool                   `json:"present,omitempty"`
}

type operatingDecisionRow struct {
	Decision                string                    `json:"decision"`
	Owners                  []operatingActorReference `json:"owners,omitempty"`
	Purpose                 string                    `json:"purpose"`
	ExpectedEvidenceTrigger string                    `json:"expected_evidence_trigger"`
	AcceptedEffect          string                    `json:"accepted_effect"`
	SourceLine              int                       `json:"source_line"`
	RawDecision             string                    `json:"raw_decision"`
}

type operatingActorReference struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
	Raw   string `json:"raw"`
}

type operatingGraphFinding struct {
	Rule       string `json:"rule"`
	Severity   string `json:"severity"`
	SourcePath string `json:"source_path,omitempty"`
	Line       int    `json:"line,omitempty"`
	GraphID    string `json:"graph_id,omitempty"`
	Team       string `json:"team,omitempty"`
	NodeID     string `json:"node_id,omitempty"`
	Edge       string `json:"edge,omitempty"`
	Member     string `json:"member,omitempty"`
	Topic      string `json:"topic,omitempty"`
	Decision   string `json:"decision,omitempty"`
	Path       string `json:"path,omitempty"`
	Detail     string `json:"detail"`
}

type operatingGraphValidation struct {
	Findings []operatingGraphFinding `json:"findings"`
	Errors   int                     `json:"errors"`
	Warnings int                     `json:"warnings"`
}

type operatingModelListResponse struct {
	Models []operatingModelDocument `json:"models"`
}

type operatingModelValidationResponse struct {
	Models     []operatingModelDocument `json:"models"`
	Validation operatingGraphValidation `json:"validation"`
}

type operatingGraphDiff struct {
	Kind             string   `json:"kind"`
	Relationship     string   `json:"relationship"`
	Team             string   `json:"team"`
	Member           string   `json:"member,omitempty"`
	Topic            string   `json:"topic,omitempty"`
	Decision         string   `json:"decision,omitempty"`
	Path             string   `json:"path,omitempty"`
	External         string   `json:"external,omitempty"`
	TargetTeam       string   `json:"target_team,omitempty"`
	SourcePath       string   `json:"source_path,omitempty"`
	Line             int      `json:"line,omitempty"`
	RuntimePath      string   `json:"runtime_path,omitempty"`
	AcceptableFields []string `json:"acceptable_fields,omitempty"`
	Suggestions      []string `json:"suggestions,omitempty"`
	Detail           string   `json:"detail"`
}

type operatingModelDiffResponse struct {
	Models []operatingModelDocument `json:"models"`
	Diff   []operatingGraphDiff     `json:"diff"`
}

type operatingModelCoverageResponse struct {
	Models   []operatingModelDocument `json:"models"`
	Coverage []operatingGraphCoverage `json:"coverage"`
}

type operatingGraphCoverage struct {
	GraphID       string                          `json:"graph_id"`
	Team          string                          `json:"team"`
	Source        operatingGraphSource            `json:"source"`
	Relationships []operatingRelationshipCoverage `json:"relationships"`
	Prompts       operatingPromptCoverage         `json:"prompts"`
	Docs          operatingDocsCoverage           `json:"docs"`
	Exclusions    []operatingCoverageExclusion    `json:"exclusions"`
}

type operatingRelationshipCoverage struct {
	Relationship       string                                 `json:"relationship"`
	RuntimeDeclared    int                                    `json:"runtime_declared"`
	GraphShown         int                                    `json:"graph_shown"`
	Matched            int                                    `json:"matched"`
	GraphOnly          int                                    `json:"graph_only"`
	RuntimeOnly        int                                    `json:"runtime_only"`
	RuntimeSubtypes    []operatingRelationshipSubtypeCoverage `json:"runtime_subtypes,omitempty"`
	ValidationRule     string                                 `json:"validation_rule,omitempty"`
	ValidationSeverity string                                 `json:"validation_severity,omitempty"`
	DiffRelationship   string                                 `json:"diff_relationship,omitempty"`
}

type operatingRelationshipSubtypeCoverage struct {
	Relationship    string `json:"relationship"`
	RuntimeDeclared int    `json:"runtime_declared"`
	Covered         int    `json:"covered"`
	RuntimeOnly     int    `json:"runtime_only"`
}

type operatingPromptCoverage struct {
	GraphMembers               int    `json:"graph_members"`
	TopicContractPresent       int    `json:"topic_contract_present"`
	TopicContractSourceMatched int    `json:"topic_contract_source_matched"`
	TopicContractContentParity string `json:"topic_contract_content_parity"`
	TopicContractSourceKind    string `json:"topic_contract_source_kind,omitempty"`
}

type operatingDocsCoverage struct {
	MermaidGraph                      string `json:"mermaid_graph"`
	RequiredSectionsPresent           int    `json:"required_sections_present"`
	RequiredSectionsTotal             int    `json:"required_sections_total"`
	TopicCatalogTable                 string `json:"topic_catalog_table"`
	TopicCatalogRows                  int    `json:"topic_catalog_rows"`
	TopicCatalogMatched               int    `json:"topic_catalog_matched"`
	TopicCatalogGraphOnly             int    `json:"topic_catalog_graph_only"`
	TopicCatalogDocsOnly              int    `json:"topic_catalog_docs_only"`
	TopicCatalogInvalid               int    `json:"topic_catalog_invalid"`
	TopicCatalogPurposeMatched        int    `json:"topic_catalog_purpose_matched"`
	TopicCatalogPurposeMismatch       int    `json:"topic_catalog_purpose_mismatch"`
	TopicCatalogPurposeMissingRuntime int    `json:"topic_catalog_purpose_missing_runtime"`
	DecisionsTable                    string `json:"decisions_table"`
	DecisionsRows                     int    `json:"decisions_rows"`
	DecisionsMatched                  int    `json:"decisions_matched"`
	DecisionsGraphOnly                int    `json:"decisions_graph_only"`
	DecisionsDocsOnly                 int    `json:"decisions_docs_only"`
	DecisionsInvalid                  int    `json:"decisions_invalid"`
	DecisionsMetadataComplete         int    `json:"decisions_metadata_complete"`
	DecisionsMetadataIncomplete       int    `json:"decisions_metadata_incomplete"`
	DecisionsAcceptedEffectWeak       int    `json:"decisions_accepted_effect_weak"`
	ExternalInputsTable               string `json:"external_inputs_table"`
	ExternalInputsRows                int    `json:"external_inputs_rows"`
	ExternalInputsBackedRows          int    `json:"external_inputs_backed_rows"`
	ExternalInputsUnbackedRows        int    `json:"external_inputs_unbacked_rows"`
	OutputsTable                      string `json:"outputs_table"`
	OutputsRows                       int    `json:"outputs_rows"`
	OutputsBackedRows                 int    `json:"outputs_backed_rows"`
	OutputsUnbackedRows               int    `json:"outputs_unbacked_rows"`
	FeedbackSteps                     int    `json:"feedback_steps"`
	FeedbackAnchoredSteps             int    `json:"feedback_anchored_steps"`
	FeedbackUnbackedReferences        int    `json:"feedback_unbacked_references"`
	GapsItems                         int    `json:"gaps_items"`
	GapsAnchoredItems                 int    `json:"gaps_anchored_items"`
	GapsTargetStateItems              int    `json:"gaps_target_state_items"`
	AdoptionValidationCommands        int    `json:"adoption_validation_commands"`
	PlanOfRecordRegistration          string `json:"plan_of_record_registration"`
	ReadmeDiscoverability             string `json:"readme_discoverability"`
}

type operatingCoverageExclusion struct {
	Kind   string `json:"kind"`
	Count  int    `json:"count"`
	Detail string `json:"detail,omitempty"`
}
