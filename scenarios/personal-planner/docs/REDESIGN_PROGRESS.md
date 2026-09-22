# Personal Planner redesign checkpoint

## 2026-09-21 — resumed North Star goal

### Changed

- Recalled the backlog, compositing contract, estimation design, mockups, source-art contract, and testing policy.
- Confirmed the repository has substantial prior Observatory shell/theme work, but the §A–§G backlog remains open.
- Confirmed the working tree contains unrelated changes outside `scenarios/personal-planner/**`; this goal will not modify them.

### Verified

- Existing UI tests and managed UI-health evidence are recorded in `docs/internal/PROGRESS.md`.
- Captured source plates exist under `docs/mockups/source-art/` for Plan, Focus, and Settings.

### Remaining

- Shared breakpoint hook and component-adaptive route composition.
- Shared procedural sky/compositor and processed captured plates for Plan, Focus, Settings.
- Global natural-language capture, mobile FAB, and scheduling/defer affordances.
- Countdown/overtime Focus timer and end-session evidence.
- Completion/estimate history, goal variance, Review calibration, and agent read model.
- Per-page functional gaps, focused regressions, targeted Test Genie runs, and final comprehensive/visual certification.

### Unverified

- The North Star end state is not yet met. No completion claim is made from existing UI-health receipts.

## 2026-09-21 — capture and scene foundation increment

### Changed

- Added SSR-safe breakpoint composition, shared ObservatoryScene layers, adaptive headers, global capture with natural-language placement through the calendar preview/apply contract, countdown/overtime Focus UI, Review calibration/reflection surfaces, Goals filters, keyed scene assets, and resolved-theme alignment for shared scenes.

### Verified

- UI type-check passes.
- UI coverage passes with 39 files and 214 tests at 97.3% statements, 85.03% branches, and 85.31% functions.
- Capture tests cover desktop shortcut, desktop trigger, mobile FAB, free-form capture, placement metadata, and clock normalization.

### Remaining

- Durable estimation/completion schema and API read model, linked actuals and dates, variance badges backed by stored data, reschedule reasons/snooze, reminders, goal auto-advancement/drift, Today overdue/momentum, full mobile interaction models, Settings drill-in redesign, and visual polish/compositor validation.

### Unverified

- Managed scenario targeted and comprehensive Test Genie runs after this increment; six-page desktop/mobile visual gate; production appearance against the mockup at both themes.

## 2026-09-21 — validation checkpoint

### Verified

- Production UI build passes after the mobile composition increment.
- The comprehensive Test Genie run `20260921-044857-e46d1ac5` completed with structure, business, performance, and experience green; unit execution is green but the run remains failed on existing unit-policy projection drift, quality lazy-chunk recovery, and UI-health interop/offscreen findings. These are recorded as evidence, not hidden.
- Experience reached L3 for the six declared pages in the comprehensive run.

### Remaining

- The comprehensive run is not a North Star certificate: UI-health reports 30 offscreen-interactive findings, and durable learning/data work is still absent.

## 2026-09-21 — stop checkpoint

### Changed

- Moved global shortcut handling into `ui/src/hooks/useKeyboardShortcut.ts`, moved Focus adaptive guidance into a shared component, added mobile-specific Plan/Goals/Focus/Settings composition, container-relative scene sizing, and a stale-chunk recovery path.

### Verified

- `pnpm -C scenarios/personal-planner/ui type-check` passes.
- `pnpm -C scenarios/personal-planner/ui test` passes: 39 files / 217 tests.
- `pnpm -C scenarios/personal-planner/ui test:coverage` passes: 96.94% statements, 85.20% branches, 85.43% functions.
- Production UI build passes.
- API targeted Go tests pass for goals, focus, and review.
- Focused Test Genie run `20260921-045444-f1ac1a5a`: unit and experience passed; quality failed on stale-chunk detection and existing quality-policy findings.

### Remaining / unverified

- Goal remains incomplete. The backlog’s durable learning schema/read model, completion/estimate history, variance badges from stored dates, reschedule reasons/snooze/reminders, goal auto-advancement/drift, Today overdue/momentum, and full Settings drill-in remain open.
- Comprehensive run `20260921-044857-e46d1ac5` is failed: UI-health reports 30 offscreen-interactive findings; quality reports stale-chunk recovery and policy findings.
- No final six-page visual/experience certification exists, and the North Star must not be marked done.

