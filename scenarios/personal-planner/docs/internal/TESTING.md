# Testing — Personal Planner

## Shared guidance

- [Test authoring standard](/docs/testing/UNIT-TEST-AUTHORING.md): boundaries,
  fixtures, and independently justified expectations.
- [Execution and validation scope](/docs/TESTING.md): focused checks and Test Genie.
- [Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md): API, UI, CLI,
  cancellation, workflow replay, and coverage configuration.

The recipes describe template mechanics. This guide owns local behavior, test
prerequisites, fixtures, and exceptions; local test sources and configuration
identify the helpers and gates this scenario currently uses.

## Scenario-specific testing

**IMPLEMENTATION-STAGE.** The strategy below remains the target verification
approach from the implementation plan §25. The first product-owned `work`,
persisted `focus` session, explicit `goals`, and measured `review` read-model
domains now have real unit, SQLite, Connect, CLI, and UI evidence. The
fixtures (F01–F16) and acceptance cases
(T01–T36) named here are the **target evidence map** — the desired
behaviors these tests must eventually prove — not tests that currently
run. Every test asserts DESIRED/EXPECTED behavior (the honest-planning
invariants), not whatever the implementation happens to do. This scenario
is workflow-heavy; see [`../concepts/FLOWS.md`](../concepts/FLOWS.md) for
the flows under test and their maturity ladder, and
[`../internal/SEAMS.md`](SEAMS.md) for the substitution boundaries every
layer below relies on.

**No real user data is ever touched.** All fixtures use recorded or
synthetic data; no real calendar account, provider connection, or
identity is exercised. Read-only provider adapters are tested against
canned responses, never a live account.

### Layer 1 — Pure domain / property tests (fast, no I/O)

The scheduling/capacity/forecast kernel is pure and deterministic, so its
strongest evidence is property tests over the §25.3 invariants. These run
with an injected fixed clock/timezone (the determinism seam) and no
database:

- **Interval algebra** — normalization is idempotent; union duration
  never exceeds the sum of component durations; adjacent half-open
  intervals do not overlap; subtraction never yields a negative duration.
- **Effort conservation** — a split/move conserves total active demand
  except through an explicit effort revision; excess over remaining
  effort is flagged as overallocated (fixtures F02, F04).
- **Recurrence identity** — routine exceptions keep a stable
  original-occurrence key after rescheduling; a moved instance carries
  that key alongside its new start (fixtures F09, F12).
- **Dependency acyclicity** — cycles are rejected; edges stay scoped to
  accessible same-workspace records or authorized source projections.
- **Deterministic proposals** — every accepted change satisfies hard
  constraints under the proposal's validated snapshot; repeated
  projection ingestion converges to the same authoritative state;
  candidate selection follows the documented tie-break order (retain
  feasible placement → meet constraints → reduce fragmentation → fit
  focus windows → avoid unnecessary movement), seeded for reproducibility.
- **Forecast non-mutation** — forecast generation leaves authoritative
  rows unchanged; it never rewrites `promised_finish`, protected
  availability, or accepted session placement (fixtures F05, F06, F15).
- **Correction, sharing, and session invariants** — a correction changes
  effective totals without deleting the original measurement (except via
  the explicit deletion workflow); removing a permission can only reduce
  or maintain visible data; any successful concurrent start sequence
  leaves at most one current exclusive session per person.

### Layer 2 — Real-DB integration tests

Repository and command tests run against a real SQLite handle (per the
per-domain repository seams) to pin behavior the fakes cannot:

- **Transactions / atomic apply** — a split/move, session transition,
  commitment revision, or proposal apply commits its allocation changes,
  conservation checks, revision bump, history, and outbox record
  together, or not at all (INV-11).
- **Unique constraints** — including the database-enforced
  **one-exclusive-session guard** (partial unique on person where state
  ∈ {running, paused} and attention = exclusive under Postgres; an
  equivalent transactional guard under SQLite); a preflight `SELECT`
  alone is insufficient (fixture F10).
- **Tenant isolation** — every read/write/job/projection/cache is scoped
  to the authorized workspace and subject (INV-01); a foreign subject
  cannot read another's records.
- **Idempotency** — a retried command (scoped by caller + command type +
  workspace, fingerprinted) returns the prior outcome and never a
  duplicate human-facing effect (INV-10); reuse of a key with a different
  payload is an error.

### Layer 3 — Adapter contract tests

Source and provider adapters are tested against recorded/synthetic
responses through their seam interfaces:

- **Source-app adapter** — normalized `SchedulingIntent` ingestion,
  revisioned projections, replay/out-of-order convergence, and stale
  source-constraint detection (fixtures F08, F16 replay paths).
- **External-calendar provider adapter (read-only)** — not part of the local
  accepted-allocation slice; provider integration remains future work.
  incremental reads, deletion reporting, moved recurring instance keeping
  its original-instance key, invalidated-cursor forced scoped resync, and
  reauth-required surfacing (fixtures F07, F09). Read-only is asserted at
  both requested scope and adapter behavior.
