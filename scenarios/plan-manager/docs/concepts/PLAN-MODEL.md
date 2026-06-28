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
| `relevant_context[]` | authored + discovered | Plan-wide setup items a fresh/resumed agent should load, inspect, or run before execution. |

## Phase

A `Phase` is a **first-class object**, not a checklist line. This is the unit the
runner walks and the unit context is scoped to.

| Field | Authored / Computed | Meaning |
|---|---|---|
| `id`, `order` | computed / authored | Identity and sequence. |
| `title` | authored | Short phase name. |
| `intent` | authored | What this phase accomplishes. |
| `required_reading[]` | migration input | Legacy phase reading preserved when importing older plans; execution-facing setup uses `relevant_context[]`. |
| `reminders[]` | authored | General + phase-specific reminders, surfaced just-in-time during the phase. |
| `references[]` | authored + resolved | Phase-scoped connected-code, requirement, or document references. A phase with no connected references must carry an explicit no-code reason during authoring. |
| `relevant_context[]` | authored + discovered | Phase-scoped setup items with kind, reason, instruction, command/argv, target, required flag, repeat policy, source, and status. New authored phases must include at least one phase-scoped relevant-context item or an explicit `NO_CONTEXT: <reason>` note. |
| `baseline_scope` | computed | The set of connected-code locations this phase touches → the exact baseline/validation command set. |
| `acceptance` | authored | Objective pass/fail for this phase. |
| `status` | computed | `todo` → `active` → `done` / `blocked`. A typed transition or inferred from acceptance + validation passing. |
| `last_validation` | computed | Most recent validation result + staleness factor for this phase. |

Decisions, findings, bug reports, records, and notes are **not** phase or
execution fields — they are typed entries in the `log` execution-log ledger (see
[The execution-log ledger](#the-execution-log-ledger) below), scoped to a
plan/execution/phase. The execution runner reads a compact `LogSummary` of them
for context and the canonical handoff; it does not store them on the phase.

**Resume point** = the earliest phase whose `status` is not `done`. Full vs.
partial completion is derived from the phase-status set — never narrated.

## Relevant Context

`RelevantContextItem` is the execution-facing context contract. It replaces a
flat required-reading list with typed setup instructions that can be rendered by
API, CLI, UI, and Markdown without asking a small agent to infer intent.

Context items may be global or phase-scoped and are classified as `skill`,
`doc`, `command`, `search`, `code_ref`, `req_ref`, or `note`. Required items
carry an explicit repeat policy such as `once_per_execution`, `on_resume`,
`every_phase`, `phase_entry`, or `as_needed`. Discovery state is auditable via
source/status fields (`authored`, `discovered`, `migrated`, `autofilled`;
`ready`, `degraded`, `unresolved`).

During authoring, discovered context starts as `ContextCandidate` session state.
Candidates carry the proposed `RelevantContextItem`, discovery concept, source,
degraded detail, and pending/accepted/rejected status. Only accepted candidates
finalize into plan or phase `relevant_context[]`; rejected candidates remain an
authoring audit trail and do not affect execution.

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

References are resolved against `code-facts`. The not-yet-built **unified code
identifier** is a planned drop-in upgrade for the locator; until then `[CODE:]` +
code-facts is the contract. References are how staleness becomes computable, so
the authoring wizard makes them mandatory at plan and phase scope. A docs-only
or process-only plan/phase may use `NO_CODE_REFS: <reason>` instead; the reason
is stored as authored context rather than pretending the plan has connected code.

## Validation, Staleness, And The Regression Anchor

- **Regression anchor** — captured *before* changes (a `git-control-tower
  baseline` snapshot for scenario-scoped work, or a `HEAD` sha + file allowlist
  for outside-scenario work). Auto-filled by the wizard; stored on the plan as
  typed fields (`strategy`, `scenario`, `baseline_name`, `head_sha`,
  `allowlist_paths`, generated `commands`). Rendered markdown is imported back
  through those fields; unstructured legacy prose is preserved as legacy/degraded
  and cannot silently become a false validation oracle.
- **Baseline scope** — derived per phase from `references[]`: the exact
  baseline/diff command set across all affected locations (not just scenarios).
  Typed anchor fields can also derive their own command set when stored commands
  are absent.
- **Staleness tier** — computed from change in referenced code since authoring:
  - `fresh` — no relevant change.
  - `lightly_stale` — small diffs in referenced code.
  - `definitely_stale` — referenced locations moved or were deleted.
- **Definition of Done** — verified against the regression anchor as an oracle
  (baseline diff exit 0), not a narrated claim.

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
forwarded types: `local` (no downstream target), `pending` (created, not yet
forwarded — the v1 default until a downstream is wired), `synced` (forwarded with
a `DownstreamRef`), `sync_failed` (a forward attempt errored). A failed/pending
sync is **never fatal**: the local entry persists and is retried with
`plan-manager log sync`.

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

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — which domain owns each part of this model
- [`DATA.md`](DATA.md) — how the model persists (durable home store)
- [`FLOWS.md`](FLOWS.md) — authoring, execution, and handoff lifecycles
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — substrate this model composes
- [`../../PRD.md`](../../PRD.md) — operational targets