## 2026-09-21 — durable learning foundation increment

### Changed

- Added additive completion/estimate history columns and tables for work items, target/completed date columns for goals and milestones, allocation linkage on manual actuals, and append-only reschedule history.
- Added the durable `review.estimation_bias` projection and a review service reconciliation that computes stored plan-to-actual variance from explicitly linked allocations.
- Added the API read model route `/api/v1/review/calibration?period=all_time|last_30d|last_90d` and changed Review’s calibration card to read persisted projection data instead of comparing unrelated daily totals.

### Verified

- `go test ./...` passes in `scenarios/personal-planner/api`.
- Calibration regression proves two linked records are reconciled and the stored `estimation_bias` value matches the API model.
- `pnpm type-check` passes.
- UI coverage passes: 39 files / 219 tests, 96.9% statements, 85.02% branches, 85.51% functions.
- Focused Test Genie run `20260921-100002-ff75067c` passed experience; unit failed only at the pre-existing coverage command threshold before the added branch tests. The local coverage command now passes after those tests.

### Remaining / unverified

- Re-run Test Genie unit after the coverage repair; the durable API route still needs live managed-scenario proof with a stored response.
- History tables are now declared and persisted, but write paths still need to capture original-estimate changes, completion records, allocation IDs, and reschedule reason codes end to end.
- Goal variance badges, calibration trend/category detail, agent-facing read contract, reminders/snooze, overdue/momentum, and remaining visual/mobile gaps remain open.

## 2026-09-21 — learning write-path increment

### Changed

- New work items now retain their original estimate and open status in durable storage.
- Completing a goal or milestone now writes its completion date; carry-forward writes an append-only `reschedule_history` row with a reason code.
- Manual actual writes now resolve and persist the first accepted allocation for the same work item/date, and the persisted allocation link is returned by the internal focus model.

### Verified

- API `go test ./...` passes after the write-path changes.
- Focus regression proves a recorded actual automatically links to `allocation-1`; calendar tests prove carry-forward remains idempotent with history storage.

### Remaining / unverified

- The public focus proto does not yet expose the allocation ID; the server-side automatic resolver keeps the link durable for current UI flows, but explicit allocation selection and correction reasons still need a public contract.
- Continue with stored goal variance/read fields, agent-facing calibration response validation, snooze/reminders, Today overdue/momentum, and the visual/compositor backlog.

### Live evidence

- Managed `make start` brought the scenario healthy in degraded mode because `notification-hub` was unavailable; the planner API itself was healthy.
- `curl http://127.0.0.1:17604/api/v1/review/calibration?period=all_time` returned the persisted projection shape with `updated_at` and `sample_size` from the live SQLite runtime.
- Follow-up Test Genie unit run `20260921-100847-1df308f1` passed. Its remaining maturity finding is the pre-existing `MISSING_INJECTABLE_SEAM` warning and advisory low-coverage debt; execution and coverage are green.

## 2026-09-21 — stored goal variance surface increment

### Changed

- Goal completion now persists `completed_date`; milestone creation preserves `original_due_date` and seeds a goal target date when one is absent.
- Added the durable review read model route `/api/v1/review/goal-variances`, deriving positive/negative day deltas from stored goal or milestone dates.
- Goals now fetch that read model and render warm early/on-time/over variance badges across active/all/completed views.

### Verified

- API regression covers stored early and late goal deltas and labels; `go test ./internal/goals ./internal/review` passes.
- UI API and Goals regressions pass: 14 tests; `pnpm type-check` passes.

### Remaining / unverified

- The goal target date is currently seeded from the first dated milestone because the existing public goals proto has no target-date field; explicit target-date editing still needs a scenario-local contract.
- Reminders/snooze, Today overdue/momentum, agent read validation, and visual/mobile certification remain open.

### Gate evidence

- Full UI coverage now passes: 39 files / 221 tests, 96.92% statements, 85.15% branches, 85.55% functions.
- Managed Test Genie run `20260921-103258-f92259a1` passed both `unit` and `experience`; unit is L2 with the pre-existing `MISSING_INJECTABLE_SEAM` warning and advisory low-coverage findings, while experience is L3.

