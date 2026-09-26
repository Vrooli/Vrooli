// Package proposals is the mutation primitive for agent-proposed changes to
// an milestone's item graph.
//
// An agent (feedback, research, or review) produces a Proposal describing a
// set of intended changes — either a flat Mutation list or a full target
// Graph that the server diffs against the current graph.json. The user then
// decides, per-mutation, which changes to accept. The Applier translates the
// accepted mutations into calls against the existing backlog, milestones,
// and execution services. Proposals NEVER write item files directly — they
// compose existing primitives so validation, cascading, and event emission
// remain a single source of truth.
//
// The package is intentionally layered:
//
//   - types.go          — wire format (Proposal, Mutation, Op, Form, ItemSpec, ItemPatch, Graph).
//   - validate.go       — pure schema + semantic validation against a current graph.
//   - normalize.go      — full_graph → mutation_list diffing.
//   - apply.go          — Applier + narrow service interfaces + per-op dispatch.
//   - errors.go         — sentinel + typed errors for callers to distinguish
//     bad-input from infra failures.
//
// Boundary rules:
//
//   - Proposals is a leaf package: it depends on backlog, milestones,
//     execution, and graph for *types and read/write primitives*, not on
//     HTTP handlers.
//   - The Applier is safe to call from any goroutine. Each mutation in a
//     batch is applied sequentially — partial application is surfaced in
//     the per-mutation Outcome so callers can present a clean accept/reject
//     UI even under failure.
//   - Attribution: every Apply call carries a Source that tags events and
//     log lines so audits can follow a mutation back to the feedback round
//     it came from. Source carries MilestoneName, FeedbackRoundID,
//     RoundNumber, RoundSlug, and Entrypoint so downstream consumers
//     (event log, agentactivity) can group mutations by the originating
//     surface.
//
// Op set:
//
// The supported ops are add_item, update_item, split_item, add_edge,
// remove_edge, change_status, change_priority, move_milestone,
// archive_item, and interrupt_in_progress.
//
// archive_item is the canonical removal path: the original plan
// intentionally excludes a remove_item op in favor of archiving (so the
// historical record stays intact and downstream consumers can still
// resolve archived references). archive_item is included as a first-class
// op here — rather than buried behind a separate "archive via existing
// endpoint" call — so a single proposal can mix archives with other
// mutations atomically and the UI can render archives in the same
// per-mutation accept/reject list as everything else.
package proposals
