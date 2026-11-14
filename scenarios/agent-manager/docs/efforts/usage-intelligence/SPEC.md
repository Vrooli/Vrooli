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

## 11. Pass 3 — accuracy (operator approved 2026-10-29)

Audit of 2026-10-29 (orchestrator + four read-only audits; memory
`agent-manager-friction-measures-are-currently-untrustworthy`): forecast, burners and coverage are
trustworthy; friction and recommendations are not. Most friction episodes are false positives and the
savings arithmetic overstates. Pass 3 makes every shown finding true before any UX redesign.

### 11.1 Findings this pass must remove

| ID | Finding (evidence) |
|---|---|
| P3-F1 | `intent_class` is argument shape, not intent: 93.5% of facts contain `shell:unparseable`; Codex `exec` input (JavaScript) is parsed as shell, giving executables `await`/`const`/`let` (2,119 facts) — `runsignal/invocation.go:312-358` |
| P3-F2 | No message source: injected user-role messages (Claude `<task-notification>`, `<teammate-message>`, hook/system notices; Codex `<codex_internal_context source="goal">`, AGENTS.md/environment context; Agent Manager wakes `[wake…]`, `[awaited result…]`) read as operator turns. The Claude codec drops `origin.kind`, `promptSource`, `isMeta` |
| P3-F3 | Outcome: the call exit code is copied onto every segment; `rg`/`grep` exit 1 (no match) counts as failure; Codex tool results never get an outcome |
| P3-F4 | Detector precision (sampled): handoff-continuation ≈0% (3,163/3,165 in goal-mode runs), fallback-after-capability ≈0% (1,703/1,709 start and end on one call), guidance-repair mostly injected messages, wrong-primitive misreads AGENTS.md rule 6 (search-hub is recall, not in-file grep), stall ≈5% (574/653 start at Codex `task_complete`), time-to-first-success ≈0%, edit-revert ≈5% (Codex delete+add rewrite), flag-hunting ≈10% (`cd`), wait-misuse ≈15% (`await` matches "wait"), capability-abandoned ≈20%, retry-after-failure ≈30%, read-then-reread ≈40% (`/proc`, logs, status files), command-failure ≈55%, abandoned-command-family ≈45% |
| P3-F5 | Episode cost: `wall_clock_ms` is one call's latency labelled "wasted"; `tokens` is all run usage between two timestamps (one episode: 115 ms, 7.66B tokens); `Turns` hard-coded |
| P3-F6 | Projection: "active" is judged against the newest attributed turn of any consumer, not now (`usageattribution/rules.go:296`); horizon stretches a burst across the week (a finished session projected at 150 pts); no cap at window headroom; recoverable reuses extrapolated past time |
| P3-F7 | Levers: fixed targets (fan-out brief 20k, context 60k) sit at or below the measured 46–60k prompt floor; fan-out never compares with doing the work in the parent; R-model sees only workload-tagged runs (interactive sessions are 78% of Claude burn) |
| P3-F8 | Calibration: Opus:Sonnet 1:0.5 is a hard-coded prior (`pricing/fallback.go` tierPriors) presented beside an uncentered R² (0.997 over 8 blocks); the model ratio is not identifiable when models share blocks |
| P3-F9 | Debt: pattern names hand-listed in four places; `measures` retry/help-recovery/repeated-work/tool-failure rates duplicate episode patterns; `run episodes` / `run messages-friction` / `run episode-trend` overlap `usage friction`; `measures help` lists an unregistered `episodes`; legacy `legacyOwnershipFree`; `BoundProjection` in `runreport`; tests pin the wrong behaviour |
| P3-F10 | The A11 change ("short worker briefs", from R-fanout) rests on a wrong mechanism: Claude sub-agents start at the prompt floor, not the parent context |

### 11.2 Targets

- **Facts.** Every fact carries a parsed command identity (`command_path`: program + subcommand words,
  e.g. `vrooli scenario logs`, `rg`, `ls`) and the intent key derives from it, never from argument
  shape. Codex `exec` JavaScript is read for its `exec_command({cmd})` strings; other code is a `code`
  fact, never shell. Each segment of a compound command gets its own outcome when knowable, else
  `unknown`; exit 1 of `rg`/`grep`/`diff`/`test` is `no-match`, not failure. Codex tool results get an
  outcome from the output's exit code.
