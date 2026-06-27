# Plan Model — Plan Manager

This is the canonical definition of the **structured plan**: the first-class
record this scenario owns. Every other document (DOMAINS, DATA, FLOWS,
INTEGRATIONS) refers back here for the shape of a plan and a phase. The model is
the foundation of the whole scenario: because a plan is structured data with
**computed** status and staleness — not a prose markdown file an agent
reverse-engineers — the runtime can hold the procedural knowledge a small/local
model would otherwise have to carry in its context.

## Purpose Of This Document

Answer, in one place:

- What is a plan, structurally? What is a phase?
- What is *authored* by an agent vs. *computed* by the runtime?
- How does the rendered markdown view relate to the structured record?
- What references, validation, staleness, and handoff state attach to a plan?

The prose skeleton this replaces is the legacy 13-section
`implementation-plan-authoring` plan. The structured model keeps the same intent
(durable, handoff-ready, regression-anchored) but makes the mechanical parts
machine-owned.

## Why Structured, Not Prose

| Prose plan (today) | Structured plan (this model) |
|---|---|
| 13 markdown sections authored by judgment | Sections + phases as typed fields; mechanical ones auto-filled |
| Status = checkboxes the agent edits | Status = computed runtime transition; markdown is a *view* |
| Staleness = unknown; agent re-reads code | Staleness = computed from referenced-code change |
| "Did this regress?" = narrated | Verified against the regression anchor as an oracle |
| Handoff = end-of-run prose dump | Canonical handoff assembled from in-flow captured state |

The structured record is the source of truth. A human-readable markdown view is
always *rendered* from it; the view is never parsed back into truth.

## Plan

A `Plan` is the top-level record.

| Field | Authored / Computed | Meaning |
|---|---|---|
| `id` | computed | Stable identifier. |
| `title`, `slug` | authored | Human-readable name + filename-safe slug. |
| `status` | computed | `draft` → `active` → `complete` / `archived`. Derives from phase status + lifecycle actions. |
| `content_hash` | computed | Hash of the structured content; powers supersession/dependency edges. |
| `created_at`, `updated_at` | computed | Timestamps (RFC3339). |
| `purpose` | authored | One-paragraph why. |
| `scope` | authored | In-scope / out-of-scope. |
| `constraints` | authored | Hard rules (e.g. greenfield) repeated into Definition of Done. |
| `non_goals` | authored | Explicit exclusions. |
| `references[]` | authored + resolved | Connected code: existing **and** proposed locations (see References). |
| `regression_anchor` | auto-filled | The "before" anchor strategy + commands (see Validation). |
| `definition_of_done` | authored + verified | Objective pass/fail criteria; the regression check is mandatory. |
| `phases[]` | authored | Ordered, first-class phases (see Phase). |
| `supersedes`, `superseded_by` | authored | Plan graph edges. |

## Phase

A `Phase` is a **first-class object**, not a checklist line. This is the unit the
runner walks and the unit context is scoped to.

| Field | Authored / Computed | Meaning |
|---|---|---|
| `id`, `order` | computed / authored | Identity and sequence. |
| `title` | authored | Short phase name. |
| `intent` | authored | What this phase accomplishes. |
| `required_reading[]` | auto-filled + authored | Skills/docs to load **for this phase**; injected on `next`. |
| `reminders[]` | authored | General + phase-specific reminders, surfaced just-in-time during the phase. |
| `references[]` | authored + resolved | Phase-scoped connected-code, requirement, or document references. A phase with no connected references must carry an explicit no-code reason during authoring. |
| `baseline_scope` | computed | The set of connected-code locations this phase touches → the exact baseline/validation command set. |
| `acceptance` | authored | Objective pass/fail for this phase. |
| `status` | computed | `todo` → `active` → `done` / `blocked`. A typed transition or inferred from acceptance + validation passing. |
| `last_validation` | computed | Most recent validation result + staleness factor for this phase. |
| `decisions[]`, `findings[]` | captured in-flow | Recorded by the agent via runner commands during execution (feeds the handoff). |

**Resume point** = the earliest phase whose `status` is not `done`. Full vs.
partial completion is derived from the phase-status set — never narrated.

## References

Every connected code location is captured with the machine-readable grammar from
`path:docs/reference/machine-readable-references.md`:

- `[CODE: path/to/file.go]` — existing code the plan depends on or changes.
- Proposed (not-yet-existing) code is referenced the same way, marked as
  `future` so staleness does not flag it as "deleted".
- `[REQ: OT-…]` — links a plan/phase to a requirement.

References are resolved against `code-facts`. The not-yet-built **unified code
identifier** is a planned drop-in upgrade for the locator; until then `[CODE:]` +
code-facts is the contract. References are how staleness becomes computable, so
the authoring wizard makes them mandatory at plan and phase scope. A docs-only
or process-only plan/phase may use `NO_CODE_REFS: <reason>` instead; the reason
is stored as authored context rather than pretending the plan has connected code.

## Validation, Staleness, And The Regression Anchor

- **Regression anchor** — captured *before* changes (a `git-control-tower
  baseline` snapshot for scenario-scoped work, or a `HEAD` sha + file allowlist
  for outside-scenario work). Auto-filled by the wizard; stored on the plan.
- **Baseline scope** — derived per phase from `references[]`: the exact
  baseline/diff command set across all affected locations (not just scenarios).
- **Staleness tier** — computed from change in referenced code since authoring:
  - `fresh` — no relevant change.
  - `lightly_stale` — small diffs in referenced code.
  - `definitely_stale` — referenced locations moved or were deleted.
- **Definition of Done** — verified against the regression anchor as an oracle
  (baseline diff exit 0), not a narrated claim.

## Handoff (Structured Layer Only)

plan-manager owns the **canonical, structured** handoff, assembled from state
captured in-flow during a run:

- phase status (→ full/partial + resume point), `decisions[]`, `last_validation`,
  candidate `findings[]`, and staleness.
- Findings are **candidate / unvalidated**; an operator triages them before they
  become real bugs.

It does **not** own the agent's free-text "prose" final handoff — that catch-all
is captured by the orchestration layer (agent-manager run transcript →
swarm-manager operating mode) and linked to the plan by reference. See
[`INTEGRATIONS.md`](INTEGRATIONS.md) for the ownership split.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — which domain owns each part of this model
- [`DATA.md`](DATA.md) — how the model persists (durable home store)
- [`FLOWS.md`](FLOWS.md) — authoring, execution, and handoff lifecycles
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — substrate this model composes
- [`../../PRD.md`](../../PRD.md) — operational targets
