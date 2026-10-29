# Usage Intelligence — target specification

Status: **APPROVED** by the operator 2026-10-29 (FEEDBACK F-1…F-4). Amendments get an ID (`AM-n`) in §9.

## 1. Destination

An operator can open Agent Manager and answer five questions in under a minute,
with numbers that agree with the provider and the transcripts on disk:

1. **Will I run out?** For each provider subscription window: percent used, reset
   time, burn rate, projected exhaustion with a confidence band, and the rate
   that would last until the reset.
2. **What is using it?** Share of the window's quota consumption by session, run,
   profile, workload, model and runner, with an explicit unattributed remainder.
3. **What should I change?** Ranked recommendations, each with an estimated weekly
   quota saving, its confidence, the evidence behind it and a concrete action.
4. **Can I trust this?** Capture coverage against the on-disk truth, with every
   uncaptured source named.
5. **Did the change work?** Quota consumed per successful unit of work, compared
   before and after a change on comparable work.

The commercial claim this enables (§6) is "do the same agent work on fewer
subscriptions". It is made only from measured evidence.

## 2. Principles

- **Provider readings first.** Forecasts use the provider's own `used_percent`
  samples. Token counts explain and attribute; they never replace a provider
  reading. Missing is unknown, never zero.
- **One derivation per fact.** One token-accounting function, one event sink with
  one hook set, one capture path. Every duplicate is deleted, not wrapped.
- **Summaries outlive events.** Usage and quota summaries are small and durable.
  Full event recall is optional and bounded by its own budget.
- **Deterministic first.** Recommendations come from inspectable rules over
  durable facts. Model inference is optional, and its output is labelled.
- **Less code.** Each workstream closes its DEBT rows. Net non-test LOC across the
  touched areas goes down (A9).

## 3. Target architecture

```text
 transcripts on disk (codex rollouts, claude projects + subagents, AM run homes,
 web-console homes)          claude statusline feed (§8 D3)
            │                              │
            ▼                              ▼
   ┌─────────────────────── Transcript Indexer ───────────────────────┐
   │ transcript_files(file_key=dev:inode, path, harness, header_id,   │
   │   parent_id, web_console_session_id, run_id?, offset, size,      │
   │   mtime, status)                                                 │
   │ incremental tail from offset with the existing codec parsers;    │
   │ AM-launched runs register their own file keys (no double count)  │
   └───────┬───────────────────────┬──────────────────────┬───────────┘
           ▼                       ▼                      ▼
   usage_counters            quota_observations      run_events + search
   (per file/turn raw        (Codex rate_limits,     (optional recall layer,
    cumulative counters,      Claude quotaLimits /    own byte budget measured
    model, ts) — durable      statusline) — durable   on its own tables; evicts
                                                       events only, no tombstones)
           │                       │
           ▼                       ▼
   Usage accounting (one function: last cumulative − baseline per session)
           │
           ▼
   Forecast · Attribution · Coverage · Recommendations  (Measures procedures)
           │
           ├── CLI `agent-manager usage …`
           ├── UI Usage page + Dashboard strip
           └── notification-hub alert when projected exhaustion precedes reset
```

Decisions this architecture makes:

- **T-1 Capture.** One per-file incremental indexer replaces snapshot import,
  tombstones, the "managed session" skip and transcript copies. Attached runs
  stay identity/authority only. They are linked to their files through
  `web_console_session_id`, and a normal process exit is `complete`, not `failed`.
- **T-2 Accounting.** Usage rows store raw cumulative counters
  (`total_token_usage` / exec `turn.completed.usage` for Codex; summed per-message
  `usage` for Claude). A session's usage is its last cumulative minus the
  cumulative before the run's first turn. The result is invariant under re-tails,
  restarts and replays. Delete `rolloutBaselineKnown`, `rolloutReceiptUsage`,
  `SetContinuationUsageBaseline`, `continuationCodexUsageBaseline`,
  `CodexUsageSnapshot`, and receipt-versus-delta authority selection for Codex.
- **T-3 Events.** Quota frames get their own event type, out of `error`. Every sink
  is built by one constructor that always applies the orchestrator's post-persist
  hooks. Projection is a durable "needs projection" mark set by every terminal
  writer, plus an idempotent sweep; it is not a fire-and-forget goroutine.
