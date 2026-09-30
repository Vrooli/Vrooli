// Package aisearch projects Offer Desk's authoritative catalog into bounded
// source records for shared search. It never invents a second catalog: every
// record is derived from catalog reads and carries a stable id, a content
// revision, a kind, a visibility, and a canonical follow-up reference.
//
// The projection is deliberately transport- and engine-neutral. A later
// reconciler can map Record into the shared ai-go/search SourceDoc shape and a
// handler can serve the same records directly; neither path needs to re-derive
// catalog semantics.
package aisearch

import "time"

// Visibility classes for catalog source records. Offer Desk is a single-tenant
// operator catalog, so every record is operator-visible. The field exists so a
// projection never silently widens or narrows what a reader may see.
const VisibilityOperator = "operator"

// Record kinds. Node records are namespaced as "node-<lowercase kind>" so the
// eight catalog node kinds stay distinguishable in one corpus.
const (
	KindEdge       = "edge"
	KindTrigger    = "trigger"
	KindFact       = "fact"
	KindEvaluation = "evaluation"
	KindProposal   = "proposal"
	KindAudit      = "audit"
	// KindDecisionContext is one owner-generated aggregate record per node that
	// carries decision state. Ordinary search cannot reconstruct "why is this
	// offer a candidate" from separate trigger, fact, evaluation and proposal
	// rows, so the projection states the current status and its recorded inputs
	// in one place.
	KindDecisionContext = "decision-context"
)

// Freshness labels for time-bounded records. A record with no freshness
// semantics leaves the field empty.
const (
	FreshnessFresh   = "fresh"
	FreshnessStale   = "stale"
	FreshnessUnknown = "unknown"
)

// Record is one bounded, searchable projection of a catalog record.
type Record struct {
	// ID is stable across reads: "node:<uuid>", "fact:<name>", etc.
	ID string
	// Kind distinguishes node kinds, relationships, and decision records.
	Kind string
	// Revision is a content hash. Re-running the projection over unchanged
	// data yields the same revision, so a reader can tell when content moved.
	Revision string
	// Visibility is the reader class permitted to see this record.
	Visibility string
	// FollowUp is the canonical owner reference an agent follows to read the
	// authoritative record.
	FollowUp string
	Title    string
	Snippet  string
	// Body is the text a search engine or lexical ranker should match.
	Body string
	// Freshness is fresh/stale/unknown for facts; empty otherwise.
	Freshness string
	// Historical marks a superseded evaluation, an audit row, or any record
	// that records what happened rather than what is currently true.
	Historical bool
	// Metadata carries the typed fields a result reader needs without a second
	// owner round-trip.
	Metadata map[string]any
}

// Snapshot is a point-in-time materialization of the catalog corpus. Its
// Generation changes only when record content changes, and MaterializedAt is
// the newest observed catalog mutation — never the read time. An unchanged
// query therefore cannot advance either value.
type Snapshot struct {
	Records        []Record
	Generation     string
	MaterializedAt time.Time
}

// IndexedCount is the number of projected records.
func (s Snapshot) IndexedCount() int { return len(s.Records) }
