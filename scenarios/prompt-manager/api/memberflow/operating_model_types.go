package memberflow

type OperatingModelSource struct {
	Path string `json:"path"`
	Line int    `json:"line"`
}

type OperatingMarkdownSection struct {
	Heading    string   `json:"heading,omitempty"`
	Line       int      `json:"line,omitempty"`
	EndLine    int      `json:"end_line,omitempty"`
	Body       []string `json:"body,omitempty"`
	Present    bool     `json:"present,omitempty"`
	Duplicates []int    `json:"duplicates,omitempty"`
}

type OperatingGraphSection struct {
	OperatingGraphBlock
	Heading string `json:"heading,omitempty"`
	Present bool   `json:"present,omitempty"`
}

type OperatingModelSections struct {
	Mission        OperatingMarkdownSection     `json:"mission,omitempty"`
	Scope          OperatingMarkdownSection     `json:"scope,omitempty"`
	OperatingLoops OperatingMarkdownSection     `json:"operating_loops,omitempty"`
	Graph          OperatingGraphSection        `json:"graph,omitempty"`
	TopicCatalog   OperatingTopicCatalogTable   `json:"topic_catalog,omitempty"`
	ExternalInputs OperatingExternalInputsTable `json:"external_inputs,omitempty"`
	Outputs        OperatingOutputsTable        `json:"outputs,omitempty"`
	FeedbackLoop   OperatingFeedbackSection     `json:"feedback_loop,omitempty"`
	Gaps           OperatingGapsSection         `json:"gaps,omitempty"`
	Adoption       OperatingAdoptionSection     `json:"adoption,omitempty"`
}

type OperatingModelDocument struct {
	ID       string                 `json:"id"`
	Team     string                 `json:"team"`
	Status   string                 `json:"status,omitempty"`
	Source   OperatingModelSource   `json:"source"`
	Sections OperatingModelSections `json:"sections"`
	Graphs   []OperatingGraphBlock  `json:"graphs,omitempty"`
}

type OperatingModelFilter struct {
	Team string
	ID   string
}

type OperatingModelListResponse struct {
	Models []OperatingModelDocument `json:"models"`
}

type OperatingModelValidationResponse struct {
	Models     []OperatingModelDocument       `json:"models"`
	Validation OperatingGraphValidationResult `json:"validation"`
}

type OperatingModelDiffResponse struct {
	Models []OperatingModelDocument     `json:"models"`
	Diff   []OperatingGraphContractDiff `json:"diff"`
}

type OperatingModelCoverageResponse struct {
	Models   []OperatingModelDocument `json:"models"`
	Coverage []OperatingGraphCoverage `json:"coverage"`
}

type OperatingExternalInputsTable struct {
	OperatingMarkdownSection
	HeaderLine int                         `json:"header_line,omitempty"`
	Headers    []string                    `json:"headers,omitempty"`
	Rows       []OperatingExternalInputRow `json:"rows,omitempty"`
	Table      bool                        `json:"table,omitempty"`
}

type OperatingExternalInputRow struct {
	ProducerTrigger string `json:"producer_trigger"`
	EntrySurface    string `json:"entry_surface"`
	Drainer         string `json:"drainer"`
	RoutingRule     string `json:"routing_rule"`
	SourceLine      int    `json:"source_line"`
}

type OperatingOutputsTable struct {
	OperatingMarkdownSection
	HeaderLine int                  `json:"header_line,omitempty"`
	Headers    []string             `json:"headers,omitempty"`
	Rows       []OperatingOutputRow `json:"rows,omitempty"`
	Table      bool                 `json:"table,omitempty"`
}

type OperatingOutputRow struct {
	Output     string `json:"output"`
	Surface    string `json:"surface"`
	Consumer   string `json:"consumer"`
	Purpose    string `json:"purpose"`
	SourceLine int    `json:"source_line"`
}

type OperatingAdoptionSection struct {
	OperatingMarkdownSection
	Commands []OperatingAdoptionCommand `json:"commands,omitempty"`
}

type OperatingAdoptionCommand struct {
	Command    string `json:"command"`
	SourceLine int    `json:"source_line"`
}

type OperatingFeedbackSection struct {
	OperatingMarkdownSection
	Steps []OperatingFeedbackStep `json:"steps,omitempty"`
}

type OperatingFeedbackStep struct {
	Text       string   `json:"text"`
	References []string `json:"references,omitempty"`
	SourceLine int      `json:"source_line"`
}

type OperatingGapsSection struct {
	OperatingMarkdownSection
	Items []OperatingGapItem `json:"items,omitempty"`
}

type OperatingGapItem struct {
	Text       string   `json:"text"`
	References []string `json:"references,omitempty"`
	SourceLine int      `json:"source_line"`
}
