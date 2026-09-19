package interfacegraph

import "context"

const (
	EvidenceProtoImport = "proto_import"
	EvidenceGoImport    = "go_import"

	DriftUndeclaredUsed       = "undeclared-but-used"
	DriftDeclaredWithoutProof = "declared-without-import-evidence"

	SeverityWarning = "WARNING"
	SeverityInfo    = "INFO"
)

type ProtoSurfaceClient interface {
	DescribeScenariosProtos(ctx context.Context, req ProtoSurfaceRequest) (*ProtoSurfaceResponse, error)
}

type ImportFactsClient interface {
	DescribeFleetImports(ctx context.Context, req ImportFactsRequest) (*ImportFactsResponse, error)
}

type ProtoSurfaceRequest struct {
	Scenarios       []string
	Limit           int32
	StabilityFilter string
}

type ImportFactsRequest struct {
	Scenarios      []string
	Limit          int32
	RepoRoot       string
	LanguageFilter []string
	UseCache       bool
}

type ProtoSurfaceResponse struct {
	Results []ProtoSurfaceResult
}

type ProtoSurfaceResult struct {
	Scenario       string
	Surface        ProtoSurface
	Error          string
	TransportWorld string
}

type ProtoSurface struct {
	Scenario             string
	Files                []ProtoFile
	CrossScenarioImports []ProtoImport
	TransportWorld       string
}

type ProtoFile struct {
	Path      string
	Stability string
}

type ProtoImport struct {
	FromFile     string
	ToFile       string
	FromScenario string
	ToScenario   string
	FromPackage  string
	ToPackage    string
}

type ImportFactsResponse struct {
	Results []ImportFactsResult
}

type ImportFactsResult struct {
	Scenario string
	Facts    []ImportFact
	Error    string
}

type ImportFact struct {
	ImportPath string
	Path       string
	Language   string
	Analyzer   string
}

type Graph struct {
	Nodes  []Node  `json:"nodes"`
	Edges  []Edge  `json:"edges"`
	Errors []Error `json:"errors,omitempty"`
}

type Node struct {
	Scenario string `json:"scenario"`
}

type Edge struct {
	FromScenario   string     `json:"from_scenario"`
	ToScenario     string     `json:"to_scenario"`
	Evidence       []Evidence `json:"evidence"`
	TransportWorld string     `json:"transport_world,omitempty"`
	Stability      []string   `json:"stability,omitempty"`
}

type Evidence struct {
	Source     string `json:"source"`
	ImportPath string `json:"import_path,omitempty"`
	FromFile   string `json:"from_file,omitempty"`
	ToFile     string `json:"to_file,omitempty"`
	Path       string `json:"path,omitempty"`
	Analyzer   string `json:"analyzer,omitempty"`
}

type Error struct {
	Source   string `json:"source"`
	Scenario string `json:"scenario,omitempty"`
	Message  string `json:"message"`
}

type DriftReport struct {
	Graph     Graph          `json:"graph"`
	Findings  []DriftFinding `json:"findings"`
	Scenarios []string       `json:"scenarios"`
}

type DriftFinding struct {
	Scenario       string     `json:"scenario"`
	Dependency     string     `json:"dependency"`
	Kind           string     `json:"kind"`
	Severity       string     `json:"severity"`
	Message        string     `json:"message"`
	Evidence       []Evidence `json:"evidence,omitempty"`
	Declared       bool       `json:"declared"`
	ActualEvidence bool       `json:"actual_evidence"`
}
