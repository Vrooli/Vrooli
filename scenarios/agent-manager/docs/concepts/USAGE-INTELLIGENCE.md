# Usage Intelligence

Usage Intelligence lets an operator answer these questions about their provider
subscriptions in under a minute, with numbers that agree with the provider and
with the transcripts on disk:

1. **Will I run out?** Per provider window: percent used, reset time, burn rate,
   projected exhaustion with a confidence band, and the rate that would last
   until the reset.
2. **What is using it?** Share of the window's consumption by session, run,
   profile, workload, model and runner, with an explicit unattributed remainder.
3. **What should I change?** Ranked recommendations, each with an estimated
   weekly quota saving, confidence, evidence and a concrete action.
4. **Can I trust this?** Capture coverage against the on-disk truth, with every
   uncaptured source named.
5. **Did the change work?** Quota per successful unit of work, before and after
   a change, on comparable work.

Product target: `OT-P0-016` in [PRD.md](../../PRD.md). Requirements:
[`requirements/26-usage-intelligence`](../../requirements/26-usage-intelligence/module.json).
This document is the durable design. The delivery effort that builds it, with
its measured baseline, debt ledger and workstreams, lives in
[`docs/efforts/usage-intelligence/`](../efforts/usage-intelligence/README.md);
that folder is temporary and is removed when the effort closes.

## Status

The design was approved on 2026-10-29 and is being implemented. T-4
(forecast, backtest and the will-exhaust alert), T-1 (the Transcript
Indexer, see the implementation notes under T-1), T-5 (attribution) and T-6
(recommendations) are implemented. Until a
section below says otherwise, the code still has the behavior it replaces:
quota frames stored as `error` events, Codex
receipt-versus-delta token accounting, and hooks applied only on the
continuation event sink. [DATA.md](DATA.md),
[effort-supervision.md](../reference/effort-supervision.md) and
[EVENT_TAXONOMY.md](../internal/EVENT_TAXONOMY.md) mark current behavior and
target behavior separately.

## Principles

- **Provider readings first.** Forecasts use the provider's own `used_percent`
  samples. Token counts explain and attribute consumption; they never replace a
  provider reading. A missing reading is unknown, never zero.
- **One derivation per fact.** One token-accounting function, one event-sink
  constructor with one hook set, one capture path. Duplicates are deleted, not
  wrapped.
- **Summaries outlive events.** Usage and quota summaries are small and durable.
  Full event recall is optional and has its own byte budget.
- **Deterministic first.** Recommendations come from inspectable rules over
  durable facts. Model inference is optional, and its output is labelled.
- **Less code.** The feature lands by replacing the capture, accounting and
  analytics paths it supersedes, so the touched areas end smaller.

## Architecture

```text
 transcripts on disk (codex rollouts, claude projects + subagents, AM run homes,
 web-console homes)          claude statusline feed
            │                              │
            ▼                              ▼
   ┌─────────────────────── Transcript Indexer ───────────────────────┐
   │ transcript_files(file_key=dev:inode, path, harness, header_id,   │
   │   parent_id, web_console_session_id, run_id?, offset, size,      │
   │   mtime, status) — incremental tail with the codec parsers       │
   └───────┬───────────────────────┬──────────────────────┬───────────┘
           ▼                       ▼                      ▼
   usage_counters            quota_observations      run_events + search
   (raw cumulative            (provider readings)     (optional recall layer,
    counters, durable)          durable                own byte budget)
           │                       │
           ▼                       ▼
   Usage accounting (one function: last cumulative − baseline per session)
           │
           ▼
   Forecast · Attribution · Coverage · Recommendations  (Measures procedures)
           │
           ├── CLI `agent-manager usage forecast|burners|recommendations|coverage|compare`
           ├── UI Usage page, dashboard strip, run usage panel
           └── notification-hub alert when projected exhaustion precedes reset
```

### T-1 Capture

One per-file incremental indexer reads every transcript root: Codex rollouts
(including `<parent>_<child>` fork files), Claude Code project files and their
`subagents/` sidechains, Agent Manager run homes and web-console homes. A file
is identified by device and inode, not by the session id in its header, so
forks and sidechains no longer collide with their parent. The indexer tails
each file from its stored offset, so a growing session is never frozen at the
size it had when first seen. Agent Manager-launched runs register their own
file keys, so their turns are counted once.

The indexer replaces snapshot import, import tombstones, the "managed session"
skip and the full transcript copies under `agent-manager/runs`. Attached runs
stay identity and authority records only. They link to their files through
`web_console_session_id`, and a normal process exit is `complete`, not
`failed`.

Implementation (`api/internal/transcriptindex`, host side in
`api/internal/orchestration/transcript_index.go`):

- **Ownership.** Each file has one reader of events. Files under an Agent
  Manager run home, and the transcript an interactive run tails, are
  `owner = run` rows: the run ingests their events and quota readings, and the
  indexer records only their usage summaries, keyed to that run (D-16), so
  usage has one source. A run-owned row registered before 2026-10-29 (never
  indexed) is read once from the start. Every other file is read only by the
  indexer. A Claude sidechain of an interactive
  run is not read by its tail, so the indexer reads it as a child run.
- **Runs.** One `imported` run per top-level file; fork rollouts and Claude
  sidechains are child runs (`parentRunId`) of their parent file's run. A
  forked Codex rollout (`session_meta.forked_from_id`) opens with a copy of its
  parent's history; the codec yields nothing for it, so the fork's events begin
  at its first line that names the fork's thread (`thread_settings_applied`)
  or a turn begun after the fork (Codex transcript version 3). The run
  is `running` while the file grew in the last 10 minutes, then `complete`.
  A session the retired snapshot importer adopted is relinked to its source
  file, its copy is deleted and it is re-read from zero, so the projection
  sweep replaces its stale read model.
- **Durable summaries.** `transcript_usage` keeps one row per provider usage
  report: raw cumulative counter, the report's own amount, model, session,
  message id, and the non-text work the request consumed since the previous
  report (images in with user/high-detail/bytes, images generated, web
  searches and fetches, audio, tool calls by kind, the Codex credits
  balance). Tool kinds come from the one table in `runsignal` that invocation
  facts also use: `shell`, `file-read`, `file-edit`, `search`, `wait`, `plan`,
  `delegate`, `web-search`, `web-fetch`, `image-generation`, `mcp`,
  `mcp:<server>.<tool>` and `scenario:<owner>` (the catalog owner of the
  scenario CLI a call runs). Quota frames go to
  `quota_observations` on the same read. Neither is evicted. Non-text work
  read after a file's newest report is saved with the file's offset
  (`pending_facts_json`), so a restart still attaches it to the next report.
  A Claude provider turn starts at the first assistant line after a prompt,
  as `turn_context` starts one for Codex.
- **Recall.** Full events and search documents are written only for files
  modified in the last 30 days and are bounded by their own budget (4 GiB),
  measured on those rows. Eviction drops the least recently active files'
  recall; projection and accounting then read the usage rows, and an explicit
  replay rebuilds events from the source file.
- **Codec changes.** A file row records the parser version that derived its
  events. A newer parser reimports from a bounded staged snapshot and atomically
  promotes it only while the prior file, usage and event evidence still match.
  A source or promotion failure preserves prior evidence. Guarded replacements
  can be inspected and explicitly retried through the exact-target owner route
  described in [DATA.md](DATA.md); uncertain event-recording outcomes cannot.
- **Coverage.** `transcriptindex.CoverageReader.Coverage(since)` walks the
  roots and joins them with the rows: per root kind and harness, files,
  indexed, with usage, run-owned, excluded, pending, unreadable (with
  reasons), bytes and captured bytes. It is served by the Measures procedure
  `UsageCoverage` (`agent-manager usage coverage`, default 7 days).
- **Report time and model.** A usage row carries its transcript line's time
  and the turn's model (Codex `turn_context.model`, Claude `message.model`).
  A row stays `unknown` only when its transcript names no model: 108 Codex
  rows and one Claude row on this host. Rows indexed before 2026-10-29 carried
  the indexing time and `unknown`; a start-up repair rewrote them from the
  transcripts once and has been removed.
- **Contention is not unreadability.** A SQLite busy or locked error while a
  file is read leaves the file `pending`. When an event write may have
  committed before an error, the indexer preserves the prior cursor and does
  not replay automatically: path, size, mtime and a short head hash cannot
  prove that the remaining source bytes are unchanged. Such an uncertain
  recording becomes an `unreadable` issue on the next pass. A durable capture
  intent precedes imported event writes; the run status and checkpoint commit
  together, while event and downstream projection effects remain separate.
  Inspect the
  exact file and reason with `agent-manager usage coverage --json` (choose a
  sufficiently early `--since`). There is no supported automatic repair for
  those unresolved cases: preserve the transcript and run evidence and seek
  an owner-level reconciliation; importing it again does not clear the guard.
  Files marked unreadable by contention before 2026-10-29 are re-read.

### T-2 Accounting

Usage rows store the provider's raw cumulative counters: `total_token_usage`
or exec `turn.completed.usage` for Codex, and summed per-message `usage` for
Claude. A session's usage is its last cumulative value minus the cumulative
value before the run's first turn. Because the result depends only on two
counter readings, it does not change when a file is re-tailed, when the API
restarts or when recovery replays a transcript. This removes the Codex
delta/baseline machinery and the receipt-versus-delta authority selection,
and every "total tokens for a run" computation calls this one function.

Read-model projection is driven by a durable "needs projection" mark that every
terminal writer sets, plus an idempotent sweep. It is not a fire-and-forget
goroutine, so a run that ends through the Reconciler or a busy database is
still projected.

### T-3 Events