- **T-4 Forecast.** Group samples by provider/pool/window and rounded `resets_at`.
  Build a monotone step series and fit burn over trailing 1h/6h/24h. The band
  combines the spread across horizons, ±1% integer rounding and historical
  volatility. Only a new `resets_at` starts a new window; within a window the series
  is the max across sessions (a lower percent is a lagging reader, F-ORCH-3), and a
  sample whose `resets_at` passed before it was observed is dropped. Status values:
  `on_track` (exhaustion after reset), `at_risk` (band straddles reset),
  `will_exhaust` (band ends before reset), `unknown` (stale > 6h or no samples).
- **T-5 Attribution.** Split each time slice's observed percent increase across
  concurrent sessions by model-weighted cached-input share. Recalibrate the weights
  each window against observed percent. Report the remainder as
  "off-disk/unattributed". Never present exact per-session percentages.
- **T-6 Recommendations.** These are rule families over durable facts. Each produces
  `{lever, target (session/profile/workload), est_weekly_quota_pct_saved,
  confidence, evidence_refs, action}`:
  - R-model: a workload or profile runs a higher-cost model where a cheaper model's
    success rate on comparable work is acceptable. Savings come from the E-2 weights.
  - R-context: a session runs near its context window for many turns without
    compaction or a fresh session (cached re-read per turn × turns).
  - R-fanout: sub-agent bursts whose children re-ingest the parent context.
  - R-friction: recurring friction episodes (read-then-reread, stall, wait-misuse,
    retries) priced in quota.
  - R-idle: polling/sleep loops and parked sessions that keep consuming.

## 4. UX

All of this follows `experiential-ui-design`. Before the UI workstream implements,
it implements the approved mockups (decision D5):
https://claude.ai/artifact/QhYD9Hn52s2X1ku4W5wyaq

**Usage page** (replaces `/stats` as the primary analytics entry; the existing Stats
panels move under it as "Details"):

```text
┌ Usage ─────────────────────────────────────────────── coverage 97% ● ┐
│ Codex · pro · weekly           ████████░░░░░░░░░ 33%   resets Nov 4 06:57 │
│ ⚠ Will run out ~Oct 30 18:00 (band Oct 30 06:00 – Oct 31 02:00)          │
│ burn 1h 2.1%/h · 6h 3.4%/h · 24h 1.5%/h   to last: ≤0.46%/h              │
│ [sparkline of used% this window with projection cone and reset marker]    │
│ Claude · max · 5h / weekly      ███░░░ 18% / ██░░ 11%   (statusline feed)  │
├ What's using it (this window) ──────────────────────────────────────────┤
│ session 1e5904ed  research orchestrator   ≈26 pts  terra  45 rollouts →  │
│ run Vega E1                               ≈2 pts   terra               → │
│ unattributed / off-disk                   ≈1 pt                          │
│ group by: session | profile | workload | model | runner                  │
├ Recommendations ────────────────────────────────────────────────────────┤
│ 1 Start a fresh session or compact 1e5904ed (185k re-read/turn)  −8%/wk  high │
│ 2 Move git-control-tower-staged-message terra → luna             −3%/wk  med  │
│ 3 Cap sub-agent fan-out in research orchestrator                 −4%/wk  med  │
│   each row → evidence (runs, episodes, numbers) and the action to take   │
├ Coverage ───────────────────────────────────────────────────────────────┤
│ indexed 97% of transcript turns (7d) · 2 files unreadable · details →    │
└────────────────────────────────────────────────────────────────────────┘
```

- **Dashboard strip.** One line per provider window: bar, status chip and projected
  exhaustion. Clicking it opens the Usage page.
- **Run detail.** Usage panel with the run's tokens (from the one accounting function),
  estimated quota points, and recommendations that target this run.
- **Phone.** The same order; cards stack; tables become lists with the share as the
  trailing value.
- **Alert.** A notification-hub notification when status becomes `will_exhaust`
  (deduped per window, re-sent if the projection moves earlier by more than 12h),
  linking to the Usage page.
- **CLI.** `agent-manager usage forecast|burners|recommendations|coverage|compare`
  with `--json`, all reading the same Measures procedures. Agent identity can read
  them.

