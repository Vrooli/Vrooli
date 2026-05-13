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
// LoadRound / SaveRound / NextRoundNumber primitives that the
// initiative-level review service also consumes.
//
// # Relationship to internal/initiativereview
//
// The initiative-level review phase (internal/initiativereview) shares
// this package's on-disk Round schema and helpers but keeps a *separate*
// Service implementation. The duplication is deliberate per the project's
// `feedback_duplicate_before_extract.md` guidance — the second use case
// ships before extraction. When a third review-owner type lands (e.g.
// research-conclusion reviews), fold both into a polymorphic
// `Owner{Type, Kind, Name}` abstraction with owner-specific spawn adapters.
// Resist pre-emptive abstraction; the current duplication is cheap and
// keeps owner-specific lifecycle clear.
//
// DOC: docs/concepts/ARCHITECTURE.md#review
package review
