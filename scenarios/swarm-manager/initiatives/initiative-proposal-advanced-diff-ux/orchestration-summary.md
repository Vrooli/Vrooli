# Orchestration Summary — initiative-proposal-advanced-diff-ux

## Origin

Created **2026-04-23** as one of three deferred initiatives after the
"Agentic Surfaces for Initiatives" foundation shipped (W1–W8 of plan
`this-sounds-great-please-calm-lobster.md`). The foundation delivered
the **baseline** proposal review UX:

- Per-mutation checkboxes with rationale per mutation
- Side-by-side before/after dependency-graph overlay
  (`InitiativeDependencyGraph` with `overlay` prop)
- Accept Selected / Reject / Revise / Dismiss actions
- Partial-accept support (accept some mutations, reject others)

This initiative layers the **advanced affordances** on top once we have
real usage data about which rough edges are worth smoothing.

## Vision

Turn the proposal review surface from "read and check" into a
collaborative authoring tool. Advanced features under consideration:

1. **Inline mutation editing** — user can tweak an `add_item` mutation's
   title/effort/priority before accepting, instead of rejecting and
   asking the agent to revise.
2. **Bulk accept patterns** — "accept all `change_priority` mutations",
   "accept every mutation scoped to initiative `foo`", "accept
   everything except `interrupt_in_progress`".
3. **Proposal revision comparison** — when the agent revises after a
   user "revise" action, show diff between proposal v1 and v2, not just
   proposal v2 vs current graph.
4. **Richer visual diff** — edge-type colors, impact radius shading on
   `change_status` mutations, collapsible clusters for large proposals.

## Why this is deliberately deferred

**We need dogfooding evidence to spec the advanced features well.** The
baseline ships with four actions (accept / reject / revise / dismiss)
and one affordance (per-mutation checkboxes). Any feature added on top
should be in response to observed pain, not speculation. Scheduling
this initiative behind `initiative-feedback-research-support` gives us
two distinct proposal sources (feedback + research) to pressure-test
the baseline before extending it.

## Architectural decisions

1. **Extend, never fork.** The advanced diff UX must extend the
   existing `proposal-review.tsx` component. Forking a parallel
   "advanced review" component would split the per-mutation primitive
   between two surfaces — exactly the kind of duplication this
   initiative should avoid.

2. **CLI parity is non-negotiable.** Every advanced affordance must
   degrade to something reviewable from
   `swarm-manager initiatives feedback-decide --mutations m1,m3`.
   Inline editing is the hardest here; the CLI path is "reject and
   ask for revision with specific text." Bulk patterns can surface on
   the CLI as multiple `--mutations` flags or filter syntax. If a
   feature is **only** usable in the UI, we designed it wrong.

3. **Preserve the mutation-list contract.** No new mutation ops. The
   advanced UX is purely presentation + input-collection on top of
   the existing `internal/proposals` primitive. If we find a feature
   requires a new op, that's a signal to push back into `proposals`
   as its own small initiative.

4. **Bulk accept is a UI convenience, not a new semantic.** Under the
   hood, "accept all `change_priority`" expands to the same
   `{kind: partial_accept, accepted_mutation_ids: [...]}` payload the
   per-mutation checkbox path produces. No new API surface.

5. **Proposal revision comparison reads from the thread.** Each
   proposal is already stored in the round's `proposals` array
   (see `feedback.Round` schema). Comparison is a pure frontend
   operation — no server changes needed. If that turns out to be
   wrong, scope expands and we renegotiate.

## Dependency reasoning

- **Depends on `initiative-feedback-research-support`.** Rationale:
  two proposal sources give us twice the dogfooding signal about
  which advanced affordances actually matter. Shipping advanced UX
  before there's a second source risks over-fitting to the feedback-
  only flow.
- **Does not depend on `swarm-manager-meta-optimizer`.** The
  optimizer consumes proposal-decision data but doesn't drive UX
  requirements. Meta-optimizer proposals reuse the same review
  component and benefit from whatever this initiative ships, but
  the reverse isn't true.

## What was explicitly ruled out

- **A dedicated "advanced review" mode toggle.** Features should be
  unified, not gated behind a mode switch. If a feature isn't good
  enough to be default-on once shipped, it shouldn't ship.
- **Rich-text rationale editing.** Rationale stays as plain text.
  Markdown rendering is fine; WYSIWYG is not.
- **Proposal diff across initiatives.** Scope is one proposal vs its
  prior revision within the same round. Cross-round / cross-initiative
  comparison is out of scope — probably belongs to the meta-optimizer.
- **New mutation ops.** Explicitly forbidden for this initiative.
- **Drag-and-drop re-sequencing of mutations.** Ordering is
  informational only (mutations apply in an order the server
  normalizes based on dependencies). Adding DnD would encourage
  bogus mental models.

## Items

1. `idea/advanced-diff-ux-spec` (priority 6, effort S) — spec sheet.
   Must quote real dogfooding complaints where possible; pure
   speculation rejected at workshop review.
2. `execute/advanced-diff-ux-implementation` (priority 6, effort M) —
   ships the spec. Depends on the idea item.

## Open questions (deferred to workshop)

- **Which advanced feature ships first?** Spec decides. Probably
  inline editing on `add_item` since it's the most common rejection-
  reason today (expected). Real data from initiatives 1 and 2 will
  confirm or refute.
- **Keyboard shortcuts for bulk accept.** Spec decides. Probably `a`
  for accept-all-selected, `r` for reject-all-selected.
- **Visual treatment of proposal-over-proposal diff.** Three candidates:
  side-by-side columns, unified diff view, toggle between versions.
  Spec + a paper prototype resolves.

## Session context

Created in the same session that shipped the agentic-initiatives
foundation, alongside `initiative-feedback-research-support` and
`swarm-manager-meta-optimizer`. Of the three deferred initiatives,
this is the lowest-priority and most UX-heavy — deliberately kept
small (2 items) because the spec is intentionally informed by
dogfooding, not predesigned. Workshop agents should NOT pre-bloat the
scope; the whole point of this initiative's pacing is to earn its
features from observed signal.
