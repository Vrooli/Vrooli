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
- Focused adaptive-chrome rerun `20260919-170222-62700a7f` initially retained one experience overflow finding on the Goals article. After switching AppShell to fill mode and constraining Goal cards with border-box/mobile stacking, experience receipt `20260919-170443-5896f992` passed at L3 Complete. UI tests remain 33 files / 178 tests; type-check and build pass. The unsupported exploratory Vitest `--runInBand` flag is not a product failure.
- Comprehensive receipt `20260919-170849-8925c691` completed 25/27: portability remains the shared trusted-base closure failure; UI-health passed but retains advisory raw-primitive/template findings, and branding passed with install-surface/public-asset advisories. After the subsequent theme/control/Plan fixes, full UI tests pass 33 files / 179 tests, type-check/build pass, Brand Manager validation passes, and focused experience receipt `20260919-172202-e3372dee` passes at L3 Complete.
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
- Secondary surface control polish: full UI suite passes **33 files / 178 tests** after adopting shared Button/Input controls across Focus, Plan, and Review; type-check and production build also pass. Managed restart reports Personal Planner healthy; notification-hub remains degraded by the known tunnel-manager freshness timeout.
- Auto appearance synchronization: targeted appearance, Settings, and Today tests pass **16/16** after adding live preference/storage event handling for configurable day/night boundaries.
- Shared form-control adoption: Today capture, Goals purpose, and Review reflection now use the shared Input/Textarea primitives; no raw input/select/textarea controls remain in non-test page source. Full UI suite passes **33 files / 179 tests**, type-check and production build pass, and focused Test Genie receipt `20260919-172834-ec054422` passes **3/3** requested phases (unit, contracts, experience). Managed restart reports Personal Planner healthy; notification-hub remains degraded by the known tunnel-manager freshness timeout.
- Secondary hierarchy and standards evidence: Plan, Goals, Focus, and Review use the shared PageHeader; all former planner-empty branches use governed EmptyState; theme chrome resolves through semantic tokens with browser-safe fallbacks; and Goals no longer overflows the 844×390 experience viewport. Full UI suite passes **33 files / 179 tests**, type-check and production build pass. UI-health receipt `20260919-174124-c14934ac` passes with runtime/manifest/interop/freshness/PWA at L5 and project standards at L3; focused experience receipt `20260919-174608-f5fe9e34` passes at **L3 Complete**. The immediately preceding combined receipt `20260919-174239-244c515e` correctly caught the Goals overflow and is retained as regression evidence, not a verdict. Managed Personal Planner remains healthy with notification-hub degraded by the known tunnel-manager freshness timeout.
- Today shared-control extraction: Today appearance, capture, task-action, timeline-view, and disclosure-close controls now use a scenario-local composition over shared Button primitives; the sidebar collapse affordance follows the same library contract. Full UI suite passes **33 files / 179 tests**, type-check/build pass, and focused shell/settings tests pass **14/14**. Test Genie receipt `20260919-175630-afdcaf8c` passes unit/contracts/experience 3/3. UI-health receipt `20260919-175707-b42b2d39` passes; runtime/manifest/interop/freshness/PWA are L5. Project standards remains L3 because intentional anchored dialogs and raw Observatory palette literals remain open standards debt.

 - Collision-aware timeline disclosure: Today timeline details now use the shared Popover’s collision-aware anchoring and portal rather than local `left/top` positioning. Dashboard regression, full UI suite (33 files / 179 tests), type-check, and production build pass. UI-health receipt `20260919-180852-24d59cb3` passes with runtime, manifest, interop, freshness, and PWA at L5; project standards remains L3 with raw capture/draft dialogs, stylesheet palette literals, and the unused HealthCard scaffold still open.
- Scene contrast and Settings density checkpoint: desktop Day/Night and Settings captures verified scene-specific ink/chrome contrast, a dark Night sidebar, and compact shared RadioGroup/Switch layouts. Full UI suite passes **33 files / 179 tests**, type-check/build pass, and UI-health receipt `20260919-182121-fef704e2` passes with runtime, manifest, interop, freshness, and PWA at L5. Project standards remains L3 with raw capture/draft dialogs, stylesheet palette literals, and the unused HealthCard scaffold still open.
- Settings geometry and timeline disclosure checkpoint: Settings planning, availability, and integration rows now span the card form area cleanly; Today timeline details remain on the shared collision-aware Popover. Full UI suite passes **33 files / 179 tests**, Settings tests pass **3/3**, type-check/build pass, and UI-health receipt `20260919-182457-5e219806` passes with runtime, manifest, interop, freshness, and PWA at L5. Project standards remains L3 with the known raw-dialog, palette-literal, unused-scaffold, and advisory text-clipping findings.
- Appearance choice surface checkpoint: Settings now uses the shared RadioGroup card variant for the Appearance choices. Full UI suite passes **33 files / 179 tests**, Settings tests pass **3/3**, type-check/build pass, and the UI-health phase inside server-owned run `20260919-182810-3560d72d` passes with runtime at L5. The enclosing comprehensive run is not a release certification: portability and other repository/provider findings remain separate from this UI change.
- Plan allocation disclosure checkpoint: Plan accepted schedule blocks now use the shared collision-aware Popover for their details instead of a detached raw dialog. Full UI suite passes **33 files / 179 tests**, Plan tests pass **15/15**, type-check/build pass, and the UI-health phase inside server-owned run `20260919-183459-d0abcefc` passes with runtime/manifest/interop/freshness/PWA at L5. The enclosing comprehensive run remains non-release due unrelated portability/docs/provider findings; its standards output still reports the broader raw-dialog aggregate and is retained for follow-up.
- Today Dialog and global theme checkpoint: Today draft/capture overlays now use the shared Dialog, with an explicit regression assertion for asynchronous portal cleanup between capture flows. The shell applies the resolved day/night appearance to every route's sidebar, bottom navigation, page canvas, and ChromeTheme base. Dashboard tests pass **9/9**, full UI suite passes **33 files / 179 tests**, type-check/build pass, and UI-health run `20260919-184929-8f08fc1e` passes with runtime/manifest/interop/freshness/PWA at L5; project standards remains L3 with the documented palette and scaffold advisories.

