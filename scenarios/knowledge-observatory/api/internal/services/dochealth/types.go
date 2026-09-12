package dochealth

// Severity classifies the impact of a DocHealth finding. Values mirror the
// proto enum in packages/proto/schemas/knowledge-observatory/v1/api.proto.
type Severity int

const (
	SeverityUnspecified Severity = iota
	SeverityInfo
	SeverityWarning
	SeverityFailure
)

// Finding is the canonical observation produced by every validator family.
type Finding struct {
	Code     string
	Severity Severity
	Message  string
	Path     string
	DocType  string
	Line     int
	Target   string
	// Fix, when non-empty, is the byte-exact replacement for Target that
	// resolves this finding (carried through from cli-health; applied
	// verbatim by the deterministic placeholder fixer).
	Fix string
}

// MisplacedDoc mirrors docvalidation.MisplacedDoc but uses the typed severity.
type MisplacedDoc struct {
	ActualPath   string
	ExpectedPath string
	Severity     Severity
	DocType      string
	Message      string
}

// MissingDoc mirrors docvalidation.MissingDoc but uses the typed severity.
type MissingDoc struct {
	DocType    string
	Path       string
	Severity   Severity
	Completion string
	RequiredBy []string
}

// Counts mirrors DocHealthCounts in the proto.
type Counts struct {
	FilesChecked     int
	MarkdownWarnings int
	MarkdownFailures int

	LocalLinks       int
	ExternalLinks    int
	BrokenLinks      int
	ExternalWarnings int
	ExternalFailures int

	MermaidValidated int
	MermaidFailures  int

	AbsolutePathHits int
	AbsoluteFailures int

	CodeFilesScanned  int
	CodeRefsFound     int
	CodeRefsBroken    int
	DocRefsFound      int
	DocRefsBroken     int
	MarkedRefsFound   int
	MarkedRefsBroken  int
	MarkedRefsSkipped int
	MarkedRefsUnknown int

	DocsInManifest    int
	DocsNotInManifest int

	// Number / derived-count lint.
	NumbersFlagged int
}

// DocHealthResult is the combined output of every validator family.
type DocHealthResult struct {
	ScenarioName     string
	SourceTemplateID string
	ManifestPath     string
	ManifestStatus   string
	HealthScore      float64
	TotalDocs        int

	MisplacedDocs []MisplacedDoc
	MissingDocs   []MissingDoc
	ExtraDocs     []string
	TemporaryDocs []string

	ContractFindings  []Finding
	ContentFindings   []Finding
	ReferenceFindings []Finding
	ManifestFindings  []Finding

	Counts Counts
}

// DocHealthOptions controls per-call validator behavior. The *bool fields are
// pointers so callers (via the proto request) can leave any of them unset and
// inherit the static server defaults.
type DocHealthOptions struct {
	StrictExternalLinks      *bool
	RequireAllDocsRegistered *bool
	SkipExternalLinks        *bool

	// Target selection. Scope is "" or "scenario" (default; target resolved
	// from the scenario name) or "path" (Path is scanned directly). A path that
	// resolves inside a scenario is promoted to that scenario (all checks);
	// a project-level path runs only generic checks.
	Scope string
	Path  string

	// Checks narrows the run to the named checks (see checkRegistry). Empty
	// means all checks applicable to the target.
	Checks []string
}