- **notification-hub delivery** — dedup/idempotent retry with quiet hours
  and no private reason in shared delivery (fixture F11 revocation, T25).

### Layer 4 — End-to-end suite

A small e2e suite covers the load-bearing loops end to end:

- the **planning → focus → review** loop (T01, T09, T14–T17), and
- **sharing security** (recipient binding, exact field mask, ID-guessing
  blocked, nested-leak blocked, expiry, revocation — fixture F11, T24).

### Evidence map: fixtures and acceptance cases

The canonical fixtures **F01–F16** (capacity/deductions, demand
conservation, fragmentation, timer-vs-actual, forecast shift, competing
initiatives, unknown/passive work, source-constraint change, provider
reset/recurrence, two-device sessions, selective sharing, timezones/DST,
honest learning, visual geometry, commitment revision, forgotten timer)
are the deterministic inputs shared across layers. The acceptance cases
**T01–T36** map those fixtures to end-to-end expectations across
foundation/persistence (T01–T02), capacity/conservation (T03–T05),
dependencies/proposals (T06–T10), forecasts/shared capacity (T11–T13),
focus/actuals (T14–T17), learning/source integration (T18–T21),
provider/temporal (T22–T23), sharing/notifications (T24–T25), what-if
side-effect isolation (T26), export/import lifecycle (T27–T28),
visual/accessibility/responsive (T29–T31), reliability/CLI parity
(T32–T34), and migration/optional-assistance (T35–T36). Each T-case is
the acceptance target the layers above must satisfy; link real local
tests to their fixture and T-case as they are added.

Record the required external services and any exceptions to the shared
harness here as they appear. Link to real local tests as they are added.
Keep execution receipts with their owning plan.

## Outcome-evidence inventory