- Latest responsive form checkpoint: Settings availability/protected-time fields and Plan routine fields now use shared `FormField` composition over library `Input`/`Select` controls, with an explicit weekday grid and unique accessible names. `make orient` reports no stale orientation metadata; full UI suite passes **33 files / 180 tests**, type-check and production build pass, and the fresh managed UI-health receipt `20260919-205447-bd9d699a` passes at L5 across all seven routes and desktop/mobile runtime coverage. The broader visual/accessibility release sweep and portability closure remain open.
- Cross-route appearance checkpoint: the shell now emits the Observatory `day`/`night` class vocabulary that its sidebar, bottom-nav, and canvas selectors consume; AppShell regression coverage proves the dark route contract. Full UI suite passes **33 files / 181 tests**, type-check/build pass, managed restart is healthy, and UI-health receipt `20260919-210311-a3d15dcc` passes **1/1** at L5 across all seven routes. The browser's non-standalone Safari chrome remains outside page control; visual oracle comparison and portability closure remain open.
- Timeline packing checkpoint: Today coverage now proves a later non-overlapping block reuses the first collision lane after a two-block overlap. Dashboard tests pass **9/9**; full UI coverage remains **33 files / 181 tests**. This protects the intended two-row geometry instead of a monotonically growing stack.
- Goals form checkpoint: Goal progress and milestone editing now use shared `FormField` composition over library controls, with explicit names that preserve accessible queries across required/optional field decoration. Goals tests pass **6/6**; full UI suite remains **33 files / 181 tests**, type-check/build pass. Today capture’s compact legacy markup and the broader visual oracle sweep remain open.

### 2026-09-19 — mobile secondary-surface spacing

- Full UI suite: **33 test files / 181 tests passed**.
- Type-check and production build passed.
- `git diff --check` passed for the responsive stylesheet change.
- Static `ui-health validate scenario personal-planner --static-only --json` reported zero static findings; its only degraded status is the intentional `runtime_not_evaluated_static_only` notice.
- Fresh managed restart completed healthy; server-owned Test Genie run `20260919-211637-5f833ee0` passed the focused `ui-health` phase at **L5** across all seven routes. The notification-hub freshness timeout remains unrelated to Personal Planner health.
- Settings hierarchy polish: focused Settings/AppShell tests passed **14/14**; full UI suite passed **33 files / 181 tests**; type-check/build passed; server-owned Test Genie run `20260919-212246-f69a1058` passed ui-health at **L5** across all seven routes.
- Responsive Observatory evidence: governed BAS captures completed at `390x844` for Day (`adf9cd60-9782-495c-ac5f-5686845820cd`) and Night (`3e4d97bc-6e02-4920-b629-14b9dafe0526`). These are runtime evidence artifacts; visual-oracle closure remains a separate judgment and fix loop.
- Route-wide shell chrome: shell/theme/a11y tests **18/18**; full UI suite **33 files / 181 tests**; type-check/build passed; managed Test Genie run `20260919-213024-045099ee` passed ui-health at **L5** across all seven routes.
- Scene palette ownership: Dashboard/theme tests **15/15**; full UI suite **33 files / 181 tests**; type-check/build passed; managed Test Genie run `20260919-213629-386da21c` passed ui-health at **L5** across all seven routes.
- Today capture shared-field contract: Dashboard tests **9/9**; full UI suite **33 files / 181 tests**; type-check/build passed; managed Test Genie run `20260919-214124-13f9ae9a` passed ui-health at **L5** across all seven routes.
- Observatory radius contract: focused Plan/Goals/Focus/Review/Settings tests **36/36**; full UI suite **33 files / 181 tests**; type-check, production build, and stylesheet diff-check passed. Managed restart is healthy on ports **17604/20003** with notification-hub degraded by its unrelated tunnel-manager freshness timeout. Test Genie run `20260919-214634-6a2c1eaa` passed focused ui-health at **L5** across all seven routes, with fourteen informational runtime confirmations.
- Responsive SettingsList surface: Settings/AppShell checks **16/16**; full UI suite **33 files / 181 tests**; type-check, production build, and diff-check passed. The managed planner is healthy on **17604/20003**; Test Genie run `20260919-215224-8c673e20` passed focused ui-health at **L5** across all seven routes with fourteen informational runtime confirmations. Notification-hub remains degraded by its unrelated tunnel-manager freshness timeout.
- Canonical Observatory appearance vocabulary: appearance/Dashboard/Settings/AppShell/controller checks **32/32**; full UI suite **33 files / 181 tests**; selector and string manifests, type-check, production build, and diff-check passed. Managed planner is healthy on **17604/20003**; Test Genie run `20260919-215915-352ca68b` passed focused ui-health at **L5** across all seven routes with fourteen informational runtime confirmations. Legacy `system/light/dark` local-storage values are covered by migration logic; notification-hub remains degraded by its unrelated tunnel-manager freshness timeout.
- Visual evidence boundary: recent governed BAS files inspected under the shared capture store are generic health fixtures, not Observatory oracle captures. They are intentionally not promoted as visual-fidelity evidence. The source-level ChromeTheme audit confirms the supported theme-color meta, safe-area fill, and PWA channels; normal Safari browser chrome remains an explicit platform limitation.

