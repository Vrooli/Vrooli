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
// `initiatives/{name}/review/decisions/{ts}-{verdict}.json`.  Round loading
// reuses the owner-neutral primitives from `internal/review`, so a single
// schema covers both surfaces and only owner-specific logic (spawn payload,
// HTTP scope, status transitions) lives here.
//
// DOC: docs/concepts/ARCHITECTURE.md#initiative-review
package initiativereview
