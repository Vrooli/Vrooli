# Orchestration Summary — initiative-feedback-research-support

## Origin

Created **2026-04-23** as one of three deferred initiatives immediately after the
"Agentic Surfaces for Initiatives" foundation shipped (W1–W8 of plan
`this-sounds-great-please-calm-lobster.md`). The foundation introduced:

- `internal/feedback` — round lifecycle, thread model, multi-turn agent runs
- `internal/proposals` — mutation primitive (`add_item`, `update_item`,
  `change_status`, `change_priority`, `add_edge`, `remove_edge`, `split_item`,
  `move_initiative`, `interrupt_in_progress`) with a pluggable apply layer
- `internal/graph.MaterializeInitiative` + listener maintaining per-initiative
  `graph.json`
- New `in_review` / `review_pending` / `needs_followup` lifecycle for items and
  initiatives (user is the sole authority on terminal transitions)
- The `swarm-manager-initiative-feedback` skill and server-side agent spawn
- A `Feedback` tab on the initiative UI with mutation-list review, per-mutation
  checkboxes, and a graph overlay showing before/after shape
- CLI surface under `swarm-manager initiatives feedback-*` and
  `swarm-manager backlog review-decide`

The foundation intentionally shipped with the **Research** chip on the feedback
dialog disabled (badge: "Coming soon"). This initiative wires it up.

## Vision

Give the user a second path into the same proposal primitive: instead of typing
feedback + attaching screenshots, start a scoped **research round** on the
initiative. The research agent investigates the initiative's item graph,
surrounding scenarios, and relevant external signal, then concludes with a
`mutation_list` proposal against `graph.json`. Review UX is identical to
feedback proposals — reuse, not fork.

This initiative exists **primarily as a second consumer of `internal/proposals`**.
If `internal/proposals` is the right abstraction, a research flow should slot in
with zero new mutation ops and no changes to the apply layer. If it needs
anything beyond a new thin skill + adapter, the abstraction is wrong and we fix
it here.

## Architectural decisions

1. **Reuse `internal/feedback` round, don't fork.** A research round is a
   `feedback.Round` whose `type` is `"research"`. Same disk layout
   (`initiatives/{name}/feedback/round-NNN-{slug}/`), same thread model, same
   lock (one agent per initiative). The type flag tells the skill which
   system prompt to load; nothing else changes.

2. **New skill, not new package.** `swarm-manager-initiative-research` loads
   the same context variables as `swarm-manager-initiative-feedback`
   (INITIATIVE_NAME, CURRENT_GRAPH, ITEM_SUMMARIES, PRIOR_FEEDBACK,
   ITEM_FOLDER_INDEX) plus a research-specific brief field on the submission.
   No new Go package — the skill carries all behavioral difference.

3. **Research concludes into a `conclusion.md` first, proposal second.**
   Pattern borrowed from the research-backlog rework (see memory entry
   `project_research_backlog_rework`). The agent writes its investigation
   notes into the round's thread, then emits a fenced JSON proposal block.
   Parse failure surfaces to the user as "ask for revision" — no silent
   failure.

4. **No direct write access to initiative files.** The research skill must
   only read from `graph.json`, `initiative.json`, item folders. All mutation
   goes through the proposal → user-decide → apply pipeline. Matches the
   forbidden-actions list already enforced for the feedback skill.

## Dependency reasoning

- No external dependencies. Blocks on nothing; the foundation is already in
  `master`.
- `initiative-proposal-advanced-diff-ux` **depends on this initiative**
  because the advanced diff UX wants two distinct proposal sources
  (feedback + research) to pressure-test inline editing and comparison. The
  dependency is captured on that initiative, not this one.

## What was explicitly ruled out

- **Research as a separate package** — covered by the "new skill, not new
  package" decision above. Keep the feedback package as the single domain
  owner of rounds.
- **Research producing raw `full_graph` proposals.** Default to
  `mutation_list` with the existing normalize-full-graph path left for the
  rare drastic restructure. Research rounds exploring broad redesigns are
  encouraged to still write a mutation list.
- **Cross-initiative research.** Scope is exactly one initiative.
  Cross-cutting research belongs on a backlog item (`kind: research`), not
  on an initiative.

## Items

1. `research/feedback-research-flow-design` (priority 3, effort S) — designs
   the exact agent responsibilities, context inputs, output shape, and the
   handoff between the feedback UI and the proposals apply layer. Produces a
   `conclusion.md` that `execute/initiative-research-entry-point` consumes as
   its plan input.
2. `execute/initiative-research-entry-point` (priority 3, effort M) —
   implementation: enable the chip, register the skill, wire the spawn.
   Depends on the research item's conclusion.

## Open questions (deferred to workshop)

- Should the research skill be allowed to call `web-search` / `web-fetch`
  resources? The feedback skill cannot; research arguably should. Workshop
  round 1 settles this and documents the allowlist.
- Should research rounds have a longer default timeout than feedback rounds?
  Probably yes — open-ended investigation takes longer than structured
  feedback. Workshop nails the exact number.
- UX copy on the chip: "Research" vs "Investigate" vs "Deep dive". Left to
  workshop.

## Session context

This initiative was created in the same session that shipped the agentic-
initiatives foundation, alongside `swarm-manager-meta-optimizer` and
`initiative-proposal-advanced-diff-ux`. Workshop agents should read those
two peer summaries to understand the full deferred-work landscape.
