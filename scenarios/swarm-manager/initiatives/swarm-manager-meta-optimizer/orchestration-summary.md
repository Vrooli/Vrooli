# Orchestration Summary — swarm-manager-meta-optimizer

## Origin

Created **2026-04-23** as one of three deferred initiatives after the
"Agentic Surfaces for Initiatives" foundation shipped (W1–W8 of plan
`this-sounds-great-please-calm-lobster.md`). This initiative's reason to
exist is a direct artifact of that plan: the feedback/review/handoff data
the foundation now captures per initiative is the raw material for a
future **meta-optimizer prompt-manager team member**.

## Vision

Build a new prompt-manager team member that mines completed-initiative
signal (feedback rounds, proposal decisions, review verdicts, per-item
handoffs) and proposes improvements to skills, tools, and orchestration
primitives. Closes the recursive-improvement loop the project is built
around: every initiative the system completes makes every future
initiative measurably better.

## Cold-start constraint (load-bearing)

**This initiative is deliberately low-priority (priority 5) and paced
behind a research item.** The meta-optimizer is cold-start sensitive: it
needs real dogfooded data to reason about. Shipping it before there are
≥5 completed initiatives with full feedback/review trails would produce
garbage proposals and burn trust in the surface.

The research item (`research/meta-optimizer-signal-audit`) exists to
enforce this pacing. It inventories what the foundation actually
produces once people use it, quantifies minimum signal quality/volume,
and gates the spec item behind a real evidence check. If the answer is
"not enough signal yet," the spec item waits.

## Architectural decisions

1. **Team member, not ad-hoc script.** The optimizer lives in
   `scenarios/prompt-manager/` as a first-class team member with its own
   skill pack, review gate, and prompt-manager UI surface. Mining scripts
   that run out-of-tree degrade to one-shot experiments; a team member
   accumulates improvement.

2. **Produces hypotheses with citations, never direct fixes.** The
   optimizer's output is structured proposals (feedback entries on
   prompt-manager's own initiatives, pull requests in skill packs with
   review gates), not autonomous edits. Same trust model as every other
   agentic surface in the system: user is the sole authority on merge.

3. **Reuses the proposal primitive from `internal/proposals`.** A
   meta-optimizer proposal targeting a skill or tooling change is
   structurally the same as an initiative feedback proposal targeting a
   graph mutation — both are mutation lists with rationale per mutation,
   both are reviewed per-mutation, both apply through existing update
   APIs. If we find the primitive doesn't generalize, that's important
   signal and we fix it before building more surfaces on top.

4. **Signal inventory before spec.** The research item blocks the idea
   item blocks the execute items. No building on speculation. This is
   baked into the `depends_on` chain.

5. **No changes to feedback / review data shape for this.** If the
   foundation's data isn't enough for the optimizer to reason about,
   that's a fix on the foundation (new fields, richer attribution), not
   a scope creep on this initiative.

## Dependency reasoning

- **Internal chain:**
  `research/meta-optimizer-signal-audit`
  → `idea/meta-optimizer-team-spec`
  → `execute/meta-optimizer-team-implementation`
  → `execute/meta-optimizer-proposal-review-ui`

  The research item gates everything. The idea item produces a spec
  document that the execute items consume. The UI item is sequenced
  after the implementation so it can be informed by real optimizer
  output, not mocks.

- **External:** depends on the agentic-initiatives foundation shipping
  (it has) **and** on enough real usage for the signal audit to
  conclude favorably. No calendar dependency; event-driven.

## What was explicitly ruled out

- **Autonomous optimizer that applies its own suggestions.** Never.
  Always gated through review — same policy as every other agent in the
  system. This constraint is load-bearing for the user's trust model.
- **Optimizer reading across scenarios beyond swarm-manager.** Scope
  starts narrow: just swarm-manager initiatives. Expansion to other
  scenarios (agent-manager, prompt-manager itself, landing-page suites)
  is a separate future initiative. Narrower scope = faster iteration.
- **Building the optimizer before the signal audit concludes.** Skipping
  the research item in the name of speed is explicitly forbidden. The
  signal audit is the go/no-go gate.
- **A dashboard of "things the optimizer could have said."** No
  speculation mode. The optimizer runs on real data only.

## Items

1. `research/meta-optimizer-signal-audit` (priority 5, effort S) —
   audits what completed initiatives produce; quantifies minimum
   signal quality/volume; produces a conclusion that explicitly says
   "ready to spec" or "not yet, need N more completed initiatives".
2. `idea/meta-optimizer-team-spec` (priority 5, effort M) — defines
   inputs, outputs, skill structure, review gate. Gated behind the
   research item.
3. `execute/meta-optimizer-team-implementation` (priority 5, effort L) —
   build from approved spec. Core skill + supporting tools + offline
   mining pipeline.
4. `execute/meta-optimizer-proposal-review-ui` (priority 5, effort M) —
   UI surface for reviewing optimizer proposals. Inherits from the
   initiative proposal review primitives; no bespoke mutation UI.

## Open questions (deferred to workshop / research phase)

- **What counts as "enough signal"?** The research item settles this.
  Candidate heuristics: N completed initiatives, M feedback rounds with
  decisions, K review rounds with rationale text. Research picks the
  real numbers.
- **Where does the optimizer live in prompt-manager's team roster?**
  Spec phase. Candidates: new team member parallel to existing prompt
  authors; or subordinate to the prompt-manager lead. Either is
  defensible.
- **Cadence:** on-demand vs continuous vs scheduled. Spec phase.
  Leaning on-demand initially (user triggers, optimizer runs, outputs
  proposals) to keep feedback tight.
- **Output format for skill proposals:** pull requests in skill packs
  (git) vs prompt-manager-native feedback rounds on the skill's parent
  initiative. Spec picks; leans toward prompt-manager-native for
  inheriting the same review primitive.

## Session context

Created in the same session that shipped the agentic-initiatives
foundation, alongside `initiative-feedback-research-support` and
`initiative-proposal-advanced-diff-ux`. This is the deepest of the
three deferred initiatives — both in scope (priority 5, largest item is
`L`) and in dependency chain (4 items, 3 sequential). Workshop agents
should treat the signal-audit research as the de facto go/no-go for the
whole initiative, not just its own item.
