# Baseline evidence — 2026-10-29

Measured read-only on this host against
`scenarios/agent-manager/data/agent-manager.db` (opened `?mode=ro`) and the runner
transcripts on disk. The one-off `baseline/` scripts named below were deleted (DEBT D-14);
`agent-manager usage coverage|burners|forecast --backtest` and the live tests
(`USAGE_COVERAGE_LIVE`, `USAGE_ATTRIBUTION_LIVE`, `AM_QUOTA_LIVE_BACKTEST`) reproduce these measures. Labels: **fact** = measured or
read in code; **hypothesis** = supported but not proven.

## E-1 Provider quota signal exists and is ignored

- Fact. Every Codex rollout `event_msg`/`token_count` frame carries
  `rate_limits.primary {used_percent, window_minutes: 10080, resets_at}`, plus
  `plan_type: pro`, `limit_id: codex`, `credits.balance` (62,500, unchanged).
  No secondary window appears since the move to weekly windows.
- Fact. The current window opened 2026-10-28 06:57Z and resets 2026-11-04 06:57Z.
  It was at 33% at 2026-10-29 04:30Z, which is 1.5–1.6 %/h on average and
  ~3.4 %/h from 19:57Z to 02:57Z. Projected exhaustion is 10-30 ~00:30Z at the recent
  rate and 10-31 ~00:30Z at the average rate, 4–5 days before the reset. Lasting
  to the reset needs ≤0.46 %/h. (`baseline/burn.py`)
- Fact. There have been five early window resets in six days: new 7-day windows began around 10-22 17:09,
  10-23 18:02, 10-25 06:10, 10-26 06:08 and 10-28 06:57Z. The previous windows ended
  at 15/8/13/27%. A forecaster must treat a drop in `used_percent` or a new
  `resets_at` as a reset at any time. (`resets_at` jitters 1–5 s per sample, so
  group windows by rounded value.)
- Fact. The Codex codec parses these frames (`api/internal/adapters/runner/codecs/codex.go:643-683,801,843-880`)
  and ~40k reached `run_events`, all from interactive runs, stored as
  `event_type=error` with `payloadKind=quota` (`api/internal/domain/run_configuration.go:432-450`).
- Fact. The `quota_observations` table has 0 rows. Only `runEventSink`
  (`api/internal/orchestration/run_operations.go:604-611`) attaches
  `observeQuotaEvent`/`observeTokenCapUsage`/`nudgeWorkflowUsage`. The fresh
  headless executor (`run_executor.go:717`), interactive (`run_execution.go:456`),
  recovery (`recovery.go:400`) and import (`import_transcript.go:1042`) sinks
  attach none. Token caps therefore do not run on fresh headless or interactive
  runs. Tests mask this because they call `runEventSink` directly
  (`token_cap_test.go:91`, `quota_observation_test.go:67`).
- Claude Code. Transcripts carry per-message `usage` but no utilization percentage. Only 429
  lines carry `quotaLimits{status, rateLimitType: five_hour, resetsAt, overageStatus}`,
  and the codec matches error text only (`claude_events.go:638-650`). The status line
  JSON receives `rate_limits.five_hour|seven_day.used_percentage` and `resets_at`
  for Pro/Max (official docs). `~/.claude/settings.json` has no `statusLine`, so
  nothing captures it.
- OpenCode and Grok have no quota fields locally; the codecs parse rate-limit error text only.

## E-2 Quota cost depends on the model

- Fact. In the current window, 1% of quota ≈ 1.95M non-cached input, 173M cached
  input and 0.47M output. The token categories are too collinear to separate (r
  0.65–0.97). Cached input alone fits best (R² ≈ 0.65).
- Hypothesis. Pooled per-model fit (84 hourly bins, 3 windows, R² 0.83), in cached
  tokens per 1%: sol ≈ 50M (44–57M), terra ≈ 100M (95–118M), luna ≈ 520M
  (270–900M). Per unit of work, terra costs ~5× luna and sol ~10× luna.
  (`baseline/cur2.py`, `baseline/reg.py`)

## E-3 What consumes the quota

- Fact. 84% of the current window's non-cached Codex tokens came from one
  web-console session, `1e5904ed-8378-458a-86d7-8dbb039cff19`. It has been open since
  10-21, has 45 rollouts (orchestrator plus sub-agents, an 8-agent burst at
  10-28 04:55Z), and a 141 MB main rollout. Its turns re-read ~185k context
  tokens each (context window 258k). The remaining share came from Vega E1/E2
  interactive runs (6.7%/4.5%) and the Vega heartbeat (3.3%). (`baseline/bysess.py`)

## E-4 Capture coverage is ~30%