- **Message source.** Every message event carries `source`: `operator`, `injected_context`,
  `notification`, `teammate`, `goal_continuation`, `wake`, `hook`, `tool_output` or `unknown`, from
  provider fields when present (Claude `origin.kind`, `promptSource`, `isMeta`) and from content
  markers otherwise (one shared classifier, reusing `isInjectedContextText`). Detectors that need the
  operator use `source == operator`.
- **Detectors.** One pattern table (id, version, cause scope, time kind, cost rule, description) is the
  only list. time-to-first-success is deleted. handoff-continuation never fires in goal-mode runs or
  across injected continuations and is informational (not priced). fallback-after-capability requires
  two distinct calls and the same `command_path` family intent. wrong-primitive is limited to recall
  intents (searching prior work/docs), not in-file search. Each other pattern takes the minimal fix in
  P3-F4's audit. Command-failure and abandoned-command-family on Vrooli-owned commands are routed to
  the owning scenario as owner.
  Operator corrections (guidance-repair) form one aggregate problem, owned by agent behaviour and
  never by a neighbouring command: a correction's subject is free text with no structured key, so
  splitting it would invent groupings. The problem lists every correction as evidence; dismissing
  it sets aside operator-correction friction as a whole (orchestrator, 2026-10-30).
- **Episode cost.** An episode is priced by its own redundant calls (their argument + result +
  residency tokens) and, where time is meaningful, by the redundant call's duration labelled as such.
  No episode reports run-wide tokens.
- **Precision gate.** A labelled corpus (true and false episodes per pattern, from real runs, reduced
  to the events the detector reads) and a test that computes per-pattern precision. A pattern below
  0.8 precision (or with fewer than 10 labelled episodes) is `quarantined`: kept in the read model,
  excluded from `usage friction`, R-friction and the UI.
- **Projection.** Active means `now − last ≤ max(15 min, 3 × the session's median request gap)`. A
  finished session projects to its points. Projection decays for active ones. The sum of projections
  and each lever's saving are capped at the window headroom (`100 − used%`) and at the lever's own
  base. Recoverable is the saving rate over remaining expected activity.
- **Levers.** The measured prompt floor (median first-request prompt per runner/model/profile) replaces
  constants. R-fanout is replaced by **R-lifetime** (worker restart at a request cap, netting the
  rebuild cost) and **R-model-mix** (sub-agents and interactive sessions on a dearer tier; shown as a
  range, "unproven" while the ratio is a prior). **R-floor** reports the floor's share and
  trimmable size. Fan-out fires only for duplicated workers.
- **Calibration.** Centered R² with a minimum block count (`max(8, 3 × parameters)`); fewer blocks
  → no R² shown; priors are labelled "prior" in the output.
- **Debt.** Every P3-F9 item is deleted, merged or derived; net non-test LOC in the API and CLI falls
  against the pass-3 baseline.
  Exception (orchestrator, 2026-10-29): `measures retry-rate`, `repeated-work-rate` and
  `tool-failure-rate` stay while `.vrooli/program-runtime/setpoint-read.py` and the
  `agent-manager-improve` skill read them; they now derive from pass-3 facts. Migrating those two
  readers to `usage friction` is the follow-up that retires them.

### 11.3 Pass-3 acceptance

| ID | Criterion | Proof |
|---|---|---|
| P3-A1 | Facts: <5% of shell facts lack a `command_path`; no fact has a JS keyword executable; segment outcomes and no-match semantics tested | unit tests + read-only SQL on the reindexed corpus |
| P3-A2 | Message source populated for new events and derived for stored ones; guidance-repair fires only on `operator` | codec tests (Claude and Codex fixtures) + SQL distribution |
| P3-A3 | Every pattern shown by `usage friction` has precision ≥0.8 on the labelled corpus; others quarantined | precision test output |
| P3-A4 | No episode carries run-wide tokens; no "wasted" label on a single call's latency | tests + live `usage friction --json` |
| P3-A5 | No finished session projects above its points; projections never exceed headroom; R-lifetime, R-model-mix, R-floor fire on the live corpus with measured floors | unit tests + live `usage recommendations` |
| P3-A6 | Calibration output: centered R², minimum blocks, priors labelled | tests + live `usage burners` |
| P3-A7 | P3-F9 debt closed; net non-test LOC falls in API and CLI | DEBT.md pass-3 rows + `wc -l` |
| P3-A8 | Go/CLI/UI tests and type-check pass; scoped Test Genie phases pass or failures shown pre-existing | command output |
| P3-A9 | Two consecutive fresh hostile review passes find nothing material | review outputs |