| Outcome | Producer / test | Current evidence | Boundary |
|---|---|---|---|
| Work validation and persistence | `api/internal/work/service_test.go`, SQLite tests | `go test ./...` passes in `api` | Work domain |
| Work transport and endpoint parity | `api/handlers/work/*_test.go`, `make endpoints` | API tests and endpoint generation pass | API ↔ proto |
| CLI capture and plan projection | `cli/domains/work`, generated CLI evidence | CLI tests pass; live `personal-planner work plan --json` returns persisted work | CLI ↔ API |
| Today live work, capacity, and focus rendering | `ui/src/pages/DashboardPage.test.tsx` | UI test passes against work recommendations, Calendar accepted-allocation data, and durable FocusService start/pause/error paths; empty schedule state remains explicit | UI ↔ Work + Calendar + Focus read models |
| Plan accepted-allocation geometry and capacity summary | `ui/src/pages/PlanPage.test.tsx`, `CalendarService.ListTodayAllocations/ListAllocations/CreateAllocation` | UI tests pass for positioned allocation, empty state, chosen work item, custom start/duration, placement action, Week projection, and outage; capacity is measured from accepted calendar rows | UI ↔ Calendar read/write model |
| Focus session lifecycle and honest active-time accounting | `api/internal/focus/service_test.go`, `ui/src/pages/FocusPage.test.tsx`, CLI domain evidence | API and CLI suites pass; focused UI tests cover start, pause/resume, and failed transition reporting | Focus ↔ Work item |
| Manual actual capture and correction provenance | `api/internal/focus/service_test.go`, `ui/src/api/focus.test.ts`, `ui/src/pages/FocusPage.test.tsx`, `api/internal/review/service_test.go` | SQLite proves effective correction plus one audit row and a bounded correction-history read; CLI/API/UI transport and correction disclosure paths pass; Review totals include manual actual minutes without increasing focus-session counts | Focus actuals ↔ Review |
| Goal prerequisite gating and milestone-derived progress | `api/internal/goals/sqlite_test.go`, `api/internal/goals/service_test.go`, `ui/src/pages/GoalsPage.test.tsx`, `ui/src/api/goals.test.ts` | SQLite proves incomplete prerequisites reject completion, completion unlocks dependents, and milestone-mode goals roll up to 100%; CLI/API/UI transport and waiting-state paths pass | Goals ↔ milestones ↔ work |
| Goal creation and explicit progress | `api/internal/goals/service_test.go`, `ui/src/pages/GoalsPage.test.tsx`, live CLI evidence | API and CLI suites pass; UI test covers goal creation; live create/list/progress returned revision 2 at 25% | Goals ↔ user-entered outcome |
| Daily review measured summary | `api/internal/review/service_test.go`, `ui/src/pages/ReviewPage.test.tsx`, live CLI evidence | SQLite read model, CLI `review daily`, and UI coverage pass; accepted planned minutes and recorded focus/manual actuals are measured separately, unrecorded time remains unknown | Review ↔ Calendar/Focus/Goals read models |
| Five-surface route contract | `ui/src/app/routes.test.tsx`, `experience/pages/*.json` | Route test passes; Plan, Goals, Focus, and Review are backed by their current read/write slices and expose explicit domain boundaries | UI experience |
| Settings product boundary | `ui/src/pages/SettingsPage.test.tsx`, locale catalogs | English rendering proves the included core, private-by-default posture, and deferred monetization hypothesis; no billing or upgrade control is presented | Product honesty; deferred commercial surface |
| Workspace planning profile | `api/internal/workspace/`, `handlers/workspace/`, `cli/domains/workspace/`, `ui/src/api/workspace.ts`, `ui/src/pages/SettingsPage.test.tsx` | Revisioned timezone, week-start, capacity, reserve, and focus defaults are validated in API/CLI/UI seams; live read/update/read and CLI read agree at revision 2 | Manual profile; provider projections remain open |
| Workspace availability | `api/internal/workspace/service_test.go`, `api/internal/calendar/sqlite_test.go`, `ui/src/api/workspace.test.ts`, `ui/src/pages/SettingsPage.test.tsx` | Overlap rejection, protected exception persistence, weekday capacity, Settings editing, and revision-aware transport are covered; CLI contracts are green | User workspace currently has no configured windows; provider projections remain open |
| Profile-backed Today capacity | `api/internal/calendar/sqlite_test.go` | Calendar subtracts protected exceptions and workspace reserve from configured local weekday windows, then computes breathing room from accepted allocations; fallback 420−60=360 and Monday 480−60 protected−60 reserve=360 are asserted | Overlapping exclusions and richer provider/routine demand remain open |
| Native routines | `api/internal/calendar/service_test.go`, `api/internal/calendar/sqlite_test.go`, `ui/src/api/calendar.test.ts`, `ui/src/pages/PlanPage.test.tsx` | Fixed and flexible routine definitions validate, persist, expand across bounded local dates, and render/create from Plan; empty live CLI routine reads remain explicit | Occurrence overrides, recurrence edit scopes, imported/provider recurrence, and routine demand accounting remain open |
| Routine occurrence overrides | `api/internal/calendar/sqlite_test.go`, `ui/src/api/calendar.test.ts`, `ui/src/pages/PlanPage.test.tsx`, CLI primitive evidence | Stable routine/date skip-once overrides persist, honor routine revision checks, disappear from generated occurrence reads, and are exposed in Plan/CLI; live unknown-routine rejection is explicit | Reschedule/edit-this-and-following/series scopes, imported/provider recurrence, and routine demand accounting remain open |
| Goal milestones | `api/internal/goals/service_test.go`, `ui/src/api/goals.test.ts`, `ui/src/pages/GoalsPage.test.tsx`, CLI primitive evidence | Criteria, due dates, validated optional work-item links, goal-scoped listing, and revision-safe open/complete state are wired through proto/API/SQLite/CLI/UI; live goal milestone listing is empty for the existing goal | Milestone prerequisites and aggregate goal-progress derivation remain open |
| Day/night scenic composition | `ui/public/public/scenes/*.webp`, BAS captures `362e0681-d5d2-4d09-86aa-3bbe2c189f9e` and `c0f10b71-ac2f-451a-9339-771453c88380` | Derived assets bundle successfully; reference-width Day and Night captures were compared against both oracles, and Night-specific secondary-text/action contrast was corrected. Residual differences are live date/data and small spacing/icon details. | Visual release gate |

### Latest receipts

- Comprehensive Test Genie run `20260919-073654-a6369545`: **26/27 phases passed**.
- The sole failed phase is `portability`; all scenario-owned phases passed,
  including `experience` at L3 and `measures` with clean domain coverage.
- Focused Unit/Contracts run `20260919-063005-becf55dd`: **2/2 passed**.
- Focused Experience run `20260919-062421-5ae4f0d5`: **1/1 passed**.
- UI coverage: **95.05% statements, 85.61% branches, 86.02% functions**;
  27 files and 122 tests passed.
- Calendar slice follow-up: UI coverage **94.82% statements, 85.52% branches,
  85.56% functions**; 28 files and 126 tests passed. Test Genie run
  `20260919-064929-a9102e19`: **3/3 focused phases passed** (unit,
  contracts, experience).
- Calendar/focus follow-up: UI coverage **94.92% statements, 85.66% branches,
  86.00% functions**; 28 files and 128 tests passed. The same focused Test
  Genie receipt remains green after the durable Today focus wiring.
- Calendar range follow-up: UI coverage **95.13% statements, 86.20% branches,
  86.72% functions**; 28 files and 132 tests passed. The latest comprehensive
  Test Genie run `20260919-074232-4986a9a3` passed **26/27 phases**; the sole
  failure remains repository-level portability. A live Connect range read
  returned the persisted accepted allocation for `2026-09-19..2026-09-26`.