## 2026-09-21 — durable next-step snooze increment

### Changed

- Added persisted `snoozed_until` state and append-only `work_item_snoozes` reason history.
- Added `POST /api/v1/work/{id}/snooze`; deferred open work is excluded from the server-owned work/day projection until its local date arrives.
- Added a Today “Not now” action for unscheduled next steps that defers them to tomorrow and refreshes the authoritative work and plan queries.

### Verified

- SQLite regression proves deferred items are hidden and the reason is persisted; full API `go test ./...` passes.
- Work API and Today regressions pass: 15 focused UI tests; `pnpm type-check` passes.

### Remaining / unverified

- Notification delivery and target-reached reminders still need a real integration path; scheduled-item carry-forward remains the existing defer path.
- Today overdue/momentum, goal drift/auto-advancement, explicit goal target editing, agent read validation, and visual/mobile certification remain open.

### Gate evidence

- Full UI coverage passes: 39 files / 223 tests, 96.94% statements, 85.48% branches, 85.75% functions.
- Managed Test Genie run `20260921-104320-2c3548a4` passed `unit` and `experience`; unit remains L2 only because of the pre-existing injectable-seam warning and advisory coverage debt, while experience is L3.

## 2026-09-21 — Today overdue and momentum increment

### Changed

- Added `/api/v1/review/today-signals`, a server-owned read model for open overdue accepted work, overdue minutes, and consecutive active-day momentum.
- Today now shows the overdue/momentum signal beside the day plan, with focused API and Dashboard regressions.

### Verified

- Review regression proves completed allocations are excluded from overdue counts while focus/manual activity contributes to momentum.
- Review API and Dashboard regressions pass: 18 focused UI tests; API review tests pass; `pnpm type-check` passes.

### Remaining / unverified

- Reminder delivery, goal drift/auto-advancement, explicit goal target editing, and visual/mobile certification remain open.

### Gate evidence

- Full UI coverage passes: 39 files / 224 tests, 96.95% statements, 85.31% branches, 85.82% functions.
- Managed Test Genie run `20260921-104825-7714dc4c` passed `unit` and `experience`; unit remains L2 with the same pre-existing injectable-seam warning and advisory low-coverage debt, while experience is L3.

## 2026-09-21 — goal drift nudge increment

### Changed

- Added `/api/v1/review/goal-drifts`, comparing stored progress with elapsed pace toward each active goal’s stored target date.
- Goals now show a warm “Behind pace” or “Ahead of pace” nudge when drift exceeds the tolerance band; on-pace goals stay quiet.

### Verified

- Review regression proves stored timestamps and progress produce a behind-pace result.
- Review API and Goals regressions pass: 18 focused UI tests; full UI coverage passes with 226 tests; `pnpm type-check` passes.

### Remaining / unverified

- Explicit goal target-date editing, target-reached reminder delivery, goal auto-advancement, and visual/mobile certification remain open.

### Gate evidence

- Full UI coverage passes: 39 files / 226 tests, 96.97% statements, 85.25% branches, 85.9% functions.
- Managed Test Genie run `20260921-105254-5dc27245` passed `unit` and `experience`; unit remains L2 with the pre-existing injectable-seam warning and advisory low-coverage debt, while experience is L3.

## 2026-09-21 — explicit goal target editing increment

### Changed

- Added the server-owned `POST /api/v1/goals/{id}/target-date` seam, including validation, persistence, revision advancement, and clear-date support.
- Goals now expose an inline target-date editor; goal drift returns active goals without dates so the empty state remains editable instead of disappearing.

### Verified

- Go service/repository validation and drift regressions pass.
- Goals REST transport and active-card save interaction regressions pass; malformed dates are rejected before persistence.

### Remaining / unverified

- Reminder delivery and target-reached reminders, goal auto-advancement, agent read-model validation, and visual/mobile certification remain open.

### Gate evidence

- Focused UI tests and type-check pass; full API regression passes.

## 2026-09-21 — in-app reminder read model increment

### Changed

- Added `/api/v1/review/reminders`, a server-derived reminder feed for upcoming accepted blocks, overdue open work, and active goals whose target date is today.
- Today renders the feed as a compact NUDGES surface and refreshes it on a one-minute cadence; the data remains honest when no reminders exist.

