package memberflow

type OperatingGraphMetadata struct {
	ID     string            `json:"id"`
	Scope  string            `json:"scope"`
	Team   string            `json:"team"`
	Mode   string            `json:"mode"`
	Status string            `json:"status,omitempty"`
	Extra  map[string]string `json:"extra,omitempty"`
}

type OperatingGraphSource struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	FenceLine int    `json:"fence_line"`
}

type OperatingGraphBlock struct {
	Metadata OperatingGraphMetadata `json:"metadata"`
	Graph    OperatingGraph         `json:"graph"`
	Source   OperatingGraphSource   `json:"source"`
}

type OperatingGraph struct {
	ID        string               `json:"id"`
	Direction string               `json:"direction"`
	Nodes     []OperatingGraphNode `json:"nodes"`
	Edges     []OperatingGraphEdge `json:"edges"`
}

type OperatingGraphNode struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Value      string `json:"value"`
	Qualifier  string `json:"qualifier,omitempty"`
	Display    string `json:"display,omitempty"`
	RawLabel   string `json:"raw_label"`
	SourceLine int    `json:"source_line"`
	Implicit   bool   `json:"implicit,omitempty"`
}

type OperatingGraphEdge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Label      string `json:"label,omitempty"`
	SourceLine int    `json:"source_line"`
}

type OperatingGraphFinding struct {
	Rule       string `json:"rule"`
	Severity   string `json:"severity"`
	GraphID    string `json:"graph_id,omitempty"`
	Team       string `json:"team,omitempty"`
	NodeID     string `json:"node_id,omitempty"`
	Edge       string `json:"edge,omitempty"`
	Member     string `json:"member,omitempty"`
	Topic      string `json:"topic,omitempty"`
	Decision   string `json:"decision,omitempty"`
	Path       string `json:"path,omitempty"`
	SourcePath string `json:"source_path,omitempty"`
	Line       int    `json:"line,omitempty"`
	Detail     string `json:"detail"`
}

type OperatingGraphValidationResult struct {
	Findings []OperatingGraphFinding `json:"findings"`
	Errors   int                     `json:"errors"`
	Warnings int                     `json:"warnings"`
}

type OperatingGraphListResponse struct {
	Graphs []OperatingGraphBlock `json:"graphs"`
}

type OperatingGraphValidationResponse struct {
	Graphs     []OperatingGraphBlock          `json:"graphs"`
	Validation OperatingGraphValidationResult `json:"validation"`
}

type OperatingGraphDiffResponse struct {
	Graphs []OperatingGraphBlock        `json:"graphs"`
	Diff   []OperatingGraphContractDiff `json:"diff"`
}

type OperatingGraphContractDiff struct {
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
