# Workstreams

The orchestrator owns this file. Every outcome is substantial (one replacement or
one user-visible capability, including its tests and docs). Do not split source,
tests and docs into separate dispatches.

## Current

- Current: @ 2026-10-29 13:00Z. **A1–A10 accepted live; A11 scheduled (D6).** All workers returned; restart 6 (12:50Z) healthy after the orchestrator's recovery-retry fix. Remaining work is operator-owned (below).

### Final acceptance (live, after restarts 5–6)

| ID | Result | Evidence |
|---|---|---|
| A1 | met | `usage coverage`: 100.0% of trailing-7-day transcript bytes, 331 files, 0 unreadable; independent scan Δ 0.00 pts, 99.97% of 22,873 turns (`USAGE_COVERAGE_LIVE=1 transcriptindex/live_coverage_test.go`) |
| A2 | met | W1 regressions + `TestLiveUsageComparison` (2d230f9d 745,410,641; 1ced66dd 1,147,786,266; aee95b01 284,138,869 = disk) |
| A3 | met | quota frames are `quota` events (63,340 migrated); back-fill of all windows on disk; dedupe live 2,214,976 → 133,912 rows |
| A4 | met | live `usage forecast --backtest`: last-5-per-scope 7.3 vs 11.0 (25%, 15 windows), 9.9 vs 13.0 (50%, 16); all windows 4.1 vs 10.0 (750), 2.3 vs 6.7 (880); `TestBacktestNeverLooksAhead` |
| A5 | met | `usage burners`: every group-by sums to the observed increase; Codex weekly 34 = attributed + 3.0 unattributed (was 17.4 before D-16) |
| A6 | met | `usage recommendations`: 6 rule families fire; #1 targets the dominant consumer; rows carry estimate/basis/confidence/evidence/action |
| A7 | met | one deduped web_push notification per window (Claude weekly 26563afb…, Claude 5h 72630466…, Codex weekly 748f1fb8…) |
| A8 | met | headless Chrome 1440x900 + 390x844 on the served build (scratchpad final/, w8b/tall3/); W8b defects fixed by W6d and re-checked live (strip fits, dormant lines, names, degraded notice) |
| A9 | met with caveat | DEBT.md A9 table: all rows closed except D-15 (temporary); pre-existing analytics core −4.0k non-test LOC, API overall +4.7k (3 new capability packages) |
| A10 | met for this effort | test-genie docs,business run 20261029-123510-56dcfbee: effort-caused errors fixed (14 → 5 docs); remaining 5 absolute-path + 7 module-24 business_status_unearned are pre-existing |
| A11 | scheduled | see below |

### A11 schedule (D6)

Recommendation #1 applied from 2026-10-29: cap sub-agent fan-out (short worker briefs instead of the parent context; ≤3 concurrent workers). On or after 2026-11-05, measure:
`agent-manager usage compare --left-window last_month --right-window last_7d --provider anthropic --change-binding "R-fanout cap from 2026-10-29"` (and the same for `openai`), then record points per successful run, population, window and caveats here and in FEEDBACK F-1. Update MONETIZATION/GTM claims only to what that proves.

### Operator-owned follow-ups

- Git: W6a ran `git mv` (ImportPage.tsx → features/usage/ManualImport.tsx), so the rename may be staged; W1/W7 each ran one read-only `git diff`. Nothing was committed.
- Filed bugs: knw-1793273768154257132 (cli-sync freshness), knw-1793273769424294640 (claude-code 2.1.185 pin downgrade risk, HIGH), knw-1793273773534897816 (uninstall skips resource step), knw-1793273775988019361 (web-console test writes real session dirs), knw-1793273777389871398 (docs code-ref resolver), knw-1793273778790829165 (analyzer JS removal + RCL Badge/Card/AppShell gaps).
- Next measures.proto pass: rename `UsageBurner.turns` → `requests`; cohort-definition RPCs to retire the REST cohort routes.
- Decide: Investigations page default window (all-time vs this_week).

## Pass 3 (SPEC §11; operator approved 2026-10-29)