## 5. Acceptance (each needs evidence in WORKSTREAMS "Accepted")

| ID | Criterion | Proof |
|---|---|---|
| A1 | ≥95% of the trailing-7-day transcript turns on this host (Codex and Claude, including sub-agents and sidechains) are indexed with usage | `agent-manager usage coverage --json` agrees within 1 pt with an independent disk scan |
| A2 | Per-session token totals equal the transcript's final cumulative counters (Codex) or summed message usage (Claude) within 0.5%, including runs 2d230f9d, 1ced66dd and aee95b01 after a re-tail and an API restart | regression tests with retained fixtures plus a live comparison script output |
| A3 | Every Codex `token_count` rate-limit frame from every path becomes a deduped `quota_observations` row; quota frames are no longer `error` events; back-fill from disk covers the last 5 windows | row counts against a disk scan; migration shown |
| A4 | Forecast handles early resets. A backtest over the last 5 windows reports median absolute error of projected exhaustion at 25% and 50% of each window, and it beats a naive linear baseline | `agent-manager usage forecast --backtest --json` |
| A5 | Attribution shares plus the unattributed remainder sum to the observed increase; group-by works for all five dimensions | measure test plus live output |
| A6 | All five rule families are implemented and fire on this week's data; the top recommendation identifies the dominant consumer; each row carries an estimate, confidence, evidence and action | live output plus rule unit tests |
| A7 | One deduped will-exhaust notification reaches notification-hub | notification record |
| A8 | Usage page, dashboard strip and run usage panel work at desktop and phone widths, operated in a browser against live data, matching approved mockups | ui-health/BAS evidence plus screenshots |
| A9 | Every DEBT row is closed or explicitly kept with a reason; net non-test LOC in the touched areas is below baseline | DEBT.md before/after table |
| A10 | PRD has a P0 Usage Intelligence target; requirements 22/23 plus a new module carry real validation refs and correct status; DATA.md and effort-supervision.md match the code | `vrooli scenario test agent-manager --phases <docs/requirements phases>` |
| A11 | Value proof: after applying ≥1 recommendation, quota per successful workload unit on a comparable workload improves, with the population, window and caveats recorded. MONETIZATION/GTM claims are updated only to what this proves | `agent-manager usage compare` output plus a doc diff |

## 6. Product and docs targets

- PRD: add `OT-P0-016 Usage Intelligence` (forecast, attribution, coverage,
  recommendations, value comparison). Fold OT-P2-002/003 into it.
- Requirements: a new module `26-usage-intelligence` with EARS requirements per A1–A8;
  re-status 22/23.
- Business docs: after A11, rewrite the paid-value hypothesis as "same work, fewer
  subscriptions", with the measured case. Until then keep the current disclaimers.

## 7. Debt posture

DEBT.md is in scope on equal footing with the features. A feature that lands
without closing the DEBT rows in its area is not accepted. New shims,
compatibility layers, dual paths or dead code are findings.

## 8. Decisions needed from the operator

| ID | Question | Recommendation |
|---|---|---|
| D1 | Who orchestrates? | **Decided (F-3):** the operator's Claude Code session (this assistant) orchestrates; sub-agents run as Claude Code agents. Codex workers only on luna, and only while Codex status ≠ will_exhaust |
| D2 | Full event recall for imported/external sessions | **Decided:** summaries always; recall layer with its own 4 GiB budget, rebuilt from source on demand |
| D3 | Claude Code quota feed | **Decided (F-3), conditional:** only as a reproducible part of `resources/claude-code` — a resource-owned `statusline` subcommand plus install/reconcile through the resource's settings path (the same way hooks are reconciled), chaining any existing user statusLine and removable on uninstall. No one-off script. Replace the stale estimate-based `docs/RATE_LIMIT_TRACKING.md` / `usage_tracking.json` design |
| D4 | Delete the REST stats layer after a fleet consumer search | **Decided:** yes |
| D5 | Mockup gate | **Approved (F-4):** https://claude.ai/artifact/QhYD9Hn52s2X1ku4W5wyaq — components only; see AM-1 |
| D6 | A11 value proof timing | **Decided:** finish A1–A10; schedule the A11 comparison |
| D7 | Web-console (Aquila) surface | **Deferred (F-4)** |
| D8 | Navigation: Usage replaces Stats; Import folds into Usage › Coverage (the indexer makes manual import unnecessary) | **Approved (F-4)** with the mockups |

