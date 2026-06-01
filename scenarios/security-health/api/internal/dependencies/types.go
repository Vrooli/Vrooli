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
	Scenario    string    `json:"scenario"`
	Ecosystem   Ecosystem `json:"ecosystem"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	SourceFile  string    `json:"source_file"`
	VulnIDs     []string  `json:"vuln_ids"`
	MaxSeverity string    `json:"max_severity"`
	LastSeen    string    `json:"last_seen"`
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

// Status reports backend availability and the last reconcile state.
type Status struct {
	Available            bool
	Ollama               bool
	Qdrant               bool
	IndexedCount         int
	VulnerableCount      int
	LastReconcileAt      string
	LastReconcileOutcome string
}

const (
	// DefaultSearchLimit is applied when a request omits Limit.
	DefaultSearchLimit = 20
	// MaxSearchLimit clamps the requested limit.
	MaxSearchLimit = 200
)