- Settings product-boundary follow-up: the localized Settings story now states
  what is included, what remains private, and which commercial capabilities
  are only future hypotheses; it presents no billing or upgrade action. The
  focused English rendering regression and full UI suite pass: **29 files / 133
  tests**, with **95.19% statements, 86.64% branches, and 86.72% functions**.
- Focused Test Genie receipt `20260919-075603-dace0ff8`: **3/3 phases passed**
  (unit, contracts, experience) after the Settings slice; unit is L2 Ready →
  L3, while contracts and experience remain L3 Complete.
- Workspace profile follow-up: API and CLI suites pass; the Settings profile
  load/save regression and transport tests pass. The full UI suite is now **30
  files / 136 tests**, with **95.42% statements, 87.16% branches, and 88.61%
  functions**. Live Connect read/update/read and `personal-planner workspace
  profile` returned the same persisted America/New_York profile at revision 2.
- Focused Test Genie receipt `20260919-081349-f61bca55`: **3/3 phases passed**
  (unit, contracts, experience). Unit remains L2 Ready → L3 with one
  evolvability warning; contracts and experience remain L3 Complete.
- Capacity follow-up: Calendar SQLite coverage now proves the Workspace
  profile is the source for Today’s available minutes and reserve accounting;
  live Calendar read returned 360 available, 45 planned, and 315 breathing
  room from the persisted 420/60 profile.
- Focused Test Genie receipt `20260919-082216-446aa291`: **3/3 phases passed**
  (unit, contracts, experience) after profile-backed capacity integration.
- Availability follow-up: UI coverage is **95.57% statements, 86.55% branches,
  87.40% functions**; 30 files and 137 tests passed. API and CLI suites pass.
  Test Genie run `20260919-083821-04a5733a` passed unit and experience; its
  contracts phase failed first on structured JSON binding and was repaired.
  The targeted contracts rerun `20260919-083858-77088cf7` is **1/1 passed at
  L3 Complete**. Live CLI read confirms availability revision 2 with zero
  configured windows/exceptions; no user schedule was seeded by validation.
- Routine follow-up: UI coverage is **95.69% statements, 85.98% branches,
  85.13% functions**; 30 files and 139 tests passed. API and CLI suites pass,
  production lifecycle is healthy, and live `calendar routines` plus bounded
  `calendar routine-occurrences` return honest empty states. Test Genie run
  `20260919-085204-a39d4be2` passed **3/3 focused phases**; unit remains L2
  Ready with the existing injectable-seam warning, contracts and experience
  are complete.
- Goal milestone follow-up: UI coverage is **95.57% statements, 85.87% branches,
  85.00% functions**; 30 files and 141 tests passed. API and CLI suites pass,
  endpoint generation and production build pass, and the live CLI lists the
  existing goal with an honest zero-milestone state. Test Genie run
  `20260919-090516-29764305` is terminal **provider_unavailable** for unit,
  contracts, and experience; no phase evidence was produced, so this is not
  treated as a product pass.
- Goal-work linkage follow-up: UI coverage is **95.58% statements, 85.96%
  branches, 85.09% functions**; 30 files and 141 tests passed. Go API and CLI
  suites, TypeScript, production build, lifecycle health, live goal/milestone/
  work reads, and live rejection of an unknown linked work item pass. Focused
  Test Genie run `20260919-092558-f15f9dd0` is terminal
  **provider_unavailable** for unit, contracts, and experience.
- Routine override follow-up: UI coverage is **95.88% statements, 86.48%
  branches, 86.14% functions**; 30 files and 144 tests passed. Go API and
  CLI suites, endpoint generation, production build, lifecycle health, and
  live unknown-routine rejection pass. The routine skip transport and Plan
  interaction are covered; Test Genie was not rerun after this slice because
  the provider-unavailable condition persisted in the preceding focused run.
- Routine reschedule follow-up: SQLite, API, and CLI tests pass; the generated
  endpoint and installed CLI expose `calendar reschedule-routine-occurrence`.
  UI coverage/build pass with **30 files, 146 tests, 95.90% statements, 86.68%
  branches, 85.88% functions**. Lifecycle health passes after restart in
  best-effort mode, with `notification-hub` and `scenario-authenticator`
  degraded by dependency freshness timeouts. No live routine was seeded merely
  for validation, so positive reschedule behavior is proven by SQLite/API/UI
  tests rather than a mutation of the user's empty runtime state.
- Responsive/accessibility follow-up: added reduced-motion and RTL layout
  safeguards plus 44px routine/milestone action targets and reran the UI
  coverage/build gate: **30 files, 144 tests, 95.88% statements, 86.48%
  branches, 86.14% functions**, production build passed. Lifecycle is healthy
  in best-effort mode; `notification-hub` and `scenario-authenticator` remain
  degraded because dependency freshness timed out. This is implementation
  evidence, not a substitute for the pending responsive/contrast capture set.