## Binary startup

`api/main_e2e_test.go` uses `api-core/boottest` to build this API, verify its
health identity in isolated storage, and check shutdown. Keep the expected
service name in the scenario test. Shared process machinery and failure
regressions live in `packages/api-core/boottest`; see its package README section
for configuration and evidence limits. The existing E2E gate runs this test.
Durable review reflection implementation checks pass locally: API `go test ./...`, CLI `UPDATE_CLI_EVIDENCE=1 go test ./...`, UI type-check/build, and UI coverage (156 tests; 96.44% statements, 86.51% branches, 85.84% functions), with focused Test Genie receipt `20260919-120927-f6dec97a` recorded above.
### 2026-09-19 — feedback-loop timeline regression

- Focused UI regression: `DashboardPage`, `SettingsPage`, and `observatoryAppearance` — 16/16 passed.
- Full UI suite: 33 test files / 178 tests passed.
- Type-check and production build passed.
- Test Genie receipt `20260919-164826-9d4babc1`: unit, contracts, and experience passed 3/3.
- The combined `test:template-library` command remains blocked by an existing generated design-token mismatch; its template-link subtests pass.

### 2026-09-20 — commitment lifecycle vertical slice

- Commitment service tests pass **2/2**; Plan commitment/API UI coverage passes **18/18**; the full UI suite passes **34 files / 188 tests**.
- API and CLI suites pass, endpoint generation is current, and live CLI smoke created, listed, and revised a commitment with preserved unknown risk/acknowledgment.
- Managed restart is healthy and execution-enabled ui-health passes **L5** across all declared routes and desktop/mobile. The remaining advisory is the intentional Night raw-hex detector; forecast/risk derivation remains explicitly unimplemented.

### 2026-09-20 — commitment boundary composer and mobile Plan navigation

- Plan tests pass **17/17** after adding regression coverage for shared Textarea metadata fields in the commitment composer.
- Full UI suite passes **34 files / 188 tests**; type-check and production build pass.
- The Plan view switcher is now bounded and horizontally scrollable on narrow screens; no billing UI was added because monetization is explicitly deferred to R3 and remains a documented hypothesis.

### 2026-09-20 — deterministic forecast outlook vertical slice

- Forecast kernel tests pass **2/2**, API `go test ./...` passes, CLI `go test ./...` passes after refreshing primitive evidence, and endpoint generation is current.
- Forecast API live smoke: `personal-planner forecasts get --local-date 2026-09-19 --timezone America/New_York --horizon-days 28 --json` returned a current forecast with input fingerprint, central/cautious dates, reserve accounting, and an explanation.
- UI forecast API/Plan tests pass **19/19**; full UI suite passes **35 files / 190 tests**; type-check and production build pass.
- Managed restart is healthy on ports **17604/20003** with notification-hub degraded by its unrelated tunnel-manager freshness timeout. Execution-enabled ui-health passes **L5** across all declared routes and desktop/mobile; remaining findings are informational runtime confirmations and the intentional Night warm-ivory raw-color advisory.

### 2026-09-20 — promise-versus-forecast risk comparison

- Forecast kernel coverage now passes **3/3**, including at-risk and cautious-only commitment boundaries; API and CLI suites pass.
- Live Connect forecast response includes the active commitment's promised boundary, forecast finish, `on_track` risk state, and explanation.
- Plan/API focused UI coverage passes **19/19**; full UI suite passes **35 files / 190 tests**; type-check and production build pass.
- Fresh execution-enabled ui-health validation passes **L5** across all declared routes and desktop/mobile profiles. Informational findings remain the standard runtime confirmations plus the intentional Night warm-ivory raw-color advisory.

### 2026-09-20 — durable forecast snapshots and change records

- Forecast schema registration and API `go test ./...` pass; CLI `go test ./...` passes.
- Live persistence smoke: the first forecast created one `forecast_snapshots` row; adding a real work item changed the fingerprint and created a second snapshot plus one `forecast_change_records` row. The Connect response returned both snapshot IDs and the material change explanation.
- Full UI suite passes **35 files / 190 tests**; type-check and production build pass. Plan Outlook renders the persisted change explanation when present.
- Fresh execution-enabled `ui-health validate scenario personal-planner --json` passes with `VALIDATION_STATUS_PASSED` at **L5**; all 15 findings are informational runtime confirmations or the documented Night warm-ivory raw-color advisory.

