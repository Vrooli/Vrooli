// Package dependencies implements the fleet Dependency & Vulnerability
// Intelligence domain — security-health's justifying feature. It walks every
// scenario's lockfiles (go.mod, pnpm-lock.yaml) into a SQLite-backed SBOM
// corpus, annotates each record with known-vuln status from osv-scanner, and
// serves structured + text queries so "which scenarios are exposed to CVE-X?"
// is a single call. A 5-minute reconcile loop keeps the corpus fresh.
//
// The SQLite corpus is the source of truth and powers deterministic TEXT and
// structured-filter search (always available). An optional semantic (AI) layer
// over Qdrant+Ollama can rank the same corpus when those resources are up; it
// degrades to TEXT when they are not.
package dependencies

// Ecosystem identifies the package manager a dependency belongs to.
type Ecosystem string

const (
	EcosystemUnspecified Ecosystem = ""
	EcosystemGo          Ecosystem = "go"
	EcosystemNPM         Ecosystem = "npm"
)

// Mode selects the retrieval strategy for a Search.
type Mode string

const (
	ModeUnspecified Mode = ""
	ModeAI          Mode = "ai"
	ModeText        Mode = "text"
)

// DependencyRecord is the indexed unit: one resolved dependency of one
// scenario, annotated with any known vulnerabilities. Mirrors the proto
// DependencyRecord message.
type DependencyRecord struct {
	Scenario        string                `json:"scenario"`
	Ecosystem       Ecosystem             `json:"ecosystem"`
	Name            string                `json:"name"`
	Version         string                `json:"version"`
	SourceFile      string                `json:"source_file"`
	VulnIDs         []string              `json:"vuln_ids"`
	MaxSeverity     string                `json:"max_severity"`
	LastSeen        string                `json:"last_seen"`
	Vulnerabilities []VulnerabilityRecord `json:"vulnerabilities,omitempty"`
}

// Key is the stable identity of a record across reconciles: a dependency is
// the (scenario, ecosystem, name, version) tuple. Re-observing it updates
// vuln annotations + last_seen rather than inserting a duplicate.
func (r DependencyRecord) Key() string {
	return string(r.Ecosystem) + "|" + r.Scenario + "|" + r.Name + "|" + r.Version
}

// Vulnerable reports whether the record carries any known vulnerability.
func (r DependencyRecord) Vulnerable() bool { return len(r.VulnIDs) > 0 }

// SearchResult is one ranked hit.
type SearchResult struct {
	Record DependencyRecord `json:"record"`
	Score  float64          `json:"score"`
}

// SearchRequest composes a free-text query with structured filters.
type SearchRequest struct {
	Query          string
	Limit          int
	Mode           Mode
	Ecosystem      Ecosystem
	VulnerableOnly bool
	NameGlob       string
}

// SearchResponse echoes which mode actually served the request.
type SearchResponse struct {
	Results  []SearchResult
	ModeUsed Mode
}

// VulnerabilitySource identifies the scanner/source that produced the evidence.
type VulnerabilitySource string

const (
	VulnerabilitySourceUnspecified VulnerabilitySource = ""
	VulnerabilitySourceOSV         VulnerabilitySource = "osv"
	VulnerabilitySourceGovulncheck VulnerabilitySource = "govulncheck"
	VulnerabilitySourcePnpmAudit   VulnerabilitySource = "pnpm-audit"
)

// Reachability captures how precisely Security Health knows the vulnerable code
// is used.
type Reachability string

const (
	ReachabilityUnspecified      Reachability = ""
	ReachabilityUnknown          Reachability = "unknown"
	ReachabilityLockfileAffected Reachability = "lockfile_affected"
	ReachabilityReachable        Reachability = "reachable"
)

// EvidenceConfidence tells downstream policy whether evidence should gate.
type EvidenceConfidence string

const (
	EvidenceConfidenceUnspecified EvidenceConfidence = ""
	EvidenceConfidenceDegraded    EvidenceConfidence = "degraded"
	EvidenceConfidenceAdvisory    EvidenceConfidence = "advisory"
	EvidenceConfidenceGating      EvidenceConfidence = "gating"
)

type AffectedVersionRange struct {
	Range        string `json:"range"`
	Introduced   string `json:"introduced"`
	Fixed        string `json:"fixed"`
	LastAffected string `json:"last_affected"`
}

type FixedVersionRange struct {
	Range   string `json:"range"`
	Version string `json:"version"`
}

type VulnerabilityRecord struct {
	VulnerabilityID    string                 `json:"vulnerability_id"`
	Aliases            []string               `json:"aliases"`
	Ecosystem          Ecosystem              `json:"ecosystem"`
	Name               string                 `json:"name"`
	Version            string                 `json:"version"`
	AffectedRanges     []AffectedVersionRange `json:"affected_ranges"`
	FixedRanges        []FixedVersionRange    `json:"fixed_ranges"`
	Severity           string                 `json:"severity"`
	NormalizedSeverity string                 `json:"normalized_severity"`
	AdvisoryURL        string                 `json:"advisory_url"`
	Summary            string                 `json:"summary"`
	Details            string                 `json:"details"`
	Source             VulnerabilitySource    `json:"source"`
	Reachability       Reachability           `json:"reachability"`
	Confidence         EvidenceConfidence     `json:"confidence"`
	Production         bool                   `json:"production"`
	DevOnly            bool                   `json:"dev_only"`
	FirstSeen          string                 `json:"first_seen"`
	LastSeen           string                 `json:"last_seen"`
	Scenarios          []string               `json:"scenarios"`
	SourceFiles        []string               `json:"source_files"`
	Remediation        string                 `json:"remediation"`
}

type VulnerabilityQuery struct {
	Ecosystem         Ecosystem
	PackageName       string
	Scenario          string
	VulnerabilityID   string
	MinimumConfidence EvidenceConfidence
	Limit             int
}

type VulnerabilityList struct {
	Vulnerabilities []VulnerabilityRecord
	Total           int
}

// Status reports backend availability and the last reconcile state.
type Status struct {
	Available            bool
	Ollama               bool
	Qdrant               bool
	IndexedCount         int
	VulnerableCount      int
	LastReconcileAt      string
	LastReconcileOutcome string
	// Vector-index coverage (Plan B). IndexedVectors / ExpectedVectors. These
	// describe the Qdrant vector index specifically; IndexedCount stays the
	// SQLite row count.
	IndexedVectors  int
	ExpectedVectors int
	// IndexReady is true when coverage ≥ threshold, i.e. AI mode is served;
	// false ⇒ search degrades to TEXT until the backfill catches up.
	IndexReady bool
}

const (
	// DefaultSearchLimit is applied when a request omits Limit.
	DefaultSearchLimit = 20
	// MaxSearchLimit clamps the requested limit.
	MaxSearchLimit = 200
)
