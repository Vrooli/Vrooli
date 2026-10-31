# Error Handling — Nutrition Planner

## Shared contract

Read the [Error Handling shared guide](/scenarios/template-manager/docs/internal/ERROR-HANDLING.md)
first. Template Manager owns the common transport contract: the Connect error
envelope, the typed sentinel-to-status mapping, the "never leak internals"
rule, and the logging correlation guidance. This document only records what is
different or additional for Nutrition Planner.

Scenario-specific authority: [`../reference/product-specification.md`](../reference/product-specification.md)
redesign R07, R17.1, R18, R21.3, R22, and R26, plus Appendix A §17.5
(`API-01`/`API-02`), §16.6 (`JOB-01`), §11 (`UX-DAT-02/03`), §18
(`SYS-01/02/03`), and §14.6 (`PLN-06`).

**Current state (audit 2026-09-22).** The codes below are the contract, not a
description of today's behavior. Two known violations are recorded as blockers
in [`REDESIGN_PLAN.md`](REDESIGN_PLAN.md) §3.2: a fresh workspace's profile
lookup returns a raw `sql.ErrNoRows` that surfaces as a generic internal error
instead of `NOT_FOUND` or an explicit unconfigured state (B2), and swap/feedback
on a workspace with no saved plan fail with an internal error while decoding an
empty plan (B3). Every page also fails with `unauthenticated` because the local
runtime has no principal (B1) — that is a configuration defect, not an
`UNAUTHORIZED` outcome a user should ever see in personal mode.

## Scenario details

### Two error planes, never conflated

The most consequential rule in this scenario is that a **valid request can fail
to produce a plan without being a transport error**. Separate:

- **Transport errors** — the request was malformed, unauthorized, missing, or
  conflicted with a revision. These map to a typed status and a stable code.
- **Domain outcomes** — the request was valid but the domain cannot give a
  complete answer: no feasible plan in the search budget, a target that is
  `unknown` because contributors are missing, a required-constraint conflict,
  or an infeasible import. These are **data**, not exceptions, and must never
  become a generic `500`, an empty success, or a silent default.

A bounded-search timeout means "no feasible plan found in this search," not
"proven infeasible." Only a directly contradictory rule set or an exhaustive
finite search may claim infeasibility, and the actual evidence must accompany
the claim (`PLN-03`/`PLN-06`, `ACT-021`/`ACT-047`).

### Stable error codes

Use these codes on the wire (envelope from `API-02`); do not invent per-endpoint
synonyms. User-facing copy is a localized projection, never the raw code.

| Code | Meaning | Retryable | Notes |
|---|---|---|---|
| `VALIDATION_FAILED` | Input rejected semantically | No | Return per-field paths; preserve all valid edits (`ACT-010`) |
| `UNAUTHORIZED` | No valid server session | No | Never expose another workspace's existence (`SYS-03`) |
| `NOT_FOUND` | Entity absent in the authorized workspace | No | Same response whether absent or foreign-owned |
| `REVISION_CONFLICT` | Expected revision no longer current | No | Return saved revision + local draft for compare/reapply (`SYS-02`) |
| `STALE_INPUTS` | A pinned input (profile, price, recipe, stock, plan) changed after a draft was generated | No | Return conflicts and `suggestedAction: revalidate_draft` (`ACT-019`) |
| `REQUIRED_CONSTRAINT_FAILED` | A required rule, equipment, lock, or explicit cap blocks the action | No | Name the rule and the affected refs; never silently relax it (`PLN-02`) |
| `MISSING_REQUIRED_EVIDENCE` | A required target/eligible claim lacks source evidence | No | Unknown evidence is not absence (`NUT-03`, `ACT-004`) |
| `UNSUPPORTED_IMPORT_VERSION` | Import schema/format not supported | No | Name the detected format/version and the supported set (`UX-DAT-02`) |
| `IMPORT_LIMIT_EXCEEDED` | Import exceeded a configured byte/entity limit | No | State the limit and the observed size (`UX-DAT-02`) |
| `PROVIDER_UNAVAILABLE` | A configured external provider cannot be reached | Yes | Report which provider, last success age, and that manual entry still works (`DATA-01/02`) |
| `BUDGET_EXCEEDED` | Per-workspace job/AI budget exhausted | No | Existing data and manual paths stay fully usable (`JOB-01`) |
| `OPERATION_CANCELED` | A job/operation was canceled | No | A late external response is retained as canceled evidence, never applied (`JOB-01`) |