### Verified

- Review persistence regression proves stored allocations, work status, and goal target dates produce the expected reminder kinds.
- Review transport and Dashboard regressions pass; type-check passes.

### Remaining / unverified

- External notification-hub delivery, target-reached focus chime, reminder preferences/quiet hours, goal auto-advancement, agent read-model validation, and visual/mobile certification remain open.

### Gate evidence

- Focused API/UI tests pass: 20 UI tests in the reminder/review/Dashboard slice; full Go suite remains green.

## 2026-09-21 — focus target cue increment

### Changed

- Focus now emits a one-shot browser chime when a countdown crosses into overtime; the cue is guarded for browsers without an audio context and never replaces the visible overtime evidence.

### Verified

- Focus regressions and UI type-check pass.

### Remaining / unverified

- Quick-pick/Pomodoro break prompts, end-of-session notes, distraction reasons, external notification delivery/preferences, goal auto-advancement, agent read-model validation, and visual/mobile certification remain open.

### Gate evidence

- FocusPage: 6 tests passed; type-check passed.

## 2026-09-21 — durable end-of-session notes increment

### Changed

- Added the `focus_session_notes` SQLite table and focus service/repository seams for validated, updatable notes attached to ended sessions.
- Focus presents a close-the-loop note form after ending; Review reads notes for the selected day and renders them as stored session evidence.

### Verified

- Focus service regression proves ended-session validation and date-filtered note retrieval.
- Focus transport, FocusPage, and ReviewPage regressions pass; type-check passes.

### Remaining / unverified

- Pomodoro cycles/break prompts, distraction reasons, external notification delivery/preferences, goal auto-advancement, agent read-model validation, and visual/mobile certification remain open.

### Gate evidence

- Focus service and 17 focused UI tests pass; type-check passes.

### Managed gate

- Test Genie run `20260921-111144-f89af999` passed `unit` and `experience`; unit remains L2 with the pre-existing injectable-seam and advisory low-coverage findings, while experience remains L3.

## 2026-09-21 — Pomodoro cycle increment

### Changed

- Focus now offers persisted `pomodoro` mode with configurable 2/4/6 cycles, explicit 5-minute break prompts, paused break time, and honest resume targets.
- Each cycle continues to use the server focus session for active-time accounting; the local UI only tracks the break countdown and cycle presentation.

### Verified

- Focus UI regression proves Pomodoro mode selection and persisted start mode; all 8 FocusPage tests and type-check pass.

### Remaining / unverified

- Distraction/pause reasons, reminder preferences/external delivery, goal auto-advancement, agent read-model validation, and final visual/mobile certification remain open.

### Gate evidence

- Full UI coverage passes: 39 files / 233 tests, 96.45% statements and 86.03% branches.
- Managed Test Genie run `20260921-111731-5b5c2771` passed `unit` and `experience`; unit remains L2 with the existing injectable-seam/advisory coverage findings and experience remains L3.

## 2026-09-21 — durable pause-reason increment

### Changed

- Added `focus_pause_events` persistence with validated reasons: interrupted, blocked, distracted, rest, and other.
- Focus records the selected reason after a successful pause transition; Review exposes date-filtered pause patterns alongside session notes.

### Verified

- Focus service regression proves reasons require a paused session and persist with the local date.
- Focus transport, FocusPage, ReviewPage, and full UI coverage pass: 234 tests, 96.48% statements, 85.93% branches.

### Remaining / unverified

- Reschedule reason codes, reminder preferences/external delivery, goal auto-advancement, agent read-model validation, and final visual/mobile certification remain open.

## 2026-09-21 — reschedule reason increment

### Changed

- Carry-forward now validates and persists `interrupted`, `underestimated`, `blocked`, `deprioritized`, and `external` reason codes in `reschedule_history`.
- Review adds a reason selector and uses the governed REST seam; legacy Connect callers retain the explicit `deprioritized` default.

### Verified

- Calendar service validation, REST transport, and Review interaction regressions pass.
- Full UI coverage passes: 235 tests, 96.49% statements, 85.87% branches; full Go suite passes.

### Remaining / unverified

