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
always *rendered* from it; the view is never parsed back into truth. Plan Manager
also keeps a durable rendered mirror file for operator and broken-server
workflows, but the mirror is still a projection: stale or edited mirror bytes are
repaired from the structured record, never silently adopted as truth.

## Plan

A `Plan` is the top-level record. Its fields form a **professional, execution-grade
implementation plan** organised into six jobs: Overview, Work Posture, Execution
Model, Validation Model, Phases, and Plan Governance. This is **not** the legacy
13-section prose plan — every field below has a concrete job for an authoring
agent, an implementation agent, or a human reviewer. Legacy concepts only become
first-class fields when they help execution or review (see [Greenfield
Posture](#work-posture--greenfieldbrownfield) and [Import
Provenance](#import-provenance--preserved-legacy-sections)).

Each field is marked **authored** (a small/local model supplies prose),
**autofilled/computed** (the runtime derives it; the agent never types it),
**imported** (recovered from markdown import), or **unmapped import
provenance** (kept verbatim because it could not be mapped).

**Overview** — why the work exists and what success looks like.

| Field | Origin | Meaning |
|---|---|---|
| `id` | computed | Stable identifier. |
| `title`, `slug` | authored | Human-readable name + filename-safe slug. |
| `status` | computed | `draft` → `active` → `complete` / `archived`. Derives from phase status + lifecycle actions. |
| `content_hash` | computed | Hash of the structured content; powers supersession/dependency edges. |
| `created_at`, `updated_at` | computed | Timestamps (RFC3339). |
| `mirror` | computed | Durable rendered markdown projection metadata: absolute path, runtime-home-relative path, rendered content hash, renderer version, rendered timestamp, freshness/repair status, and last error when degraded. |
| `purpose` | authored | One-paragraph why this plan exists. |
| `problem_statement` | authored | The concrete problem / need / gap this plan closes. Mandatory for implementation plans. |
| `target_outcome` | authored | The end state once the plan is done — what is observably true. Mandatory for implementation plans. |
| `scope` | authored | In-scope / out-of-scope. |
| `non_goals` | authored | Explicit exclusions. |
| `assumptions` | authored | Preconditions taken as given (environment, prior work, access). |
| `assumption_risks[]` | authored (optional) | Structured assumptions with an "if wrong → then" `mitigation` each, rendered as the two-column table under **Assumptions & Risks**. Authored as `assumptions` lines carrying an `-> <mitigation>` suffix. Empty renders nothing. |

**Work Posture** — automatic Greenfield/Brownfield derivation (never hand-authored).

| Field | Origin | Meaning |
|---|---|---|
| `work_posture` | autofilled | `greenfield` or `brownfield`. Derived from scenario maturity; **default greenfield**. |
| `work_posture_source` | computed | How posture was decided: `default`, `service_maturity`, `explicit_override`, `import_legacy`. |
| `work_posture_detail` | computed | Human-readable derivation note (e.g. fallback reason, sunset warning). |

**Change Boundary** — the plan's blast radius, the source of truth for posture, anchor, and validation scope.

| Field | Origin | Meaning |
|---|---|---|
| `change_boundary.acceptance_allow[]` | authored | Repo-relative path globs this plan may change (e.g. `scenarios/plan-manager/**`, `packages/proto/**`, `docs/**`). Mandatory for implementation plans unless an operator-only reason is recorded. **There is no `scope` field and no `primary_scenario` flag** — scenario identity is *derived* from these globs. |
| `change_boundary.acceptance_deny[]` | authored (optional) | Path globs this plan must **not** change. Guardrails only — they never widen validation scope. |
| `change_boundary.operator_only_reason` | authored (optional) | Why a plan legitimately carries no `acceptance_allow` (operator-only / no-code work) — the boundary analogue of the references `NO_CODE_REFS` escape. |

**Execution Model** — what to load and how to build.

| Field | Origin | Meaning |
|---|---|---|
| `relevant_context[]` | authored + discovered | Plan-wide setup items (skills, docs, discovery searches, commands) a fresh/resumed agent loads before execution. |
| `references[]` | authored + resolved | Connected code: existing **and** proposed locations (see References). |
| `constraints` | authored | Hard rules. The Greenfield block is **not** authored here — it is injected by posture. |
| `prohibited_approaches` | authored (optional) | Approaches that are explicitly off-limits, only when genuinely relevant. |
| `technical_approach` | authored | Design rationale: the chosen approach and why, not a phase list. Mandatory for implementation plans. |
| `decisions[]` | authored (optional) | Pinned plan-time contract decisions (`title` + `statement`), rendered `D1..Dn` under **Approach & Decisions** — do not relitigate during execution. Distinct from execution-time `log decision-add` entries. Empty renders nothing. |

**Validation Model** — how regressions and done-ness are proven.

| Field | Origin | Meaning |
|---|---|---|
| `regression_anchor` | autofilled | The "before" anchor strategy + commands (see Validation). |
| `validation_strategy` | authored | How the plan proves it works: baseline approach, phase validation expectations, what evidence counts. Mandatory for implementation plans. |
| `final_validation_commands[]` | authored | The exact commands a reviewer runs at the end (scenario test, baseline diff, focused suites). |
| `definition_of_done` | authored + verified | Objective pass/fail criteria; the regression check is mandatory. |

Validation also includes an execution-grade **plan quality** pass. This is
separate from the authoring structure gate: legacy imports remain non-destructive,
but validation flags imported/thin phases that lack ordered steps, phase
validation, or objective acceptance, and flags migrated setup context that still
contains markdown-fence residue or malformed generated commands. Import should
promote obvious legacy phase sections (`Objective`, `Checklist`, `Validation`,
`Definition of done`) into structured phase fields before preserving any
unmapped prose as legacy provenance.

### Baseline sets

New implementation plans may carry an optional `baseline_set` intent derived
from their Change Boundary: a stable collection name, explicit behavioral
scenario targets, selected repository paths, execution-start capture policy,
and compatibility state. Plan Manager owns this intent and its phase policy;
Git Control Tower owns the underlying collection, Test Genie anchors, and
source-evidence mechanics.

At execution start, Plan Manager persists an immutable baseline-set checkpoint
on the execution: resolved scenario/path inventory, capture timestamp, required
coverage counts, and complete/partial/degraded state. Resume reads that
checkpoint rather than recapturing or deriving a new before-state. Required
behavioral coverage must be complete before it is treated as a usable oracle.
Selected source paths are captured as **informational source evidence** only;
their changes never substitute for a Test Genie regression verdict.

The checkpoint also keeps the collection branch, each member's baseline/run and
capture status, and metadata-only path-snapshot references. This is recovery and
operator provenance: source bytes remain private in GCT. Validation captures a
bounded current “after” snapshot through GCT, requests a typed phase-filtered
path delta, and records it as a non-oracle child alongside the behavioral
collection diff. Final Definition-of-Done requires the persisted collection to
be complete and runs a durable typed full-inventory collection diff; it never
replays the rendered collection command through a shell.

For a phase with a narrow validation scope, Plan Manager intersects that scope
with the execution checkpoint's captured target inventory (falling back to
authored intent only before an execution exists) and dispatches one typed GCT
collection-diff operation for the selected members. The operation is durable
and idempotent, and the
behavioral aggregate remains `unknown`/not-comparable whenever required
coverage is incomplete; source evidence is presented separately.

Legacy regression anchors remain readable and execute under their existing
single-scenario semantics. Rendered new plans show a compact baseline-set
summary rather than a wall of generated child commands.

**Phases** — ordered, first-class units of work (see [Phase](#phase)).

| Field | Origin | Meaning |
|---|---|---|
| `phases[]` | authored | Ordered, first-class phases. |
| `risks_hazards` | authored (optional) | Plan-wide risks/hazards and their controls, only when relevant. |

**Plan Governance** — lineage and import bookkeeping.

| Field | Origin | Meaning |
|---|---|---|
| `supersedes`, `superseded_by` | authored | Plan graph edges. |
| `import_provenance` | imported | Where an imported plan came from + original format (see Import Provenance). |
| `preserved_legacy_sections[]` | unmapped import | Unmapped import sections kept verbatim, never silently dropped. |

## Phase

A `Phase` is a **first-class object**, not a checklist line. This is the unit the
runner walks and the unit context is scoped to.

| Field | Origin | Meaning |
|---|---|---|
| `id`, `order` | computed / authored | Identity and sequence. |
| `title` | authored | Short phase name. |
| `intent` | authored | What this phase accomplishes. |
| `affected_areas[]` | authored | The files/dirs/surfaces this phase touches, as orienting prose lines (complements typed `references[]`). |
| `change_boundary` | authored (optional) | Optional per-phase boundary refinement. **Narrows** the plan-level boundary for phase-specific checks; it never widens the plan blast radius. |
| `validation_scope` | authored | Every phase in a multi-scenario plan declares either `narrow` with an explicit boundary or `full_plan` with a rationale. References never implicitly narrow validation. |
| `relevant_context[]` | authored + discovered | Phase-scoped setup items with kind, reason, instruction, command/argv, target, required flag, repeat policy, source, and status. New authored phases must include at least one phase-scoped relevant-context item or an explicit `NO_CONTEXT: <reason>` note. |
| `steps[]` | authored | Ordered implementation steps — the concrete sequence an implementation agent follows. Mandatory for implementation phases. |
| `expected_outputs[]` | authored | The artifacts/outputs this phase should produce (new files, generated code, passing suites). |
| `validation` | authored | The method of checking this phase — the commands/checks run to confirm it. Distinct from `acceptance` (the outcome gate). |
| `acceptance` | authored | Objective pass/fail outcome for this phase. |
| `risks_hazards[]` | authored (optional) | Phase-specific risks/hazards and controls, only when relevant. |
| `handoff_notes` | authored (optional) | What the next phase depends on / what a resuming agent must know. |
| `reminders[]` | authored | General + phase-specific reminders, surfaced just-in-time during the phase. |
| `references[]` | authored + resolved | Phase-scoped connected-code, requirement, or document references. A phase with no connected references must carry an explicit no-code reason during authoring. |
| `required_reading[]` | migration input | Legacy phase reading preserved when importing older plans; execution-facing setup uses `relevant_context[]`. |
| `baseline_scope` | computed | The set of connected-code locations this phase touches → the exact baseline/validation command set. |
| `status` | computed | `todo` → `active` → `done` / `blocked`. A typed transition or inferred from acceptance + validation passing. |
| `last_validation` | computed | Most recent validation result + staleness factor for this phase. |

`acceptance` and `validation` are **distinct and must not be identical**:
`validation` is the *method* (the commands/checks you run); `acceptance` is the
*outcome gate* (what must be true for the phase to count as done). The authoring
wizard rejects a phase whose `acceptance` is a verbatim copy of its `validation`.

Decisions, findings, bug reports, records, and notes are **not** phase or
execution fields — they are typed entries in the `log` execution-log ledger (see
[The execution-log ledger](#the-execution-log-ledger) below), scoped to a
plan/execution/phase. The execution runner reads a compact `LogSummary` of them
for context and the canonical handoff; it does not store them on the phase.
Rendered plans include an **Execution Feedback** section with the default
`plan-manager log ...` capture workflow so a small execution agent sees the
feedback contract even before it asks the runner for just-in-time context.

The runner also computes a **phase feedback checkpoint** from phase-scoped log
entries. A `done` transition is recommended and accepted only after the phase has
captured durable feedback (decision, finding, bug report, or record) or an
explicit no-feedback note titled `Phase feedback reviewed: none`; degraded or
operator-reviewed cases must carry an explicit feedback override reason. This
checkpoint is computed from the log ledger, not authored onto the phase.

**Resume point** = the earliest phase whose `status` is not `done`. Full vs.
partial completion is derived from the phase-status set — never narrated.

## Authoring Interaction Contract — Form, Not Wizard

The authoring session is a **form with a derived cursor, not a wizard with
state**. The write layer is order-free: any section or phase field may be
submitted at any time, and the "current" position is always derived
(`firstUnfilledMandatory` / `nextIncompletePhaseID`), never stored as a stage.

- **Full disclosure.** Every guided step carries a `checklist` — the complete
  requirement set for the touched scope (all 7 phase fields on a phase step;
  the session-wide mandatory/gate map on section and review steps) with live
  `filled`/`missing`/`violation` status. No requirement is ever revealed only
  after submitting the previous one.
- **Batch granularity.** `SubmitFields` carries one field, a whole phase, or
  the whole plan in one call under one session lock and one save. Items apply
  independently (never all-or-nothing); each returns accepted/rejected +
  violations + a one-line parse summary. The single-field RPCs
  (`SubmitSection`, `SubmitPhaseField`) are wrappers over the same apply path.
- **No stage gating.** Quality enforcement stays at field-level validators
  (acceptance ≠ validation, reference-kind routing, `NO_CONTEXT:` reasons,
  boundary globs) and the `validate`/`finalize` structure gate. A
  quality-rejected write is **not applied** and says so loudly.
- **Both context paths satisfy the context gates**: typed `context-submit`
  items (executable/loadable setup) and prose `relevant_context` notes /
  `NO_CONTEXT:` skip reasons. A violation-rejected context item reports
  `accepted: false` and leaves the gate open — never success-shaped output.
- **Honest finalize.** The finalize response names the physical store (SQLite
  path), the stamped workspace, and the **computed** mirror publish result
  threaded from the write itself — `fresh` or `write_failed` (+ reason), never
  the read-model default `unknown`. An unretrievable read-back is a hard
  error; an idempotent re-run is flagged `already_finalized`.

Concurrency: every authoring mutation runs lock → reload → modify → save on a
per-session lock (`withSessionLock`), so pipelined or parallel CLI calls
cannot lose writes.

## Change Boundary — acceptance_allow / acceptance_deny

A plan is **general over repository change boundaries**: it can target one
scenario, multiple scenarios, shared packages, docs, root tooling, or any mixed
set of repo paths. The `change_boundary` field is the first-class contract for
that blast radius and the **single source of truth** the runtime derives posture,
regression-anchor intent, validation scope, and execution reminders from.

- **Vocabulary is `acceptance_allow` / `acceptance_deny`**, deliberately aligned
  with Swarm Manager's backlog change boundary so the two compose without
  translation. There is **no public `scope` field** and **no `primary_scenario` /
  `affected_scenario` flag** — those were rejected. Scenario identity is *derived*
  from the allow globs (and references), never authored as a top-level plan
  identity.
- **`acceptance_allow`** lists the repo-relative path globs the plan may change.
  It is mandatory for newly authored implementation plans unless an explicit
  operator-only / no-code reason is recorded (`operator_only_reason`).
- **`acceptance_deny`** is optional and rendered when present. Deny globs are
  **guardrails only**: they never widen validation scope; they flag forbidden
  edits and render/validate as pre-execution constraints.
- **Affected scenarios** are derived: a glob under `scenarios/<name>/...` yields
  `<name>`; every other glob (`packages/**`, `docs/**`, root tooling) is a
  **repo-level path** that has no scenario baseline oracle today.
- **Unresolved placeholders** (`<scenario>`, `<path>`, `<branch>`, `<allowed
  path>`) are invalid in a finalized boundary or anchor.

A phase may carry its own optional `change_boundary` that **narrows** the plan
boundary for phase-specific checks; it can never widen the plan's blast radius.

**Substrate honesty.** `git-control-tower` baselines and `test-genie` suites are
scenario-keyed today. Plan Manager consumes that limitation honestly: it derives a
scenario baseline **oracle** for each affected scenario and an **informational**
repo/path diff for non-scenario allow globs, and never pretends the repo diff is a
pass/fail oracle. Native multi-path baselines are tracked as deferred substrate
work (see `docs/internal/PROBLEMS.md`).

## Work Posture — Greenfield/Brownfield

Every plan carries an automatic **work posture**. Authoring agents do **not** type
a Greenfield block; the runtime derives the posture and the renderer injects the
exact block. This guarantees the Greenfield/Brownfield guidance is consistent and
never contradicts the scenario's real maturity.

Posture is **aggregate and conservative** across *every* scenario the change
boundary and references touch — not a single associated scenario. If **any**
affected scenario is `pilot`, `production`, or `sunset`, the rendered posture is
Brownfield and `work_posture_detail` names the causing scenario(s). Posture is
Greenfield only when all affected scenarios are greenfield/absent, or when no
scenario is resolved at all (non-scenario paths default to Greenfield with an
honest detail).

**Per-scenario derivation rules** (centralized; tested for every maturity enum
value). The signal is each affected scenario's `.vrooli/service.json` `maturity`
field, resolved through a posture seam:

| Scenario `maturity` | Posture | Source | Detail |
|---|---|---|---|
| absent / unrecognized | `greenfield` | `service_maturity` | matches `.vrooli/schemas/service.schema.json` default |
| `greenfield` | `greenfield` | `service_maturity` | — |
| `pilot` | `brownfield` | `service_maturity` | limited live use; preserve external contracts |
| `production` | `brownfield` | `service_maturity` | serving real data; preserve external contracts |
| `sunset` | `brownfield` | `service_maturity` | sunset detail: scenario is being retired; prefer non-invasive changes |
| no resolvable scenario | `greenfield` | `default` | fallback detail explaining no scenario was associated |
| explicit override | as set | `explicit_override` | reserved for a future override signal |
| legacy import | preserved as imported | `import_legacy` | imported plans keep their stated posture if present |

The **default is Greenfield** unless a scenario maturity proves otherwise. Plan
Manager's own `.vrooli/service.json` omits `maturity`, so its plans are Greenfield.

When posture is `greenfield`, the renderer **always** emits this exact block (the
code-like tokens keep their backticks in Markdown):

> **This is greenfield work.** Do not include compatibility shims, legacy
> wrappers, dead code, unused re-exports, `// removed` comments, or renamed
> `_unused` variables.

When posture is `brownfield`, the renderer emits a conservative block, e.g.:

> This plan targets a deployed or limited-live scenario. Preserve external
> contracts and data unless the plan explicitly authorizes a breaking change.

If an author submits constraints that contradict the derived posture (e.g. a
greenfield plan that asks for compatibility shims), validation **flags the
conflict** rather than rendering contradictory guidance.

## Import Provenance & Unmapped Import Sections

Markdown import is a **canonicalization path**, never a lossy one. When a markdown plan is
imported:

- `import_provenance` records the `source_path`, `imported_at`, and
  `original_format` (e.g. `legacy_markdown`) so a reviewer knows the plan was
  imported, not authored fresh.
- Import sections that map cleanly become first-class fields (e.g. *Problem
  Statement* → `problem_statement`, *Target End State* → `target_outcome`,
  *Testing Plan* → `validation_strategy`, *Risks + Mitigations* → `risks_hazards`).
- Sections that do **not** map are preserved verbatim in
  `preserved_legacy_sections[]`, each carrying `heading`, `content`, an optional
  `mapped_to`, and `preservation_reason = "unmapped_legacy_section"`. They render
  under an **Unmapped Import Sections** block so nothing silently disappears.

Import is **non-destructive**: the source Markdown file is never moved or deleted.

## Rendered Markdown — Stable Review Artifact

The rendered Markdown is the **stable human-review artifact**. It must look
professional, coherent, and complete enough that a human can judge whether the
plan is better than a legacy plan without reading raw JSON, proto, or authoring
session internals. It is a deterministic *view* — never parsed back into truth.

**Plan render order** — nine reader-question clusters in fixed order (present
sections only; Work Posture is always shown). Field identity is unchanged:
clusters are a render grouping over the same structured fields, emitted as
`###` subsections. The wizard asks in the same order the artifact renders.

1. Title / status / content-hash
2. Purpose — why does this plan exist? (an abstract, not a restated Problem)
3. Problem — what is wrong today?
4. Outcome — what is observably true when done?
5. Approach & Decisions — the technical approach / design rationale (and, when
   present, pinned plan-time contract decisions)
6. Boundaries — what may I touch, what must I not do? Subsections: Scope,
   Non-Goals, Constraints, Prohibited Approaches, Work Posture (always shown),
   Change Boundary (always shown for implementation plans)
7. Assumptions & Risks — subsections: Assumptions, Risks / Hazards
8. Verification — how do we prove it works? Subsections: Regression Anchor,
   Validation Strategy, Definition of Done (plan-level gates only; phase
   acceptances are not restated there)
9. Execution Setup — what do I load before starting? The global setup context
   groups (Load Skills / Read Docs / Run Discovery Searches / Run Commands /
   Inspect References / Operator Notes), References, and the one-line
   Execution Feedback pointer at the typed `plan-manager log` commands
10. Phases
11. Import Provenance / Preserved Legacy Sections (only when present)
12. Plan Graph

**Phase render order:** heading → status → intent → affected areas → phase context
setup → ordered steps → expected outputs → phase validation → acceptance criteria →
risks/hazards (when present) → handoff notes (when present) → baseline scope →
references.

This order, the automatic Greenfield block, and the import-preservation rule are
covered by **golden tests** (wizard-authored render, legacy-import render, and
render → parse → render idempotence) so the renderer, parser, wizard, model, and
this document cannot silently drift apart.

## Relevant Context

`RelevantContextItem` is the execution-facing context contract. It replaces a
flat required-reading list with typed setup instructions that can be rendered by
API, CLI, UI, and Markdown without asking a small agent to infer intent.

**Skill-pack discovery is direct relevant context.** `author skill-pack` runs
`prompt-manager discover <concepts...> --type skill --json`, parses the returned
skills/read command/budget status, and upserts those skills as global
`RelevantContextItem`s. There is no candidate queue, discovery batch, accept/reject
pass, or finalization blocker. Missing skill context is a warning/advisory; the
hard execution-grade gates remain boundary, regression anchor, references (or
`NO_CODE_REFS`), and phases.

Search-hub is intentionally direct. When docs, records, code locations, or prior
work would help, the agent runs `search-hub query "<intent>" --type record,doc,skill`
itself, inspects the native confidence/attribution, and submits only durable
references or setup context back into plan-manager.

Context items may be global or phase-scoped and are classified as `skill`,
`doc`, `command`, `search`, `code_ref`, `req_ref`, or `note`. Required items
carry an explicit repeat policy such as `once_per_execution`, `on_resume`,
`every_phase`, `phase_entry`, or `as_needed`. Discovery state is auditable via
source/status fields (`authored`, `discovered`, `migrated`, `autofilled`;
`ready`, `degraded`, `unresolved`).

The repeat policy **defaults from scope**: a phase-scoped item defaults to
`phase_entry` (it is loaded each time that phase begins), a global item to
`once_per_execution`. An unset `--repeat` flag means "let the server pick the
scope default" — the CLI no longer hard-codes `once_per_execution`, which had
silently mis-set phase context. `once_per_execution` on a phase-scoped item is
contradictory and is corrected to `phase_entry`; other explicit policies are
honored. A `command`/`search` item must carry an `instruction` (and a `reason`)
before it is accepted, never after.

During authoring, discovered skill-pack items are already accepted relevant
context. Manually submitted context uses the same item model and can be updated
or removed before finalization.

Execution injects relevant context through `start`, `status`, `context`,
`resume`, `continue`, and `next`. `context` can inspect a requested phase
without changing the runner pointer; `resume` reuses an existing execution when
possible, creates one for a plan when needed, and emits on-resume plus
phase-entry setup before work continues. `continue` is the preferred small-agent
loop: it resumes or starts without advancing and returns a single recommended
next action from the API-owned `GuidedStep`.

Marking a phase `done` is validation-gated. The execution runner requires the
last stored phase validation result to be `pass` with `fresh` staleness before it
will persist `done`; degraded/offline work must pass an explicit
`validation_override.reason` through the transition request. The override is an
auditable exception, not hidden prose.

`references[]` stays separate. References are the strong validation/staleness
locator set; relevant context is the setup checklist for execution.

Finalized plans are still repairable through narrow structured mutations. Use
the persisted `plans context-list/update/remove` and
`plans reference-list/update/remove` commands to fix a bad setup item or locator
without rewriting the whole plan or editing the rendered mirror. These mutations
operate on the structured record, preserve phase identity, recompute derived
metadata, and republish the mirror from the source of truth.

Legacy `Required Reading` sections are import-only compatibility. Markdown
import preserves phase-level raw `required_reading[]` as provenance, converts
those lines into `relevant_context[]` with `source=migrated`, and renders future
views only through `Global Execution Setup` / `Phase Context Setup`.

## References

Every connected code location is captured with the machine-readable grammar from
`path:docs/reference/machine-readable-references.md`:

- `[CODE: path/to/file.go]` — existing code the plan depends on or changes.
- Proposed (not-yet-existing) code is referenced the same way, marked as
  `future` so staleness does not flag it as "deleted".
- `[REQ: OT-…]` — links a plan/phase to a requirement.

References are **connected locators**, not regex-scraped prose. The authoring
wizard recommends direct `search-hub query "<intent>" --type record,doc,skill`
when discovery would help, but it does not mirror search-hub results into
authoring state. The author inspects native search-hub confidence/attribution
and submits durable `[CODE:]`, `[DOC:]`, or `[REQ:]` locators manually, or records
`NO_CODE_REFS: <reason>` when there are genuinely no connected references. As more
Answer-projection providers (architecture-cartographer AI search, symbol/slice/
coupling leaves) register with search-hub, discovery silently improves with no
plan-manager change. The not-yet-built **unified code identifier** is a planned
drop-in upgrade for the locator; until then `[CODE:]` + the Answer projection is
the contract. References are how staleness becomes computable, so the wizard makes
them mandatory at plan and phase scope. A docs-only or process-only plan/phase may
use `NO_CODE_REFS: <reason>` instead; the reason is stored as authored context
rather than pretending the plan has connected code.

**Kind/path semantics are validated at submit time.** A reference whose declared
kind obviously contradicts its target is rejected before it enters session state,
not silently accepted and surfaced later: a documentation path (`*.md`, a
`docs/` segment) tagged `[CODE:]`, or a source file (`*.go`, `*.ts`, `*.proto`, …)
tagged `[DOC:]`, fails with an actionable message naming the right marker. The
same gate applies to phase references and to `code_ref`/`doc` relevant-context
items. Ambiguous targets (a `[REQ:]` id, a bare scenario path with no extension)
are left to the author — only an unambiguous mismatch is blocked. The same gate
runs when a suggested candidate is accepted with an inline edit, so a mislabeled
locator never reaches the references section.

## Validation, Staleness, And The Regression Anchor

Readiness/preflight is the shared execution-grade contract across drafts and
persisted plans. Author `validate`, preview guidance, finalize, and persisted
durable validation operations all consume the same readiness evaluator for deterministic plan
quality and structured setup checks. The pure quality kernel lives in
`planmodel`; the application-level readiness layer adds optional dependency
checks for CLI command validity and context/reference resolution. Deterministic
quality failures block finalize before persistence. External dependency outages
degrade honestly instead of becoming passes, so an operator can distinguish "the
plan is thin" from "the resolver is unavailable."

Persisted validation is **producer-owned**. `validate start` records a durable
Plan Manager ticket and exact producer argv; the agent starts and waits through
Git Control Tower or Test Genie using that producer's native contract, then runs
`validate sync <operation-id>` for one nonblocking typed reconciliation. Plan
Manager never waits for or recreates producer work. `validate show` only reads a
ticket. The former `validate run` and `verify-dod` routes return migration
guidance rather than executing a hidden worker.

- **Regression anchor** — **boundary-native** typed **intent** at authoring, fresh
  **snapshot** at execution start. New plans author the `change_boundary`
  strategy: affected scenarios and the tiered command set are *derived* from the
  plan's `change_boundary`, not from a hand-authored single scenario. Authoring
  records only intent (the `baseline_name` derived from the plan slug, a
  `head_sha` placeholder captured at execution start) — never a git-control-tower
  call, never stale. Derived commands are **tiered and labelled**:
  - one `git-control-tower baseline snapshot status --wait --json` +
    `git-control-tower baseline diff --wait` pair per affected scenario — these
    are verdict **oracles**;
  - one informational `git diff --stat [<head_sha>] -- <repo paths>` for the
    non-scenario allow globs — **informational only**, never a pass/fail oracle
    until a path-baseline substrate exists.

  The actual "before" collection is captured fresh when execution *starts* (a
  plan is often authored days before it runs), but never behind the agent's
  back: the runner renders one `baseline collection capture --name … --member …`
  command, Git Control Tower prints the native one-shot wait/recovery command,
  and `exec baseline-sync <execution-id>` records its typed state. The legacy
  `scenario_baseline` / `head_sha_allowlist` strategies remain
  **import/read-only** for pre-cutover plans; unstructured legacy prose is
  preserved as legacy/degraded and cannot silently become a false validation
  oracle.
- **Baseline scope** — derived from the `change_boundary` first (affected
  scenarios + repo paths), supplemented by `references[]`: the exact
  baseline/diff command set across all affected locations (not just scenarios).
- **Staleness tier** — computed from change in referenced code since authoring:
  - `fresh` — no relevant change.
  - `lightly_stale` — small diffs in referenced code.
  - `definitely_stale` — referenced locations moved or were deleted.
- **Definition of Done** — requires a fresh synchronized, selector-free final
  collection diff for the complete captured inventory; a phase subset can never
  certify completion.

## Handoff (Structured Layer Only)

plan-manager owns the **canonical, structured** handoff, assembled from state
captured in-flow during a run:

- phase status (→ full/partial + resume point), `last_validation`, and staleness.
- a compact `LogSummary` plus the run's `log_entries` (decisions, candidate
  findings, bug reports, records, notes), read from the `log` domain through the
  read-only `LogLedger` seam — the handoff no longer embeds separate
  `decisions`/`candidate_findings` lists.
- Findings are **candidate / unvalidated**; an operator triages or promotes them
  (`log promote`) before they become real bugs.

It does **not** own the agent's free-text "prose" final handoff — that catch-all
is captured by the orchestration layer (agent-manager run transcript →
swarm-manager operating mode) and linked to the plan by reference. See
[`INTEGRATIONS.md`](INTEGRATIONS.md) for the ownership split.

## The Execution-Log Ledger

While a plan executes, an agent produces typed work products. These are **not**
plan/phase fields — they are entries in the `log` domain's single durable ledger
(`api/internal/planlog/`, table `log_entries`). One table holds five DISTINCT
entry types that must never be conflated:

| `LogEntryType` | Meaning | Downstream |
|---|---|---|
| `decision` | An in-flow design decision (feeds the handoff). | local-only |
| `finding` | An **unvalidated** candidate observation (a possible bug). | local-only |
| `bug_report` | A defect **deliberately filed** to the issue tracker. | forwarded → scenario-qa |
| `record` | Reusable learning/work captured for the learning loop. | forwarded → swarm-manager |
| `note` | A lightweight progress/context note. | local-only |

A finding is **not** a bug and a bug is **not** a record. Findings file as
triage `candidate` and are never auto-promoted.

**Sync status** (`LogSyncStatus`) tracks downstream forwarding for the two
forwarded types: `local` (no downstream target), `pending` (created but the
downstream was unavailable), `synced` (forwarded with a `DownstreamRef`),
`sync_failed` (a downstream rejection or non-unavailability error). A
failed/pending sync is **never fatal**: the local entry persists and is retried
with `plan-manager log sync`.

**Severity** (`LogSeverity`, optional on findings/bug reports): `info`, `low`,
`medium`, `high`, `critical`.

**DownstreamRef** records the result of forwarding an entry — `system`
(`scenario-qa` | `swarm-manager`), `kind`, the downstream `reference`/id once
synced, last `detail`, and `synced_at`.

**Idempotency & dedup.** Every entry carries an optional `idempotency_key`; a
retry with the same key returns the existing entry instead of creating a
duplicate. Findings/decisions without an explicit key also dedup by
(execution, attribution run id, type, normalized title). Both are enforced by
partial UNIQUE indexes in the schema, so concurrent retries cannot double-file.

**Supersession & promotion.** `supersedes_id` links a correction to the entry it
corrects. `PromoteEntry` turns a finding into a `bug_report` or `record` while
**preserving** the original finding (marked triage `promoted`) and linking the
new entry back via `promoted_from_id` — promotion never mutates or discards the
finding.

The neutral Go types live in `api/internal/planmodel/log.go` (`LogEntry`,
`LogSummary`, the enums, `SummarizeLog`) so the execution domain can read a
compact summary through a seam without importing the log package. The wire types
live in `packages/proto/schemas/plan-manager/v1/shared/model.proto`
(`LogEntry`/`LogSummary`/`DownstreamRef` + the enums) and the service in
`packages/proto/schemas/plan-manager/v1/log/log.proto` (`LogService`).

## Invariants

These are load-bearing rules. Tests enforce them; changing them is a deliberate
contract change, not an incidental edit.

1. **Markdown is a view, not the source of truth.** The structured record is
   authoritative; rendered Markdown is always projected from it and is never
   parsed back into truth during normal operation. Import is the only parse path,
   and it produces a structured record (adoption), not a live edit channel.
2. **Work posture is runtime-derived, not agent-authored.** The Greenfield/
   Brownfield block is injected from `work_posture`; the wizard never asks an
   author to write it. Posture derivation is centralized behind one seam and
   covered for every `maturity` enum value.
3. **Unknown imported sections are mapped or preserved, never silently dropped.**
   Every legacy section either maps to a canonical field or lands in
   `preserved_legacy_sections[]`.
4. **Wizard-required sections and renderer sections stay in lockstep.** Every
   mandatory authoring field has a renderer section (or a documented non-rendered
   reason), and every renderer section the renderer emits is recoverable by the
   parser. Golden tests fail if a required field stops rendering or the renderer
   emits a section the parser cannot recover.
5. **`acceptance` ≠ `validation`.** A phase's outcome gate and its checking method
   are distinct fields and must not be identical.
6. **The change boundary is the blast-radius source of truth, expressed as
   `acceptance_allow` / `acceptance_deny`.** There is no public `scope` field and
   no `primary_scenario` / `affected_scenario` flag. Scenario identity is derived
   from the allow globs; `acceptance_deny` is a guardrail that never widens
   validation scope; finalized boundary/anchor data may not contain unresolved
   `<placeholder>` tokens.
7. **Posture is aggregate and conservative.** It is derived from every scenario
   the boundary and references touch — if any affected scenario is pilot/
   production/sunset the plan is Brownfield. No code path uses "first scenario
   wins."
8. **Non-scenario repo/path diffs are informational, never oracles.** Validation
   classifies scenario baseline checks as oracles and repo/path diffs as
   informational until a path-baseline substrate exists; a repo diff never
   produces a false PASS.

The code/proto surfaces that must stay in lockstep with this document:
`packages/proto/schemas/plan-manager/v1/shared/model.proto` (wire contract),
`api/internal/planmodel/types.go` (neutral kernel), `api/internal/planproto`
(conversion), `api/internal/plans/render.go` (renderer),
`api/internal/planmodel/parse.go` (parser), `api/internal/plans/posture.go`
(posture derivation), and `api/internal/authoring` (wizard order + gates).

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — which domain owns each part of this model
- [`DATA.md`](DATA.md) — how the model persists (durable home store)
- [`FLOWS.md`](FLOWS.md) — authoring, execution, and handoff lifecycles
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — substrate this model composes
- [`../../PRD.md`](../../PRD.md) — operational targets