- Experience-floor regression closure: focused Dashboard tests pass **9/9**, production build passes, and final managed Test Genie receipt `20260919-221722-0f9da741` passes **1/1 at L3 Complete**. It follows failed receipts `20260919-220409-22f36e6a` (Goal overflow plus Today negative-y geometry), `20260919-220943-22bc155c` (Goal cleared; Today remained), and `20260919-221225-c67dc897` (compact card exposed landscape action/link floor failures). The final run is clean; notification-hub remains degraded only by the recurring tunnel-manager freshness timeout.
- Feedback-loop shell/mobile closure: full UI suite passes **33 files / 182 tests**, type-check and production build pass. The first post-change experience run correctly caught the compact Auto control at **39.19px**; after raising its minimum to 44px, receipt `20260919-224444-d0d0d602` passes **1/1 at L3 Complete** with no blocking findings. This validates the route scroll reset, bounded sidebar geometry, Plan title disclosure, and mobile tap-target correction together.
- Review mobile hierarchy polish: Review tests pass **6/6** and the full UI suite passes **33 files / 182 tests**. The required date-field semantics remove the misleading optional annotation, mobile summary spacing is tightened, and the Today rejected-capture test waits for the mutation boundary. Type-check/build pass; experience receipt `20260919-225048-e7243910` passes **1/1 at L3 Complete** with no blocking findings. BAS mobile capture `bas-capture://eaf677c2-0d07-46c2-b4f2-701651c3f4eb/screenshot` is retained as runtime evidence, not visual-oracle certification.
- Settings language polish: Settings tests pass **4/4** and the full UI suite passes **33 files / 182 tests**. The product-specific description is covered by a regression assertion, type-check/build pass, and experience receipt `20260919-225504-3d9ad5bf` passes **1/1 at L3 Complete**. Mobile BAS evidence `bas-capture://0246c0c5-148e-4b6b-b8ff-984852f80742/screenshot` confirms the Settings route renders without the previous scaffold-like copy.
- Plan mobile placement hierarchy: Plan tests pass **15/15** and the full UI suite passes **33 files / 182 tests**. The mobile capacity strip is compacted, the Plan description is tightened, and placement fields are explicitly required. Type-check/build pass; managed restart is healthy; experience receipt `20260919-230345-ea56ef9c` passes **1/1 at L3 Complete**. BAS mobile evidence `bas-capture://2a9f8db0-239a-42e5-a855-9d14137f05c7/screenshot` is runtime evidence, not visual-oracle certification.
- Focus mobile shell polish: Focus tests pass **5/5** and the full UI suite remains **33 files / 182 tests**. Horizontal overflow is constrained at the shell/content boundary, the Focus work-item selector is required, and the actuals header date is kept on one line. Type-check/build pass; experience receipt `20260919-231123-d338d184` passes **1/1 at L3 Complete**. BAS mobile evidence `bas-capture://f77e6f4a-aa19-400e-8c7c-5672ed30128c/screenshot` confirms the full bottom navigation and required selector in the built route.
- Night Observatory visual checkpoint: Dashboard tests pass **9/9**, full UI suite passes **33 files / 182 tests**, type-check/build pass, and experience receipt `20260919-231835-7c5809dd` passes **1/1 at L3 Complete**. The built Night checkpoint `bas-capture://fdce80d1-6f46-45d7-a84b-aa57914c07a7/screenshot` confirms the approved warm-ivory next-action surface against the dark scene. The artifact has a direct-route scroll-position limitation and is evidence of the corrected region, not whole-surface visual-oracle certification.
- Observatory shell proportion checkpoint: AppShell and shell accessibility tests pass **13/13**, type-check/build pass, managed restart is healthy, and experience receipt `20260919-232657-b2e36929` passes **1/1 at L3 Complete**. The fresh Day runtime capture `bas-capture://a2c22589-6670-4d1c-8e6c-be1490647fbe/screenshot` confirms the 144px baseline sidebar and tighter reference-like main gutter. The unrelated notification-hub/tunnel-manager freshness timeout remains outside Personal Planner health.
- Goals progress control checkpoint: Goals tests pass **6/6**, the full UI suite passes **33 files / 182 tests**, type-check/build pass, managed restart is healthy, and experience receipt `20260919-233228-3d735d37` passes **1/1 at L3 Complete**. The raw range input is now covered by the shared Slider contract; progress persistence is commit-based rather than per-drag-frame.
- Plan timeline clipping checkpoint: Plan tests pass **15/15**, full UI suite passes **33 files / 182 tests**, type-check/build pass, and execution-enabled UI-health validation reaches **L5** across all seven routes and desktop/mobile profiles with zero `visual_text_clipped` findings. Experience receipt `20260919-234036-858caffe` passes **1/1 at L3 Complete**. Short timeline blocks retain title/popover detail while removing only cramped secondary metadata.
- Today focus-view geometry checkpoint: Dashboard tests pass **9/9**, type-check/build pass, managed restart is healthy, and experience receipt `20260919-234425-8377334b` passes **1/1 at L3 Complete**. The Focus ruler and grid now use the active span rather than a fixed eleven-column layout.
- Observatory Day/Night oracle checkpoint: full UI suite passes **33 files / 182 tests**, production build pass, managed restart is healthy, and experience receipt `20260919-234855-762c41c2` passes **1/1 at L3 Complete**. Reference-width runtime evidence is Day `bas-capture://bb5fe5a4-42bf-4b8e-8243-0485ed08b8c6/screenshot` and corrected Night `bas-capture://c4a0ee34-6d71-45cb-9c17-f6ff202f5839/screenshot`. The checkpoint specifically verifies the Night ivory action surface and its light secondary action; live data/date differences from the concept remain intentional.
- Responsive Observatory checkpoint: governed 390×844 captures confirm Day `bas-capture://4d98656a-a5e8-49ff-9949-62eff34ef197/screenshot` and Night `bas-capture://db77a784-55c0-41dd-91fa-08c801c0e18d/screenshot`. Visual inspection confirms no unintended outer shell border, correct bottom safe-area treatment, left-focused scenery, readable action card, and appearance-specific browser chrome.
- Settings scenery contract closure: focused appearance/Settings/Dashboard tests pass **19/19**; full UI suite passes **33 files / 184 tests**; type-check and production build pass. Managed restart is healthy on **17604/20003**. Execution-enabled `ui-health validate scenario personal-planner --json` passes at **L5** across all seven routes and desktop/mobile profiles; the only finding is the intentional Night warm-ivory raw-color advisory.
- Browser chrome branding closure: comprehensive Test Genie run `20260920-000129-85e490c3` completed **25/27**, with portability and performance failed; the scoped branding rerun `20260920-001027-c46e8d58` passes **1/1**. HTML and manifest theme colors now agree, and a dark-scheme fallback is declared. `brand-manager provider validate personal-planner` passes with Install Surface complete; only the known generated-token custom-font advisory remains.
- Initial-load performance closure: route-level loading split Plan, Goals, Focus, and Review into deferred chunks; the initial JS bundle is **804 KB** versus the previous roughly **880 KB**. Scenery now loads only the active panorama initially. Full UI suite passes **33 files / 184 tests**, type-check/build pass, and server-owned Test Genie run `20260920-002132-eec37121` passes the performance phase **1/1 at L3 Complete**. A single earlier full-suite async capture failure reproduced cleanly on focused rerun and the final full suite passed.
- Settings section hierarchy checkpoint: focused Settings/AppShell tests pass **17/17**, full UI suite passes **33 files / 184 tests**, type-check/build and diff-check pass. Execution-enabled `ui-health validate scenario personal-planner --json` passes at **L5** across all seven routes and desktop/mobile profiles; Test Genie experience run `20260920-002814-7492f69c` and current-build performance run `20260920-002931-67e3b25a` each pass **1/1 at L3 Complete**. The only UI-health advisory remains the intentional Night action-surface raw-color detector.
- Dynamic mobile chrome checkpoint: ThemeProvider tests pass **7/7**, full UI suite passes **33 files / 185 tests**, type-check/build and diff-check pass, and the managed runtime is healthy. Media-specific `theme-color` declarations are now regression-tested against the resolved Night appearance. Fresh execution-enabled ui-health validation remains **L5** across all seven routes and both viewport profiles.
- Plan routine-editor geometry closure: Plan tests pass **15/15**, full UI suite passes **33 files / 185 tests**, type-check/build and diff-check pass, managed restart is healthy, and fresh execution-enabled ui-health validation remains **L5** across all seven routes and desktop/mobile. Current-build Test Genie performance run `20260920-004309-3f0d8b52` passes **1/1 at L3 Complete**. The routine editor now has explicit responsive columns and safe label wrapping after BAS exposed an actual desktop label collision.
- Plan Capacity perspective: Plan tests pass **16/16**, full UI suite passes **33 files / 186 tests**, type-check and production build pass, and `git diff --check -- scenarios/personal-planner` passes. The new view is covered for accepted work, provider holds, freshness, the capacity meter, and the read-only allocation list; managed runtime/UI-health evidence is still pending for this bundle.
- Plan Month/Timeline perspective: Plan tests pass **16/16**, full UI suite passes **33 files / 186 tests**, type-check and production build pass, and scoped diff-check passes. Month and Timeline interaction coverage uses the accepted range fixture; the current Commitments gap is documented as a missing domain contract rather than represented by a fake UI state.
## 2026-09-20 — Forecast history

