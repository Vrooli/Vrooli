// Package aisearch is architecture-cartographer's thin consumer of the shared
// retrieval engine in packages/ai-go/search. It owns only the cartographer-
// specific concerns: turning the DERIVED domain map of every scenario into an
// indexable corpus (domain_index.go — the Source adapter, embedding-text
// composition, and result projection) and the search/reindex orchestration
// surface the Connect handlers call (service.go). The index/store/reconcile
// core — embedding, the Qdrant vector store, drift reconciliation, the sync
// loop, and env config — lives in github.com/vrooli/ai-go/search.
//
// Corpus design (the headline goal): retrieval must be TERM-AGNOSTIC. A caller
// who does not know Vrooli's vocabulary asks "how does authoring work in
// plan-manager?" and gets the `plan-manager/authoring` domain back — matched on
// the domain's responsibility/purpose PROSE, never on the literal word
// "domain". So the embedding text folds scenario + name + responsibility +
// purpose + owns-data + glossary (the natural-language identity), while the
// structured facets used only for filtering/projection (archetype, surfaces,
// paths, authority) ride along as Meta and are NOT embedded as prose.
package aisearch

// DefaultCollection is the Qdrant collection architecture-cartographer indexes
// its per-scenario domains into. Single named-dense vector layout (the shared
// engine's CollectionSpec). Fresh — the cartographer had no Qdrant substrate
// before this provider, so there is no legacy collection to stay compatible
// with.
const DefaultCollection = "architecture-cartographer-domains"

// domainKind is the logical collection the domain records belong to.
const domainKind = "domain"

// idPrefix namespaces cartographer point IDs inside the shared engine so its
// natural keys never collide with another consumer's in a shared collection.
const idPrefix = "architecture-cartographer:"

// DomainRecord is the canonical view of a single scenario domain — the unit
// that gets embedded, stored, and returned by Search. It is the cartographer's
// projection of internal/domains.DerivedDomain, flattened so embedding-text
// composition, ContentHash, and result projection are pure and testable.
type DomainRecord struct {
	// ID is the stable leaf id "<scenario>/<name>" (e.g. "plan-manager/authoring").
	ID string `json:"id"`
	// Scenario owns the domain (e.g. "plan-manager"). A metadata facet.
	Scenario string `json:"scenario"`
	// Name is the domain name (e.g. "authoring").
	Name string `json:"name"`
	// Responsibility is the human-authored semantic anchor — the primary
	// retrieval signal and the snippet shown to callers.
	Responsibility string `json:"responsibility,omitempty"`
	// Purpose captures why the capability exists for users/operators.
	Purpose string `json:"purpose,omitempty"`
	// OwnsData is the authored data-ownership note.
	OwnsData string `json:"ownsData,omitempty"`
	// Glossary is the optional canonical vocabulary (type/function names).
	Glossary []string `json:"glossary,omitempty"`
	// Archetype is the primary archetype (reporting / service / mutation / …).
	// A metadata facet (filter/projection), NOT embedded as prose.
	Archetype string `json:"archetype,omitempty"`
	// Surfaces are the code-evidence surfaces. A metadata facet.
	Surfaces []string `json:"surfaces,omitempty"`
	// Paths are the repo-relative path prefixes/globs the domain owns.
	Paths []string `json:"paths,omitempty"`
	// Authority is the ladder rung that defined this domain (a metadata facet
	// carried for provenance, e.g. "domains_doc").
	Authority string `json:"authority,omitempty"`
	// Confidence is the authority confidence ("high" | "low" | "missing").
	Confidence string `json:"confidence,omitempty"`
}

// SearchMode tags how Search should retrieve results. Kept local (not
// pkg.SearchMode, which has no "ai" member) so the cartographer's wire vocab is
// ai/text, mapped to the engine's dense/text in service.go.
type SearchMode string

const (
	ModeAuto SearchMode = "auto" // prefer ai, fall back to text
	ModeAI   SearchMode = "ai"   // ai only; error if unavailable
	ModeText SearchMode = "text" // text only; never embed
)

// DomainHit is the per-result projection returned by Service.Search.
type DomainHit struct {
	ID             string   `json:"id"`
	Scenario       string   `json:"scenario"`
	Name           string   `json:"name"`
	Responsibility string   `json:"responsibility,omitempty"`
	Purpose        string   `json:"purpose,omitempty"`
	Archetype      string   `json:"archetype,omitempty"`
	Paths          []string `json:"paths,omitempty"`
	Score          float64  `json:"score"`
	ScorePercent   int      `json:"scorePercent"`
	// Weak is the regime-aware weak-match flag, computed once in Service.Search
	// via the shared engine's regime-aware weak labeling (pkg.LabelWeakForMethod).
	Weak bool `json:"weak"`
}

// SearchResponse wraps results with the request echo + retrieval method.
type SearchResponse struct {
	Results  []DomainHit `json:"results"`
	Total    int         `json:"total"`
	Query    string      `json:"query"`
	Method   string      `json:"method"`             // "ai" | "text"
	Reranker string      `json:"reranker,omitempty"` // active reranker leg or "none"
}

// StatusReport describes search backend availability.
type StatusReport struct {
	Available            bool   `json:"available"`
	Ollama               bool   `json:"ollama"`
	Qdrant               bool   `json:"qdrant"`
	IndexedCount         int    `json:"indexedCount"`
	LastReconcileAt      string `json:"lastReconcileAt,omitempty"`
	LastReconcileOutcome string `json:"lastReconcileOutcome,omitempty"`
	Reranker             string `json:"reranker,omitempty"`
}