Provider quota frames have their own `quota` event type instead of `error`.
Every event sink is built by one constructor that always applies the
orchestrator's post-persist hooks: quota observation, token caps and workflow
usage nudges. Each quota event produces one deduplicated `quota_observations`
row. Claude Code's utilization percentage, which transcripts do not carry, comes
from a `statusline` subcommand owned and installed by `resources/claude-code`.

### Account scope and launcher limits

Quota observations now have an optional opaque account scope. A blank scope
means unknown; it does not identify a shared account. Account scope partitions
observation identity, heartbeat deduplication, compaction and explicit scoped
reads. Legacy rows migrate as unknown, and their original observation IDs stay
stable for replay. Forecast and backtest accept an exact `account_scope`;
without it they read only legacy unknown-scope history. Scoped forecasts do not
use provider-wide activity to infer account idleness. Attribution and alerts
still consume only legacy unknown-scope history. An exact `account_scope`
filter on the bounded quota observation read selects only that scope; the
caller must independently prove the selected launch uses it. The read computes
freshness at request time, including an elapsed reset.

Neither current Claude statusline samples nor current Codex transcript quota
frames establish account identity, so their observations remain unscoped. No
current reading can yet be represented as verified for a Web Console launcher.
`GET /api/v1/pricing/live-quota?agent=codex` performs a bounded, coalesced local
app-server read, returning provider windows without account identifiers. Its
`likely` match reflects this Agent Manager process's login, not a proven future
launcher account. `agent=claude-code` uses the newest coherent, recent
statusline sample; recent contradictory samples from different sessions make
the estimate unavailable. Stale or elapsed windows are not reported as current.
Unsupported agents and failed reads return unavailable, never fabricated zero.
The endpoint runs against the Agent Manager host and cannot be projected onto a
different remote target. Codex responses are cached for one minute and failures
for ten seconds; Claude timestamps are the original sample time.
The installed Codex app-server schema exposes an optional account ID alongside
`account/rateLimits/read`. The live reader discards that identifier and does
not establish its relationship to a future launcher. Do not associate existing
history with an account or treat a hostname, provider name or copied home as
proof. The Web Console launcher
currently runs operator-owned shell shortcuts in a PTY, whereas the existing
managed app-server is a distinct launch mode. A prospective read from some
other app-server cannot prove the credentials, executable or remote target that
an arbitrary shortcut will use. A future integration must bind an exact launch
intent and target to a provider-returned account scope, revalidate that binding
at launch, and test wrappers, environment overrides and account changes. The
provider's `accountId` is optional and must remain backend-only; its absence
prevents verified account matching. The installed Codex schema also supports multiple limit buckets,
so classify 5-hour and weekly windows by duration, not by primary/secondary
position. On 2026-10-29, a bounded read using the installed Codex app-server
in the investigating agent's inherited environment returned an account ID and
one 10080-minute window; no 300-minute window was present. Only booleans and
window lengths were retained as evidence. This proves neither that the next
Web Console launch uses that account nor that all Codex accounts report the
same windows. No credentials, account ID, or utilization percentage were
recorded. The launcher must not use this unrelated sample as its own reading.

### T-4 Forecast

Implemented in `api/internal/usageforecast` and served by the Measures
procedures `UsageForecast` and `UsageForecastBacktest`.

**Readings.** The forecaster reads `quota_observations` through
`QuotaReadingSpans`: one row per distinct (provider, pool, window, reset minute,
used percent, standing) with its first and last observation time. A reading
without a reset time (Codex rollouts before 2025-11 carried `resets_at: null`)
is grouped per hour of observation instead, because its percent repeats across
windows. The table is deduplicated (SPEC A3): `RecordQuotaObservation` skips a
reading that repeats the scope's preceding row (same reset minute or reading
hour, percent and standing) within `pricing.QuotaHeartbeat` (5 minutes). The
preceding row, not the newest, is compared, because back-fill and concurrent
sessions write out of time order. Rows stored before deduplication are removed
once per start by `CompactQuotaObservations`. It runs in the background,
pages each scope by time and deletes in batches of 500. In each run of
identical readings it keeps the first row, the last row and one row per
heartbeat, so every span's first and last time, and therefore the forecast
and backtest, are unchanged. A second pass deletes nothing. Freed pages return
to the file through incremental auto-vacuum.

**Windows.** Readings are grouped by provider, pool and window length, then by
reset time: readings whose `resets_at` differ by at most 2 minutes are one
window (the provider jitters it by 1–5 s). Only a new reset time starts a new
window. A lower percent under the same reset time is a lagging reader (F-ORCH-3),
so the series is the running maximum. A reading whose reset had passed when it
was observed is rejected at ingest (`pricing.ErrResetElapsed`, from
`QuotaObservation.Validate`, so the statusline feed and every codec quota path
share it). Readings without a reset time start a new window when
used percent falls by 2 or more points, or when the window has run its nominal
length plus 5 minutes. No window outlasts its nominal length: the 2025-10-30 →
2025-11-23 "5h" window was the reset-less readings grouped as one. Window
lengths are rounded to the hour, because Codex reported 299 and 10079 minutes.
The window starts at `resets_at − window_minutes`, where the provider's used
percent is zero.

**Series and burn.** The step series holds the first time each higher percent
was reported. Burn is measured over the trailing 1h, 6h and 24h ending **now**,
not at the newest reading, clipped to the window start. A horizon needs at
least 10 minutes of history and a real reading at or before its start
(F-ORCH-5). A horizon with no reading inside it is **stale**: it shows as
stale, not as a current rate. The forecast reports the last reading's
percent and time beside the window maximum. These differ when the newest
reader lags.

**Projection.** The projected burn from now to the reset is the burn since the
window began, shaped by the operator's burn profile and scaled by a
calibration:

- *Profile.* Every used-percent increase in the scope's readings before the
  window began is attributed to its hour of the week (UTC). Each hour of the
  week has few samples, so its share is shrunk halfway toward its hour of the
  day. A history shorter than 4 weeks is mixed toward uniform in proportion,
  and 20% always stays uniform. With no history the profile is uniform, and the
  projection is the linear burn since the start. The window's own readings are
  excluded: they would teach the profile that only the hours already burned
  are active.
- *Intensity.* The used percent since the start, divided by the profile weight
  of the elapsed hours.
- *Calibration.* Take the scope's 10 newest completed windows with an observed
  outcome that ended before the newest reading. At the same fraction of each
  one's life, compare the burn that followed with what this projection would
  have said. The factor is the median ratio, and the quartiles bound it. With
  fewer than 3 windows the factor is 1, and the status reason says
  "uncalibrated".

The run-out is the time at which the profile-weighted burn from now uses the
remaining percent.

**Band.** The band scales the intensity by the union of three ranges:

- the calibration quartiles;
- the ±1 point integer rounding of the burn since the start (`(Δ ± 1) / Δ`);
- the spread of the current horizons around what the profile expected over
  them, counting a horizon only where it differs by more than its own
  rounding, and then by its nearer edge.

The fast edge (with one point less to go) gives the band start, and the slow
edge (with one point more) gives the band end. A zero slow edge means the band
never ends. The burn that lasts to the reset is `remaining / time to reset`.

| Status | Meaning |
|---|---|
| `on_track` | The band starts after the reset |
| `at_risk` | The band straddles the reset |
| `will_exhaust` | The whole band ends before the reset, or the provider reports 100% |
| `idle` | The newest sample is older than 6 hours inside its own period, and no Agent Manager run, active session or indexed usage of that provider's runner came after it: the window lasts at this pace. Needs the activity source; without it the window stays `unknown` |
| `unknown` | No samples, the newest sample is older than 6 hours with work since (or unmetered use cannot be ruled out), the reset has passed with no reading of the new window, or there is less than 10 minutes of history |

A scope whose newest window ended more than one nominal window length ago is
no longer read (an old account or pool) and is not a current window.

A window that reports a standing only (for example Claude `quotaLimits` without
a percentage) reports that standing: `exhausted` is `will_exhaust`, `limited` is
`at_risk` and `available` is `on_track`. It has no projection. When the window
began before the previous window's scheduled reset, the forecast carries an
early-reset note with the previous window's final percent.

**Backtest (A4).** For each completed window, the backtest projects at 25% and
50% of the window's actual life with the same `project` code the live forecast
uses. It compares the projected percent at the window's end with what
happened, and when the window ran out it also compares the run-out time. The
baseline is the naive linear rate since the window start. Both the model and
the baseline project to the window's actual end. The replay is time-ordered:
a checkpoint sees only readings up to it, a profile built before the window
began, and calibration windows that ended by then (`TestBacktestNeverLooksAhead`).
A window is complete when a later window began or its reset has passed. It is
scored only when its outcome was observed: it exhausted, or its newest reading
is within 6 hours of its end. A checkpoint whose newest reading is stale is
skipped. Two populations are reported per checkpoint: `last_5_windows_per_scope`
(SPEC A4: the five newest scored windows of each provider/pool/window) and
`all_windows`. Validity comes from the smaller recent population and is
`unreliable` below 5 scored windows.

Measured 2026-10-29 on a copy of the live table (2.21M rows, 1,061 completed
windows, 26 not scored), median absolute error of the final percent:

| Population | At | Windows | Forecast | Linear baseline |
|---|---|---|---|---|
| last 5 per scope | 25% | 15 | 7.3 pts | 11.0 pts |
| last 5 per scope | 50% | 16 | 9.9 pts | 13.0 pts |
| all windows | 25% | 750 | 4.1 pts | 10.0 pts |
| all windows | 50% | 880 | 2.3 pts | 6.7 pts |

