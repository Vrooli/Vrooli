// Package proposals is the mutation primitive for agent-proposed changes to
// an initiative's item graph.
//
// An agent (feedback, research, or review) produces a Proposal describing a
// set of intended changes — either a flat Mutation list or a full target
// Graph that the server diffs against the current graph.json. The user then
// decides, per-mutation, which changes to accept. The Applier translates the
// accepted mutations into calls against the existing backlog, initiatives,
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
//   - Proposals is a leaf package: it depends on backlog, initiatives,
//     execution, and graph for *types and read/write primitives*, not on
//     HTTP handlers.
//   - The Applier is safe to call from any goroutine. Each mutation in a
//     batch is applied sequentially — partial application is surfaced in
//     the per-mutation Outcome so callers can present a clean accept/reject
//     UI even under failure.
//   - Attribution: every Apply call carries a Source that tags events and
//     log lines so audits can follow a mutation back to the feedback round
//     it came from. This is carried through but unused until W4 wires the
//     feedback package on top.
package proposals
