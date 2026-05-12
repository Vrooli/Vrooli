// Package initiativereview owns the post-completion review phase for an
// initiative as a whole.
//
// When every backlog item in an initiative reaches a terminal status, this
// package spawns a review agent that assesses whether the initiative's stated
// goal was actually delivered, surfaces regressions, and (optionally) proposes
// follow-up items. The user then inspects the review round and renders a
// terminal verdict (`accept`, `fail`, or `followup`) — the initiative's
// status only flips to `completed` / `failed` / `needs_followup` through this
// package's Decide path. Nothing else may write a terminal initiative status.
//
// Lifecycle:
//
//	active ─▶ in_review ─▶ review_pending ─▶ completed | failed | needs_followup
//	         (agent          (awaiting user
//	          gathering       verdict)
//	          evidence)
//
// Round storage parallels the backlog review layout: rounds live under
// `initiatives/{name}/review/round-NNN.json` and decisions under
// `initiatives/{name}/review/decisions/{ts}-{verdict}.json`. Round loading
// reuses the owner-neutral primitives from `internal/review` (Round,
// LoadRound, NextRoundNumber, SaveRound) so the schema is unified; only
// owner-specific logic (spawn payload, HTTP scope, status transitions,
// initiative-scoped context attachments) lives here.
//
// # Relationship to internal/review
//
// Item-level and initiative-level reviews share the on-disk round schema
// but intentionally keep *separate* Service implementations. This is a
// deliberate duplicate-before-extract per the project's
// `feedback_duplicate_before_extract.md` guidance: the second use case
// (initiatives) ships first, then a third review-owner type (likely
// research-conclusion reviews from the initiative-feedback-research-support
// initiative) should trigger the extraction to a polymorphic
// `review.Owner{Type, Kind, Name}` abstraction with one round-lifecycle
// implementation and owner-specific spawn adapters.
//
// Do NOT attempt the polymorphic refactor pre-emptively. The current
// duplication is cheap (round lifecycle is ~200 LOC in each place) and
// the extraction risk grows with every premature abstraction that has to
// be un-abstracted once the real owner-variation pattern is visible.
//
// # Lock coordination
//
// The review service acquires `internal/initiativelock` (the same
// `.feedback-lock` file the feedback service uses) before spawning, so a
// feedback round in flight blocks review and vice versa. Release happens
// on round terminal or provisional-path failure. See service.go's
// startReview for the acquire/swap/release sequence.
//
// DOC: docs/concepts/ARCHITECTURE.md#initiative-review
package initiativereview