- `go test ./...` passed in `api`.
- `UPDATE_CLI_EVIDENCE=1 go test ./...` passed in `cli` and refreshed primitive evidence for `forecasts history`.
- `pnpm exec vitest run src/api/forecasts.test.ts src/pages/PlanPage.test.tsx` passed: 19 tests.
- `pnpm exec tsc --noEmit && pnpm build` passed.
- `make endpoints` passed.
- Managed `make restart` reached healthy on API 17604/UI 20003; notification-hub remained a known degraded dependency because tunnel-manager freshness timed out.
- `personal-planner forecasts history --json --limit 5` and the raw Connect list endpoint returned the two persisted forecast snapshots, including the recorded change explanation.

## 2026-09-20 — Goals editorial hierarchy

- `pnpm exec vitest run src/pages/GoalsPage.test.tsx` passed: 6 tests.
- `pnpm exec tsc --noEmit && pnpm build` passed.
- Managed `make restart` reached healthy on API 17604/UI 20003; notification-hub remained a known degraded dependency because tunnel-manager freshness timed out.
- Execution-enabled `ui-health validate scenario personal-planner --json` passed at `L5`; the existing intentional Night warm-ivory raw-color advisory remains the only finding class.

## 2026-09-20 — Focus ritual hierarchy

- `pnpm exec vitest run src/pages/FocusPage.test.tsx` passed: 5 tests.
- `pnpm exec tsc --noEmit && pnpm build` passed.
- Managed `make restart` reached healthy on API 17604/UI 20003; notification-hub remained a known degraded dependency because tunnel-manager freshness timed out.
- Execution-enabled `ui-health validate scenario personal-planner --json` passed at `L5` across desktop/mobile routes.