- Review date follow-up: Review now requests an explicit browser-local
  `YYYY-MM-DD` and supports adjacent-day navigation; Review/API tests pass.
  Full UI coverage/build pass: **30 files, 147 tests, 95.95% statements,
  86.79% branches, 85.14% functions**. This does not claim weekly review,
  correction capture, or carry-forward semantics.
- Scoped Test Genie receipts after the Review date and responsive fixes:
  contracts `20260919-100018-ddce9ae3` passed at L3 Complete after removing
  the redundant start-minute binding; experience `20260919-100212-c5f51fa2`
  passed at L3 Complete with live browser capture and cleared the mobile
  `floor_no_document_horizontal_overflow` failure. Unit `20260919-100931-6ac90b65`
  passed at L2 Ready after restoring Scenario Dependency Analyzer readiness;
  it retains one injectable-seam warning and coverage advisories. The earlier
  unit retry `20260919-100843-98fd17c9` correctly exposed a TypeScript test
  failure, which was repaired before the passing receipt.
- Combined focused receipt `20260919-101007-c977ea9e`: **3/3 phases
  passed**. Unit is L2 Ready with one `MISSING_INJECTABLE_SEAM` warning and
  45 coverage advisories; contracts is L3 Complete; experience is L3 Complete
  with browser-backed structure reconciliation. This is focused evidence, not
  the broader certification/maturity gate.
- Visual checkpoint at 1585×992: Day and Night captures were compared once
  against the approved mockups; the second pass corrected the shell to a
  narrow Planner/Observatory rail, Today navigation, editorial content width,
  card geometry, scene layering, and Day/Night active appearance. The
  remaining bounded difference is live-date/sample-data content and residual
  library spacing/icon details; the separate mobile checkpoint below covers
  the responsive action/timeline treatment, while a broader accessibility
  review remains open.

- Weekly Review follow-up: the generated review contract exposes
  `GetWeeklySummary`; API tests prove seven daily summaries aggregate accepted
  planned minutes and recorded activity without inferring completion. CLI
  evidence, endpoint generation, live weekly read, UI coverage/build, and
  focused receipt `20260919-102400-ce2d3e68` passed before the actuals slice;
  the newer combined receipt is recorded below.

This inventory is not a release claim. The comprehensive Test Genie gate,
visual evidence registration, and remaining provider/routine plus richer goal
milestone/link domains are still required for completion. Brand validation has
no warning findings after the token hardening; informational notes remain for
the unassigned brand-manager marker and system-font posture.

- Manual actuals follow-up: Focus now records approximate or observed active
  minutes, lists them by local date, and corrects them with revision checks plus
  an `actual_corrections` audit row. Review sums manual actual minutes with
  focus-session active time while keeping the session count separate. API,
  CLI, UI coverage/build, endpoint generation, lifecycle health, and live empty
  actuals read pass. UI validation has **30 files, 151 tests, 96.21%
  statements, 86.38% branches, and 85.35% functions**, with type-check and
  production build passing. Focused Test Genie receipt
  `20260919-103942-536a7288` passed **3/3 phases**: unit L2 Ready, contracts
  L3 Complete, and experience L3 Complete. Unit retains the known injectable
  seam/coverage advisories; notification-hub remains degraded. The live
  workspace was not seeded with a validation actual.

- Post-cleanup receipt `20260919-104957-5fe628a7` passed API, proto,
  contracts, unit, and experience phases. It is not a release pass overall:
  portability and security failed, and channel-conformance was
  provider-unavailable. Structure is clean after removing the stale empty
  capabilities handler directory; CLI contracts are L3 Complete.

- Goal prerequisite/rollup follow-up: milestone creation now persists
  prerequisite edges; completion returns a failed-precondition error while a
  prerequisite is open, and goals configured for milestone progress derive a
  completed/total rollup. UI validation has **30 files, 152 tests, 96.24%
  statements, 86.52% branches, and 85.5% functions**, with type-check and
  production build passing. Test Genie receipt
  `20260919-112043-f6897478` passed **3/3 phases**: unit L2 Ready, contracts
  L3 Complete, and experience L3 Complete.

- UI Health/visual checkpoint follow-up: the mobile text-entry controls now
  satisfy the 16px browser zoom floor; `ui-health validate scenario
  personal-planner --json` returned `VALIDATION_STATUS_PASSED`, zero
  `visual_focus_zoom_risk` findings, and zero blocking findings. Provider
  maturity remains L0 because the legacy template-slot and component-adoption
  findings are still present. A BAS Day checkpoint was captured at 1585x992
  (`9285fef8-2aed-4db2-8b8d-f7ea67e5f875`, screenshot
  `a935b414-2d09-4138-8eda-24751a03a4be/.../step-01-cd87d6f0-fd3e-4665-831c-6ebec45aa156.jpg`);
  the live date/sample content differs from the oracle as expected, while the
  remaining responsive and release-certification evidence stays open. The later
  mobile checkpoint at 390px (`bde58633-a3c2-49a5-8b13-c91b8a9a0947`) exposed and
  then verified the stacked action-row fix plus vertical agenda treatment.