- Reminder preferences/external delivery, goal auto-advancement, agent read-model validation, conflict/breathing-room guardrails, and final visual/mobile certification remain open.

## 2026-09-21 — completion-to-goal increment
### Changed
- Added durable work completion through Today and a server-owned completion endpoint.
- Completion records actual-versus-original estimate history, advances linked milestones when prerequisites permit, and completes milestone-led goals when all milestones are complete.
### Verified
- SQLite completion regression proves work status, completion history, milestone advancement, and goal auto-completion persist together.
- Full Go suite and UI coverage pass: 239 tests, 96.58% statements, 85.61% branches, 86.09% functions.
- Managed Test Genie run `20260921-114822-7f582f12` passed unit and experience; experience is at maximum maturity.
- Calendar reschedule history now uses the repository ID seam; full Go suite and unit gate remain green.
### Remaining / unverified
- External notification-hub delivery, agent read-model validation, energy-aware suggestions, wins/gratitude, trend sparklines, and final visual/mobile certification remain open.

## 2026-09-21 — explicit agent read-model increment
### Changed
- Added `/api/v1/agent/read-model`, a dedicated machine-facing projection containing the persisted `estimation_bias` calibration and generation timestamp.
- Added client transport coverage and retained the existing Review calibration route for human-facing use.
### Verified
- Review service, handler compilation, UI type-check, API transport tests, full UI coverage (240 tests), and managed Test Genie run `20260921-115345-edb4ed06` pass.
### Remaining / unverified
- External notification-hub delivery, energy-aware suggestions, wins/gratitude, trend sparklines, and final visual/mobile certification remain open.

## 2026-09-21 — durable wins increment
### Changed
- Added a separate `review_wins` store and Review API for one concise daily win or gratitude line.
- Review now loads and saves the win independently from guided reflection, preserving both signals in durable history.
### Verified
- Win service persistence and validation, API transport, Review interaction, full Go suite, and UI coverage pass.
- UI coverage: 242 tests, 96.62% statements, 85.48% branches, 86.36% functions.
- Managed Test Genie run `20260921-115904-6bcb1124` passed unit and experience; experience remains at maximum maturity.
### Remaining / unverified
- External notification-hub delivery, energy-aware suggestions, trend sparklines, original-estimate editing, and final visual/mobile certification remain open.

## 2026-09-21 — original-estimate memory increment

### Changed

- Added a Plan-day editor for remaining effort with explicit reason codes: new information, scope changed, blocked, or better understood.
- Added a REST mutation that updates remaining minutes without overwriting `original_estimate_minutes` and writes each change to `work_item_estimation_changes`.

### Verified

- SQLite persistence, service validation, transport, UI interaction, full Go suite, and UI coverage pass.
- UI coverage: 244 tests, 96.65% statements, 85.45% branches, 86.15% functions.

### Remaining / unverified

- External notification-hub delivery, energy-aware suggestions, trend sparklines, and final visual/mobile certification remain open.

## 2026-09-21 — capture and review signal increment

### Changed

- Reconciled the backlog with the existing tested natural-language parser, global desktop shortcut/mobile FAB, and SSR-safe breakpoint hook.
- Replaced the Review calibration bar with a real SVG sparkline over the stored accuracy trend, including a truthful single-observation fallback.

### Verified

- Review and Plan focused tests pass; full UI coverage passes with 244 tests, 96.65% statements, 85.32% branches, and 86.15% functions.
- Natural capture, global capture, breakpoint behavior, and calibration transport remain covered by the existing UI suite.

### Remaining / unverified

- Energy-aware suggestions, external notification-hub delivery, final visual/mobile certification, and the deeper compositing/settings work remain open.

## 2026-09-21 — durable reminder preferences increment
### Changed
- Added durable reminder preferences with enable/disable, quiet-hour boundaries, and configurable lead time.
- Reminder reads now enforce the saved preference and quiet-hour policy; Settings exposes the controls through a persisted REST seam.
### Verified
- Reminder preference persistence, validation, quiet-hour suppression, API mapping, Settings interaction, and full UI coverage pass.
- UI coverage: 237 tests, 96.56% statements, 85.82% branches, 85.96% functions.
- Managed Test Genie run `20260921-114008-3865cb48` passed unit and experience; experience is at maximum maturity.
### Remaining / unverified
- External notification-hub delivery, goal auto-advancement, agent read-model validation, conflict/breathing-room guardrails, and final visual/mobile certification remain open.

