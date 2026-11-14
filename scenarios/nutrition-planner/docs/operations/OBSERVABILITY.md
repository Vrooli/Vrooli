# Observability — Nutrition Planner

This document records logs, metrics, telemetry, health checks, and
business/product signals for the scenario.

The working product name is **Nooch** (formerly Daily; decision D-042). Scenario id is `nutrition-planner`.

## Purpose Of This Document

Use this document to answer:

- What signals tell us the scenario is healthy?
- What signals tell us users are getting value?
- Which logs or metrics should an operator inspect first?
- What telemetry gaps remain before deployment or monetization?

**The rule that shapes this document.** This scenario holds private personal data — what a person eats, their
dietary restrictions and allergies, supplement schedules, body-related
targets, receipts, and prices. Observability must prove the system is
healthy and honest **without becoming a second, less-protected copy of that
data**. `SYS-04` and `OPS-02` are therefore constraints on what may be
emitted, not just product features: logs carry operation IDs, timings,
counts, error codes, and safe revision references, and deliberately omit
recipe text, receipt contents, labels, medical notes, and access tokens.

The same rule applies to the product's own success signals: they are
collected only with consent, in aggregate, and are never estimated or
fabricated to look better.

## Signals

| Signal | Type | Source | Purpose | Threshold |
|---|---|---|---|---|
| API `/health` | health | API | Process liveness and API readiness | healthy under the lifecycle health check |
| UI `/health` | health | UI server | UI bundle/server reachability | responds during the lifecycle health check |
| Database access | health | API | SQLite reachable and readable | any failure marks the API unhealthy, not "empty data" |
| Migration version | health | API boot | Recorded schema version matches the binary's expectation | mismatch is a named startup failure (`SYS-05`) |
| Job execution | health | job adapter | Jobs are making progress and not stuck | no job `running` past its declared budget |
| Provider configured vs unavailable | health | provider/AI adapter | Distinguish "not configured" (supported) from "configured but failing" (degraded) | `not_configured` is not an outage; `unavailable` is surfaced with a reason |
| test-genie result | validation | `make test` | Scenario correctness evidence | required phases pass |
| Authenticated request succeeds | health (synthetic) | `nutrition-planner workspace list` or a `ListWorkspaces` call | `/health` stays healthy while every workspace RPC returns `401` today (blocker B1); only a real authenticated call proves the app is usable | any `unauthenticated` in the local personal mode is a configuration failure |
| Capability report (redesign) | health | capabilities/diagnostics endpoint | What is actually configured — image generation, personal-planner link, assisted import — so the UI never shows a dead button (R21.3, R25.4) | reports configured state without leaking credentials; the current capabilities registry still advertises the template's `audio-tools` entry |

`not_configured`, `unavailable`, and `never_run` are deliberately distinct.
Collapsing them would be the same class of error as showing a missing
nutrient as zero, and it would hide the manual-fallback path the product
depends on.

## Logs

| Log | Source | How To Read | Details |
|---|---|---|---|
| API request/operation log | lifecycle-managed API | `make logs` | One entry per operation: operation ID, operation type, workspace-scoped safe reference, outcome, duration. Request logging uses the deterministic clock seam in tests. |
| Job run log | job adapter | `make logs` | One entry per run: job type, state transition, attempt count, counts, duration, safe error code. |
| UI server log | lifecycle-managed UI | `make logs` | Production bundle server logs only. |
| AI/provider request log | adapter | `make logs` | Provider name, task type, outcome, token/request counts, budget state. **Never the prompt or response body.** |
| Audit-equivalent domain record | database, not a log file | CLI/UI query | The durable event history (purchases, consumption, corrections, stock). Not subject to log retention. |

**Correlation.** Every user-visible error and every job carries an
`operationId`. A support conversation should be answerable by that ID
alone, so the user never has to paste their diet into a bug report.

**Prohibited in logs at any level below a deliberate bounded debug
capture:** recipe text, ingredient names, receipt contents, product/label
images or OCR text, dietary restrictions or allergy details, supplement
schedules, nutrient totals attributable to a person, access tokens,
provider credentials, full generation prompts containing user details,
private food photos, and signed media URLs (R25.4). A failed parse logs the error class and source type,
never the payload that failed.

## Metrics