## 2026-09-20 — Review reflection hierarchy

- `pnpm exec vitest run src/pages/ReviewPage.test.tsx` passed: 6 tests.
- `pnpm exec tsc --noEmit && pnpm build` passed.
- Managed `make restart` reached healthy on API 17604/UI 20003; notification-hub remained a known degraded dependency because tunnel-manager freshness timed out.
- Execution-enabled `ui-health validate scenario personal-planner --json` passed at `L5` across desktop/mobile routes.

## 2026-09-20 — Responsive Plan chrome reconciliation

- Focused Experience run `20260920-020212-5ef76e09` correctly failed on portrait Plan document overflow; after wrapping the perspective switcher, `20260920-020534-53cee519` exposed the same issue at 844×390 landscape.
- Short-landscape four-column wrapping corrected the second profile. Authoritative receipt `20260920-021020-455eea56` passed `1/1` at **L3 Complete**, with clean structure reconciliation and no findings.
- Focused Plan/Review tests passed 24/24 before the final CSS-only breakpoint refinement; the production build passed after it, and the managed runtime is healthy on API 17604/UI 20003. Notification-hub remains degraded by the unrelated tunnel-manager freshness timeout.
- Visual checkpoint artifacts: Settings desktop/mobile and Review desktop BAS captures are recorded in PROGRESS; these are runtime evidence, not substitutes for the approved Today visual oracle.

## 2026-09-20 — Local typography evidence

- `pnpm exec vitest run` passed: **35 files / 190 tests**.
- `pnpm type-check` and `pnpm build` passed; `git diff --check -- scenarios/personal-planner` passed.
- `brand-manager provider validate personal-planner --json` passed at **L3** with zero findings after adding the local system-backed `Planner Sans` face.
- Execution-enabled `ui-health validate scenario personal-planner --json` passed at **L5** across all declared routes and desktop/mobile profiles. Remaining non-render findings are the intentional Night action-surface raw-color advisory and generic shared-component text-box measurements.

## 2026-09-20 — Cross-route Observatory theme parity

- A live managed-bundle Playwright check visited `/settings` and `/`; both resolved `data-resolved-theme=dark`, `--color-background: #141321`, and `--color-surface: #201e30`. This catches the prior defect where secondary routes inherited the default blue library palette.
- Runtime screenshots `/tmp/personal-planner-settings-theme.png` and `/tmp/personal-planner-today-theme.png` were inspected. Settings now uses the Observatory Night canvas, raised surface, amber selected state, and lilac section accents; Today and Settings share the same Night token family.
- `pnpm exec vitest run` passed **35 files / 190 tests**; `pnpm type-check` and `pnpm build` passed; managed restart reports Personal Planner healthy on API `17604` / UI `20003`.
- Final execution-enabled `ui-health validate scenario personal-planner --json` passed at **L5**. Findings are 14 runtime confirmations plus the intentional `standard_no_raw_hex` Night action-surface advisory; `git diff --check -- scenarios/personal-planner` passed.
- Server-owned Experience receipt `20260920-023409-c7221f57` passed **1/1** at **L3 Complete**, with clean structure reconciliation and no findings.
- A fresh 390×844 managed-browser check reports no horizontal overflow (`scrollWidth=390`, `innerWidth=390`) and produced `/tmp/personal-planner-settings-mobile-theme.png` and `/tmp/personal-planner-today-mobile-theme.png` for visual inspection.

## 2026-09-20 — Settings editorial header parity

- `pnpm exec vitest run src/pages/SettingsPage.test.tsx src/layout/AppShell.a11y.test.tsx` passed: **6 tests**.
- `pnpm type-check` and `pnpm build` passed.
- Managed `make restart` reached healthy on API `17604` / UI `20003`; notification-hub remained degraded because tunnel-manager freshness timed out.
- Live managed Settings capture: `/tmp/personal-planner-settings-editorial.png`.
- Experience receipt `20260920-025021-032b6ada` passed **1/1** at **L3 Complete** with clean structure reconciliation.

## 2026-09-20 — Today timeline compact disclosure

- `pnpm exec vitest run src/pages/DashboardPage.test.tsx` passed: **10 tests**.
- `pnpm exec vitest run` passed: **35 files / 190 tests**.
- `pnpm type-check`, `pnpm build`, and `git diff --check -- scenarios/personal-planner` passed.
- Managed `make restart` reached healthy on API `17604` / UI `20003`; notification-hub remained degraded because tunnel-manager freshness timed out.
- Final live managed Today capture: `/tmp/personal-planner-today-final-duration.png`; compact timeline labels are legible and full titles remain disclosed on interaction.
- Experience receipt `20260920-030058-bdc96885` passed **1/1** at **L3 Complete** with clean structure reconciliation.