Before this model (longest-horizon burn) the same history scored 8.0 vs 7.9
and 5.2 vs 5.1. Caveats:

- The profile granularity was chosen on this data. Hour of the week alone and
  hour of the day alone also beat the baseline on all windows. On the 15-window
  recent cell, however, they scored 11.6 and 13.3 against 11.0. The
  hierarchical blend was chosen for bias-variance reasons and is the only one
  that wins every cell. The recent cells are small and sensitive.
- Only one scored window exhausted, so the run-out time error (16.2h vs 6.8h
  at 25%) is one sample.
- The secondary weekly scope in July 2026 holds overlapping windows: two reset
  times are active at once, which most likely means two credentials. The
  readings carry no account, so they share a scope.

Reproduce with `AM_QUOTA_LIVE_BACKTEST=1 go test ./internal/adapters/database/
-run TestLiveQuotaBacktestAndCompaction -v`. The same run proves that the
compaction deletes 2,081,088 of the 2,214,976 rows (133,888 remain). It shrinks
the table and its indexes from 1,322.7 to 124.2 MiB, and the backtest stays
identical.

### T-5 Attribution

Implemented in `api/internal/usageattribution` and served by the Measures
procedure `UsageBurners` (`agent-manager usage burners`).

**Usage.** Attribution reads one source, `transcript_usage`: the indexer
records a row per provider report for every file, operator sessions and
Agent Manager runs alike (run-owned rows carry the run's id), so no report is
counted twice and run events are never read. Codex reports are folded into
10-minute buckets per file; a bucket keeps its highest cumulative counter and
the counter before its first report, so `runsignal.AccountUsage` credits
exactly the bucket's usage. Claude reports count once per message.

**Blocks.** The window's step series gives the observed increases; one-point
steps are merged into blocks of at least 3 points, because integer rounding
moves one step's usage by up to a point. Each block's increase is explained
by the usage that ran in it.

**Weights.** A turn's cost is its tokens priced per category and model. The
ratios come from the price book when the model is priced; a subscription
model the runner resources resolve to an unpriced name uses its tier prior:
Codex sol:terra:luna 10:5:1 per cached token (the E-2 per-model fit), Claude
opus:sonnet:haiku by list-price ratio, categories cached:input:creation:output
1:20:25:100. A per-model least-squares fit on integer steps proved collinear
with an off-disk rate (it priced terra at zero on the live window), so the
models share one scale per window. Prices are looked up concurrently per
request and memoized for 10 minutes across requests (`usageattribution.PriceBook`);
a stale price serves while it refreshes. A component neither the price book
nor a manual override prices stays unpriced; the retired historical-average
fallback (a JSON scan of `run_events` for cost fields no codec writes) is
deleted.

**Calibration.** Off-disk use only ever adds to a block's increase, so the
window's scale is the 25th percentile (increase-weighted) of each block's
increase over its usage cost (`fitted`). A block's usage explains at most its
increase at that scale; the rest is unattributed. A fit reports a centered R²
(against the blocks' mean increase) and needs at least `max(8, 3 × fitted
parameters)` blocks with usage; the scale is the only fitted parameter. With
fewer, the lowest block ratio scales (`scaled`), no R² is reported and what it
does not explain stays off-disk. Model and category ratios are priors (price
book or tier ratios, e.g. Opus:Sonnet 1:0.5), never fitted, and the output says
so; models that share blocks cannot be told apart from a window's own data. The
scale, per-model cached tokens per point and their source are reported per window.
On 2026-10-29 the Codex window calibrated to terra ≈138M and luna ≈690M
cached tokens per point (E-2: 100M and 520M).

**Shares.** A turn's points are its block's explained increase times its
share of the block's cost. Group-by session (web-console session or
top-level transcript session with its sub-agents and forks; each Agent
Manager run is its own session), run, profile, workload, model, runner,
modality (the turn's primary non-text work) or tool kind (split by call
count). For every grouping, burners plus the unattributed remainder equal the
observed increase. Rows past the list and rows under `fold_below_points`
(0.1) fold into one "N smaller items" row the API sums (`other_burners`,
`other_points`); the page renders that row as sent. Usage after the newest
reading is reported as pending.
Work the provider meters outside the token counters (Codex credits, web
searches, image generation) gets its own line and is never added to the
points. A Vrooli scenario call is local work, not provider-metered: its count,
failures and output weight are in the tool friction profiles below. Points are estimates and are presented as approximate.

### T-6 Recommendations

Implemented in `api/internal/usageattribution` (`Recommend`, `actions.go` and
`rules.go`) and served by `UsageRecommendations` (`agent-manager usage
recommendations`). It is **one ranked list of owner-level actions** (SPEC
§12.4 P4-A6) merged from four sources:

| Source | What it contributes | Owner kinds |
|---|---|---|
| `lever` | quota levers over each provider's longest current window (or one window named by `provider`, `pool`, `window`) | harness setting, session (still running only), workload |
| `tool` | `toolfriction` findings (failure, degraded, shape-guessing, output-weight, polling, slow) and R-skill | command, scenario, skill |
| `opportunity` | repeated sequences (adopt / investigate / write a program) and bypasses | program, scenario |
| `intervention` | operator corrections the accuracy gate (`interventions.ProvenGate`) admits | skill or doc |

Each `UsageAction` carries owner (`owner_kind`, `owner`), a title of at most 60
characters, the plain action, rationale, evidence refs, drill-down sessions, the
effect **in its own unit** (`points`, `calls`, `ms`, `tokens`, `interventions`)
with a basis (`would have saved` when every part has finished, `projected` only
when a part is still running, `range` for an unproven move, `per 10k tokens
trimmed`, `observed`), a confidence and the command or measure that verifies it.
One behaviour is listed once: bypasses come from opportunities (which name what
agents reached), not the tool finding; skill loads are R-skill, not the
`prompt-manager skill read` output-weight finding; output-shape and
skill-reload opportunities stay in `usage opportunities`.

**Ranking.** Each action removes a *share* of its own kind of harm: quota
levers of their window's observed points; tool findings of their finding's
largest share over that kind's base (`toolfriction.Totals.Base`, the same base
the friction list states; a root-cause roll-up states its other kinds' shares
beside the score: "also degraded 67 calls = 0.3% of the window's 24878 Vrooli
CLI calls"); opportunities of the window's hand-run Vrooli CLI calls or bypass
calls; operator corrections of operator messages. `score` (one rule,
`toolfriction.Score`, shared with the friction list) is listed below the next
paragraph.

Actions carry what makes them concrete: a root-cause roll-up is one action
naming the root and listing its lines (`subs`), a shape-guessing action names
the alias to add or the help to fix (`guesses`), an output-weight finding states
the tokens the largest call printed against those the model read, and a finding
or bypass made mostly by one run states that run's share (`top_run_share`);
its cost counts each run at most 10x the median run, which the score basis says.

Two scales, never interleaved: the list is the quota levers first (the
**quota scale**), then the friction actions (the **friction scale**), each
group ranked by its own score, then a live session's own advice. Every
`score_basis` starts with its scale and states the absolute cost beside the
share ("friction scale: 22.8% = √(share 17.7% (189 calls) of the window's 1070
polled Vrooli CLI calls × breadth 29.4%: 78 of 265 runs)"; "quota scale: 26.5%
(13.0 pts) of the openai 10080m window's 49.0 observed points"; tokens and
time read "96.8M tokens", "8m 31s"), so a reader sees that a 26-call
shape-guessing action and a 96.8M-token output-weight action rank by their
shares, not their sizes. The CLI and the UI head each group with its scale.

- quota lever: `share` (its quota share). Quota levers are **not additive**:
  levers of one window that price the same runs (a sub-agent restart, its
  starting context, its model) price the same quota there. Each lever records
  every run it prices and names the higher-ranked levers of its window it
  shares runs with ("Not additive: it shares runs with #1 on 108, #7 on 96 of
  its 120 runs, …; take the larger saving per run, not the sum"); the window's
  outcome combines them, each run once at its largest lever
  (`CombinedRecoverable`). A lever's verify command carries every evidence run
  it names and says how many runs it prices ("its 10 evidence runs (of the 108
  runs it prices) against the runs since the change"); its right side is the
  same runner or model filter over a custom window from the time of the change,
  which cannot hold the left side's runs (a `this_week` right side did);
- any action with a run count (tool findings, opportunities, R-skill):
  `score = √(share × breadth)`, `breadth = runs showing it ÷ distinct runs in
  the window's tool calls`. The geometric mean keeps the scale of a share, which
  a plain product would not (two fractions multiplied fall under the floor),
  and equals the share when the harm spreads over every run. One run's jq loop
  holding most of a harm kind scores `√(0.3 × 1/200)`; a control-plane defect
  blocking 16 runs scores `√(0.05 × 16/200)`, so it ranks above it;
- operator corrections: no run count exists, so the share alone.

`score_basis` prints the inputs: `friction scale: 7.5% = √(share 11.3% (543
calls) of the window's 4805 polled calls × breadth 5.0%: 14 of 281 runs)`. Kinds of harm weigh
equally, a stated assumption: no measured exchange rate between quota, failed
calls, wall-clock and operator attention exists, and converting a failed call
into the quota of one model request (≈0.002 pts on this host) ranked every tool
finding below the floor, which contradicts "price in the unit of the harm"
(SPEC §12.2). A range is scored at its bottom (0) and ranks by its top after
the point estimates. An action whose *score* is under
`toolfriction.MaterialScore` (1%; a range: whose top is) is counted
(`below_materiality`), not listed; the friction list applies the same floor to
the same score. The floor reads the score, not the share alone: a park refusal
in 28 of 283 runs holds 0.5% of the Vrooli CLI calls and 209 minutes of
inline waits, and a share-only floor hid it. A live session's own advice (owner kind
`session`) ranks after every aggregate: it is one run's and ends with the run.
A providers-degraded summary whose promised notes the output never printed is
an action for that CLI to print them ("Make search-hub print the
degraded-provider notes it promises").
A still-running session already priced in a lever aggregate is that aggregate's
drill-down, never a second action for the same quota.

Polling of a command whose owner declares a blocking sibling is an action for
the skills and docs that send agents to it and the command's help: "Agents poll
`X` where `Y` blocks: name it in the skills and docs that send agents to `X`,
and in its help and not-done output (30 of 90 polled calls); no blocking sibling
is shown to apply to the other 60" (owner kind `skill`). Its effect is the polls the
sibling stands in for (`Finding.Steered`), not every poll: a sibling declared to
block only after another command (below, polling) claims only the polls that
followed one. With no poll covered, the command's owner is told to add a
blocking wait. A
shape-guessing action whose help lookups outnumber its wrong-shape calls says
the usage is not memorable at the call site (AGENTS.md/help), then names any
guessed alias, named only with three calls of evidence (`minGuessCalls`): one
agent's `--title` is not the CLI's to fix. A finding a line attributes to another scenario than the
command's CLI (a park refusal printed inside `test-genie runs wait`) is that
scenario's action, not the command's. A rule's row count is taken after merging, and a merged action
keeps every finding's `fingerprints`, so each stays reportable.

A bypass and a tool finding about the same target of the same owner (they share
a target word) merge into one action that lists both (a bypass of test-genie's artifacts and a finding
that `test-genie runs artifacts` is guessed at become "Add or repair artifact
access in test-genie"). It takes the larger score: the signals overlap, so they
do not add. A lever's saving is priced at the one cap the operator is told
(the median of the members' best caps): every member is re-evaluated at it and
one that nets nothing there drops out. The growth levers (R-context,
R-lifetime) state one cap per runner, fitted in the provider's longest current
window: a 5-hour and a weekly window see the same sessions and would otherwise
tell one runner two caps. A sequence opportunity counts occurrences as periods
of its shape (a loop needs one full repeat, at most 8 steps a turn) and calls
saved as the calls of the subject's own steps less one per period, at most
(steps − 1) per occurrence: other commands a merged stretch runs in between are
not credited to it; one made only of the commands AGENTS.md mandates (journal note, recall,
skill read) states round-trips and result tokens saved, not work removed. Its
rationale opens with the program-library verdict (adopt / investigate / write).
The response carries `scoring` (these sentences) and `coverage` (what each
source saw: windows' attributed vs observed points, tool-call outcome coverage,
the opportunity corpus and library, the intervention gate: "cannot tell, not
none" while it admits nothing).

**Projection.** A target is active when `now − last request ≤ max(15 min, 3 ×
its median request gap)`, judged against now. A finished one contributes what it
would have saved; an active one is projected at its rate `r = points /
max(span, 1 h)` decayed over the median session length τ (4 h when unknown)
across the time R left: `r·τ·(1 − e^(−R/τ))`, with all active targets' future
use capped pro rata at the window's headroom (`100 − used%`).
`recoverable_points` is the part of a lever still to be had before the reset (0
when every part has stopped); alerts name the top quota levers by it
(`AlertActions`).

**Window outcome.** A request that names a window gets `outcome`
(`UsageWindowOutcome`): what its listed quota levers recover together
(`usageattribution.CombinedRecoverable`: a run several levers act on counts
once, at the largest lever's points for it, and a session gives at most its
projected future points, so levers on one quota never double count) and the
forecast with that taken out (`usageforecast.Forecast.WithSaving`): the saving
is spread over the remaining burn in the hour-of-week profile's shape, giving
the used percent at the reset, whether it lasts, else the moved run-out and
how far short of the reset it is, and the cut of the current burn (24 h rate,
else central) needed to last. The Usage page renders these; it computes none.

**Growth model** (R-context for sessions, R-lifetime for sub-agents, one
function `capSaving`). A run's requests split into segments at compactions (a
request under half the size of the one before). In a segment of `n` requests a
prompt grows `g = (last − first)/n` a request; capping lives at `k ≥ 20`
requests saves `g·n(n − k)/2` cached-equivalent tokens and costs `(n/k − 1)`
rebuilds (the measured floor × creation weight plus 3 re-orientation requests
at the run's mean prompt). A run that compacts is priced on its own regrowth
after each compaction, never as a session that stops growing; the `k` netting
the most over all segments is chosen, priced at the run's points per token.

- **R-context:** one action per runner (owner: `<runner> session context`, the
  auto-compact threshold) summing the sessions' best-cap savings, with the share
  of session quota at prompts above 200k (Anthropic's long-context price
  threshold), the compactions already done and the median cap. A still-running
  session that has **never compacted** also gets its own action: restarting now
  makes each of its R expected remaining requests `last − floor` cheaper, less
  one rebuild (both regrow alike). A session that compacts is never told to
  compact.
- **R-lifetime:** the same model over sub-agents, per runner, with the share of
  sub-agent quota after request #k.
- **R-model-mix:** per runner, role and dearer-tier model outside a workload: a
  range from 0 to `weekly points × (1 − tier-prior ratio)`, `unproven`.
- **R-floor:** per runner, model and role started in 3+ runs: what each 10k
  tokens cut from the measured first request saves (every request re-reads it).
- **R-idle:** turns whose tool calls are mostly waits with under 600 output
  tokens a request; the saving keeps one request per idle stretch.
- **R-image:** each viewed image (≈1.3k tokens, estimated) is written once and
  re-read by the later requests of its segment (measured per run); a downscale
  saves half.
- **R-model:** a workload whose cheaper model succeeded within 10 points over
  3+ runs in 14 days (owner: the workload).
- **R-skill:** per skill loaded in 3+ runs, its context tax (result tokens ×
  requests that re-read it before a compaction) and repeated loads. Loads with
  unknown result tokens count no tax; the rationale and the rule's reason say
  "token counts unknown for N loads", so a quiet R-skill is not read as light
  skills.

Every family reports whether it fired and, if not, why (including "only under
the materiality floor").

### Tool friction profiles

Package `toolfriction` profiles every command agents ran in a window (default
the last 7 days; `usage friction --since`), per `command_path`, owned by the
catalog scenario of the CLI (`vrooli` for the control plane, external for
`rg`, `sed`, `python3`…). It is a pure aggregation over invocation facts
(`invocation-fact.v32`: per-command outcome `success | failure | no-match |
refused | unknown`, exit code, duration, `result_tokens`, `output_bytes`,
`residency_requests`, `output_event_id`; Codex rollout `CommandExecution`
results joined to their `exec` call (one per `exec_command`, matched by its
literal command, so commands of one path join their own results); a command's
duration is held by one of its segments, the last that is neither a pipe reader
nor an `echo`/`printf` (`status x | jq` on status, `wait x && echo done` on
wait), else its last, never copied to each, so a window's time sums to the time
its commands ran; the same segment holds the command's output, so a window
counts each output once: `result_tokens` are estimated from the text the model
was handed (a result's output, else its text blocks: Codex exec returns its
output only as blocks), and `output_bytes` is what the command printed, which
Codex may have cut first. In a Codex exec batch each command with a nested
result takes the tokens it could have shown (its printed bytes, capped by its
`max_output_tokens`) and the script's own `code` unit the rest of the result;
claims beyond the result scale down to it. The other segments of a command
(`skill read x; plans list --json` prints one inseparable output, held by
`plans list`), and a command with no nested result beside a script unit (a
forked rollout copies its parent's calls without them), have unknown tokens and
no bytes) and the features of each call's and
result's text (error line and signature, notice, truncation, shape guess,
post-filter, jq paths, polled key, bypass owner and method, an example
command). `toolfriction.Extract` reads those once, when the run is projected
(head and tail of each result; a local port is resolved to its scenario then),
and the fact row stores them; a segment of a nested Codex command is read from
its own pipeline of that command's argv, so `status treasury && status
money-ledger` stores two commands and two poll keys (`tool_*` columns, `tool_profiled = 1`, partial
index on `occurred_at`); `UsageAttributionRepository.ToolCalls` is SQL over
those rows and never reopens `run_events`. It reads **only facts at the current
`InvocationFactVersion`**: an older fact's features were made by older rules
(missing, or meaning something else), and scoring them by the current rules
would report what no rule found. Older facts are counted instead: the coverage
line of `usage friction`, `measures get tool-friction` and `usage opportunities`
ends "N runs (M calls) awaiting re-derivation; findings exclude them"
(`stale_runs`, `stale_calls`) until the projection sweep re-derives them.
`result_tokens` carry their basis (`measured | estimated | unknown`): a result
whose stored text is empty although the command printed bytes (a nested Codex
command whose output was kept only as a size) is estimated from those bytes (an
upper bound where the harness cut it); a result never paired, or a skill load
that stored neither text nor size, is **unknown**, never 0 tokens. Coverage
counts the commands whose result no segment sizes ("result tokens unknown for N
commands", `unknown_token_commands`); the other segments of a compound command
whose output sits on its holder segment are not unknown. `code` facts (Codex JavaScript) are not
commands and are left out. A command under a manifest group keeps its next
word as the `command_path` (`test-genie runs wait`, `runs start`) even when the
manifest does not list it.

| Profile field | Rule |
|---|---|
| failure rate | failed ÷ calls that succeeded or failed; no-match, refusals, a verify command's verdict and interrupts (exit ≥ 128) are not failures. A command declares `semantics.kind: verify` in its manifest, on a command or on a durable-run `exception` (`test-genie runs wait`, `runs wait-all`, `execute`); the control plane's are read from `cli/manifest.json` (`vrooli scenario test`, `vrooli package test`). Its non-zero exit is a verdict ("suite execution completed with failures") unless it never reached one: a wrong-shape call, or an error line naming a tool error (unavailable, saturated, refused, denied, not found, timed out…) that is not a line of the command's report (a test mark `✓`/`✗`/`×`, `PASS`/`FAIL`/`ok`, `--- `, a listed `- ` or `* ` item: "✓ … preserves unavailable paging reasons" is a passing test, not an unavailable tool); only then is its error line stored. The read applies the same rule to stored error lines, so facts projected before it neither fail nor cluster on a report line. Verdicts are in neither the failures nor the failure-rate denominator |
| error signature | the first line of a failure's text a CLI prints as its error (`Error: …`), else the first with an error word, else a JSON `error`/`reason`/`detail`/`message` value; ids, numbers, paths, quoted values and times normalised away; a three-part error code (`identity_refresh_unavailable`) is the signature whatever sentence wraps it. A failure that printed no error line (`exit code <n>`) names nothing but its command, so it clusters per command path, never pooled across commands onto the control plane |
| cluster owner | the Vrooli CLI the line names as its subject (`test-genie admission is saturated`); else the CLI that printed more than half of the cluster's calls (a notice in a compound call is held by the segment whose scenario the line names, else the call's dominant scenario CLI, never `vrooli` when a scenario CLI ran); else the control plane the CLIs share (a line printed by a dozen CLIs through the shared client, none above half, naming no scenario). A line names a scenario by its CLI name or its display name ("the Agent Manager identity broker") |
| degradation | a line a Vrooli CLI prints while working that says it degraded or fell back, or that a named CLI is unavailable (`agent-manager park unavailable … waiting inline instead`), whatever the call's outcome; priced in the **calls affected** and, where durations exist, the **wall-clock** of those calls that are not blocking waits (share kind `wall-clock`, over the window's Vrooli CLI time). A blocking wait (`test-genie runs wait`, `vrooli scenario wait`) lasts as long as its target whatever the degradation, so the time of a park refused inside one is stated as held inline ("3h 10m held inline by those blocking waits (the wait itself, not a saving)") and never priced: parking would free the session, not shorten the wait. A summary that "some providers degraded — see notes below" names them from the lines below (`▸ agent-manager.runs — degraded: …` → `[degraded: agent-manager.runs]`), so each degraded provider is its own finding. A summary whose output names no degraded provider promised notes it did not print: the finding says so plainly ("printed \"… some providers degraded — see notes below\" 745 times in 77 runs, but the output read never printed the notes naming the degraded providers: print them (provider and reason) or drop the promise"); a live `search-hub query` on 2026-10-30 printed the promise and no notes |
| root cause | failure and notice lines that name the same failing dependency (the word before `broker` or `refresh`: `identity_refresh_unavailable`, "the Agent Manager identity broker did not answer", park notices citing `identity_refresh_unavailable`) are one cluster listing each line as a sub-signature; it counts affected calls once, takes its owner from the scenario its lines name (most calls), else from its largest line, and is identified by the root, so a line joining it later does not change its fingerprint |
| shape guessing | priced in **wrong-shape calls**, the costly part: a call that failed for its shape is made again. Help lookups are cheap reads, stated beside it and never priced, so a command looked up often but never called wrongly is no finding (`vrooli-memory journal note`: 8 wrong-shape calls and 19 help lookups in 1,098 calls outranked the identity outage while help lookups were priced and breadth counted every run that called it). Wrong-shape failures and help lookups are counted and named apart (`WrongShape`, `HelpLookups`; the sentence gives those help lookups beside every help lookup of the path and its share of calls, each labelled: "4 were help lookups around a failure or retry (of 5 help lookups in all, 50% of its calls)"): a failure that says the shape is wrong (unknown command/subcommand/flag, `did you mean`, missing or too many arguments, a usage dump), and help lookups around a call that went wrong: within five calls of one run, a call of the same program failed before the lookup and the program ran again after it, or the same command path ran before and after it (a retry). A lookup with neither is reading the manual, however often agents do it (`test-genie runs`: 62 of 64 calls were `--help` navigating its subcommand index, with no failed or retried call around them). The unknown command or flag an error names is kept with the CLI's `did you mean` suggestion (`runs` → `run`) so the finding says "add the alias or fix the help" |
| output weight | median, p90 and max result tokens; truncation share (harness markers); context tax = result tokens × requests that re-read them before compaction; share of calls piped through `head`, `tail`, `grep`, `rg`, `jq`, `python`, `sed`, `awk`, `cut` or `wc`; jq paths read after `--json` |
| latency | total duration of calls with a known duration (a chain's time once, on the segment holding it); p90, timed calls and slow excess only over calls whose duration is their own: a segment that holds the time of a chain (`a && b && c`, several command segments sharing one result) is not timed, while a pipe reader shares nothing (`status x \| jq` is status's own time). A `slow` finding needs a query-shaped command (last word `status`, `list`, `get`, `show`, `read`, `state`, `board`, `signals`, `health`, `describe`, `inspect`, `timeline`, `freshness`, `ps`, `ls`, `query`, `search`, `info`, `version`, `locks`: waits, watches, log follows, tests, builds and starts take as long as their work), five or more timed calls, p90 over 5 s, and time beyond 5 s per call at least 0.1% of the window's Vrooli CLI time; priced in that excess milliseconds |
| polling | one read command (last word `status`, `list`, `get`, `show`, `wait`, `state`, `board`, `signals`…) re-querying the **same target**: the key keeps ids, names and paths and drops only volatile flag values (`--after`, `--since`, `--cursor`, `--limit`, `--tail`, `--timeout`…) and timestamps, so `effort signals <effort> --after 57` then `--after 59` is one feed read again while `validate get <run-a>` then `<run-b>` is two targets. A call that blocks until done (its manifest declares it blocking, or a `--wait`/`--timeout` flag) is never polling unless the wait itself is known to have failed or timed out and is asked again on one target; an unknown outcome is not a timeout (`workflow-health validate wait`: 1,449 unknown, 48 success). Run three or more times in one run with every gap under ten minutes; calls inside `sleep`, `watch`, `until`/`while` loops or `time.sleep` are counted as wrapped. A **reshape** is not a poll: a repeat within 30 s of the series' last call that cuts the output another way (filtered or not, other jq paths) re-reads the result just printed, so it neither joins nor breaks the series; a repeat in the same shape stays a poll however soon. Run 0bbbdd88 read `test-genie runs show <id> --json`, then piped the same finished result through two jq filters 4 s and 6 s later. The rule is the data's: of 1,776 polled calls (v31, 7 days) 229 were such fast reshapes, 54 of them `test-genie runs show`'s; a "terminal state in the earlier output" rule was rejected because read commands share no state vocabulary (a generic match flagged 54 `vrooli scenario status` repeats, which poll a start in progress). `runs show` polling fell 123 → 42 calls, `vrooli scenario status` 200 → 189, `agent-manager run get` 40 → 29. A polled command whose owning CLI's manifest declares a **blocking** command that replaces it names that command (`semantics.blocks.replaces`, full command paths, in `cli/manifest.json`, a scenario manifest or its durable/streaming `exceptions`; `runsignal.BlockingSiblings`, in declaration order). A verb is never evidence: `agent-manager run await-result` re-reads a parked result and blocks on nothing, so `run get` is offered `run events --follow` (`blocks.flag: "--follow"`); `vrooli scenario status` → `vrooli scenario wait`, `test-genie runs show` → `runs wait`/`wait-all`/`follow`, `program-runtime programs get` → `programs wait`, `device-control device state` → `device watch`. A listing or a read of something else is not replaced by a wait on one target (`run list`, `device list`, `vrooli scenario logs` name none; `vrooli scenario logs --follow` streams until interrupted, and needs `--force-follow` in an agent's shell, so it is no bounded wait). A mutation that can wait (`workflows execute --wait`) starts new work and declares no `blocks`. Whether a call itself blocks (so is not a poll) reads the same declaration (`runsignal.Blocks`), besides `--wait`/`--timeout`. A **precondition** is structured, never parsed from prose: `blocks.after` lists the command paths that must have acted on the target first (`vrooli scenario wait`: `vrooli scenario start`/`restart`, `make start`/`restart`; with none in flight it returns at once). Such a sibling is claimed only for the polls that followed one of them on the same target (the first non-flag word after the command path; `make -C scenarios/x` reads `x`) in the same run, inside the poll stretch or within ten minutes before it; a start that names no target (`make restart` in a scenario directory) cannot be shown to be the same target and is counted apart, not claimed. The rest are polling with no blocking sibling ("`vrooli scenario wait` blocks only after … of the same target, which 47 of them followed (23 more followed one that names no target, which is not claimed); 87 polled calls have no blocking sibling"). The manifests are parsed once into one index (paths, semantics, siblings) that is rebuilt when a manifest is added, removed or changed (stat'ed at most every 5 s), so a fact's `manifest-index:<hash>` names the manifests it was resolved against |
| bypass | `sqlite3` on a scenario's database (`scenarios/<s>/data/…`, `data/<s>.db`), an HTTP client to a **non-health route of the API port** (`port_key` `API_PORT` in the control plane's process registry; UI ports and `/health`, `/ready`… are smoke checks, not bypass), or a read under `~/.vrooli/{state,data}/vrooli/<s>`, by command or read tool. A run **working on** the scenario it reads (developing or reviewing it) does not bypass it (`Call.OwnWork`, `toolfriction.WorksOn`): of the run's tool calls whose first 2,000 characters name any scenario tree (`scenarios/<name>/`: commands, file paths, working directories), at least half name that scenario's, outside its data directory. Read from the run's `tool_call` events, for the runs that reached a scenario's internals only. On 2026-10-23..30 38 of the 44 runs that ran sqlite3 on agent-manager's database were its own development and review (59–97% of their scenario-naming calls name agent-manager's tree); runs reaching test-genie's port or workflow-health's state name those trees in 0–24%. Each bypass finding and opportunity states what it set aside ("36 sqlite3 on its database in 6 runs (285 more in 38 runs working on agent-manager itself are not counted…)") and the coverage line counts them; bypass shares are over the window's remaining bypass calls |

Findings (`toolfriction.Finding`) are owner-level problems: a failure or
degradation cluster with three or more calls, a profile with three or more
shape guesses, heavy output (p90 ≥ 8k or max ≥ 40k result tokens, a printed output of ≥ 40k tokens that the harness cut before the model read it, or ≥ 1% of the window's context tax and ≥ 5M re-read tokens; the description gives both what was printed and what the model read), nine or more
polled calls, a slow query, and bypass calls of one scenario and method in
three or more runs (`toolfriction.MinPatternRuns`, the opportunities' rule
too: one agent's habit is not a pattern). Each is priced in the unit of its harm (calls, command time, or
context tokens) and states a **share per kind of harm** over that kind's base
(`shares`; one `Totals.Base(kind)` names every base, so the friction list, the
recommendations, the bug report, the CLI and the UI print the same words):
failures over the window's Vrooli CLI failed calls, degradations over its
Vrooli CLI calls, wrong-shape calls over its Vrooli CLI wrong-shape calls, polls over
its polled Vrooli CLI calls, slow excess over its Vrooli CLI time, output
weight over its context tax (every command's), bypasses over its bypass calls.
A failed `grep` or a slow build is nobody's to fix here and does not dilute a
Vrooli CLI's share. A root-cause roll-up holds one share per kind (identity:
"failure 91 of 95 calls (each run at most 10x the median run's) = 13.6% of the
window's 668 Vrooli CLI failed calls; degraded 86 calls = 0.6% of the window's
15430 Vrooli CLI calls"), never both kinds' calls over one base. Every count in
it is of **calls**: each line lists the calls that printed it ("failure
"identity_refresh_unavailable" in 95 calls"), a line printed twice by one call
counts once, a call that failed and warned is under both lines, and a share the
per-run cap cut says how many of how many calls it counts. Findings rank by `score = √(cost_share × breadth)`
(`cost_share` is the largest share; breadth the share of the window's runs
that show its harm, `HarmRuns`: the runs that guessed a command's shape or
polled it, not every run that called it), the score the recommendations use; the friction list shows
those at or over the 1% floor and counts the rest (`below_floor`) and those
the limit cut (`beyond_limit`): "Listed 20; 466 more under the materiality
floor (score 1%), 0 more past --limit".
**Concentration:** a finding reports `TopRunShare`, the largest single run's
share of its uncapped cost, and counts each run's contribution at most 10x the
median of the runs that contributed (60 bypass calls in one loop beside three
runs of 2 count 20, not 60; runs that hold the command but none of its harm do
not pull the median to zero): one looping run is not a pattern. A finding of one run keeps its cost and
says 100%. When its cost counts less than its sentence names, in any unit, it
says which is which ("its cost counts 607 of those 948 calls, each run at most
10x the median run's"; "its cost counts 142.7M of those 240.0M tokens, …"). The same cap (`toolfriction.Capped`) prices every
opportunity's calls saved. A command's finding says how many more of its calls
the window holds at an older fact version ("271 more calls of it await
re-derivation at the current fact version and are not counted"): a count over
every version (1,342 `journal note` calls) differs from the finding's (1,071)
by exactly those. A profile ranks by its largest share of failed calls, command time
or context tax. The fingerprint is kind and normalised signature (the command
path for findings about one command, the scenario and method for a bypass, the
root cause for a rolled-up cluster), never the evidence and never the owner:
the owner is a majority vote that can flip between weeks, and a flip must not
reset a reported or dismissed status or file a second bug. A second command
path reporting the same failure does not reopen it either. (Fingerprints
changed with this rule on 2026-10-30; `friction_problem_status` held no rows
then, so nothing was re-keyed.) Every list
prints its coverage ("N commands, outcome known for X%"); an empty list is
read beside it, never as "no friction".

`ToolFriction` (`measures get tool-friction | tool-profiles | error-signatures`,
runtime binding `agent-manager/measures/tool-friction`) serves the window's
rates, profiles, clusters and findings from one computation; `owner` narrows
them to one scenario (`agent-manager.friction-digest` reads it per scenario).

### Friction

`UsageFriction` (`agent-manager usage friction`) lists the tool findings of
its window, each with its owner, a plain sentence, counts, cost, evidence runs
and redacted example commands, and a durable status: `open`, `reported` (with
the scenario-qa bug id) or `dismissed`, in `friction_problem_status`; then the
operator interventions (below), whose section is always present.
`usage friction-report <fingerprint>` files a tool finding through the report-bug
writer path (`PublishScenarioQABug`, idempotency key
`agent-manager-friction/<fingerprint>`, honesty flags `speculative-cause` and
`repro-not-attempted`) and records the bug id. It files once per fingerprint:
an already reported finding returns its bug, concurrent reports of one
fingerprint are serialized, and the inbox returns the existing bug for the
key, so a retry after the status write failed records the bug filed before.
`usage friction-dismiss <fingerprint>` sets it aside. The two actions are
`FrictionService.ReportFriction` and `DismissFriction`; each request carries
`since`, the start of the window the finding was listed in (the CLI's
`--since`, the page's `window_start`; empty is the default week), and the
fingerprint is looked up there in one read. Errors: an unknown fingerprint is
`NotFound`, an unreachable bug inbox or status store `Unavailable`, a
malformed `since` `InvalidArgument`, anything else `Internal`. Every tool window (`usage friction`, `usage opportunities`, the
`ToolFriction` RPC and its measures) starts at most 90 days back
(`MaxLookback`; an older `--since` or custom range is clamped), ends by now,
and a start at or after its end is refused. One load of a window serves every
reader for a minute (`WithToolCallMemo`): its bounds are floored to the minute,
so the friction list, the recommendations and the opportunities a page refetches
together, and a report or dismiss right after, share one read; concurrent reads
wait for the load in flight, which runs apart from its callers (a caller that
gives up stops waiting without cancelling it; the load has its own 2-minute
timeout) and is not kept when it fails or panics. The memo keeps at most
200,000 calls (about one week), drops an entry a minute after it loaded, and a
new slot's load drops the earlier slots of its span. Windows longer than eight
days (the 30-day recall windows) are not memoized: one alone outgrows the
bound, so each read loads its own. Supervision, the will-exhaust alert, the
friction board and the measures share one store per database, so one memo.

Friction episodes were deleted on 2026-10-30. The last pattern,
`handoff-continuation`, was informational (stored, never priced or shown) and
no pattern had passed the precision gate since the tool-friction profiles
replaced the command patterns, so the derivation, its precision corpus, the
episode table (dropped by a run-once migration), the per-run episode RPC and
CLI verb, the episode cohort and comparison measures, and the episode half of
cohort selection went with it.
Tool and command friction is read from the profiles above; operator corrections
are the classified interventions below. Self-report spans (assistant phrases such
as "blocked"; classifier `self-report.v4`) stay in the run report
(`selfReportSpans`) and the conformance corpus. Fact classifier history that
still applies: facts v27 stores the tool-friction features on command facts at
projection; the next word under a manifest group stays in `command_path`.

### Parser version and silence

`transcript_files.parser_version` records the version of the events a transcript
parser derived (`runner.TranscriptVersioner`; the Codex codec is at 3, the Claude
codec declares none). `ToolFriction` coverage counts the commands of runs whose
file records an older version than the harness's current one apart from the
rest: "N commands from runs read by an older parser; outcomes there may be
missing" (`older_parser_commands`); runs with no transcript record are counted
on their own ("N commands from runs with no transcript record, whose parser is
unknown", `no_transcript_commands`). A low known-outcome
share over those runs is a parser gap, not evidence that commands succeeded
(SPEC AM-12a). Nothing is re-derived: interactive Codex transcripts are kept
seven days, so the older evidence leaves the window by itself.

### Operator interventions

`api/internal/interventions` classifies every operator message that followed
agent work (the run's opening message is `initial`, not an intervention) with
ai-gateway's decision capability (`DecideBatch`, role `decide.accurate`: Winnow,
hosted Jev and Clef when available, then local Laya, then the `classify.fast`
LLM enum fallback). One choice question over eight categories (`early_stop`,
`false_inability`, `wrong_claim`, `quality_miss`, `too_dense`,
`instruction_ignored` are friction; `scope_change`, `approval_or_info` are not);
the state is `{preceding_agent_text, operator_message}`, bounded to the
message's first 2,000 and the agent text's last 1,000 characters (Winnow reads the
corpus unclipped; retried at 60% when a decision model could not fit it). Each category names its owner
(`Category.Owner`).

`operator_interventions` holds one row per message event and
`ClassifierVersion` (a digest of the question set; other versions are pruned at
start): category, per-category probabilities, method, provider, model, the
agent work since the previous operator message (`work_requests`,
`work_tool_calls`, `work_tokens`), and `repeat_of_event_id`. A background job
walks the last 30 days in six-hour slices, then follows new messages and
re-walks hourly. It runs every minute while `ProvenGate` admits nothing, on
purpose: it collects the classifications the gate is measured on and that a
category shows once the operator admits it. Each pass is capped: it scans at
most `DefaultPassMessages` (128) user-role messages (a page never takes a pass
past it), recorded ones included, so at most that many go to the classifier
and an hourly re-walk is spread over passes; a recorded message's payload is
never read. No run starts after `DefaultPassBudget` (45 s), so a pass overruns
by at most the classifier call in flight, and a pass stopped inside a page
resumes at the first message it left. A run's events are read once per pass,
only up to its last message in that pass. If ai-gateway or its models do
not answer, the message is `unclassified` with the reason and retried after an
hour per attempt (at most a day): coverage drops, nothing reads as "no
friction". A forked Codex rollout's replay of a recorded message (same
timestamp and text) is `duplicate`.

A **repeated report** is a `quality_miss` that shares distinctive words (four
letters or more, generic words removed, at least a quarter of the shorter
message's) with a `quality_miss` among the previous three operator messages of
the run, with agent tool calls in between. It is deterministic over stored
categories because the decision state holds one message; it inherits the
`quality_miss` gate and has no labelled recall yet.

A category is shown only for a model in `interventions.ProvenGate`: precision
≥ 0.8 and recall ≥ 0.7 on the labelled hold-out and on the untouched fresh random
validation set, which must hold ≥ 10 gold rows of that category (P4-A4 amendment 2).
A pooled "operator friction (category unproven)" bucket (`PooledFriction`,
`Gate.ShownPooled`) is shown under the same bars for friction probability mass.
`ProvenGate` is empty: Winnow-E4B, tuned on the train split, shows no category
(only quality_miss has enough validation rows and it fails precision), and pooled
friction passes the hold-out (P 0.92, R 0.75) but misses validation precision
(P 0.73, R 0.72), so no category or pooled bucket is shown (`docs/efforts/usage-intelligence/p4-w4-interventions.md`, "Classifier
benchmark"). `usage friction` carries an "Operator interventions" section
(`UsageFrictionResponse.interventions`, rendered by the CLI and the Usage ›
Friction tab): per admitted category the count, owner (`Category.Owner`),
interventions per agent-hour, the summed agent work before them, repeat
reports, and example messages (redacted, 240 characters) linking to their run;
a pooled bucket renders as "operator friction (category unproven)" with each
per-category hint labelled unproven. When nothing is admitted it prints the
number of operator messages, the classified share and why ("no category has
passed the accuracy gate yet"), never "no friction". `measures get usage.interventions` reports, per friction category,
what was classified and, only for proven categories, interventions per
agent-hour (agent-hours: ten-minute run slots with a provider request in
`transcript_usage`, divided by six), repeats and the median agent tokens before
the correction; a final row gives coverage. Re-measure with
`AGENT_MANAGER_LIVE_DECIDE=1 go test ./internal/interventions -run TestLiveInterventionBenchmark -v`;
it fails until `ProvenGate` matches the measurement.

Codex message sources use structure before text
(`domain.ClassifyCodexUserMessage`): user-role text in a sub-agent thread
(`session_meta` `thread_source=subagent` or a `source.subagent` spawn) is
`teammate`; an item whose content provenance names no `user` item, or whose
content item kinds name no `user.text`/`user.image`, is injected context.
`<send_user_message_question_reply>` is the operator's answer to an agent
question (operator).

### Opportunities

`api/internal/opportunities` mines the stored tool calls (the same
`ToolCalls` read as the tool profiles, no event payloads) for what agents
repeatedly do by hand, as records for the owner who can act. It is served as the
registry-only measure `usage.opportunities` (`measures/opportunities.go`; no
proto) and `agent-manager usage opportunities [--since RFC3339] [--json]`, which
groups by owner and ranks by calls saved. Four kinds, each needing three
distinct runs:

- **sequence**: per run, the ordered Vrooli `command_path`s (consecutive
  duplicates collapsed; tools outside Vrooli, help lookups and bare
  executables such as `prompt-manager` with no subcommand are not steps), 3- and 4-grams,
  arguments dropped, kept when seen in 3+ runs. Grams whose occurrences share
  two or more steps of a run (shifted or rotated views of one behaviour) merge
  into one maximal motif. Merged grams need not occur together, so a stretch of
  a run the grams cover need not hold every step: the subject is the stretch
  shape (a repeated loop reduced to one period, steps in the order observed)
  that the most runs run in full, and only the places a stretch runs the
  subject in full are occurrences, evidence runs and savings; a step belongs to
  one motif. Each motif is
  matched against the program library read from program-runtime
  (`programlibrary`: `ListLibrary`, contracts with their called binding ids). A
  program that calls the commands of a strict majority of the motif's steps
  (*binding* basis) is `adopt <program>`, with program runs (calls of
  `program-runtime library run <program>`; a help, `library get` or skill read
  naming it is not a run) against manual occurrences. A program that calls half or fewer, or is linked only by
  *affinity* (its scenario owns a step and names a rare word of the commands
  of at least two steps: one shared word, "list" in "a request list", links a
  program to any listing step), is
  `investigate` (lower confidence), never "adopt". No match is `candidate`; an
  unreadable library is `unknown`. A `vrooli-memory journal note` closes a
  slice of work: it may end a motif, but the next task's steps never join it.
  A program can only pick a later step whose argument came from an earlier
  step's output, so before the library verdict the motif's result dependency is
  measured (`opportunities/dependency.go`): per later step, the argument words
  that differ between occurrences (a word the same in every occurrence is fixed
  in the program) are looked up as whole identifiers in the output text of the
  earlier steps of that occurrence (facts carry `output_event_id`; text read
  through `ToolOutputs`, only for an even sample of at most
  `MaxDependencySamples` = 30 occurrences; it selects the result's error,
  output and block text, never the raw JSON copies, cut at 30,000 characters,
  Claude Code's inline output limit). Output text that could not be read
  leaves the motif `unmeasured` with the read error as its reason. Words the agent already typed in an
  earlier step of the occurrence are not counted (an output echoing them is not
  their source), and the output searched is its results: document bodies
  (CDATA, each to its own `]]>`; one a cut output left open runs to the end)
  and backticked suggestions (backticks paired within a line; a stray one hides
  nothing) are removed. Other prose around a result
  still counts, so the rate is an upper bound on "picked from the top hits".
  The verdict uses the 95% Wilson interval of the rate over the sampled steps
  against `MinDependency` = 50%: a lower bound at or above it keeps the library
  verdict (candidate, adopt, investigate); an upper bound below it is
  `independent` (agents run the steps together but choose each later step
  themselves, so a program cannot pick it); an interval spanning it is
  `undetermined`, stating the rate and sample size. `independent` and
  `undetermined` are not candidates and claim no calls or tokens saved. With varying arguments but no
  readable output text the verdict is `unmeasured`, also with no savings. A
  motif whose steps never vary needs no output text. The rows carry
  `dependency_measured` and `dependency_used`; the recommendation list omits
  `independent` and `unmeasured` sequences. Owner: the adopted program's scenario, or
  prompt-manager for a skill ritual, else the scenario with most steps.
- **bypass**: `toolfriction` bypass features grouped by owner and method, with
  the tables (sqlite3), URL paths (ids folded) or state directories reached,
  from the stored redacted command: "add a CLI command or RPC for these".
- **output-shape**: per command path, the jq paths agents extract after
  `--json`: "add `--fields` or a focused flag". Only jq is recorded; python
  one-liner field access is not.
- **skill-reload**: `prompt-manager skill read <name>` again within one run and
  one compaction segment (`residency_segment`); a reload after a compaction is
  not counted. A read whose output went through a filter (`skill read x | rg
  y`) never put the skill in context: it is neither a load nor a reload, here
  and in R-skill (`Call.LoadsSkill`; `research-coordination` fell from 54 to 20
  reloads). Extra loads and their own result tokens, owned by prompt-manager
  (a reload whose output sits on another segment of its command, fact v32,
  counts with its tokens stated unknown: "tokens unknown for 3 of them").

Every opportunity names as evidence the runs that showed it most (first seen
on a tie), not the first runs read. The coverage counts command calls only
(file-read tool calls are read for bypasses), the count the friction coverage
states for the same window.
  Segments are real for a call at invocation fact version 30 or later whose
  run's transcript the current parser read (`Call.SegmentsKnown`: Claude
  transcripts get compaction events only after re-import at parser 1, Codex at
  2). In a run read by an older parser or with no transcript record every load
  sits in segment 0, so its repeats are counted apart as "residency unknown"
  with nothing saved, and R-skill counts them without "still in context". The
  sentence counts sessions per claim: "read again 2 times while still in
  context in 2 sessions …; 3 repeats in 3 sessions whose facts record no
  compactions (residency unknown …)".
  A compaction event's `tokensAfter` is 0 (unknown): neither transcript states
  the size after it; the next request's usage event carries that.

Every kind counts its number on **one basis**, named for what the change does
(`opportunities.savedBy`, the row's `saved_label` and `saved_basis`, printed by
the CLI after the number and in the recommendation's rationale; the field is
`calls_saved` for all): a sequence has **calls saved**, each full occurrence's
steps less the one program call that would run them; a bypass **calls
replaced**, its direct reads, each by one supported command or RPC call; an
output shape **extractions removed**, jq pipelines a focused flag makes
unnecessary, one per call (the command call stays); a skill reload **reads
saved**, reads of a skill still in context in runs whose compactions are known.
Every kind counts them by the one per-run cap (`toolfriction.Capped`, with
`top_run_share`), and when the cap cut the count the basis says how many of how
many ("67 of 82 counted, each session at most 10x the median session's"), so
occurrences and the number never differ unexplained. The CLI groups rows by
owner in the measure's rank order and adds up nothing the measure did not.

Every row carries evidence run ids, redacted example commands, and the check
that verifies it (the same view over a window after the change). The coverage
row states calls, Vrooli calls, runs, and library size, or that the library was
unavailable; empty lists print it. Token figures are upper bounds.

## Windowed totals and run scope (Details)

Every token total over a time window (`RunCost`, the runner/model/profile/
workload breakdowns, the per-run distribution, `WorkloadEfficiency`) sums
`transcript_usage` rows by `observed_at` (request time). A run that started 30
days ago and ended today contributes only its last-7-day requests to a 7-day
total, so Details agrees with the provider's readings and with a plain `SUM`
over the transcript rows. The lifetime counters on `invocation_read_model_runs`
serve per-run views (run detail, cohort rows) only.

Two rules keep the sums honest. A harness session copied into several harness
homes (the same Codex rollout under two web-console homes) is several
transcript files with identical requests; a request equal to one in an earlier
file of the same session counts once. And dollars exist only for per-token
(`metered`) runs: a run's charge is apportioned to the window by the share of
its tokens requested in it. Subscription work reports quota points, not dollars.

Run statistics (success rate, throughput, run time with median and p90) count
only runs Agent Manager executed (`codec_pipe`) by default. The filter
`InvocationFilter.scope` selects `agent_manager`, `operator_sessions` (imported,
attached and interactive sessions) or `all`; tokens count every request unless a
scope is chosen. An imported or attached session with no usage, and a session
copy, are never counted as runs.

## Surfaces

- **Usage page** replaces `/stats` as the primary analytics entry. It shows the
  forecast per window, the consumers, the recommendations and coverage; the
  former Stats panels move under Details, and manual import folds into
  Coverage. Which window is which is the API's to say (`usageforecast.Mark`):
  each forecast window carries `dormant` (its newest reading predates its
  period: the reset passed or the period began after it; no reading at all is
  not dormant), `primary` (the longest window of its provider pool,
  `usageforecast.Longest`, the same rule the recommendations use, so a
  provider's second pool keeps its own) and `focus` (the most urgent window
  with a reading in its period: will run out soonest, then may run out
  soonest, then most used). The page and the dashboard strip collapse dormant
  windows to a line, the focus band shows the requested window or the focus,
  and the quota tile lists the live primary windows; none re-derives them.
- **Dashboard strip:** one line per provider window with a bar, a status chip
  and the projected exhaustion; it opens the Usage page.
- **Run usage panel:** the run's tokens from the one accounting function,
  estimated quota points (requests, not turns) and the actions whose evidence or drill-down names the run.
- **Supervision:** a watched run's `friction_score` is `toolfriction.ScoreRun`
  over its command calls projected at the current fact version (the share that
  failed, guessed a command shape or polled, each call once; `friction_commands`
  is the denominator). One source and one rule: `RunToolFriction` reads only
  facts at the current fact version and says how far that projection reaches
  (`RunScore.Through`, the watermark at that version), so a run projected only
  at an older version has nothing to score and is unavailable. The program
  input states "9 of 10 commands had friction (a command counts once): 8
  failed, 0 guessed a shape, 3 polled". A run is scored only when it has commands and at least
  80% of them have a known outcome (`RunScore.Scorable`); otherwise its friction
  is unavailable (cannot tell), never quiet, and it neither signals nor enters
  the program's friction list: both evaluators apply one rule
  (`SubjectSummary.FrictionReadable`), the `supervision-evaluate` program also
  drops rows whose run summary is `friction_unavailable`. Both apply one
  precedence: deadline, event count, quiet time, then friction (the program
  calls it `stalled`); unavailable friction only removes that run from the
  friction trigger, so the other triggers still fire, and with none firing the
  decision is quiet. A parity table feeds the same subjects to both across
  every trigger. It feeds the watch trigger and the `supervision-evaluate`
  program input `run_tool_friction` (formerly the unfed `friction_episodes`).
- **Alerts:** `usageforecast.Alerter` evaluates the current windows every 5
  minutes and publishes `agent_manager.usage.notification.v1` through Vrooli
  Events. Categories are declared in `.vrooli/notifications.json`, which also
  sets `appDelivery` (Settings › Notifications renders the shared
  `AppNotificationSettings`; `main.tsx` calls `initAppNotifications()`):

  | Category | Default | When |
  |---|---|---|
  | `usage-will-exhaust` | on | A window becomes `will_exhaust`; sent again only if the projection moves earlier by more than 12 hours. The body names the window, the projected date, and the window's #1 quota lever with its saving, ranked from the Usage page's store and tool calls. |
  | `usage-back-on-track` | on | A window with a standing will-exhaust warning has been `on_track` or `idle` for 30 minutes, or has reset (at once). Sent once per warning (its dedupe key names the warning's time); it says the earlier warning no longer applies. After a withdrawal the window is not warned again in the same period unless the projection moves 12 h earlier than the warned one. |
  | `usage-at-risk` | off | A window becomes `at_risk`; once per window period. |
  | `usage-digest` | off | Once per day from 08:00 local (a restart later than 12:00 skips that day; the zone is `AGENT_MANAGER_TIMEZONE`, then `TZ`, then the host's `literal:/etc/localtime`): up to 3 live windows, most urgent status first, each with its top action and what it could still save before the reset. |

  Every window alert's `context_url` is `/usage?window=<provider>/<pool>/<window>`.
  Dedupe state lives in `usage_forecast_alerts`, saved only after a publish
  succeeds, so a failed publish is retried: will-exhaust rows are keyed by the
  window key and deleted when the warning is withdrawn; at-risk and digest rows
  use the reserved key prefixes `at-risk:` and `digest:`, and the withdrawal hysteresis uses `safe:` and `withdrawn:`; reserved rows older than 30 days are pruned each evaluation. The #1 action comes
  from the recommendations service through one adapter
  (`usageattribution.AlertActions`); an alert is still sent without one. When
  Vrooli Events is not resolved, no alert is sent.
- **CLI:** `agent-manager usage forecast|burners|recommendations|friction|friction-report|friction-dismiss|coverage|compare|reindex`
  with `--json`, reading the same Measures procedures as the UI
  (`UsageForecast`, `UsageForecastBacktest`, `UsageBurners`,
  `UsageRecommendations`, `UsageFriction`, `UsageCoverage`, `UsageCompare`). Agent identity
  can read them. Window titles match the UI: "Claude · 5-hour", "Codex ·
  weekly", "Claude · weekly · Opus" (slot-only pools `primary`, `secondary`,
  `five_hour`, `seven_day` are dropped); titles that collide among the
  windows shown append the pool id ("Codex · weekly · primary" / "Codex ·
  weekly · secondary"). `burners --by <dimension> [--since] [--run]`,
  `recommendations [--run] [--pool P --window W]`, `friction [--status]`, `coverage [--since]` and `compare
  --left-*/--right-*` (the `measures` cohort selector: cohort, run ids,
  filter, window). `compare` reports attributed points per successful run on
  each side and is `insufficient` below five successful runs per side.
  A run's weighted non-cache tokens (and its direct children's) or its redacted
  invocation facts are `run report <id> [--children] [--weights] [--facts]`;
  `usage reindex [<run-id>] [filters] [--full]` rebuilds durable invocation
  evidence (refresh by default). They replace `usage run`, `run tokens`, `run
  invocation-facts` and `run replay-invocation-facts | refresh-invocation-facts
  | replay-invocation-corpus`. The same usage procedures are also served by
  `measures get usage.<name>` through the measures-go registry, from the same
  compute path; `usage burners` is the one command for quota consumers (token
  footprint by capability is `measures get token-attribution`, a different
  question). The command and measure reference is
  [operational-stats.md](../reference/operational-stats.md).
- The REST statistics layer (`/api/v1/stats/summary` and its drill-down routes)
  is removed; Measures is the only analytics contract.

The UI is built inside Agent Manager's existing shell and tokens. On phone
widths the bottom navigation holds a short, fixed set of destinations (the
limit is stated in requirement REQ-P0-036), with the rest behind a More menu.

## Acceptance evidence

Requirements in module 26 that cite this document as a manual validation are
proven by live evidence recorded with the effort: coverage against an
independent disk scan (A1), one deduplicated will-exhaust notification record
(A7), and the before/after value comparison (A11). When the effort closes, that
evidence moves into this section.

- **A1 (2026-10-29).** `TestLiveCoverageAgreesWithAnIndependentDiskScan`
  (`USAGE_COVERAGE_LIVE=1`) compares the coverage report from the live
  database with a scan that shares no indexer code: every root's file count
  and bytes match, captured bytes agree within 0.00 points, and 99.97% of the
  22,873 trailing-7-day turns are captured (Claude home 20,238 of 20,244).
- **A5/A6 (2026-10-29, before the API restart).**
  `TestLiveUsageAttributionDryRun` (`USAGE_ATTRIBUTION_LIVE=1`) copies the
  live facts into a temporary database, applies the usage-row repair and
  attributes the current windows: every group-by sums to the observed
  increase (Codex weekly 34 = 16.6 attributed + 17.4 unattributed, the
  unattributed part being pre-2026-10-29 Agent Manager runs whose legacy
  events are excluded); all six rule families fire; the top recommendation
  targets the dominant consumer.
- **D-16 (2026-10-29, before the API restart).** The same dry run then
  indexes the 81 run-owned files (1.16 GB, 34,529 usage rows, no host
  writes) with the production indexer: Codex weekly 34 = 16.58 + 17.42
  unattributed becomes 30.99 + 3.01; Claude windows are unchanged (23 = 21.28
  + 1.72, 27 = 27.00 + 0.00).