## 9. Amendments

- **AM-1 (F-4).** The mockups define the new and changed components only. They
  are built inside Agent Manager's existing shell, layout and tokens; the existing
  UX is improved, not replaced. The mockup's own chrome (sidebar, header) is not a
  target.
- **AM-2 (F-4).** The UX workstream also finds and fixes existing Agent Manager UX
  issues and adopts React Component Library assets where they fit. A known
  example: the phone bottom navigation carries every section. It must hold at most
  5 destinations, with the rest behind a menu (hamburger/"More"). Each fix is
  recorded in WORKSTREAMS Findings with before/after evidence.
- **AM-3 (F-6).** Usage accounting and attribution cover non-text work. This includes images viewed
  (screenshots, attachments, browser captures), images generated (for example by a runner's image
  tool or an image scenario), web search/fetch, audio, and any other provider-metered tool. For each
  runner, research what the transcripts record (counts, token or credit charges, a separate quota
  pool) (R-1). Capture it in the indexer (W2), and expose a `modality`/`tool kind`
  dimension in attribution and recommendations (W5). Where the provider meters such work outside
  the token counters, it gets its own line and is never folded silently into text tokens. Where
  it is unknown, it stays unknown.


## 10. Pass 2 (operator F-7, approved F-8, 2026-10-29)

Proposal with live captures, mockups and the code audit: https://claude.ai/artifact/TWQD4VtTu5H6RQQokE48Yw
(copy in `mockups/pass2/usage-pass-2.html`). The mockups bind like D5/AM-1: they define the changed
components; the existing shell, tokens and RCL components stay. Numbers marked as placeholders in the
mockups are layout only.

### 10.1 Decisions (all approved as recommended, F-8)

| ID | Decision |
|---|---|
| P2-D1 | Findings page is replaced by **Usage › Friction**: one list grouped by problem, in plain words, with owner, count, cost (time and points), "Report bug" (files to scenario-qa with evidence; row becomes Reported) and "Dismiss". `/findings` redirects to it. The generic-text finding generator and the unused review fields (operator decision, effectiveness) are deleted; the fingerprint is detail, not headline |
| P2-D2 | Investigations leaves the main navigation; reached from a run's Investigate action and from Health. Status, diagnosis and coverage codes are shown as words. The back end stays |
| P2-D3 | Fix the web-console test that writes fixture sessions into the real sessions folder (F-W2-1), skip test output in the indexer, and delete the imported fixture runs plus the fixture session folders under an operator deletion grant |
| P2-D4 | Sub-agent runs fold under their parent in the Runs list ("▸ N sub-agents"); an origin filter (All, Agent Manager, Operator sessions, Workflows) can show them flat |
| P2-D5 | Usage notifications: "runs out early" (on), new "back on track" (on; withdraws a stale warning), "may run out" (off), daily 8 AM digest (off). Every alert names the window, the date, and the #1 action with its saving, and opens /usage with that window selected. Agent Manager adopts app delivery (`appDelivery`, Settings › Notifications with RCL `AppNotificationSettings`, `initAppNotifications`) |
| P2-D6 | Every token and quota total over a time window reads `transcript_usage` bucketed by request time; per-run lifetime counters remain only for per-run views. Windowed readers of the lifetime counters are deleted |
| P2-D7 | Subscription work shows quota points; dollars only for per-token billed runs. "unknown 0.00 + subscription 0.00" is removed |
| P2-D8 | Details defaults to Agent Manager runs. Tokens and quota count everything; success rate, throughput and run time count only runs Agent Manager executed. Filter: Agent Manager runs, Operator sessions, Everything. Orchestrator ruling (P2-W1): "Agent Manager runs" = `codec_pipe` (headless) runs; interactive sessions, including ones Agent Manager launched in a web console, are human-paced and count as operator sessions |
| P2-D9 | No enforcement in this pass; design budgets/auto-compaction/quota-aware scheduling after A11 |
| P2-D10 | Details keeps the summary tiles and the model and profile tables; the panels that repeat "Where it went" (runner, tool, capability, cost distribution, run classes) are deleted; fallback insights become a Health item; the time window lives in the URL |
| P2-D11 | REST/RPC twins are consolidated in this pass after a fleet-wide consumer search: AM's CLI and UI move to RPC, every REST twin without an outside caller is deleted; twins with outside callers are listed and kept |
| P2-D12 | Naming: subscriptions by plan ("Claude · weekly", "Codex · 5-hour"), runners by tool ("Claude Code", "Codex", "OpenCode"); quota always "pts" (1 pt = 1% of a window); "requests" (not turns) everywhere, including the proto field `UsageBurner.turns` → `requests` |
| P2-D13 | Runner health checks (probe) live on the Health page only; the dashboard shows a one-line health summary linking there |