- Comprehensive receipt `20260919-113113-be49c7ab`: **25/27 phases passed**
  in 286 seconds. Structure, contracts, UI Health, API, architecture,
  dependencies, unit, workflow, business, experience, proto, branding,
  templates, and the other product phases passed; portability and security
  failed. Orientation is restored to **9/9 complete**. This remains a
  non-release result until the two failed phases and the broader UI-health
  maturity findings are resolved or explicitly bounded.

- A subsequent server-owned run `20260919-113728-69471a83` reproduced the
  same **25/27** result in 285 seconds, with portability and security still
  the only failed phases. The attempted focused invocation was absorbed by
  the active default run, so no narrower phase receipt is claimed.

- Security hardening follow-up: `security-health validate scenario
  personal-planner --json` now returns `VALIDATION_STATUS_PASSED` after the
  workspace CLI stopped using `strconv.Atoi` before an int32 conversion. A
  regression test rejects values outside the proto int32 range; the CLI and
  API suites pass. Advisory G104/G115 findings and dependency/toolchain
  vulnerability intelligence remain visible, but no ERROR finding remains.

- Selective carry-forward follow-up: Calendar now owns a transactional,
  idempotent carry-forward command. SQLite proves source-history preservation,
  overlap rejection, and retry identity; the CLI exposes
  `calendar carry-forward`; Review exposes an explicit per-placement action
  with target-day capacity preview. UI validation passes **30 files, 154
  tests, 96.36% statements, 86.66% branches, 85.37% functions**, with
  type-check and production build passing. API and CLI suites pass; the live
  runtime is healthy in best-effort mode with `notification-hub` degraded.

- Focused carry-forward receipt `20260919-115722-0b180c9a` passed **3/3**:
  unit L2 Ready, contracts passed, and experience L3 Complete. Unit retains
  the known injectable-seam advisory; the receipt is focused evidence, not a
  comprehensive release certification.