- Fact. Over the last 7 days (files >10 KB), web-console rollouts were 64 files /
  3.19 GB, of which 15 matched a run and **≈5% of bytes** were covered. AM-launched
  rollouts were 65 files / 1.12 GB at 100%. The total is **≈30%**. (`baseline/wc_detail.txt`)
- Fact. Runs by mode: attached 14,065, imported 4,909, codec_pipe 1,183,
  interactive 403.
- Fact. Attached runs come from `packages/cli-core/cliutil/agentlauncher.go:370` →
  `POST /api/v1/runs/attach`. They are identity and authority only by design
  (`attached_run.go:24-28`) and have ~2 events each. The reconciler marks every normal
  exit `failed` (`reconciler.go:683-699`, `reconciler_recovery.go:28-58`): 13,799
  failed versus 149 complete.
- Fact. A live attached run makes the sweep skip the whole web-console session
  (`transcript_scheduler.go:460`). Nineteen are live, on processes up to 12 days old,
  covering 10 sessions including `1e5904ed`.
- Fact. Import is snapshot-once, keyed by header session id
  (`import_transcript.go:416,722-750`, `transcript_scheduler.go:302`), and never
  refreshed. Example: session `01a1640a` was imported at 6.4 MB and is now 192 MB. Codex
  fork files (`<parent>_<child>`) and Claude `subagents/*.jsonl` sidechains collide
  on the parent id (71 Claude sidechain files lost in 7 days). Each import also
  writes a full duplicate copy into `agent-manager/runs` (5.6 GB).
- Fact. The eviction budget (`ImportedCorpusMaxBytes` 8 GiB, `levers.go:716`) is
  compared with the whole DB's live bytes (10.5 GB), and only imports can be evicted
  (`reconciler_storage.go:55-99`). Every import is therefore evicted at once: all 4,909 imported
  runs have 0 events. Tombstone rows block re-import. Their read-model token totals
  survive but describe the stale snapshot.

## E-5 Token totals are wrong

- Fact. A Codex parser started mid-transcript emits the session's cumulative total as
  one delta. `rolloutBaselineKnown: true` with no prior total
  (`codex.go:271,492`, `rolloutUsageDelta` `:926-944`) also stamps
  `reconciliationComplete=true`. Interactive `Tail()` creates a fresh parser per turn
  (`interactive/coordinator.go:635-660,737-750`, `interactive/tail.go:169,186`), and
  so does recovery `drainTranscript` (`recovery.go:274-287`). The guard
  `continuationCodexUsageBaseline` (`run_continuation.go:39-90`) needs a
  `CodexUsageSnapshot`, and 0 events carry one.
- Fact. Run `2d230f9d` (Vega E2): the read model reports 6.29B. The true cumulative at the last
  captured event was ~372M, a **~17× overcount**. Capture also stopped 54 min before the run
  ended, losing ~125M. Codec_pipe run `aee95b01`: 568M versus 284M on disk (2×), from
  one recovery replay right after an API restart. (`baseline/sel.py` reproduces
  `SelectCanonicalUsage` exactly.)
- Fact. Read-model projection is a fire-and-forget goroutine
  (`invocation_read_model.go:543-552`). It is skipped when the status append fails
  (`SQLITE_BUSY`) or when the Reconciler writes the terminal state directly
  (`recovery.go:205,228,255`). Of 97 runs that ended after the last restart, 92 have no row.
- Fact. A run's total tokens are computed 8 different ways (DEBT D-5).

## E-6 Surfaces and authority

- Fact. Every `measures` call returns 403 for agent identity. Measures are Connect
  POSTs that are missing from `agentReadPOSTRoute`
  (`handlers/authorization_routes.go:137,354-414`). `run stats` works only because
  it is the REST `GET /api/v1/stats/summary`, which the UI no longer uses.
- Fact. The UI has a `/stats` page (`ui/src/features/stats/`, ~2.6k tsx lines)
  including TokenAttributionBreakdown. No page shows quota, forecast, coverage or
  recommendations. The dashboard shows no cost or friction.
- Fact. PRD: cost tracking is P2 "planned". `requirements/22-cost-tracking` and
  `23-durable-run-analytics` are "planned" despite being largely implemented.
  `docs/business/MONETIZATION.md` and `GO-TO-MARKET.md` disclaim savings and call a
  limits dashboard future discovery. `docs/concepts/DATA.md:119-123` and
  `docs/reference/effort-supervision.md:454-468` contradict the code.
- Fact. The API is 151k non-test / 133k test Go lines. The analytics core is ~17.6k
  non-test lines, excluding investigation, findings and effort.

## W8 acceptance evidence