## 2026-09-21 — forecast seam hardening increment
### Changed
- Forecast persistence now accepts an injected ID generator, removing direct UUID generation from the service path and making snapshot/change-record writes deterministic in tests.
### Verified
- Forecast seam regression and the full Go suite pass.
- Managed Test Genie unit + experience run `20260921-112953-baf1c860` passed; experience is complete and unit execution is green.
### Remaining / unverified
- Reminder preferences/external delivery, goal auto-advancement, agent read-model validation, conflict/breathing-room guardrails, and final visual/mobile certification remain open.

### Gate evidence

- Managed Test Genie run `20260921-112247-cfc3de78` passed `unit` and `experience`; unit remains L2 with the existing injectable-seam/advisory coverage findings and experience remains L3.

## 2026-09-21 — energy, mobile, and visual compositing increment

### Changed

- Added a tested Plan heuristic that identifies deep-work language and offers a one-tap morning placement suggestion.
- Added a mobile Today next-step bar in the thumb zone, preserving the existing draft, focus, and capture actions while keeping capacity secondary.
- Reframed Settings as “Calibrate your instrument” and added mobile drill-in navigation for its dense sections.
- Chroma-keyed the Plan, Focus, and Settings captured plates into transparent WebP assets and wired desktop/mobile/day/night scene paths through the shared Observatory scene layer.
- Reconciled the visual backlog with the existing shared procedural sky, ambient effects, page-specific scene compositions, and mobile page variants.

### Verified

- UI type-check and strings check pass.
- Full UI coverage passes: 39 files, 246 tests, 96.67% statements, 85.65% branches, 86.24% functions.
- Dashboard mobile regression passes; Plan and Settings focused suites pass.
- Managed Test Genie run `20260921-123042-2b44b2fa`: unit, structure, business, performance, and experience passed. Quality reported existing TypeScript/Makefile findings and did not pass.

### Remaining / unverified

- Quality-health findings remain: one injectable seam advisory, broad low-coverage advisories, TypeScript dangerous-pattern detection, lazy-chunk recovery detection, and Makefile quality-target detection.
- External notification-hub delivery and final visual/manual certification remain outside the automated evidence above.

## 2026-09-21 — quality and runtime-surface hardening

### Changed

- Replaced the local lazy-route recovery wrapper with the shared `installChunkReloadGuard()` contract in `main.tsx`.
- Repaired the scenario Makefile quality targets through the Quality Health fixer, then corrected their API/UI working directories.
- Removed non-null assertions from production and test paths touched by the planner surfaces.
- Added shortcut intent relay for the global capture keyboard hook.
- Extracted Review calibration into a reusable component, replaced raw Today action buttons with the governed Button primitive, and kept Settings’ page-section navigation semantically scoped.
- Hardened AppShell route-entry scroll reset across the real shell scroll container, descendants, document, and asynchronous layout settling.

### Verified

- Quality Health now clears lazy-chunk recovery, Makefile quality gates, shortcut relay, and project-standard primitive/component findings.
- UI type-check and strings check pass; full UI coverage passes with 39 files and 247 tests at 97.02% statements.
- Focused AppShell, Dashboard, Review, and Settings tests pass.

### Remaining / unverified

- Quality Health still reports TypeScript assertions in the broader UI source/test surface and an uncovered runtime contract advisory.
- UI Health’s browser validator still reports 34 `visual_offscreen_interactive` errors despite route-entry reset hardening; this remains the next runtime-layout repair target.
- Comprehensive post-hardening Test Genie certification and final visual/manual inspection remain pending.

## 2026-09-21 — final mobile Review runtime certification

### Changed

- Constrained the shared Observatory and Review mobile layout with box sizing, bounded grids, and min-width contracts.
- Fixed Review’s durable daily-win adapter to accept the API’s capitalized JSON response shape; missing calibration trends and optional list responses now degrade safely instead of crashing the page.
- Added regression coverage for the legacy daily-win response shape.

### Verified