- Focused reflection receipt `20260919-120927-f6dec97a` passed **3/3**: unit L2 Ready, contracts L3 Complete, and experience L3 Complete. Unit retains the known injectable-seam advisory; this is focused evidence, not comprehensive release certification.
- Focused post-visual receipt `20260919-122802-b345ac2d` passed **3/3**: unit L2 Ready, contracts L3 Complete, and experience L3 Complete. It covers the responsive/Night visual changes; unit retains the known injectable-seam advisory.
- Focused correction-history receipt `20260919-124822-0ef8ab99` passed **4/4**: unit, contracts, experience, and measures. Domain coverage is clean; unit and measures retain their non-blocking maturity advisories (`MISSING_INJECTABLE_SEAM` and `measures.tier-fallback`).
- Follow-up contract/measure receipt `20260919-125606-b33f4a95` passed **2/2** after simplifying the `actualId` CLI binding; CLI contracts are now L3 Complete and measures retain only the tier-fallback advisory. Live `personal-planner focus corrections --json` returns the bounded empty history response `{}`.
- Comprehensive receipt `20260919-122840-a6789844` passed **26/27** phases. Security now passes; the sole failed phase is portability. Narrow rerun `20260919-123437-8b8b6355` reports the repository-level trusted-base closure error `agent-manager -> workspace-sandbox`, outside Personal Planner's declared dependency surface.
- Latest comprehensive receipt `20260919-124928-a592b095` also has exactly one failed phase—portability—with all 26 other phases passing. The previous narrow portability receipt remains the authoritative diagnostic for the unchanged shared trusted-base closure; scenario-owned correction-history contracts/measures were separately rerun after this full run.
- Measures rerun `20260919-124657-216206ea` passes with domain coverage clean; remaining maturity is `measures.tier-fallback` because the current bounded list measures use best-effort parameter extraction. The persisted actual-corrections, manual-actuals, milestone, and routine substrates now have explicit measures or narrowly scoped relationship-table waivers.
- Post-slice UI structural check: `pnpm run type-check` and `pnpm exec vitest run src/pages/PlanPage.test.tsx` pass (10/10). `pnpm run test:template-library` remains red only on the existing composed design-token equality test: the scenario’s intentional Observatory palette differs from the current default-kit output; the three other template checks pass.
- Adopted Test Genie receipt `20260919-130105-4151a134` completed with 26/27 phases passing after the range-component extraction. Unit, contracts, and experience passed; portability remains the sole failed phase for the shared trusted-base closure. The receipt also confirms CLI measure metadata is contract-complete with informational tier-fallback findings.
- Provider connection slice evidence: API `go test ./...`, CLI `UPDATE_CLI_EVIDENCE=1 go test ./...`, UI coverage/build, and live CLI create/sync/list all pass. The live fixture sync returned three synthetic imported events, 120 busy minutes, and revision 2; this is contract evidence only and does not prove a real provider or capacity projection.
- Focused post-integration Test Genie receipt `20260919-132005-d4e4f27f` completed with 26/27 phases passing. Contracts, unit, experience, and measures passed; portability remains the sole failed phase for the shared trusted-base closure. Measures now reports eight informational tier-fallback entries, including `provider_connections`.
- Imported-capacity projection evidence: Calendar SQLite regression proves active imported intervals are unioned with accepted allocations without double-counting overlap; Dashboard regression proves Today surfaces the external event/minute fact. The projection is synthetic-fixture evidence only until a real provider adapter supplies durable cursor/freshness semantics.
- Google adapter evidence: `go test ./internal/integrations` passes HTTP fixtures for calendar pagination, timed/all-day/transparent normalization, final sync-token capture, and HTTP 410 full-reset signaling. This is adapter-contract evidence, not live OAuth/provider certification.
- Focused Google adapter receipt `20260919-133813-6f7796c4` passed **4/4**: unit, contracts, experience, and measures. Unit remains L2 with the known `MISSING_INJECTABLE_SEAM` advisory; measures remains L1 with three `measures.tier-fallback` advisories. The receipt is scenario-owned contract evidence, not live OAuth/provider certification.
- Today quick-capture evidence: the UI transport and Dashboard tests cover name-first creation, optional description/estimate/source, persistence refresh, and failed-save honesty. Full UI validation passes **32 files, 163 tests, 97.08% statements, 86.32% branches, and 85.59% functions**; production build passes. Managed runtime smoke created and re-read `Capture flow verification` through WorkService. Receipt `20260919-135129-21650def` passes **4/4** requested phases; experience is L3 Complete. The run retains the known unit injectable-seam and measures tier-fallback advisories.
- Today next-action evidence: Dashboard now uses an accepted allocation as the source of the promoted next task and exposes its accepted time/duration rather than silently presenting a newer unplaced capture. Focused Dashboard/API tests pass; after rebuilding and restarting the managed runtime, receipt `20260919-135543-3105b2ef` passes **4/4** with experience L3 Complete. Unit injectable-seam and measures tier-fallback advisories remain non-blocking maturity debt.
- Placement-preview evidence: Calendar SQLite tests prove the deterministic preview moves past accepted and imported busy intervals; CLI contract evidence and Plan UI tests cover preview-before-accept and explicit preview-only state. API `go test ./...`, CLI `UPDATE_CLI_EVIDENCE=1 go test ./...`, UI coverage/build, and live `personal-planner calendar preview-placement` pass. Receipt `20260919-140559-0191f573` passes **4/4** requested phases (unit/contracts/experience/measures); unit retains the known injectable-seam advisory and measures retains tier-fallback advisories. This slice is not full optimizer or stale-revision certification.
- Durable proposal evidence: SQLite proves persisted proposal application, schedule-revision rejection, and idempotent retry; Plan’s stale acceptance test keeps the proposal visible state honest. Live CLI preview returned proposal `04731791-b1a1-4984-81db-b36fdb111d26` at revision 2, apply returned allocation `72920bfe-00d1-4f40-b767-27c4647da9ba`, and the previous proposal retry returned the same allocation id. API and CLI suites pass, UI coverage/build passes with 32 files/165 tests, and Test Genie receipt `20260919-141655-24ee566f` passes **4/4**. This remains a bounded one-item proposal slice, not full multi-item optimizer certification.
- Proposal-geometry evidence: Plan renders the requested and proposed local times as a labeled comparison and keeps acceptance separate; the 11-test Plan suite and production build pass. Receipt `20260919-141852-be7dd21a` passes **4/4** requested phases with Experience L3 Complete. Unit injectable-seam and measures tier-fallback advisories remain non-blocking maturity debt.
- Multi-item proposal evidence: Calendar SQLite tests prove stable request-order placement from real remaining effort, atomic batch apply, and idempotent retry. Live CLI preview produced proposal `82409669-b986-4a13-b7eb-32b6f2ad8262` at revision 3 for two work items; apply returned two accepted allocations and the exact retry returned the same ids. API and CLI suites pass; full UI coverage passes **32 files, 170 tests, 97.20% statements, 85.28% branches, and 86.22% functions**; production build passes. Receipt `20260919-142916-abab670a` passes **4/4** with Experience L3 Complete. This is bounded one-day batch scheduling, not full R1 horizon optimization.
- Demand-conservation evidence: Calendar SQLite regression proves accepted planned minutes are subtracted from `work_items.remaining_minutes` for subsequent previews and transactional single/batch applies reject over-demand proposals. Managed runtime CLI smoke returned a blocked multi-item proposal with both already-scheduled items unresolved and a blocked single placement with `0 minutes of remaining effort`. API `go test ./...` passes; receipt `20260919-143414-8fb01c4d` passes **4/4** requested phases. This does not yet cover split/setup policies or a full horizon optimizer.
- Responsive shell follow-up: `pnpm run type-check` passes, focused Dashboard coverage passes **9/9** after adding actionable timeline and overlap-lane regressions, and the full UI suite remains green at **170 tests** before the new focused regression. Brand Manager candidate generation/pick/apply and `brand-manager provider validate personal-planner` pass. BAS captures at 390px mobile and desktop Settings confirm the fixed left-focal scenery, removed mobile shell inset, canonical observatory mark, and shared SettingsList card layout. The managed restart is healthy for Personal Planner; notification-hub remains degraded by the known tunnel-manager freshness timeout.
- Plan/Focus and template-cleanup evidence: Plan now renders a labeled 09:00–19:00 axis, keeps accepted block width proportional to duration, lanes overlaps, and exposes accepted allocation details; a 15-test Plan suite covers the short-width/action path. Focus uses the shared editorial page hierarchy and its five-test suite remains green. `template-manager detemplate personal-planner --dry-run` and the real command both report zero blocks/lines/deletions after removing ambiguous `notes` example copy. API `go test ./...` passes; UI coverage/build pass with **172 tests** and **97.17% statements**. Test Genie run `20260919-151509-052a5ea1` is the current server-owned comprehensive run and must be waited to terminal before its result is recorded.
- Branding follow-up: Brand Manager reported one exact target defect in its generated 192px maskable icon (five pixels outside the safe zone); the generated 512px target was compliant. The 192px target was deterministically normalized from the compliant 512px render through `image-tools ops resize`, after which `brand-manager provider validate personal-planner` cleared `declared-icon-targets` and focused branding receipt `20260919-152717-8d4582ff` passed. The remaining `custom-font-loaded` INFO is a validator false positive over generated `--font-size-*` token names; the actual primary families are generic `sans-serif`/`monospace` and no custom font is intended.
- Current comprehensive receipt `20260919-151509-052a5ea1` completed **25/27**: portability retains the shared trusted-base closure failure, and UI Health's runtime check hit a transient connection refusal while the managed UI was restarting. Focused experience receipt `20260919-152304-d63202f9` and the follow-up branding receipt above both pass; the full receipt is not a release certification.
- Local appearance/control follow-up: focused Dashboard, Settings, and appearance tests pass **15/15**; `pnpm run type-check` and production build pass. Auto appearance is now based on configurable local transition times rather than OS color preference, Today collision lanes use hour-consistent interval math, and the timeline detail is anchored to the selected block. Settings planning controls use shared Input/Select/Button primitives; a 44px mobile minimum fixed the first experience failure (`experience.floor_tap_target_size`). Experience receipt `20260919-154148-961aa0b1` passes; its remaining `experience.capture_unavailable` note is an evidence-coverage advisory, not a product failure.
- Ongoing feedback-loop shell/Today slice: full UI coverage passes **178 tests** with **97.83% statements, 85.83% branches, and 85.45% functions**; type-check and production build pass. The first experience rerun caught below-floor new controls before the managed rebuild; after raising the sidebar collapse and timeline view controls to 44px and rebuilding/restarting through the lifecycle, receipt `20260919-160637-4e3fe0bc` passes experience **L3 Complete**. The earlier failed receipt `20260919-160256-fcdbbe8d` is retained as regression evidence, not a release verdict.
- Focused shell/controls receipt `20260919-161944-a00cb4cb` passes **3/3**: unit, contracts, and experience. It covers the library-owned shell refactor, completed PWA metadata, mobile collapse affordance, Review component extraction, and shared Goals controls. Unit retains the known injectable-seam advisory.
- Comprehensive receipt `20260919-162020-12be5a0e` completed **26/27** phases. UI-health passed its runtime/PWA/branding checks; the remaining UI findings are shared adoption/template debt and secondary raw-control cleanup. The sole terminal phase failure is portability, a repository-level trusted-base closure outside Personal Planner’s declared resource surface.

## Binary startup

`api/main_e2e_test.go` uses `api-core/boottest` to build this API, verify its
health identity in isolated storage, and check shutdown. Keep the expected
service name in the scenario test. Shared process machinery and failure
regressions live in `packages/api-core/boottest`; see its package README section
for configuration and evidence limits. The existing E2E gate runs this test.
Durable review reflection implementation checks pass locally: API `go test ./...`, CLI `UPDATE_CLI_EVIDENCE=1 go test ./...`, UI type-check/build, and UI coverage (156 tests; 96.44% statements, 86.51% branches, 85.84% functions), with focused Test Genie receipt `20260919-120927-f6dec97a` recorded above.