### 10.2 Pass-2 behaviour targets

- **Forecast states.** A window with no reading newer than 6 h and no running work that could spend it
  is **idle** ("idle, lasts at this pace") — never "unknown". "Unknown" stays for windows where
  unmetered use cannot be ruled out. A reading whose reset time is already in the past at observation
  time is rejected at ingest.
- **Grouped recommendations.** Recommendations for one target are returned as one group with its
  levers; the group's saving is computed by the server without double counting overlapping levers
  (each lever's saving is applied to what remains after the larger ones). Each group carries used-so-far
  points, the projected points this week and the share of the window's burn. Recommendations with
  near-zero saving are folded.
- **Focus.** `/usage?window=<provider>/<pool>/<window>` selects that window. A will-exhaust window shows
  the focus band: used, burn now, affordable burn, cut needed, the saving of the shown actions and the
  honest consequence (new projected date; whether that reaches the reset).
- **Friction rows** carry a one-line plain description derived from the fingerprint's evidence (command,
  exit/fallback pattern, typical duration) and an owner.
- **"Where it went"** folds rows under 0.1 pts and never shows a raw UUID as the label.
- **Details** shows units (B/M/K), median and p90 run time with a correct sample size, and section-level
  error/retry states with deadlines.

### 10.3 Pass-2 acceptance

| ID | Criterion | Proof |
|---|---|---|
| P2-A1 | Details tokens for 7d equal `SUM(transcript_usage)` for the same window within 1%; run-time sample size is non-zero when runs finished; throughput, success and run time count only AM-executed runs; no windowed measure reads lifetime counters | measures tests + live CLI/Measures output vs a read-only SQL sum |
| P2-A2 | 0 fixture runs remain; the web-console test writes only to a temp dir; the indexer skips test output | SQL count, test, grant id |
| P2-A3 | Alerts: four usage categories declared; will-exhaust body names the #1 action; back-on-track sent once when a notified window turns on_track/idle/reset; app delivery on with a Notifications settings tab; stale-reset readings rejected; idle state live | unit tests + live notification record and receipts + browser capture; on-phone pairing is the operator's check |
| P2-A4 | Grouped recommendations with non-double-counted savings, used-so-far vs projected, folded small items; friction descriptions and owners | unit tests + live `usage recommendations` |
| P2-A5 | Friction tab replaces Findings; Report bug files to scenario-qa; finding generator and review fields deleted | tests + live capture + bug id |
| P2-A6 | Pages match the mockups: Usage Overview focus, Friction, Details, Dashboard, Runs (filters, folding), Investigations/Health placement, run detail split, naming rule P2-D12 | headless Chrome 1440×900 and 390×844 captures; no horizontal overflow; phone targets ≥ 44 px |
| P2-A7 | DEBT pass-2 rows closed or kept with reason; net non-test LOC falls in API, CLI and UI vs the pass-2 baseline; 0 eslint errors in `ui/src`; type-check and touched Go/CLI/UI tests pass | DEBT.md table + command output |
| P2-A8 | REST/RPC consolidation recorded with the consumer search and the kept list | search output + endpoints.json diff |
| P2-A9 | Scoped Test Genie phases pass for changed docs/business/unit areas, or failures are shown pre-existing | `vrooli scenario test agent-manager --phases …` |
| P2-A10 | Two consecutive fresh hostile review passes over the pass-2 changes find nothing material | review outputs |