## 2026-09-20 — Compact timeline visual-health hardening

- `ui-health validate scenario personal-planner --json` passed at **L5** after the Plan compact-block adjustment; the prior two `visual_text_clipped` warnings are no longer present. Remaining findings are 14 advisory runtime confirmations plus the intentional `standard_no_raw_hex` advisory.
- `pnpm type-check`, `pnpm build`, and `git diff --check -- scenarios/personal-planner` passed.
- Managed restart reached healthy on API `17604` / UI `20003`; notification-hub remained degraded by tunnel-manager freshness timeout.
- Experience receipt `20260920-030714-41b9f378` passed **1/1** at **L3 Complete** with clean structure reconciliation.

## 2026-09-20 — Mobile secondary-surface and chrome evidence

- 390×844 managed-browser audit captures: `/tmp/personal-planner-goals-mobile-audit.png`, `/tmp/personal-planner-focus-mobile-audit.png`, and `/tmp/personal-planner-review-mobile-audit.png`. Goals, Focus, and Review retain their editorial hierarchy, shared controls, scrollable content, and safe-area bottom navigation.
- `pnpm exec vitest run src/layout/AppShell.test.tsx src/theme/ThemeProvider.test.tsx` passed: **19 tests**.
- `pnpm exec vitest run` passed: **35 files / 191 tests**.
- `pnpm type-check`, `pnpm build`, and `git diff --check -- scenarios/personal-planner` passed.
- The new shell regression verifies Night `meta[name="theme-color"]`, `--rcl-status-fill`, and the rendered `planner-status-fill` strip.

## 2026-09-20 — Plan timeline disclosure readability

- `pnpm exec vitest run src/pages/PlanPage.test.tsx` passed: **18 tests**.
- `pnpm exec vitest run` passed: **35 files / 190 tests**.
- `pnpm type-check` and `pnpm build` passed.
- Managed `make restart` reached healthy on API `17604` / UI `20003`; notification-hub remained degraded because tunnel-manager freshness timed out.
- Live managed `/plan` capture: `/tmp/personal-planner-plan-after-title-fix.png`. Short blocks show duration labels; full titles remain available via title/accessible label/popover.
- Experience receipt `20260920-024551-942e7e1d` passed **1/1** at **L3 Complete** with clean structure reconciliation.

## 2026-09-20 — Day panorama contrast correction

- Reduced the Observatory Day panorama scrim from the flattening 88% middle
  surface wash to a lighter semantic blend, preserving the approved scene's
  warm atmosphere and content readability without changing Night behavior.
- `pnpm test` passed: **35 files / 191 tests**.
- `pnpm type-check` and `pnpm build` passed.
- `ui-health validate scenario personal-planner --json` passed at **L5** with
  no `visual_text_clipped` findings; remaining findings are informational
  runtime confirmations and the existing raw-palette advisory.
- Managed `make restart` reached healthy on API `17604` / UI `20003`; the
  unrelated notification-hub/tunnel-manager freshness timeout remains.

## 2026-09-20 — Sidebar route-entry scroll hardening

- AppShell route reset now clears every descendant scroll container inside the
  library-owned sidebar, protecting the full navigation list across deep-link
  and route transitions.
- Focused AppShell validation passed: **15 tests**.
- `pnpm test` passed: **35 files / 195 tests**.
- `pnpm type-check` and `pnpm build` passed.
- `git diff --check -- scenarios/personal-planner` passed.

## 2026-09-20 — Mobile browser-chrome viewport contract

- `ui/index.html` now follows the Web Console safe-area contract with
  `interactive-widget=resizes-content`, `viewport-fit=cover`, explicit zoom
  behavior, and `color-scheme=light dark`.
- `pnpm test` passed: **35 files / 195 tests**.
- `pnpm type-check` and `pnpm build` passed.
- `git diff --check -- scenarios/personal-planner` passed.
- Managed `make restart` reports the planner healthy on API `17604` / UI
  `20003`; it continues in degraded mode only because notification-hub hit the
  unrelated tunnel-manager freshness timeout.

## 2026-09-20 — Responsive scenery and query-aware route reset

- Landscape Day/Night captures at 844×390 passed readiness and visually
  confirmed a fixed, left-focal Observatory scene without squashing; Night
  retains visible stars:
  `bas-capture://3db06971-b5e1-408e-a9f4-7de0774118cb/screenshot` and
  `bas-capture://0417de08-55d2-4772-773d7b1fdea9/screenshot`.
- AppShell route-entry reset now responds to both pathname and URL search,
  preventing appearance/deep-link changes from carrying stale scroll state
  through the main surface or sidebar.
- `pnpm exec vitest run src/layout/AppShell.test.tsx` passed: **15 tests**.
- `pnpm test` passed: **35 files / 195 tests**.
- `pnpm type-check`, `pnpm build`, and `git diff --check -- scenarios/personal-planner` passed.
- `ui-health validate scenario personal-planner --json` passed at **L5**.
- Managed planner restart reached healthy on API `17604` / UI `20003`; the
  unrelated notification-hub/tunnel-manager freshness timeout remains.

