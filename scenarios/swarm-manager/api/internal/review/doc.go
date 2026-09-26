// Package review owns the post-completion review phase for a single
// backlog item.
//
// Given a completed execution, the service spawns a review agent that
// assesses whether the item's stated goal was delivered, surfaces
// regressions, and — if needed — drives additional evidence-gathering
// rounds via RequestMoreEvidence. Terminal verdicts on the item itself
// land via the backlog `review-decide` endpoint (see internal/backlog),
// which is the only path to a terminal item status.
//
// Round storage lives under the item's own folder at
// `<item_dir>/review/round-NNN.json`, written through the shared
// LoadRound / SaveRound / NextRoundNumber primitives.
//
// DOC: docs/concepts/ARCHITECTURE.md#review
package review