| Metric | Status | Details |
|---|---|---|
| Requirement coverage | active | Tracked through `requirements/` and test-genie coverage artifacts. |
| Operation duration | required | Per operation type and per job; the basis for the performance budgets in [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md). |
| Validation categories | required | Counts by error code (`VALIDATION_FAILED`, `REQUIRED_CONSTRAINT_FAILED`, `MISSING_REQUIRED_EVIDENCE`, `STALE_INPUTS`, `REVISION_CONFLICT`, etc.). Counter labels only; no field values. |
| Stale-write conflicts | required | Rate of `REVISION_CONFLICT` / `STALE_INPUTS` by operation type. A rising rate means the UI is editing against old revisions. |
| Calculation version | required | The versioned nutrition/planner policy in force. Assessments are keyed by input revisions and evaluator version (`ARCH-03`). A version change must be observable, not silent. |
| Import counts | required | Created/updated/skipped/conflicted per staged import; a non-zero `conflicted` count is a normal, reviewable outcome, not an error. |
| Job retries and failures | required | Retry counts, terminal failures, cancelations, and budget-exceeded events. |
| Provider/availability by adapter | required | Label by adapter and reason class only. Backs `OPS-02` and `ACT-060`. |
| Generation jobs (redesign) | required when D7 lands | Count and duration by state (queued, running, awaiting_review, approved, failed, canceled, rejected), attempts, and dedup hits (R18.2, R25.4). Job and asset ids only — never the prompt text when it contains user details. |
| Budget reservations (redesign) | required when D7 lands | Reserved, settled, released, and pending-unknown amounts per period and unit; a reservation that never settles must remain visible (R18.3). |
| Media fallback reasons (redesign) | required | Counts by treatment chosen and fallback reason code (missing asset, load failure, wrong appearance, incompatible revision, rejected). Asset ids only; never signed URLs or private paths (R17.1, R25.4). |
| Sync conflicts and outbox replays (redesign) | required | Outbox depth, replay outcomes (applied, no-op, conflict), and conflict rate by operation type (R22). |
| Calendar link sync (redesign) | required when D7 lands | Link states (pending, linked, failed, conflict), retries, and reconcile-by-key hits for the personal-planner adapter (R24.3). |
| Planner timing | required | Search duration, candidate counts, and budget-hit outcomes per run (R25.4). |
| Product activation | deferred | Collected only with consent and only in aggregate; see below. |
| Cost telemetry | deferred | Only meaningful once R2 provider/AI usage exists; bounded per workspace by `JOB-01`. |

## Alerts / Health

Lifecycle health checks cover API and UI. Beyond those:

- **A provider `not_configured` is not an alert.** It is the supported
  default at R0 and must not appear as an error or an outage.
- **A provider `unavailable` is a user-facing degraded state, not a page.**
  It belongs on the affected surface with a reason and an age, alongside
  the manual fallback. Paging an operator for a flaky third-party API
  trains them to ignore alerts (`OPS-02`).
- **A partial or unknown assessment is not an error.** It must not
  increment an error metric or emit an error-level log, or normal honest
  operation will look like an outage (`NUT-03`).
- **Duplicate application of a purchase, consumption, import, or replan is
  a real alert.** Idempotency is broken if it happens; only reversal or
  correction can repair it.
- **A migration version mismatch is a real alert.** It means the binary and
  the database disagree and should fail closed at startup (`SYS-05`).
- **A job stuck `running` past its budget is a real alert.** It blocks
  progress and may hold stale inputs.

**What "users are getting value" looks like here.** Deliberately narrow, because the honest answer is that most value signals
require personal data this scenario should not emit. These signals are
collected **only with consent**, in aggregate, and are never faked or
backfilled with fixture numbers:

| Question | Signal | Notes |
|---|---|---|
| Is setup survivable? | Time and interaction count to a first usable plan | The headline usability goal (`C02`, `DEC-05`); if setup is abandoned, nothing else matters. |
| Do recommendations land? | Acceptance rate and voluntary swap reasons, with an explicit unknown bucket | Missing feedback stays unknown, never treated as noncompliance (`DOM-07` #13). |
| Does it displace the default? | Planned versus recorded spend, kept as two labeled numbers | Never presented as proven savings without a comparable baseline (`CST-03`). |
| Is nutrition honest? | Coverage and missing-data status per configured target | Partial coverage must stay visible; a complete-looking number over unknowns is the failure mode this scenario exists to avoid (`NUT-03`). |
| Is the low-maintenance path working? | Scheduled drafts produced; redundant questions dismissed | A scheduled run never replaces an accepted plan or marks food eaten (`JOB-02`). |

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| No backup target is registered for the SQLite database | A non-regenerable dataset has no automated durability | Register with `data-backup-manager` before real use; see the Runbook. |
| No operation-duration metric yet | The ~2s search budget cannot be checked in production | Build with the first real domain, not after. |
| No calculation-version signal yet | A policy/algorithm change could alter assessments unnoticed | Required before the nutrition/planner versions diverge. |
| No provider availability metric yet | `OPS-02` and `ACT-060` cannot be checked | Build with the R2 adapter boundary. |
| Health reports healthy while the product is unusable | Every workspace RPC returns `401` in the local runtime; lifecycle health checks never notice | Add the authenticated synthetic check above when the authentication profile lands (B1). |
| Settings data health calls the wrong route | The diagnostics panel fails with a parse error, so data-health findings are invisible | Fix blocker B6 (`/api/v1/diagnostics`). |
| No media, generation, reservation, outbox, or calendar signals | Redesign failure modes (silent fallbacks, runaway spend, stuck replays, duplicate events) would be invisible | Build each signal with its surface (REDESIGN_PLAN D4–D7). |
| Product usage telemetry | Cannot validate adoption or monetization | Before any external claim, and only in a consenting, aggregate, no-personal-content form. |
| Cost telemetry | Cannot evaluate provider unit economics | Before R2 budgets are enabled for real. |
| No signal distinguishing "no data entered" from "adapter failed to load" | An empty collection and a broken pipeline can look alike | Before the first non-manual data source. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates and release checklist
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business validation signals
- [`../business/GO-TO-MARKET.md`](../business/GO-TO-MARKET.md) — validation experiments
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — what must never be emitted
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — budgets and measurements
- [`../reference/product-specification.md`](../reference/product-specification.md) — R17.1, R18, R22, R24, R25.4, and Appendix A sections 2.3, 16.6, 18.4, and 19.2
