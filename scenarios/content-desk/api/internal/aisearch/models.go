// Package aisearch projects Content Desk's authoritative records into bounded
// source records for shared search. It never invents a second corpus: every
// record is derived from its owning store read (drafts, their current
// authority, publish history, the capability catalog, and campaigns with their
// launch assets) and carries a stable id, a content revision, a kind, a
// visibility, and a canonical follow-up reference.
//
// The projection is deliberately transport- and engine-neutral. A later
// reconciler can map Record into the shared ai-go/search SourceDoc shape and a
// handler can serve the same records directly; neither path needs to re-derive
// owner semantics.
package aisearch

import "time"

// Visibility classes for editorial source records. Content Desk is a
// single-tenant operator surface, so every record is operator-visible. The
// field exists so a projection never silently widens or narrows what a reader
// may see.
const VisibilityOperator = "operator"

// Record kinds. Drafts are live editorial work; publish records are what was
// actually released. Capabilities, campaigns and claims project the current
// marketing state owned by their own domains.
const (
	KindDraft         = "draft"
	KindPublishRecord = "publish-record"
	KindCapability    = "capability"
	KindCampaign      = "campaign"
	KindClaim         = "claim"
	// KindClaimSummary is one owner-generated aggregate over the launch-cited
	// claims: the evidence-backed set that is safe to use and the remaining
	// gaps. It is not a claim itself and never stands in for an unqualified
	// claim's own record.
	KindClaimSummary = "claim-summary"
	// KindWorkSummary is one owner-generated aggregate classifying the remaining
	// marketing work as launch (before release) versus ongoing (capability
	// improvement), with the binding next action. It states the classification a
	// board read derives; it never invents a priority.
	KindWorkSummary = "work-summary"
	// KindCampaignPerformance is one owner-generated aggregate over the retained
	// measurement state for the launch material: the real metric readings when
	// samples exist, or an explicit unavailable verdict when none do. It never
	// renders missing samples as a zero.
	KindCampaignPerformance = "campaign-performance"
)

// Record is one bounded, searchable projection of an editorial record.
type Record struct {
	// ID is stable across reads: "draft:<uuid>", "publish:<uuid>".
	ID string
	// Kind distinguishes live drafts from historical publish records.
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
	// Freshness is fresh/stale/unknown for time-bounded records; empty
	// otherwise. Content Desk's editorial projection does not assign
	// freshness, so it is left empty here rather than fabricated.
	Freshness string
	// Historical marks a publish record: it records what happened rather than
	// what is currently true.
	Historical bool
	// Metadata carries the typed fields a result reader needs without a second
	// owner round-trip.
	Metadata map[string]any
}

// Snapshot is a point-in-time materialization of the editorial corpus. Its
// Generation changes only when record content changes, and MaterializedAt is
// the newest observed source mutation — never the read time. An unchanged
// query therefore cannot advance either value.
type Snapshot struct {
	Records        []Record
	Generation     string
	MaterializedAt time.Time
}

// IndexedCount is the number of projected records.
func (s Snapshot) IndexedCount() int { return len(s.Records) }