Captured 2026-10-29 ~11:35-11:50Z by W8b against the live UI (http://localhost:21238) and API after restart 4. Tooling: headless Google Chrome via the repo's Playwright 1.62.1 (script `/tmp/claude-1000/-home-matthalloran8-Vrooli/76d970de-82d0-4c52-b559-ceadb6acbf61/scratchpad/w8b/cap.mjs`, `/tmp/claude-1000/-home-matthalloran8-Vrooli/76d970de-82d0-4c52-b559-ceadb6acbf61/scratchpad/w8b/tall3.mjs`); ui-health/BAS were not used (no capture surface for a dark-only scenario UI). The app is **dark-only** (index.html pins `class="dark"`), so no light-mode captures exist. Run used: `1a2ae9a2-177a-4721-8c98-03ffdff49db5` (imported Claude Code run).

### A8 browser evidence (1440x900 and 390x844; full-height screenshots)

Screenshots, all under `/tmp/claude-1000/-home-matthalloran8-Vrooli/76d970de-82d0-4c52-b559-ceadb6acbf61/scratchpad/w8b/tall3/`: `{desktop,phone}-usage.png` (Overview), `-usage_tab_coverage.png`, `-usage_tab_details.png`, `-dashboard.png`, `-runs_RUN_tab_usage.png`, `-runs_RUN_tab_report.png`, `desktop-findings.png` (phone findings did not finish in the final run; first-pass viewport shots exist at `/tmp/claude-1000/-home-matthalloran8-Vrooli/76d970de-82d0-4c52-b559-ceadb6acbf61/scratchpad/w8b/phone-findings.png`). Settings > Model pricing: `/tmp/claude-1000/-home-matthalloran8-Vrooli/76d970de-82d0-4c52-b559-ceadb6acbf61/scratchpad/w8b/desktop-settings_pricing.png`, `/tmp/claude-1000/-home-matthalloran8-Vrooli/76d970de-82d0-4c52-b559-ceadb6acbf61/scratchpad/w8b/phone-settings_pricing.png` (viewport only; not inspected for scroll behaviour). Crops: `/tmp/claude-1000/-home-matthalloran8-Vrooli/76d970de-82d0-4c52-b559-ceadb6acbf61/scratchpad/w8b/crop-recs.png`, `/tmp/claude-1000/-home-matthalloran8-Vrooli/76d970de-82d0-4c52-b559-ceadb6acbf61/scratchpad/w8b/crop-bottom.png`, `/tmp/claude-1000/-home-matthalloran8-Vrooli/76d970de-82d0-4c52-b559-ceadb6acbf61/scratchpad/w8b/c-phone-attr.png`.

Observed working: five window cards with status chips, projected run-out text and band, chart with projection cone and reset marker, 1h/6h/24h/to-last burn; Coverage chip "100% - 7 days"; "What's using it" with 8 group-by tabs (Session, Run, Profile, Workload, Model, Runner, Modality, Tool kind), unattributed row, metered-outside-counters lines and a "How this is estimated" disclosure; 20 ranked recommendations with confidence chips and WHY/EVIDENCE panel; coverage grid; dashboard usage strip; desktop dashboard click-through to `/usage` (final URL `http://localhost:21238/usage`); run Usage tab shows quota points, cached/new/output tokens per window. No horizontal overflow at either width (scrollWidth == clientWidth on all pages).

Console/network: every page logs one `503 GET /api/v1/health` (API reports `degraded`: "in_flight_runs ... context deadline exceeded; workflow_accounting: context deadline exceeded", start-up compaction still running); run pages add `400 GET /runs/<id>/diff` ("imported run has no sandbox", console error "Failed to load diff"). No other failed requests. Raw logs: `/tmp/claude-1000/-home-matthalloran8-Vrooli/76d970de-82d0-4c52-b559-ceadb6acbf61/scratchpad/w8b/result.json`.

Not verified: phone burners group-by select (not operated), "did not fire" lines (none rendered in the captured data; not operated), light theme (none exists), Settings pricing scroll behaviour, phone /findings full page, Alert in-app surface (not built).

UI defects (do not fix in W8b):
1. Phone / dashboard strip: the progress bar is cut off at the right edge for every window row, and on "Openai primary 300m" the "0%" label overlaps "resets Sep 9, 4:28 PM". Expected per mockups/Dashboard.dc.html: one line per window, bar and chip fitting the 390 px width.
2. Dashboard and Usage list two Openai windows whose last reading is 49 days old (reset Sep 9 / Sep 16) as cards with "Forecast unknown". Expected: hide or collapse windows with no reading in the current period.
3. Phone dashboard shows "Health: Agent Manager is unreachable" while the API answers; the health probe returns 503 (degraded during start-up compaction) and the UI treats degraded as unreachable. Header chip "Degraded - Live" is red on every page.
4. Recommendation evidence links are nine identical "sub-agent" links with no identifying text (crop-recs.png); expected the run tag or id per link.
5. Recommendation 2 reads "142k re-read per turn over 24436 turns" for session 1e5904ed, which is implausible for one session; verify the turn count (W5 data, not UI).
6. Usage Details tab: header cells (Runner, Tokens, Runs, Success, Cost, Avg Time / Profile) measure under 32 px high (small hit targets); run pages' run-id link "1a2ae9a2" is under 32 px.
7. Run detail pages show "Loading run usage..." and "No cost events recorded for this run" together before data arrives; the latter persists next to a populated Usage panel (desktop-runs_RUN_tab_usage.png).
8. Overview takes about 10 s to populate attribution and recommendations while the API is compacting (`agent-manager usage burners --json` 9.5 s); the page shows spinners only.

### A7 notification evidence

`agent-manager usage forecast` at 11:33Z: Claude seven_day "WILL RUN OUT ~Sat Oct 31 18:57 UTC". `usage_forecast_alerts` (read-only) holds one row per window: `anthropic|seven_day|10080m|2026-11-04T15:00Z` projected 2026-10-31T09:19:32Z notified 2026-10-29T07:03:35Z; `anthropic|five_hour|300m|2026-10-29T10:10Z` projected 07:42 notified 07:03:35Z; `openai|primary|10080m|2026-11-04T06:57Z` projected 2026-10-31T04:51Z notified 06:38:55Z. notification-hub (`notification-hub notifications list --limit 500 --json`) holds exactly one notification per row, all `NOTIFICATION_STATE_DELIVERED` with a web_push receipt, `context_url` https://agent-manager.itsagitime.com/usage:
- Claude weekly: `26563afb-8b0e-4530-bf4d-03ec59cfa2ed`, dedupe_key `anthropic|seven_day|10080m|2026-11-04T15:00Z|2026-10-31T09Z`, receipt `5b347d73-593a-433e-b7ff-f7c200253de1`, urgency high.
- Claude 5h: `72630466-5d69-4d53-884e-649700e62678` (07:03:36Z).
- Codex weekly: `748f1fb8-6424-41a2-92c8-8723dc6f83fb` (06:38:56Z).
Dedupe: the later projection (Oct 31 18:57 versus the notified Oct 31 09:19) did not move earlier, so no re-send; `count(*) where dedupe_key like 'anthropic|seven_day%'` = 1. The event type constant is `agent_manager.usage.notification.v1` (api/internal/usageforecast/alert.go:21); the notification-hub CLI does not expose the topic field, so topic equality was not read directly. Gap: the Claude 5h notice fired at 69% used with a projection of 07:42 inside the same window, which is a past-dated projection by now.

### Bugs filed to scenario-qa (report-bug)
- F-ORCH-1 cli-sync treats existing binary as fresh: knw-1793273768154257132
- F-ORCH-2 claude-code resource.json pins 2.1.185 vs host 2.1.312: knw-1793273769424294640
- F-W3-1 resource uninstall never runs the resource uninstall step: knw-1793273773534897816
- F-W2-1 web-console Go tests write rollouts to the real state dir: knw-1793273775988019361. Cause found: `newCodexTailerTestServer` (scenarios/web-console/api/codex_tailer_test.go:57) never sets a temp state root, so `sessionCodexSessionsDir` (pty_local_unix.go:242) resolves the real one; tests at codex_tailer_test.go:102,167,212,246 and codex_tailer_stale_test.go:25 create the rollout files (about 900 each).
- F-W0-2 docs phase mis-resolves repo-root ref at DATA.md:4: knw-1793273777389871398
- F-W6c-3 JS dep removal verb, RCL Badge/Card/AppShell gaps (filed as one `unknown` entry): knw-1793273778790829165

### Docs manifest hygiene
`docs/manifest.json`: added `reference/effort-supervision.md`, `reference/token-attribution.md`, `reference/component-library-gaps.md` to the reference section, and a new `efforts` section registering the nine effort documents (README, SPEC, GOAL, WORKSTREAMS, WORKER, EVIDENCE, DEBT, FEEDBACK, R1-non-text-usage), all `completion: optional`. All 66 manifest paths exist on disk. The docs phase itself was not re-run (no scoped phase assigned), so the misplaced_doc finding is unconfirmed cleared. Backup of the prior file: `/tmp/claude-1000/-home-matthalloran8-Vrooli/76d970de-82d0-4c52-b559-ceadb6acbf61/scratchpad/w8b/manifest.before.json`.