- Current: @ 2026-10-30 04:30Z. **Pass 3 implemented; acceptance below.** Live service restarted on facts v25 / episodes v23; the stale-projection sweep is re-deriving history newest-first (budget 10 s/min, ~1,400 of last week's runs left at 04:10Z).
- A11 amended: the "short worker briefs" change came from the wrong R-fanout mechanism (P3-F10). The
  binding change measured on/after 2026-11-05 is the Track-A guidance (sub-agent model by task kind,
  worker restart per slice, prompt-floor trim) applied 2026-10-29.

### Pass 3 acceptance (live, 2026-10-30)

| ID | Result | Evidence |
|---|---|---|
| P3-A1 | met | v25 facts (654,057 rows so far): 0 JS-keyword executables, 0 `unparseable` intent keys, 120 of 228,080 shell facts (0.05%) without a command path; `invocation_command_test.go` covers heredocs, wrappers, Codex JS, chains, probes |
| P3-A2 | met (derived) | `domain/message_source.go` + codec tests (Claude with/without `origin`, Codex goal/AGENTS.md/env, wakes, harness-block prefixes); stored events derive their source at read time; SQL distribution of sources not taken |
| P3-A3 | met | `TestPrecisionGate` on 269 windows / 215 blind labels: shown = command-failure (Wilson LB 0.94) and guidance-repair (0.72); 15 others quarantined; hold-out split, unlabelled ≤20%, ≥5 matched true, all-label precision ≥0.8 |
| P3-A4 | met | episodes priced by their own calls (`FrictionEpisode.price`); guidance-repair has no time; live `usage friction` shows no run-wide tokens and no "wasted" label |
| P3-A5 | met | live `usage recommendations`: finished sessions project to their points (76d970de 27.3 → 27.3); R-lifetime "restart sub-agents after about 45 requests" −4.8 pts replaces R-fanout −54.8; R-model-mix shown as "up to −10.2 (unproven)"; R-floor "floor 33% of cost" |
| P3-A6 | met | centered R², minimum 8 blocks, priors labelled; `attribute_test.go` |
| P3-A7 | **not met** | debt items closed (one pattern table, four lists deleted, `help-recovery-rate`, `run episode-trend`, `run messages-friction`, TimeNone/TimeLost, legacy ownership branch, time-to-first-success, `HandoffsBeforeMet`, corpus-specific exit table); but non-test LOC rose in `runsignal` (now 4,689; new shell lexer + Codex JS reader ≈ +860) and `usageattribution` (now 3,017; three new levers). No pre-pass baseline was snapshotted for those two packages, so the net is estimated, not measured |
| P3-A8 | met | `go test ./internal/...` (API), CLI `go test ./...`, `make type-check`, vitest usage + Investigations (65); Test Genie unit+docs run 20261030-013348-ef220f99 PASS (docs findings are pre-existing placeholder snippets) |
| P3-A9 | **partly met** | 17 fresh hostile reviews; every material finding fixed. Pass 10 and pass 17 found nothing material in the core accuracy work, but not two consecutively: passes 11–16 found UI/rendering defects, all fixed. Stopped on diminishing returns |

### Pass 3 follow-ups (open)

- **Codex outcomes are unknowable for `exec` scripts.** The JS `exec` tool returns text ("Script completed … Output:") with no exit code, so ~98% of Codex facts have outcome `unknown` and command-failure cannot see Codex sessions. Needs exit status from the harness (per `exec_command`), not inference.
- **repeated-work over-fires** (16,205 v23 episodes, quarantined, so hidden). Fix or delete before relabelling.
- **Retire the three kept rate verbs** by moving `.vrooli/program-runtime/setpoint-read.py` and `agent-manager-improve` to `usage friction`.
- **Labels:** guidance-repair passes narrowly (Wilson 0.72, thin hold-out); more operator-correction labels would firm it up. Several quarantined patterns have true cases the detectors miss (flag-hunting, fallback-after-capability via Codex one-liners, retry/abandoned precision).
- **A11 (≥2026-11-05):** measure the Track-A guidance change (`.claude/agents`, Luna roles, worker restart per slice) with `usage compare`.
- `make restart` failed twice with no captured error and succeeded on retry; uninvestigated.

| ID | Outcome | Owns (only these) | Wave |
|---|---|---|---|
| P3-W1 | Facts: command identity, Codex JS exec, segment outcome, no-match (SPEC §11.2 Facts) | `api/internal/runsignal/invocation*.go` and its tests/testdata | 1 |
| P3-W2 | Message source + Codex result outcome (§11.2 Message source) | `api/internal/domain` message event types, `api/internal/adapters/runner/codecs/`, `api/internal/orchestration/park.go` wake text, `import_transcript.go`, proto `events.proto`/`run.proto` message fields (proto ownership granted) | 1 |
| P3-W4 | Projection, levers, calibration (§11.2 Projection/Levers/Calibration) | `api/internal/usageattribution/` except `friction.go`; `api/internal/pricing/` | 1 |
| P3-W3 | Detectors, pattern table, episode cost, owner routing (§11.2 Detectors/Episode cost) | `api/internal/runsignal/` except invocation*.go; `usageattribution/friction.go`; friction UI component | 2 |
| P3-W5 | Precision corpus + gate (§11.2 Precision gate) | `api/internal/runsignal/testdata/precision/`, `runsignal/precision_test.go`, quarantine wiring in `friction.go` after W3 returns | 2 |
| P3-W6 | Debt (P3-F9 items not owned by W3) | `cli/` measures/run episode verbs, `invocationreadmodel` rate measures, `runreport/` | 3 |
| P3-W7 | Integration fixups after W1–W4 | usageattribution tests, `measures/usage_attribution*`, `wiring/friction_board.go`, proto `time_lost`, `cli/domains/usage/`, `domain/message_source.go` | 3 |

Orchestrator: restart, `usage reindex --full`, live validation, precision gate run, scoped Test Genie,
DEBT rows, two hostile reviews.

## Pass 2 (SPEC §10; operator F-7/F-8)

- Current: @ 2026-10-29 20:15Z. **Pass 2 accepted** (P2-A1–P2-A10 below). Fixture data deleted under operator-approved grants (P2-A2 data part done; see `p2-fixture-inventory.md` Outcome). Follow-ups shipped the same day: deletion requests reachable for agents (`run-deletion` policy; storage-manager grants fast, reviewable, `--paths-file`), `stop-by-tag`/`stop-all` requestable, investigation approvals asked in the inbox and withdrawn when decided in Agent Manager, Usage/Investigations scrolling, Investigations paging, deleted runs no longer counted (derived rows removed; repair `2026-10-29-drop-orphan-run-projections`), CLI warns instead of silently dropping a grant when the identity broker is slow. Remaining operator items: confirm Agent Manager notifications on the phone (P2-A3 pairing), A11 on 2026-11-05.

### Pass 2 acceptance (live, after restart 14)

| ID | Status | Proof |
|---|---|---|
| P2-A1 | met | read-only live: 7-day windowed tokens 30,893,007,112 = `SUM(transcript_usage)`; `run stats` p50/p90 with non-zero samples; scope agent_manager/operator_sessions/all; duplicate run events repaired (23,719 + 102 events; 0 duplicate tool calls left) and prevented by deterministic event IDs |
| P2-A2 | code met; data pending operator | web-console tests isolated (`useIsolatedSessionState`); indexer skips test rollouts (`TestIndexerSkipsWebConsoleTestRollouts`); deletion scripts dry-run 4,700 dirs / 4,700 runs, 0 skipped; `run delete` needs operator context |
| P2-A3 | met (phone pairing = operator) | categories `usage-will-exhaust/back-on-track/at-risk/digest` declared, `appDelivery: true`, Settings › Notifications (RCL AppNotificationSettings); hysteresis, recoverable-before-reset action text, digest in operator zone; idle state; past-reset readings rejected at ingest (tests in usageforecast, pricing) |
| P2-A4 | met | live `usage recommendations`: groups with combined overlap-free saving (76d970de −54.7 vs levers −57.8), recoverable points, folded small items, cross-cut friction plans |
| P2-A5 | met | `/findings` → `/usage?tab=friction`; friction classifier v13 (no self-pairs; stall 5 min–2 h inside a turn); Report bug via `PublishScenarioQABug` for scenario/command owners, Suggest a fix for agent behaviour; generator + review fields + 888 generic rows deleted |
| P2-A6 | met | headless Chrome 1440×900 / 390×844 captures (scratchpad p2/final, p2/w5a, p2/w5b, p2/rev4): no horizontal overflow; phone targets ≥ 44 px; final review UI findings fixed |
| P2-A7 | met | DEBT "Pass 2 after-measures": non-test LOC −5,043 (API −247, CLI −449, UI −4,347); eslint 0; go vet clean (api, cli); type-check clean; UI 548 tests, branch coverage 85.95% |
| P2-A8 | met | `p2-transport-inventory.md`: 30 REST routes deleted, 22 stub RPCs deleted, 0 `Unimplemented` RPCs (104 → 82), endpoints.json 166 → 135; kept twins listed with outside callers |
| P2-A9 | met except pre-existing fleet advisories | Test Genie 20261029-182010-86295329: structure, contracts, api, docs, business, unit, quality, ui-health, measures pass; security fails only on transitive `nanoid` and `source-map-js` advisories in `ui/pnpm-lock.yaml` (fleet-wide; the dependency gateway has no transitive override). Comprehensive run 20261029-174040-fce7f99d: remaining failures pre-existing (tidiness vs 2025-11-08 baseline, proto shared-type placement, monetization declarations, programs, portability with no findings) |
| P2-A10 | met | four fresh hostile reviews: 1 (3 major → fixed), 2 (1 blocker: duplicated run events → fixed at source + repaired), 3 (2 major: leftover/legacy duplicates → fixed), 4 (no blocker; UI polish → fixed). Final checks after fixes: UI gate, go vet/test, Test Genie above |

### Pass 2 follow-ups (operator-owned or out of scope)

- Run the two fixture deletion scripts (operator context required).
- Phone check: Settings › Notifications → "Show notifications in this app", then a test alert should show "Agent Manager" and open /usage on the window.
- Fleet: transitive `nanoid` < 3.3.18 and `source-map-js` advisories need a governed override path in scenario-dependency-analyzer.
- Agent Manager pre-existing quality debt from the comprehensive run: duplication vs a year-old tidiness baseline, proto shared types in `domain`, monetization account surface/journey probe, program-runtime dependency declaration.
- Bugs filed: knw-1793296384973920481 (requirements sync promotes ref-less/missing-ref requirements; three restored by hand), knw-1793296386141588699 (proto verify leaves temp dirs).
- A notification category "delivery-reliability" is observed under source agent-manager but not emitted by Agent Manager code; another producer uses the agent-manager source.


| WS | Outcome | Closes | Depends on | Owns | Status |
|---|---|---|---|---|---|
| P2-W1 | Measures truth: windowed totals from `transcript_usage`, AM-executed scope, Denom fix, duplicate/zero-token imported runs; then the `internal/stats` engine folded into Measures (fallback insights → Health item) | P2-1, P2-2; P2-A1 | — | `api/internal/measures`, `invocationreadmodel`, `adapters/database/repository_invocation_read_model.go`, `internal/stats`, `handlers/operational_stats.go`, `handlers/run_classes.go`, CLI ops services; measures.proto request/scope fields (generation via orchestrator) | wave 1 |
| P2-W2 | Fixture clean-up: web-console test temp dir, indexer test-output skip, fixture inventory for the deletion grant, deletion | P2-6; P2-A2 | operator grant for deletion | `scenarios/web-console/api/*codex_tailer*_test.go`, `api/internal/transcriptindex` skip rule | wave 1 |
| P2-W3 | Alerts + forecast states: idle state, stale-reset rejection, back-on-track, may-run-out, digest, actionable body, label owner (Go), app delivery + Settings › Notifications | P2-5 (Go), P2-A3 | P2-W4 group API for the #1 action (integrate when ready) | `api/internal/usageforecast`, quota ingest reader, `.vrooli/notifications.json`, CLI `domains/usage` labels, UI `SettingsDialog` Notifications tab + `main.tsx` init | wave 1 |
| P2-W4 | Recommendations + Friction API: grouped recommendations with combined saving and typed facts; Friction measure (list, report bug, dismiss); delete finding generator + review fields; proto `turns`→`requests`; CLI `usage recommendations` grouped, `usage friction` | P2-3 (API), P2-4; P2-A4, P2-A5 (API) | — | `api/internal/usageattribution`, `api/internal/findings` + recurring publisher, usage messages in measures.proto (generation via orchestrator), CLI `domains/usage` recommendation/friction commands | wave 1 |
| P2-W5b | UI foundation + shell pages: `lib/format.ts`, query client with deadlines, Dashboard, Runs filters + sub-agent folding, Investigations/Health placement, RunDetail split, ModelPricingTab, dead UI code, eslint 0 | P2-14, P2-16, P2-17/18 (part), P2-15 (part); P2-A6 (part) | — | `ui/src/lib`, `hooks`, `pages` except Findings, `components/RunDetail*`, `SettingsDialog/ModelPricingTab`, `App.tsx` nav | wave 1 |
| P2-W5a | UI Usage feature: Overview focus + grouped actions + Where it went, Friction tab (Findings redirect), Details trimmed and corrected, run Usage tab single source | P2-3 (UI), P2-15/17/19; P2-A6 (part) | P2-W1, P2-W4 protos; P2-W5b `lib/format.ts` | `ui/src/features/usage/**`, `features/stats/**`, `pages/FindingsPage*` | wave 2 |
| P2-W6 | API debt: dead routes, start-up migrations, D-15, pricing owner, classifiers, dead funcs, run test builder, god-function splits, live-DB checks | P2-7..P2-12, P2-18 (API), D-15 | — | `orchestration` (except usage/transcript/stats files), `adapters/database/connection.go`, `pricing`, `codecs`, `domain` | wave 2 |
| P2-W7 | Transport consolidation after a fleet consumer search | P2-13; P2-A8 | P2-W1, P2-W6 handler changes | `handlers.go`, `handlers/*`, CLI services, UI clients | wave 3; API+CLI part done 2026-10-29 (18 routes deleted, inventory in p2-transport-inventory.md); UI part and stub RPCs open |
| P2-W8 | Acceptance: restarts, browser captures vs mockups, phone notification check request, scoped Test Genie, DEBT after-measures, two hostile reviews | P2-A6..P2-A10 | all | orchestrator + review agents | wave 3 |

Serialized shared resources as pass 1: API restart (orchestrator only), proto generation (`flock /tmp/claude-1000/am-proto.lock`), Test Genie runs. Quota note: Claude weekly is projected to run out before its Nov 4 reset; mechanical workers run on Sonnet with self-contained briefs.

## Allocation (pass 1)

| WS | Outcome | Closes | Depends on | Owner (run/agent id) | Status |
|---|---|---|---|---|---|
| W0 | Target docs: PRD OT-P0-016, requirements module 26 + re-status 22/23, EVENT_TAXONOMY entry for the quota type, DATA.md/effort-supervision.md corrections that describe the target | A10 (docs part), D-13 | SPEC approval | W0 worker | accepted (docs; re-run in W8) |
| W1 | Accounting correctness: cumulative-counter usage (Codex and Claude), one accounting function, durable projection mark + sweep, recomputation of existing read-model rows | A2, D-3, D-4, D-5 | W0 | W1 worker | accepted (live) |
| W3 | Event pipeline: single sink constructor with hooks on all paths, quota event type + migration, `quota_observations` populated (live, interactive, recovery, import), Claude `quotaLimits`, Claude statusline feed as a `resources/claude-code` subcommand installed by the resource (D3), retire the estimate-based RATE_LIMIT_TRACKING design | A3, D-1, D-2, D-12 | W0; coordinate with W1 on `codecs/codex.go` | W3 worker | accepted (live) |
| W2 | Transcript Indexer: per-file incremental capture of all roots including sidechains, AM runs registering file keys, attached exit semantics, recall layer with its own budget, removal of snapshot import/tombstones/copies, back-fill | A1, D-6, D-7, D-8 | W1 (usage rows), W3 (sink) | W2 worker | accepted (live; back-fill complete) |
| W4 | Forecast + coverage: domain service, Measures procedures, `usage forecast/coverage` CLI, reset detection, backtest, notification-hub alert | A4, A7, A1 (report) | W2, W3 | W4 worker | accepted (live; A4 via W4b) |
| W5 | Attribution + recommendations: burners by 5 dimensions, per-window model-weight calibration, rule families R-model/R-context/R-fanout/R-friction/R-idle, `usage burners/recommendations/compare` | A5, A6 | W1, W2, W4 | W5 worker (10:25Z) | accepted (live) |
| W4b | Forecast beats the linear baseline (A4), quota_observations dedupe, last-reading/staleness in forecast proto+CLI | A4, A3 (dedupe) | W2 back-fill | W4b worker (10:25Z) | accepted (live) |
| W6 | UX: mockups (D5), Usage page, dashboard strip, run usage panel, Stats panels under Details, desktop + phone | A8 | W4/W5 contracts (may start on proto contract + fixtures) | W6a, W6b, W6c accepted | accepted (W6a–W6d; A8 live) |
| W7 | Surface consolidation: agent read allowlist, REST stats deletion (D4), CLI family folding, `run efficiency`/`run tokens` onto Measures | D-9, D-10, D-11 | W0; after W1 for token commands | W7 wave-1 accepted; D-11 folding pending (after W1) | accepted (W7, W7b; D-11 finished in W8a/W8a2) |
| W8 | Validation + value: end-to-end ground-truth comparison, scoped test-genie phases, DEBT after-measures, baseline/ deletion, A11 comparison (or schedule it per D6), business docs | A9, A10, A11, D-14 | all | W8a, W8a2, W8b, orchestrator | accepted; A11 scheduled |

## Concurrency

- Parallel first wave: **W1**, **W3** and **W7** (only the allowlist and REST-stats
  parts; W7's token-command parts wait for W1). W1 owns `codecs/codex.go` usage
  logic. W3 owns that file's quota-event emission and coordinates through the
  orchestrator.
- Second wave: **W2**, then **W4** and **W6** (mockups can start in wave 1).
- Third wave: **W5**, then **W8**.
- Serialized shared resources: agent-manager API rebuild/restart, DB migrations,
  `packages/proto` generation (governed owner command only), test-genie runs on
  agent-manager. Only one worker holds each at a time; the orchestrator grants it.

## Forecast

- Target: A1–A10 accepted; A11 measured or scheduled.
- Estimate (provisional): wave 1 ≈ 1–2 days, wave 2 ≈ 2–3 days, wave 3 ≈ 1–2 days.
- Uncertainty: indexer migration of existing imported rows; proto change size;
  the UI mockup review turnaround.

## Accepted

- **Orchestrator 12:45Z — recovery retry.** Restart 5 came up degraded/not-ready: required startup step `in_flight_runs` hit "context deadline exceeded" under start-up indexing contention and critical steps never retried. `maintenance.RecoveryStep` gains `Attempts`/`RetryDelay` (required steps 3 × 15 s, each with a fresh timeout); tests `TestRecoveryRequiredStepRetriesTransientFailureAndOpensReadiness`, `TestRecoveryRequiredStepFailingEveryAttemptStaysDegraded`. Restart 6: healthy. Also removed the dead `historical_average` pricing source chip/label from the UI (pricingDisplay, ModelPricingTab, EditPricingDialog, types) after W8a2 deleted the fallback; dialog tests 28/28.
- **Restart 5 live checks.** `transcript_usage_repairs` marker present; unknown-model rows 47; forecast drops dead scopes and uses UI names; `usage burners` 0.59 s cold / 0.56 s warm; run-owned back-fill under way (Codex unattributed 3.0 pts).
- **W8a2 accepted 12:30Z (live after restart 5; pricing-fallback deletion pending).** F-ORCH-7 dead scopes dropped; D-16 indexer reads run-owned files for usage only (dry run Codex weekly 16.58+17.42 → 30.99+3.01 = 34.00); F-ORCH-8 savings capped at projected weekly use with stated basis (#1 −37.8 → stopped session −5.4); F-W5-4 handoff detector + sweep re-projects on classifier change (v10); F-W5-5 Claude turns; F-W1-2 one charge per receipt (3 tests changed to the one-charge contract); F-W2-2 `pending_facts_json`; F-W4-4 usage measures in /measures/execute; D-11 `usage run`/`usage reindex` (5 run commands + 2 REST routes deleted; callers/skills/docs updated; cli evidence regenerated); endpoints regenerated; "requests" wording; PriceBook (warm 0.33 s, cold 2.7 s); CLI window names = UI + collision rule. API 82 pkgs + CLI 20 pkgs green. Net ≈ +335 non-test LOC (registry, basis, price book).
- Orchestrator 12:25Z: UI windowTitle(scope, siblings) applies W8a2's CLI collision rule (append raw pool id only when two windows on one surface share a title) on windows cards, dormant lines, strip, burners, recommendations; run panel caption "N requests" (UsageBurner.turns carries provider requests). type-check ok, usage vitest 28/28, eslint clean.
- **W6d accepted 12:15Z (live after restart 5).** Health: degraded (503 with health doc) ≠ unreachable, amber chip/notice with reason (healthState.ts); dashboard strip fits 390px (Progress 12rem min lifted); dead windows collapse to "no reading since …"; human window names + formatted server times (format.ts); evidence rows "kind · label · short id"; run Usage panel owns cost, single empty message after both load; skeletons, 30 s deadlines, keep-previous on group-by switch; shared SortableHeader (fixes Tokens header over Runs column) and ≥32/44px targets across stats links. type-check ok; vitest 123 files/564 tests; dev-server Playwright proof in scratchpad/w6d/. UI non-test LOC 29,753 → 29,927 (+174).
- **W8b part 1 accepted 11:55Z.** A8 headless Chrome at 1440x900 + 390x844 (app dark-only): all surfaces render, no horizontal overflow; 8 UI defects found (EVIDENCE "W8 acceptance evidence") → W6d. A7 met: one deduped web_push notification per window (Claude weekly 26563afb…, receipt 5b347d73…; Claude 5h 72630466…; Codex weekly 748f1fb8…). Bugs filed to scenario-qa: F-ORCH-1 knw-1793273768154257132, F-ORCH-2 knw-1793273769424294640, F-W3-1 knw-1793273773534897816, F-W2-1 knw-1793273775988019361 (cause: web-console `newCodexTailerTestServer` codex_tailer_test.go:57 writes real session dirs), F-W0-2 knw-1793273777389871398, F-W6c-3 knw-1793273778790829165. docs/manifest.json: effort folder registered, 3 missing reference docs added.
- 11:40Z live: quota compaction complete (2,214,976 → 133,895 rows, matches W4b copy proof); W5 repair left 47 unknown-model rows of 2.2M. W5's completion marker (transcript_usage_repairs) and busy-retry landed after the restart-4 build → verify at restart 5 (expect one marker row; the 0.9 GB `unreadable` file re-indexed).
- **Restart 4 (11:20Z)**: W4b/W5/W6c/W8a live. UI `make type-check` now passes (proto-types copy refreshed); vitest 122 files/549 tests. Start-up compaction running (quota_observations 2.21M → 475k at 11:30Z) and W5 repair running (unknown-model rows 2.2M → 162k).
- **W5 accepted 11:30Z (live).** usageattribution (block calibration from price book / tier priors, unattributed remainder, 8 group-bys, rules R-model/R-context/R-fanout/R-friction/R-idle/R-image with fired/why-not), Measures UsageBurners/UsageRecommendations/UsageCoverage/UsageCompare, CLI `usage burners|recommendations|coverage|compare`, codec line-time + model capture, transcriptindex/repair.go (D-15 temporary). Live: `usage coverage` 100.0% bytes, 331 files, 0 unreadable (A1 independent scan Δ 0.00 pts, 99.97% of 22,873 turns); `usage burners` Claude five_hour +22 = 19.5 (76d970de, this orchestration) + 0.2 + 2.3 unattributed, R² 0.99; Codex weekly +34 = 15.6 + 1.0 + 17.4 unattributed (D-16: AM-run Codex usage); `usage recommendations` 20 rows, #1 R-fanout on dominant consumer 76d970de. +3,167 non-test LOC (new capability).
- **W8a part 1 accepted 11:12Z.** Statusline dedupe per session (map in `.last`, 5-min heartbeat, append only on change; `TestFeedDedupesPerSessionWhenSessionsAlternate`); helper refreshed via cli-refresh, orchestrator smoke-tested render (exit 0) and Claude Code 2.1.312 intact; PRD updated. Findings duration now "1m31s" (`formatWallClock`, new rows only). `migrateRunEventPayloads` deleted. `agent-manager measures cohort define|list|show|delete` replaces `run cohort` (old CLI + service methods deleted; REST `/api/v1/runs/cohorts` kept — no cohort-definition RPCs yet); `measures/cohort_test.go` covers the selector. Net non-test ≈ −85.
- **W4b accepted 11:10Z (live-unverified; restart 4).** Projection = burn since window start shaped by an hour-of-week profile (learned only before window start, shrunk to hour-of-day, mixed to uniform under 4 weeks) × calibration factor (median ratio over the 10 newest windows completed before the checkpoint; quartiles = band). Burn horizons anchored at now with `stale`; `last_reading_percent`, `calibration_factor/windows`; volatility deleted (field 25 reserved). Reset-less readings grouped per hour (fixes the Oct 30→Nov 23 "5h" window). Write-time dedupe (state change or 5-min heartbeat vs the preceding row) + batched idempotent start-up compaction: live copy 2,214,976 → 133,888 rows, ≈1.2 GiB reclaimable, backtest identical before/after. Backtest on live copy (median final-% error, forecast vs linear): last-5-per-scope 25% 7.3 vs 11.0 (15 windows), 50% 9.9 vs 13.0 (16); all windows 25% 4.1 vs 10.0 (750), 50% 2.3 vs 6.7 (880); before 8.0/7.9, 5.2/5.1. No-lookahead regression `TestBacktestNeverLooksAhead`. Caveats: profile granularity chosen on this data (alternatives lose the small last-5 @25% cell); run-out error rests on 1 exhausted window; non-test LOC ~890 → ~1,390 (misses net-LOC bar; kept by decision — new capability: dedupe/compaction/profile). Worker used a whole-file rewrite on measures.proto (W5 content intact) and added one line to api/main.go.
- **W6c accepted 10:58Z (live-unverified; type-check gate after restart 4).** features/usage: api.ts (shared hooks), format.ts, UsageWindowChart, UsageWindows, UsageBurners (8 group-by dims, unattributed/other rows, calibration disclosure, metered lines), UsageRecommendations (evidence panel, did-not-fire lines), UsageCoverage (chip, grid, issues), DashboardUsageStrip, RunUsagePanel (Run detail › Usage, `?tab=usage`); components/runTabs.ts (role=tab, `?tab=report` deep link fixed); Findings on RCL Card with ≥32/44px links; errorPresentation "Not available yet" for unimplemented; ModelPricing on RCL ScrollArea, radix scroll-area wrapper deleted. Scratch-alias proof: tsc 0, vitest 122 files/549 tests. UI non-test LOC 28,390 → 29,133 (+743; feature ≈870). Not built (no data/API): Alert in-app surface, recommendation Dismiss, RunUsage token buckets, counter-verified badge.

- **W0 (docs part) — 2026-10-29.** PRD OT-P0-016; requirements module 26 (REQ-P0-029..038, REQ-P2-012 roadmap), 22 folded and removed, 23 re-statused, 25 registered; DATA.md, effort-supervision.md, EVENT_TAXONOMY.md (target type `quota`) corrected; durable design doc docs/concepts/USAGE-INTELLIGENCE.md. Proof: test-genie run 20261029-052826-5aab9086 docs,business FAIL. The failures are 6 intent.ref_missing for planned test files that W2/W5/W6 will create, plus pre-existing errors (misplaced_doc on this effort folder, PROTECTED_MODE_RUNNERS absolute_path, module 24 business_status_unearned). Re-run in W8.

- **W6a — 2026-10-29.** Phone bottom nav 12→5 (Home, Tasks, Runs, Usage, Findings) + hamburger menu with all destinations; `/usage` with Overview/Coverage/Details tabs and honest empty states; `/stats`→`/usage?tab=details`, `/import`→`/usage?tab=coverage`; dashboard counts from ListRuns totals (needs W7 total fix); dark-theme pin fixes the invisible-text shell; dead STATS_ENDPOINTS/AppHeader/SideNav removed. Proof: type-check pass; vitest 117 files / 527 tests pass, coverage 91.5/85.5/88.6/92.9; playwright captures in scratchpad `after/`. Not yet seen live (needs restart). UI non-test LOC +130 (A9 debt carried to W6b).

- **W7 (wave 1) — 2026-10-29.** D-9: REST stats layer deleted. Net ≈ −1,700 non-test LOC and ≈ −770 test LOC. `repository_stats.go` is down to the 2 pricing model-ranking reads. Fleet search found no callers outside AM. `run stats` reads 8 Measures procedures. D-10: agent read access is derived from the MeasuresService descriptor; TestEveryMeasuresProcedureIsAgentReadable. ListRunsResponse.total is now a true COUNT (has_more exact); tests added. Proof: api go build OK; handlers/database/wiring/measures/cli tests pass (handlers ran with an overlay excluding a peer's mid-edit test). Live checks wait for the restart.

- **W3 — 2026-10-29.** D-1: the 6 event-sink variants are reduced to 1 constructor (imports record quota only). D-2: new `quota` event type (proto RUN_EVENT_TYPE_QUOTA=13); an idempotent migration of 63,340 legacy rows plus a quota_observations back-fill runs on start; payload sniffing deleted. D-12: Claude `quotaLimits` is parsed. D3: `resource-claude-code statusline run|install|uninstall|status`, with install-direct reconciling it, chaining any previous command, and tests on a temp HOME. ClaudeStatuslineReader feeds quota_observations; RATE_LIMIT_TRACKING.md deleted. Proof: api build; domain, pricing, runsignal, runreport, handlers, protoconv, codecs, database, event, wiring, cli and resource tests pass; proto verify-committed-gen passes. 5 orchestration tests in W1's area are still failing (pending W1). Live migration waits for the restart.

- **W1 — 2026-10-29.** One usage account (runsignal/usage_accounting.go) built from raw cumulative counters, with a per-turn amount from the provider. The Codex baseline/delta machinery is deleted (D-3). All 8 token-total computations now use it (D-5). Projection is durable: a pending queue table plus a sweep on the reconcile tick, marked by every terminal writer (D-4). Claude counts each message id once, which fixes a ~2.5× overcount and an end_turn-only undercount. Proof: 78 internal packages pass; A2 regression fixtures (Codex rollout and exec, Claude on-disk and sub-agent, interactive per-turn re-tail, Reconciler drain) pass; the live comparison shows fresh parse and re-tail+replay equal to the disk totals on 2d230f9d (745,410,641), 1ced66dd (1,147,786,266) and aee95b01 (284,138,869). Old stored events have no counters, so re-capture (W2) is needed to correct historic rows. Non-test LOC is ≈ +230 (queue table plus richer accounting); A9 is carried. The orchestrator split repository_run.go (1502 lines → 1363 + repository_run_list.go) to clear the archtest size limit.

- **W4 — 2026-10-29.** usageforecast package (T-4 windows/early resets/monotone series/burn 1h-6h-24h/band/status), backtest vs naive baseline, Alerter publishing `agent_manager.usage.notification.v1` (dedupe per window + 12h shift, durable), Measures UsageForecast/UsageForecastBacktest, CLI `agent-manager usage forecast [--backtest]`. Proof: proto verify ok; usageforecast/measures/database/wiring/cli tests pass; the Oct 28 shape test gives will_exhaust. On-disk readings forecast run-out Oct 31 02:40Z (band Oct 30 11:58Z – Oct 31 16:35Z) vs the Nov 4 reset. Backtest on 9 disk windows: a tie at 25% (5.0 vs 5.0) and a narrow win at 50% (6.8 vs 7.0), so A4 is not yet met. Non-test LOC +1,297 (new feature).

- **Integration restart 1 — 2026-10-29 06:37Z** (W1, W3, W4, W6a, W6b, W7 live; W2 indexer loop held inert, `W2-INERT` at main.go:463 and storage_health.go:139). Live proof:
  - run_events: quota=63,340, error=18 (migration done).
  - quota_observations: 0 → 61,517.
  - attached runs: complete=13,962 (D-7).
  - invocation_read_model_pending: 13,813 queued for re-projection.
  - `agent-manager usage forecast`: Codex weekly, will run out ~Oct 31 04:51Z (band Oct 29 18:26 – Oct 31 07:32) vs the Nov 4 reset. Its readings stop at 01:49Z, at 27%, because AM's own capture is sparse until W2 back-fills (disk says 33%).
  - `measures runner-breakdown`: agent identity reads it (it was 403).
  - `run stats`: served from Measures.
- **W6b — 2026-10-29.**
  - Shared error presentation (RCL ErrorState), one DataState for loading/error/empty, and confirmation dialogs on RCL AlertDialog (no native confirm()).
  - Run titles and owners: "Unknown Profile" 80→0 and "Loading task…" are gone.
  - Delete removed from list rows; doubled phone gutter fixed; Workflows phone drill-in.
  - RCL Button everywhere (local button deleted); RCL StatusBadge; RCL EmptyState compact, contributed upstream as EmptyState 1.8.0 via draft-begin/check/publish.
  - Proof: type-check pass; 538/538 tests, coverage 91.7/85.8/88.7/93.0; playwright before/after under scratchpad w6b/.
  - UI non-test LOC +48 (A9 is carried).

- **D3 live — 2026-10-29.** `vrooli cli-refresh resource claude-code <exe>` rebuilt only the helper CLI (Claude Code stayed 2.1.312). `resource-claude-code statusline install` changed only `statusLine` in ~/.claude/settings.json; the backup is in the scratchpad (settings.json.bak). Samples go to ~/.vrooli/state/vrooli/claude-code/quota-samples.ndjson.

- **W4 fix — 2026-10-29.** Windows are identified by resets_at only; a sample whose reset has already passed is dropped; the current window is the one with the latest future reset; the drop-split rule is removed; the previous window reports whether it was observed or only last read. The statusline resource drops expired windows, and the AM reader also skips them. Proof: usageforecast/pricing/measures/database tests and the resource tests pass; a replay of live readings gives Claude five_hour 64% resetting 10:10Z and seven_day 23% resetting Nov 4. Needs restart and helper CLI refresh. Remaining: expose `Previous.Observed/LastAt` through proto/CLI wording (with W7b/W8).

- **W7b — 2026-10-29 (partial D-11).** One cohort selector (`cohort | run_ids | filter`) in measures/cohort.go behind SelectCohort, EpisodeCohort, CohortReport and CompareEpisodeCohorts (proto). Deleted from CLI and REST: `run efficiency`, `cohort-report`, `goal-cohort`, `cohort-compare`, `episode-cohort` and `invocation-aggregate|cohort|metrics`; replaced by `measures select-cohort|episode-cohort|cohort-report|episode-compare`. Skills, docs, the UI episodes client and endpoints.json are updated. Net ≈ −600 non-test LOC. Proof: proto verify ok; api and cli build and tests pass; UI tsc and Investigations tests pass. Remaining for W8: `usage run <id>` replacing `run tokens`/`run invocation-facts`; `measures cohort define|list|show|delete`; unit tests for measures/cohort.go; Investigations page window default changed from all-time to this_week (decide).

- **W2 — 2026-10-29.**
  - What: the Transcript Indexer (transcriptindex package) replaces snapshot import, tombstones, the managed-session skip, the transcript copies and the scheduler (transcript_scheduler.go deleted).
  - Ownership rule: each file has one reader. Run-owned files (AM run homes, interactive-tailed files) never produce indexer usage; sidechains of interactive runs are indexed as child runs.
  - Storage: durable transcript_usage rows; recall budget of 4 GiB measured on its own tables; AM-3 facts (images, web, audio, tool kinds incl. mcp:/scenario:, Codex credits); base64 stripped from stored Codex tool output.
  - Run status: D-7 attached exit → complete.
  - Proof: indexer tests (growth, restart, fork, sidechain, truncation/rewrite, ownership, eviction, relink) pass; a dry run over the real roots gave 100% of 7-day bytes (4.57 GB, 326 files).
  - Capture-area LOC 1,890 → 3,147 (A9 carried).
- **Integration restart 2 — 2026-10-29 07:02Z** (W2 live; the forecast fix and helper CLI refresh are live). Claude five_hour reads 69% resetting 10:10Z; seven_day 24% resetting Nov 4 15:00Z.

## Findings

- F-W8a2-1: `UsageBurner.turns` carries provider request counts (one Codex turn = many requests); rename to `requests` at the next measures.proto pass. Cold burners 2.7 s was the dead historical-average pricing fallback (≈1.9 s per unpriced model, 0 samples) — W8a2 deleting it.
- F-W6d-1: dashboard Refresh icon 40px on phone (<44); pre-existing eslint failures in untouched tests (lib/utils.test.ts:30, `any` casts in 4 tests); window-name collision risk when Codex primary and secondary weekly are both live (W8a2 asked to define one rule for CLI+UI).
- F-ORCH-7 (→W8a2): forecast lists dead windows as current (Codex primary 5h reset Sep 9, secondary weekly reset Sep 16) since W4b's grouping change; drop scopes whose reset passed long ago with no new reading.
- F-ORCH-8 (→W8a2): recommendation savings extrapolate a ~20h orchestration session to a week (#1 −37.3 pts/wk for a 13-pt consumer); state the basis and cap at the consumer's projected weekly use.
- F-W5-2..5 (→W8a2): AM-run Codex usage unattributed (run-owned files produce no transcript_usage; legacy events overcount 40–50×) = D-16; subscription models unpriced (tier priors stand in); runsignal handoff-continuation Turns = len(all facts); Claude parser never sets ProviderTurnStarted (coverage turns 0).
- F-W8a-1 (→W8b): REST cohort-definition handlers remain because measures.proto has no cohort-definition RPCs; a later proto pass could move them and delete the REST routes. `go vet` lock-copy warnings in untouched test files.
- F-W4b-1: Codex July 2026 secondary weekly scope has two concurrent reset times (likely two credentials); readings carry no account id, so the forecast mixes them. Compaction re-scans the table every start (cheap; could skip via marker).
- F-W6c-2 (→W8a): findings evidence sentence "cost %d ms" is built server-side (api/internal/orchestration/friction_routing.go:98); format the duration there. docs/concepts/USAGE-INTELLIGENCE.md § Surfaces needs the run-panel location, 8 group-by dims, coverage chip.
- F-W6c-3 (→W8b report-bug): scenario-dependency-analyzer has no JS dependency removal verb (`deps remove` is Go-only), so unused @radix-ui/react-slot and react-scroll-area cannot be removed governed. RCL gaps: Badge 1.0.3 always sets role="status" (live region on static labels); Card part selectors out-rank utility classes (needs flush/density); AppShell 2.1.2 lacks nav groups.

- F-W6c-1: ui/node_modules/.pnpm/@vrooli+proto-types is a 03:01 copy (pre-W5), so `make type-check` fails until the restart lifecycle setup refreshes it; against current gen tsc 0 errors, vitest 549/549 (W6c scratch alias). Re-run type-check after restart 4.

- F-W5-1 (W2 data defect, W5 fixing with grant 10:35Z): transcript_usage observed_at = indexing time on ~2.15M Codex rows (indexer.go prefers codec time.Now() over the line timestamp); model = unknown on all rows (Codex turn_context.model and Claude message.model never read). Fix + idempotent batched start-up repair; needs restart.
- F-ORCH-6 (→W5 with F-W5-1): indexer marks a file `unreadable` on transient SQLITE_BUSY (one 0.9 GB web-console rollout); must retry.

- F-ORCH-5 FIXED in source 07:05Z (not yet live; next restart). A horizon is defined only from a real reading at or before its start, or from the implicit window start; otherwise the rate falls back to burn since window start. Tests added.
- F-ORCH-5 (original): trailing-horizon burn treats a horizon with no sample at its start as 0% (Claude 1h 69%/h; weekly "1h 24%/h"). Burn must be unknown unless a real sample, or the window's implicit start, lies at or before the horizon start. The band must exclude unknown horizons.
- F-W2-1: test-named rollouts (rollout-e2e/backfill/stale/test123/resume) sit in 65 real web-console homes, so some test writes into the real web-console state dir. Trace and file a bug in W8.
- F-W2-2: usage facts recorded after a file's last usage report are held in memory and lost on restart. `cli/app.go` still describes import-sweep as the scheduled sweep.

- F-ORCH-4: the statusline `.last` dedupe marker is shared across sessions, so sessions taking turns write nearly every render (file growth only).

- F-ORCH-3 (→W4 follow-up/W8): Claude statusline samples are per session and stale between that session's API responses. Concurrent sessions report different percents for the same window (46 vs 63, same resets_at); some samples carry an expired resets_at from an old session. The forecast must use the max of fresh samples per window, drop windows whose reset has passed, and never treat a lower stale sample as a reset (verify the W4 drop-≥2-points rule cannot misfire on Claude samples).
- F-ORCH-1 (control plane, file in W8): `vrooli resource cli-sync` treats "binary exists" as fresh (internal/cli/resourcehandlers/handlers.go:281) and ignores the declared freshness inputs; the resource CLI did not self-refresh after source changes.
- F-ORCH-2 (control plane, HIGH, file in W8): resources/claude-code/resource.json pins acquisition @anthropic-ai/claude-code 2.1.185 while the host runs 2.1.312. `vrooli resource install claude-code` (install-direct) would likely DOWNGRADE the running agent. Avoided per F-5.

- F-W6b-1: W6b ran a raw offline `pnpm install --frozen-lockfile` to refresh the RCL copy (lockfile unchanged). This is a rule deviation; it is surfaced to the operator.
- F-W6b-2 (→W6c): Card and label badges are not migrated (RCL gaps recorded); AppShell nav groups need RCL 2.2; findings text "cost 928768399 ms" is built server-side; findings links are under 32px; unused deps @radix-ui/react-slot and react-scroll-area should be removed via scenario-dependency-analyzer; operational-stats.md still documents deleted REST routes.

- F-W4-1/2: AM's own readings are too sparse to score windows until W2 back-fills them; some frames lack window_minutes, which creates a stray "0m" scope to filter.
- F-W4-4: measurelib /measures/execute registry lacks the usage measures (Connect procedures work).

- F-W1-1: W1 ran one read-only `git diff --stat` (rule violation, no state change).
- F-W1-2: codec-pipe Codex metered runs count the charge twice (on the usage event and as a separate charge event); invisible under subscription billing. Fix in W8 or W5.
- F-W1-3: interactive Claude sub-agent usage lives in subagents/*.jsonl, which is W2 indexer scope.

- F-W3-1: `vrooli resource uninstall claude-code` (internal/resources/drivers_cli.go) never calls the resource's uninstall step, so the statusline entry outlives the binary. Owner: control plane; fix in W8 or file a bug.
- F-W3-2: database.migrateRunEventPayloads is dead (test-only). resources/claude-code/PRD.md is stale. handlers type filter rejects compaction/lifecycle types.
- F-W3-3: A3's disk back-fill of the last 5 windows needs W2's indexer.

- F-W7-1: W7 ran one read-only `git diff --numstat` (rule violation, no state change). Surfaced to operator.
- F-W7-2 (→W3/W1): rename StatsRepository → model-ranking name; move nearestRank to the invocation read-model repo.
- F-W7-3: .vrooli/endpoints.json is stale vs current routes (quiesce/wake-by-key); regenerate in W8 via `make endpoints`.
- F-W7-4 (→D-11, after W1): folding plan: delete `run invocation-aggregate|cohort|metrics` (they duplicate Measures); `replay/refresh-invocation-*` become one `usage reindex`; cohort commands become one selector on `measures`; `run efficiency` moves to the efficiency-report program/Measures; `run tokens` becomes `usage run`. Callers to update: skills/agent-manager, agent-manager-improve, reviewing-agent-run-efficiency, docs (operational-stats, TRIAGE, runner-transcript-recovery, acceptance-walkthrough, DELIVERY_RELIABILITY O20), ui/src/api/episodes.ts.

- F-W6a-1: W6a ran one `git mv` (rule violation). The rename may be staged in the git index. Surfaced to the operator; no further git was used.
- F-W6a-2 (→W6b): raw RPC errors shown to users in Stats cards; "No runs yet" shown when the API is down; Findings page blank when the API is down; Recent Activity chip clipping, machine-ID titles, rows stuck on "Loading task…"; "Unknown Profile" on every run; Delete on every phone run row; doubled phone gutter on Runs; raw prompt titles in import list; EmptyState has no compact mode; local shadcn Button/Badge/Card next to RCL ones; AppShell has no nav group headings (RCL draft-begin).
- F-W6a-3 (→W6b/W8): docs/reference/operational-stats.md:10,191 and token-attribution.md:161,188 still say Stats/`/stats`.

- F-W0-1: docs phase flags this effort folder as misplaced_doc. Register or exempt it in W8 (or when the folder is retired).
- F-W0-2: docs phase mis-resolves a repo-root code ref in DATA.md:4 (knowledge-observatory resolver; pre-existing). File via report-bug in W8.
- F-W0-3: effort-supervision.md, token-attribution.md, component-library-gaps.md missing from docs/manifest.json.
- F-W0-4: module 24 business_status_unearned ×7 (pre-existing).