- UI type-check and production build pass.
- UI coverage passes: 39 files, 248 tests, 97.03% statements, 85.25% branches, 85.61% functions.
- Mobile browser probe at 390px confirms Review renders and document/body width remain 390px.
- Direct `ui-health validate scenario personal-planner --json` passes with zero error findings.
- Final managed Test Genie run `20260921-134344-40b72666` passes all 7 phases: unit, quality, structure, business, performance, experience, and ui-health; 0 failed, 0 skipped.
- `git diff --check -- scenarios/personal-planner` passes.

### Known advisory posture

- Test Genie retains non-blocking unit low-coverage and injectable-seam advisories.
- Quality Health retains its detection-only TypeScript type-assertion warning and runtime coverage-gap info finding; no quality phase errors remain.

## 2026-09-21 — Today mobile composition refinement

### Changed

- Re-composed Today on phone widths so the day timeline owns the first useful fold, the next-step action follows it in the thumb zone, and capacity/plan status stay out of the mobile Today surface where the mockup places them in Plan.
- Improved the mobile next-step title contrast and made the Today signal strip resilient to narrow-width wrapping.

### Verified

- Headless rendered captures at 1440×1000 and 390×1400 show the desktop two-column Observatory and mobile timeline-first composition; the mobile capture also shows the scene, bottom navigation, action card, and signals without horizontal overflow.
- `pnpm -C scenarios/personal-planner/ui type-check` passes.
- `pnpm -C scenarios/personal-planner/ui test` passes: 39 files / 248 tests.
- Managed Test Genie run `20260921-212730-9c6b8bdb` passes `unit` and `experience`; experience is L3/complete, while unit remains L2 only for the known injectable-seam and advisory coverage findings.

### Remaining / unverified

- Plan, Goals, Focus, Review, and Settings still need the same adversarial desktop/mobile Day/Night rendered comparison against the oracle, especially scene framing and fold hierarchy.
- No final six-page visual/manual certification exists yet; the North Star remains incomplete.

## 2026-09-21 — Plan mobile capacity refinement

### Changed

- Replaced the mobile Plan three-column capacity strip with the mockup’s compact “Today’s capacity” card: planned versus total available time plus a proportional ring.
- Kept the desktop command-chart capacity strip unchanged and covered the mobile branch with a regression.

### Verified

- Rebuilt and restarted through `make restart`; the managed scenario returned healthy in degraded mode because `notification-hub` could not refresh its unrelated tunnel-manager dependency.
- Live 390×1000 Day render shows the persisted capacity response as a 0% ring with `0 min / 360 min`; the desktop Day/Night renders remain two-column command-chart surfaces.
- UI type-check passes; UI suite passes: 39 files / 249 tests.
- Managed Test Genie run `20260921-213342-3b6f53f8` passes `unit` and `experience`; experience is L3/complete, unit remains L2 for the known injectable-seam/advisory coverage findings.

### Remaining / unverified

- Plan still needs a stronger mobile placement affordance when real work exists and a deeper check that the scene plate reads as a diorama rather than a narrow edge strip.
- Goals, Focus, Review, and Settings still need adversarial desktop/mobile Day/Night rendered comparison and refinement.

## 2026-09-21 — Focus mobile aperture mode

### Changed

- Added a server-session-backed mobile Focus mode that removes the shell tabs, global capture FAB, page header, catch-up ledger, and generic card surface while a session is active.
- Reframed the timer, evidence, and persisted controls inside the captured scope plate; Night now exposes procedural stars through the aperture while Day keeps the warm sky visible.
- Added a regression asserting that an active session selects the dedicated `focus-active` composition.

### Verified

- Live 390×1000 Day and Night captures show the full-screen scope composition with thumb-zone Resume/End controls and no navigation chrome.
- Full UI suite passes: 39 files / 249 tests; type-check passes.
- Managed Test Genie run `20260921-214727-fbb85fc2` passes `unit` and `experience`; experience is L3/complete, unit remains L2 for the known injectable-seam/advisory coverage findings.
- The managed scenario is healthy; repeated lifecycle restarts remain degraded only because notification-hub cannot refresh its unrelated tunnel-manager dependency.

### Remaining / unverified

- Focus still needs a running-session/overtime capture and desktop comparison against the oracle’s through-scope treatment.
- Plan placement/diorama, and adversarial Goal/Review/Settings surface refinement remain open; no final six-page visual certification exists.