## 2026-09-20 — Persistent validation-data cleanup

- Read-only inspection identified three smoke work items, smoke focus sessions,
  a smoke commitment/goal, generated proposal/forecast history, and fixture
  provider connections in the persistent local database.
- The database was backed up before deleting those exact validation records;
  planning profile/schema state was retained. No production code or test
  fixtures were removed.
- Fresh managed Plan capture confirms the honest empty state: `0 min` planned,
  `360 min` available, `360 min` breathing room, and no routines:
  `bas-capture://94041309-c71c-4203-928b-c79d81c64be2/screenshot`.
- Managed restart reports the planner healthy on API `17604` / UI `20003`;
  notification-hub remains degraded by the unrelated tunnel-manager timeout.

## 2026-09-20 — Cross-surface appearance transition contract

- Secondary planner surfaces now transition background, foreground, border, and
  elevation together when the resolved appearance changes.
- The transition layer is disabled under `prefers-reduced-motion: reduce`.
- The shared main scroller now publishes bottom `scroll-padding` for fixed
  mobile navigation and safe-area space.
- `pnpm test` passed: **35 files / 195 tests**.
- `pnpm type-check`, `pnpm build`, and `git diff --check -- scenarios/personal-planner` passed.
- `ui-health validate scenario personal-planner --json` passed at **L5**.
- Managed planner restart is healthy on API `17604` / UI `20003`; the known
  notification-hub/tunnel-manager freshness failure remains unrelated.
- Fresh Review capture at 1585×992 with `networkidle` passed readiness and
  visually confirmed the full sidebar: `bas-capture://2839fac0-519e-461f-b325-e525b3af2d29/screenshot`.
- The managed restart reached healthy on API `17604` / UI `20003`; the
  unrelated notification-hub/tunnel-manager freshness timeout remains.

## 2026-09-20 — Compact Observatory appearance control

- Day’s Auto/Day/Night control is icon-first, with only the selected label
  expanded; all three buttons retain accessible names, pressed state, and
  native titles.
- Fresh BAS Day capture at 1585×992 with `networkidle` passed readiness and
  visually confirmed the compact control:
  `bas-capture://835f453b-34f9-4bd3-8b17-4f4fc8e471e8/screenshot`.
- `pnpm test` passed: **35 files / 195 tests**.
- `pnpm type-check` and `pnpm build` passed.
- `git diff --check -- scenarios/personal-planner` passed.

## 2026-09-20 — Day panorama atmospheric blend refinement

- The Day-only scene scrim was reduced so the cloudy sky and mountain texture
  remain visible without weakening the light-theme text contrast.
- Fresh BAS capture at 1585×992 with `networkidle` passed readiness and was
  visually inspected: `bas-capture://c1904fae-6c06-4273-9fcc-9ecdbc31603b/screenshot`.
- `pnpm test` passed: **35 files / 194 tests**.
- `git diff --check -- scenarios/personal-planner` passed.
- `ui-health validate scenario personal-planner --json` passed at **L5** with
  the existing informational runtime confirmations and raw-palette advisory.
- Managed restart reached healthy on API `17604` / UI `20003`; the unrelated
  notification-hub/tunnel-manager freshness timeout remains.

The AppShell URL-override regression now passes as part of the full suite and
asserts the rendered sidebar baseline alongside the Night shell class and
document theme attribute. The final full-suite count for this checkpoint is
**35 files / 194 tests**.

Fresh BAS reference-width captures after the managed restart:

- Day: `bas-capture://19b4e8a1-a161-470a-bba4-2f18544d2e02/screenshot`
- Night: `bas-capture://194b7019-aa13-43c0-8644-b9df775c1dea/screenshot`

Both were captured at 1585×992 with `networkidle`; visual inspection confirms
the sidebar and Today surface now resolve together in each appearance.

## 2026-09-20 — Shared appearance resolution

- URL appearance overrides now resolve in `ThemeProvider` before AppShell
  mounts, so deep-linked Day/Night captures cannot leave the sidebar, bottom
  navigation, status strip, or browser chrome in the prior theme.
- Focused ThemeProvider/AppShell/Dashboard validation passed: **31 tests**.
- `pnpm test` passed: **35 files / 193 tests**.
- `pnpm type-check` and `pnpm build` passed.
- `ui-health validate scenario personal-planner --json` passed at **L5** with
  only informational runtime confirmations and the existing raw-palette
  advisory.
- Managed `make restart` reached healthy on API `17604` / UI `20003`; the
  unrelated notification-hub/tunnel-manager freshness timeout remains.

## 2026-09-20 — Sidebar resize range correction

- The AppShell resize separator now exposes `aria-valuemin=128` and
  `aria-valuemax=280`, retaining the 144px default and adding 8px/40px
  keyboard steps.
- `pnpm exec vitest run src/layout/AppShell.test.tsx` passed: **13 tests**.
- `pnpm test` passed: **35 files / 192 tests**.
- `pnpm type-check` and `pnpm build` passed.
- Managed `make restart` reached healthy on API `17604` / UI `20003`; the
  unrelated notification-hub/tunnel-manager freshness timeout remains.