`BUDGET_EXCEEDED`, `PROVIDER_UNAVAILABLE`, `STALE_INPUTS`, and
`REVISION_CONFLICT` also carry the redesign's new operations (R21.3): starting
image generation beyond the reserved cap returns `BUDGET_EXCEEDED` before any
dispatch (R18.3); an unconfigured or failing image-tools or personal-planner
adapter returns `PROVIDER_UNAVAILABLE` and the capability endpoint reports the
disabled state so the UI shows an honest explanation instead of a dead button
(R21.3, AT-043); applying an Explore suggestion, a swap, or a shopping diff
against a changed plan returns `STALE_INPUTS` or `REVISION_CONFLICT` with a
refreshed choice, never an overwrite (R11.3, R14.3). No new wire codes are added
for the redesign.

### Identity, idempotency, and retry safety

- Every mutation carries an **operation identity** and an **idempotency key**
  scoped to workspace + operation type. Repeating the same key with the same
  payload returns the original result; the same key with a materially different
  payload returns a conflict — never a silent overwrite (`DOM-07` #12,
  `FIX-07`, `ACT-053`).
- A commit whose response was lost is resolved by the key or by re-reading the
  operation result — never by blindly creating a second purchase, consumption,
  import, or replan (`SYS-01`, review script G).
- A **conflict is distinct from a network error** and must be presented
  differently, because the recovery is different (`SYS-01`).
- The following effects are all-or-nothing: profile activation, plan apply,
  import/restore apply, purchase plus stock update, preparation plus batch
  creation, and intake correction plus batch adjustment (`SYS-02`). If the
  platform spans stores or queues, record a durable operation and use a tested
  outbox/compensation strategy — do not call unrelated writes atomic.

### Media failures are fallbacks, not errors

When a meal image is missing, fails to load, belongs to the wrong appearance, or
is incompatible with the recipe revision, the presentation chooser moves to the
next eligible treatment (scene → cutout → editorial → minimal) without a broken
image icon, a toast, or a layout jump (R17.1, R26). Record a **safe diagnostic**
— asset id, treatment attempted, fallback reason code, counts — for
observability; never a signed URL, private file path, or recipe text. The user
sees a polished meal, not an error.

### Offline and sync conflicts

Offline shopping checks, manual items, and timer controls are queued in the
outbox with their base revision and desired state (R22). On reconnect:

- A replay whose desired state already holds is a no-op, not a conflict (two
  devices checking the same row must not toggle it twice, AT-035).
- A changed requirement (for example more rice needed after the row was picked
  up) keeps the fulfilled amount and surfaces the extra need as a reviewable
  item, not a `REVISION_CONFLICT` dialog (R14.3, AT-034).
- A true conflict keeps the local edit visible with a pending marker and an
  actionable resolution; it is never silently discarded.
- The UI distinguishes cached data, local pending actions, and server-confirmed
  saves (R07). A failed load is never shown as an empty account.

### Undo and correction are not errors

Undo is a **new operation with a known precondition**, not a state rewind. When
later dependent actions prevent exact reversal, return a correction path and
explain the affected facts. Preserve the original event and its correction link
for auditability (`SYS-02`, `INV-02`, `ACT-050`).

### Solution- and data-quality messages

A suggestion should be concrete and actionable, not a dead end:

- "The soy milk has no B12 value recorded. Add its label or leave B12 coverage
  unknown." (`NUT-06`)
- "No reviewed soy-free candidate exists. Add a matching meal or review the
  restriction." (`PLN-06`)
- "Your food settings changed after this draft was created." (`STALE_INPUTS`)

Prefer "Fits your current rules" over "Safe for your allergy," and never turn
an `unknown` into a reassuring zero or a green badge (`NUT-03`, `UX-VIS-04`).

### Import and restore failures

- Parse and preview must not mutate live data. Apply all selected valid changes
  in one transaction; a multi-record import with invalid entries defaults to
  rejecting the whole import, with an explicit "Import valid records only" that
  shows exactly what is omitted and preserves rejected content for correction
  (`UX-DAT-02`, `ACT-026`).
- A restore validates all references and creates a **recoverable checkpoint
  before** replacing domain content; on failure the previous workspace remains
  intact and the job is resumable/idempotent with a terminal outcome
  (`UX-DAT-03`, `SYS-05`, `ACT-028`).
- Never identify duplicates by name alone; same-ID different-content imports use
  the chosen conflict policy with references remapped (`ACT-027`).

### What must never appear in an error

- Another workspace's existence, content, or identifiers (`SYS-03`).
- Provider tokens, full private label images, raw recipe text, receipt contents,
  or medical notes in logs (`SYS-04`).
- A fabricated numeric default, a fake live price, or a success toast without a
  corresponding persisted result (`UX-VIS-04`, `DEC-05`).

### Open implementation decisions

The typed sentinel names, the exact HTTP/Connect status mapping, and the error
envelope's JSON field names are selected during repository fit (`ARCH-01`) and
recorded back into [`DECISIONS.md`](DECISIONS.md). This document fixes the
codes and semantics; it does not fix the transport spelling.