## 2026-09-21 — Plan mobile placement affordance

### Changed

- Added a mobile-only, real-data-gated thumb-zone `+ Place work` action when the server returns available work.
- The action reveals the existing persisted placement form; it does not create a parallel mobile-only workflow or fabricate work.
- Added a compact overflow affordance for secondary planning views and reserved bottom safe-area space so the fixed action does not cover content.
- Added a mobile regression for the placement action and overflow affordance.

### Verified

- UI type-check passes; production build passes; UI suite passes: 39 files / 250 tests.
- Live 390×1000 Plan Day render with an empty persisted work list correctly omits the action rather than presenting a false affordance; capacity remains the real `0 min / 360 min` response.
- Managed Test Genie run `20260921-215454-869daef6` passes `unit` and `experience`; experience is L3/complete, unit remains L2 for the known injectable-seam/advisory coverage findings.
- Managed restart returned healthy; the shared notification-hub dependency refresh remains the only degraded-host condition.

### Remaining / unverified

- Need a live capture with a persisted work item to inspect the action bar’s visual treatment and scroll target.
- Plan still needs a stronger desktop/mobile diorama scene treatment; Focus still needs a running-session/overtime capture and desktop comparison; no final six-page visual certification exists.

## 2026-09-21 — keyed observatory plates and command-chart affordances

### Changed

- Converted the Plan desk, Focus scope, Settings instrument, and Plan night landscape source plates from green-screen RGB to transparent WebP assets while preserving the canonical PNG source art.
- Completed the layered compositor wiring so the Plan desk aperture can reveal the lake/procedural sky and the Focus/Settings apertures can reveal the live observatory sky and FX.
- Added the missing desktop Plan `+ Place work` command-chart action; it reveals the same server-backed placement form used by mobile.
- Tightened Day active-Focus evidence/fact contrast after inspecting the real through-scope render.
- Updated the source-art processing record with the generated asset mapping.

### Verified

- Fresh live renders show Plan desk texture without green spill, Settings instrument scenery on Night desktop/mobile, and Focus’s brass scope with live stars through the Night aperture.
- Fresh live Focus Day render shows the countdown timer, persisted session facts, and thumb-zone controls through the scope; the active session is server-backed.
- UI type-check passes; production build passes; UI suite passes: 39 files / 251 tests.
- Managed Test Genie run `20260921-221525-17aafaee` passes `unit` and `experience`; experience is L3/complete, unit remains L2 for the existing injectable-seam and advisory coverage findings.
- Managed lifecycle returned healthy after the standard shared notification-hub/tunnel-manager best-effort degradation path.

### Remaining / unverified

- Need a live persisted-work render to inspect the desktop/mobile placement actions with real backlog data; tests cover both responsive branches.
- Focus still needs a live running-session/overtime capture; Goals and Review need final adversarial scenic comparison; no comprehensive six-page visual/manual certification exists.

## 2026-09-21 — shared observatory background repair

### Changed

- Fixed the shared scene compositor’s Day/Night selectors: the renderer emits `appearance-day` and `appearance-night`, while lake, comet, balloon, and star-trail rules were targeting stale `scene-day`/`scene-night` classes.
- Added the missing Day and Night panorama layers to `ObservatoryScene`, bringing Plan, Focus, Goals, Review, and Settings onto the same layered sky contract already used by Today.
- Refined the procedural comet and balloon primitives into restrained atmospheric shapes with glow, gradient, and motion instead of a bare line and star glyph.

### Verified

- UI suite passes after adding a regression for both shared panorama layers: 39 files / 251 tests.
- Production build passes; managed restart returns the scenario healthy. The unrelated notification-hub dependency remains degraded while its tunnel-manager refresh times out.

### Remaining / unverified

- Fresh six-page Day/Night desktop/mobile visual certification is still required, including a running Focus session and overtime capture.

### Comprehensive validation receipt

- Test Genie run `20260921-223710-17d6b491` completed with 25/27 phases passed in 338 seconds. UI health, performance, unit, experience, business, and structure passed; the terminal failures were portability (`local_clean: false` with no reported blocker details) and the pre-existing docs-health debt (1 required-doc error, 553 warnings, 25 infos). No visual or application phase failed.