## 12. Pass 4 — find the friction (operator approved 2026-10-30, F-10)

Evaluation of 2026-10-30 (orchestrator, raw-transcript corpus of 60,921 tool calls over 7 days, two
read-only audits; memory `agent-manager-friction-measures-are-currently-untrustworthy`). Pass 3 removed
the false positives by hiding 15 of 17 detectors; the result is now dominated by **false negatives**.
`usage friction` shows one problem (0.06 pts) while the transcripts show large, owned, fixable friction.
Forecast, burners and per-session facts were verified accurate and are not reworked here.

### 12.1 Findings this pass must remove

| ID | Finding (evidence) |
|---|---|
| P4-F1 | Codex command outcomes are dropped at parse time. Web-console rollouts carry `event_msg`/`item_completed` `CommandExecution` items with `exit_code`, `status`, `duration` and `aggregated_output` for every command (44,431 in 76/88 files; 3,155 failed). The rollout `event_msg` switch (`codecs/codex.go:748`) handles no `item_completed`. `measures tool-usage`: `exec=142782 (0 success, 0 failed)`. SPEC P3-F3/WORKSTREAMS "Codex has no exit code" is wrong |
| P4-F2 | Real scenario friction is invisible: Codex scenario-CLI calls 890/10,347 failed — `identity_refresh_unavailable`/`web_console_session_unverified` 908 commands in 5/6 sessions; Agent Manager park unavailable inside `test-genie runs wait` 161 commands / 252 min inline (409 "runner type is unavailable", "no session ID"); unknown command/flag ≈518; capacity `over_subscribed`/admission saturated ≈533. `command-failure` saw 4 episodes all week; none recurred, so none shown |
| P4-F3 | Operator corrections: ≈1 in 6 real operator messages corrects or re-steers. The single regex (`runsignal/struggle_detectors.go:129`) recalls 10/26 hand-labelled corrections (early-stop 1/4, false-inability/over-asking 1/6). A second classifier (`selfreport.go` + rule pack) covers the same fact. The gate measures precision only. Corrections are priced in quota tokens of the correction turn, not in operator attention or pre-correction wasted work. Codex sub-agent reports arrive in the `user` role |
| P4-F4 | No tool-level view: `burners --by tool_kind` puts 78% in `shell`; there is no command/scenario dimension. `error-patterns`, `file-reread-rate` have sample 0; `tool-failure-rate`/`external-tool-share` unavailable (classified 83.8%); `capability-efficacy` sees 34 receipts vs 1,260 `search-hub` calls in transcripts |
| P4-F5 | Recommendations are per-session post-mortems: 14 groups, mostly stopped sessions labelled "projected"; titles are raw first prompts; zero-value rows (R-image 0.0). R-context assumes a fresh session never regrows and ignores compactions already done (1e5904ed had 47, 76d970de 6). Cross-session aggregates that matter are absent: Claude week cost at prompt >200k = 70%, sub-agent requests after #45 = 33%, Opus sub-agents = 30% |
| P4-F6 | Debt (audit 2026-10-30): 15 quarantined detectors ≈1,000 prod + 850 test LOC + 2.9 MB corpus; dead `Shown(...)` branches in `measures.go:144-147,1309-1364`; three shell parsers (`catalog.go:39` `shellSegments`, `invocation_shell.go:27` `lexShell` cyclomatic 90, `invocation_script.go`); two correction classifiers; three tool-kind taxonomies (`invocation.go:287`, `transcriptindex/facts.go:209`); duplicate cache-read ratio rules (`pricing/fallback.go` 1/20 vs `WithDefaultCacheRatios` 1/10); `FormatDuration` outside `usagelabel`; `finding_recurrence_rate` over a "deleted" findings table |
| P4-F7 | Surface sprawl (audit 2026-10-30): 28 hand-written `measures` subcommands (12 RPCs without a UI caller); single-dimension breakdowns duplicate `run stats` and `efficiency-report`; `burners` ≈ `token-attribution`; `usage run` ≈ `run report`; `run investigate` ≈ `investigation`; `findings` is a one-verb group already redirected; `ui/src/features/stats` survives a removed REST layer; four overlapping concept docs (USAGE-INTELLIGENCE 707, FAILURE_TOPOGRAPHY 426, operational-stats 299, token-attribution 231); `agent-manager-improve` setpoints stale |

### 12.2 Principles added

- **Silence needs proof.** A list with no findings is valid only beside its coverage: commands with a
  known outcome, operator messages classified, sessions analysed. Unknown coverage reads "cannot tell",
  never "no friction". Every shown detector has a recall estimate as well as precision.
- **Owner level first.** Findings aggregate across sessions to whoever can fix them: a command path
  (`program subcommand`), a scenario, a skill, a harness setting. Sessions and runs are drill-down
  evidence only.
- **Price in the unit of the harm.** Failed calls, wall-clock, context tax (result tokens × requests
  that re-read them before compaction), operator interventions; quota points only where quota is the harm.
- **Every finding is actionable and checkable.** Owner, plain action, evidence refs, expected effect,
  and the `usage compare` cohort that will verify it.
- **The baseline comes first.** Non-test LOC of every touched package is recorded in DEBT before work
  starts; net LOC falls (pass 3 lacked a baseline).

### 12.3 Workstreams

| WS | Target | Depends |
|---|---|---|
| P4-W1 Facts | Parse Codex `item_completed` `CommandExecution` into per-command facts (outcome, duration, output size), joined to the enclosing `exec` call. Per-call facts gain `result_tokens` and `residency_requests` (requests until the next compaction). Claude harness refusals (`<tool_use_error>Blocked`, hook errors) are their own outcome, not command failure. One shell parser (`invocation_shell.go` owns segments; `ResolveCatalog` re-expressed on it); one tool-kind taxonomy (runsignal). Capability usage derives from `command_path` facts, receipts become a cross-check | — |
| P4-W2 Tool friction profiles | One profile per `command_path` per window: failure rate (no-match excluded) with error-signature clusters (normalised first error line + owner); shape-guessing (unknown command/flag, `did you mean`, `--help` followed by a retry); output weight (median/p90/max result tokens, truncation share, context tax, share of calls post-filtered by `head`/`grep`/`jq`/`python`, JSON paths extracted after `--json`); latency (total, p90) and polling (same command ≥3× within 10 min, sleep/until wrappers); bypass (direct `sqlite3`/`curl`/state-dir access to the owning scenario's internals). Ranked by cost; each profile files to scenario-qa with evidence, deduplicated against open bugs. Replaces `error-patterns`, `file-reread-rate`, `tool-failure-rate`, `external-tool-share`, `retry-rate`, `repeated-work-rate` and the quarantined structural detectors | W1 |
| P4-W3 Opportunities | Command-sequence mining: `command_path` 3–4-grams across ≥3 sessions, arguments templated, matched against the program library → "adopt existing program" or "program candidate". Bypass clusters → capability candidates (which tables/endpoints agents reach for). Repeated skill loads in one session → skill-size/split or digest candidates. Candidates are records for prompt-manager/program-runtime, never auto-built | W2 |
| P4-W4 Operator interventions | Classify `source=operator` messages with the preceding agent turn by a local model (Ollama, Haiku fallback; ≈400 messages/week). Categories: early stop, false inability/over-asking, wrong claim/recall miss, quality miss, too dense, scope change (not friction), approval/information (not friction). Each category maps to an owning skill or doc. Measures: interventions per agent-hour, agent work since the previous operator message. Labelled set ≥150 with precision and recall per category. Deletes the regex and `selfreport` operator-correction pack. Codex inter-agent `user`-role messages get `source=teammate` | W1 (source) |
| P4-W5 Recommendations | Owner-level levers from cross-session aggregates (context >200k share, sub-agent lifetime, model mix, skill-load weight, top tool profiles), each with a verifying `compare` cohort. R-context uses the R-lifetime growth model and counts existing compactions. Finished sessions say "would have saved"; zero-value rows are dropped; labels are short titles. Per-session lists move to drill-down | W1, W2 |
| P4-W6 Debt and surface | Delete the 15 quarantined detectors, their tests and corpus, the `Quarantined`/`Shown` machinery and the dead `measures` branches as W2/W4 replace them. `measures` subcommands generated from `AllMeasureDefinitions` (`measures get <id>`); breakdowns → one `--by`; delete `run stats`; merge `burners`+`token-attribution`, `usage run`→`run report`, `run investigate`→`investigation`, `findings`→`friction`; remove `features/stats`; one cache-ratio rule; `FormatDuration` into `usagelabel`; one concept doc plus reference; skills (`agent-manager-improve`, `reviewing-agent-run-efficiency`) migrated, then the three kept rate verbs retire | W2, W4 |
| P4-W7 Proof | A seeded recall fixture from this week's corpus (the P4-F2 clusters and the 26 labelled corrections) must be found by the live pipeline. Live acceptance after reindex. Hostile reviews ask explicitly for user-visible strings, totals, and "what did it miss" | all |

Out of scope for agent-manager, filed separately (owners fix them; W2 must find them first):
`identity_refresh_unavailable` in web-console Codex sessions (control plane / web-console); park
refusal for interactive runs (agent-manager orchestration); unbounded CLI output (unpiped single calls
returning 70k–262k tokens: `prompt-manager skill`, `prompt-manager team`, `device-control audit`,
`browser-automation-studio executions`, `test-genie validation`, `react-component-library components`,
`scenario-dependency-analyzer scan`, `plan-manager plans`, `vrooli locks`, `agent-manager task`);
non-unique skill `local-experimentation`; unregistered memory scope `unit-health-usage`.

Amendment AM-12a (2026-10-30, P4-W1c): provider-derived events of **interactive** runs are not re-derived after a codec change in this pass. Interactive transcripts are retained 7 days (`run_state` max_age), so re-derivation could only reach the window that ages out by itself (276/307 interactive Codex runs with stored events already lack their transcript). Facts record the transcript parser version that produced them, so older-codec evidence reads as "older codec", never as "no outcome". P4-A1 is measured per command (distinct result event) over Codex commands whose events were produced by the current parser (imported runs re-imported by P4-W1b, and all runs since the change); stored interactive history from before 2026-10-30 is reported separately and leaves the 7-day window by 2026-11-04. Implemented at read time (P4-W6c, no fact-version bump): `ToolFriction` coverage joins each run to `transcript_files.parser_version`, counts the commands of runs whose recorded version is below the harness's current one (or that have no record) as `older_parser_commands`, and prints "N commands from runs read by an older parser; outcomes there may be missing". Build trigger for re-derivation (design option (a) in WORKSTREAMS P4-W1c: deterministic transcript event ids + anchor re-sequencing, replacing the imported drop-and-re-read): interactive transcript retention raised to the 30-day recall window, or a codec change must be proven on interactive data within 7 days.

### 12.4 Pass-4 acceptance

| ID | Criterion | Proof |
|---|---|---|
| P4-A1 | ≥95% of Codex commands in the 7-day corpus have a known outcome; `measures tool-usage` shows Codex success/failure | SQL on reindexed corpus + live output |
| P4-A2 | Live `usage friction` finds every seeded P4-F2 cluster with its owner and a fileable report; empty lists print coverage | live output vs fixture |
| P4-A3 | Tool profiles exist for every Vrooli `command_path` with ≥20 calls; the top 10 by cost match an independent transcript recount within 10% | independent script + live output |
| P4-A4 | Operator-intervention classifier: precision ≥0.8 and recall ≥0.7 per shown category on a held-out split | test output |
| P4-A5 | Opportunities list includes the orientation sequence (journal → search-hub → skill) and the test loop with `test-genie.iterate` as "adopt existing" | live output |
| P4-A6 | Recommendations: no stopped session labelled "projected"; R-context counts compactions; owner-level levers shown with a verifying cohort | tests + live output |
| P4-A7 | P4-F6 and P4-F7 closed; net non-test LOC falls against the recorded baseline in API, CLI and UI | DEBT rows + `wc -l` |
| P4-A8 | Tests, type-check and scoped Test Genie phases pass | command output |
| P4-A9 | Two consecutive fresh hostile reviews find nothing material, one of them a recall review | review outputs |
