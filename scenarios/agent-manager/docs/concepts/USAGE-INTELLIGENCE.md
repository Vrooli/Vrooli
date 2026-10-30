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
  sidechains are child runs (`parentRunId`) of their parent file's run. The run
  is `running` while the file grew in the last 10 minutes, then `complete`.
  A session the retired snapshot importer adopted is relinked to its source
  file, its copy is deleted and it is re-read from zero, so the projection
  sweep replaces its stale read model.
- **Durable summaries.** `transcript_usage` keeps one row per provider usage
  report: raw cumulative counter, the report's own amount, model, session,
  message id, and the non-text work the request consumed since the previous
  report (images in with user/high-detail/bytes, images generated, web
  searches and fetches, audio, tool calls by kind including `mcp:<server>.<tool>`
  and `scenario:<name>`, the Codex credits balance). Quota frames go to
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
  committed before an error, the indexer preserves the prior cursor and
  replays the source from its start under the tick byte budget to reconstruct
  parser state; event and usage identities prevent duplicate evidence. A
  source changed since the failed read, a source larger than the tick budget, or a permanent
  parse failure leaves an `unreadable` issue instead of guessing. Inspect the
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
observed increase. Usage after the newest reading is reported as pending.
Work metered outside the token counters (Codex credits, web searches, image
generation, `scenario:<name>` calls) gets its own line and is never added to
the points. Points are estimates and are presented as approximate.

### T-6 Recommendations

Implemented in `api/internal/usageattribution` (`Recommend`) and served by
`UsageRecommendations` (`agent-manager usage recommendations`). The rules
read each provider's longest current window, or one window alone when the
request names it (`provider`, `pool`, `window`; savings are then stated over
that window's length and R-model, which states weekly outcomes, does not fire
for a window shorter than a week).

**Levers and plans.** A rule's finding is a *lever* about one target (a
session, a workload, or a friction problem). Levers for the same target in
the same window fold into one *plan*, because they reduce the same use: the
largest lever applies to the target's projected use first and each next lever
to what remains, so `combined = projected × (1 − Π(1 − saving / projected))`
and each lever's `marginal_points` (what it adds after the larger ones) sum to
the combined saving. The combined saving is below the sum of the levers and
never exceeds the target's projected use. A plan carries `used_so_far_points`,
`projected_points`, `share_of_window_burn`, `still_running` and `basis_hours`.
Plans saving under 0.1 pt are counted (`folded_groups`, `folded_points`), not
listed.

**Projection.** A saving is stated against the target's projected use over the
window's horizon. A target is active when `now − last turn ≤ max(15 min, 3 ×
the session's median request gap)`, judged against now (never another
consumer's newest turn); a finished target projects to its points, so its
saving is what the lever would have saved of its use. An active one adds its
rate `r = points / max(span, 1 h)` decayed over the median session length τ (4 h
when unknown) across the time R left in the window: `r·τ·(1 − e^(−R/τ))`. The
future use of all active targets together is capped pro rata at the window's
headroom (`100 − used%`), and each lever at its own base.
Each lever carries typed facts for its rule (`UsageLeverFacts`: context,
lifetime (the `fanout` field), idle, image, friction, model or floor (the `context` field)), never a free-form metric map. Every
rule family reports whether it fired and, if not, why from the data.
A saving is stated over the window's whole horizon at the target's current
rate (`at_this_rate_points` is a projection, not a forecast); `recoverable_points`
(on each plan and lever) is the part still to be had before the reset: the
saving times the future share of the projection (`(projected − used) /
projected`, the saving rate over the expected remaining activity), and 0 for a
target that has stopped. Friction plans are a cross-cut view (`cross_cut`): their points are
already inside the session plans that ran them, so clients never sum savings
or shares across target kinds. `usageattribution.Service.TopPlan` returns the largest plan for one window; it
is the function an alert asks for "the #1 action".

- **R-model:** over the last 14 days a workload ran on a cheaper model with at
  least 3 runs and a success rate within 10 points of its pricier model. The
  saving is the pricier model's weekly points on that workload times
  `1 − price ratio`.
- **R-model-mix:** sessions and sub-agents outside a workload (interactive
  ones included) on a dearer tier (opus, sol, terra) that has a cheaper tier
  (sonnet, terra, luna), priced per turn's model, with runs counted per model. The saving is a range from 0 to `points × (1 − ratio)`;
  the lever claims none (`saved_points` 0), its confidence is `unproven` and the
  ratio is a tier prior, not fitted.
- **R-context:** a session's own provider requests (usage reports; a Codex
  turn makes many) average at least 120k prompt tokens over at least 50
  requests. The target is the measured floor (the median first-request prompt
  of the runner, model and role over the window). The saving is its
  cached-input share of points times `1 − floor / average`, less the cost of
  rebuilding the floor.
- **R-lifetime:** a sub-agent with at least 40 requests whose prompt grew from
  its own first request `S` to `L` over `n` requests (`g = (L − S)/n`; the
  measured floor `F` only prices the rebuild, so a worker that starts above `F`
  is not credited for the excess). Capping its life at
  `k ≥ 20` requests saves `g·n(n − k)/2` tokens and costs `(n/k − 1)` rebuilds
  (`F × creation weight` plus 3 orientation requests at the mean prompt); the
  `k` with the largest net is chosen and the lever fires only when it is
  positive (at least 5% of the worker's tokens). It replaces R-fanout: fan-out
  would fire only for duplicated workers (same brief fingerprint), which the
  facts do not carry, so it is gone.
- **R-floor:** at least 50 requests and at least 20% of the session's tokens
  are the floor, in the same weighted units as the session's tokens (the first
  request of a run writes the floor to cache, the rest re-read it). Informational (low confidence, no saving claimed): the
  trimmable size is unknown.
- **R-friction:** a persisted friction fingerprint recurs at least 3 times in
  runs that consumed the window (a *friction problem*, below; dismissed ones
  are listed but not recommended). Episodes are priced as their turns' share
  of the run's turns times the run's points, capped at the run's points;
  half is assumed avoidable. Only episodes of the current episode classifier
  are read; a classifier change re-projects history through the reconcile
  sweep. A `handoff-continuation` episode counts only the calls after a
  terminal answer and before the next user message (informational and never priced from v14),
  so work the operator asked for is never friction.
- **R-idle:** at least 20 provider requests whose tool calls are mostly waits
  (spawning a sub-agent is work, not a wait) with under 600 output tokens per report, or requests of a parked run; 70% avoidable.
- **R-image:** at least 30 viewed images or 10 high-detail images (R-1). An
  image is ≈1.3k input tokens (estimated), written to the cache once (creation
  weight) and re-read while it stays in context; downscaling halves it.

Plans are ranked by combined saving. The dominant consumer is the session with
the largest share of its own window, since points of different subscriptions
are not comparable.

### Friction problems

Episode definitions the problems rest on (classifier `friction-episode.v16`).
`runsignal.PatternSpecs()` is the only list of patterns: id, classifier
version, cause scope, time kind, cost rule, description template and
quarantine. A *stall* is an idle gap of 5 minutes to 2 hours that the agent
owed: a tool result was in and the next tool call came late. A tool running,
the operator or an injected message, a terminal or evidence-only message, a
park, wake or stop, and gaps over two hours (a suspended machine or an
outage) are never stalls. Wait loops and re-reads stay within one working
stretch (no silence over 30 minutes, no boundary). A tool call counts once
however many copies of its event were recorded. `handoff-continuation` is
informational (reported on a run, never priced or shown) and never fires in a
goal run or when the next user message is an injected one. A *command failure* is scenario friction: a
Vrooli project-owned command (resolved through the catalog: scenario CLIs,
`vrooli ...`) fails with an error, whatever the cause: the agent misused it
(bad flag, wrong arguments) or the scenario or its infrastructure was broken or
unavailable (capacity broker lock, read-only filesystem, unreachable API). A
no-match, an interrupt, a help probe, a documented informational verdict and any
command outside the project (`ls`, `rg`, `python`, `git`) are not command
failures. One failing call is one episode, owned by the scenario and command
path, described as "`<command>` failed: <first error line>". Each classifier
bump prunes older versions once at start-up.

An episode is priced by its own redundant calls: `turns` is their number,
`tokens` their argument and result tokens, `wall_clock_ms` their own duration
("redundant call time"; a stall's is its "idle gap"). No episode carries
run-wide tokens. A pattern that fails the precision gate
(`runsignal/testdata/precision`, `go test ./internal/runsignal -run Precision`)
is `Quarantined` in its spec: kept in the read model, excluded from usage
friction, R-friction and the UI.

Classifier history: v23 (facts v25) guidance-repair is unpriced (the preceding call is not redundant work); episode reads honour workload and classifier-version filters. v22 (facts v25) existence probes (`command -v`, `which`, `type`, `hash`) are not calls of the probed program. v21 (facts v24) a heredoc inside `$(…)` no longer breaks parsing on an apostrophe; a non-literal element in a Codex `exec_command` argv makes the command unresolved; run scope applies to fact, episode and finding reads. v20 agent-behaviour episodes (operator corrections) carry no command owner, so they group as one problem and are never filed against a scenario. v19 harness blocks in front of another injected form keep that form's source; a second report of the same events counts against precision, distinct failing calls in one labelled span each count; recall counts labelled episodes found. v18 (facts v23) a unit after `&&` or `||` no longer inherits a failed call's exit (the failing unit is unknown; `;` and pipeline tails still inherit it); the precision gate needs at least 5 matched true detections and a 95% Wilson lower bound of precision of at least 0.7, with recall checked from 3 true labels (edit-revert and wait-misuse leave the shown set); cohort selection and cohort episode reads filter to shown patterns in SQL before the limit, with hidden episodes counted separately. v17 manifest command semantics now reach facts (the lookup used a doubled owner key, so no `semantics` declaration ever applied); a verify command's non-zero exit (`effort lint`, `effort epoch-check`) is its verdict, not a failure; command-failure shown. v16 command-failure is scenario friction: one episode per failing project-owned command call, any cause, owner scenario + command path, first error line in the description (labels re-judged under this rule, `relabel: scenario-friction-2026-10-30`). v15 operator corrections must target the agent's prior action, corpus-specific exit-code table removed, quarantined and informational patterns are excluded (runsignal.Shown) from run friction scores, durability evidence, capability efficacy, usage friction, cohort episodes and the episode endpoints; only the precision test and the durable read model read them unfiltered. The precision gate reports detections outside any label as unlabelled, fails a pattern whose unlabelled detections exceed 20%, and also requires all-label recall; quarantine reasons point at testdata/precision/results.json instead of quoting numbers. v14 one pattern table, command-path outcomes, message
sources, redundant-call cost, `time-to-first-success` deleted. v13 a stall is an
idle gap of 5 min–2 h inside an active turn (not waiting on a tool, the
operator, a park or a resume); wait loops and slow starts stay within one
working stretch. v12 a tool call is counted once however often its events were
duplicated and never pairs with itself; a pipeline filter (`| head`) is not a
file read. v11 wall clock is the time the friction wasted, per pattern;
read-then-reread pairs only within one working stretch. v10 handoff-continuation
stops at the next user message.

Problems count only runs the current episode classifier has analysed;
`UsageFriction` returns `runs_analysed` and `runs_total` for the windows so a
client can say it is still re-analysing while history is re-projected. A
problem's `time_kind` says what its time measures: `redundant` (the
redundant calls' own duration) or `idle` (a stall's gap: waiting, not wasted
work).
Reporting or dismissing a fingerprint no current problem has is rejected.

`UsageFriction` (`agent-manager usage friction`) lists the problems R-friction
prices: every episode with one fingerprint in runs that consumed quota in the
current window, with three or more episodes. Each carries one plain sentence
(`usageattribution.Describe`, derived from the pattern, the preferred and used
commands, and the median duration — never the generic "review the
fingerprint" text), its owner (the command, the scenario, or "Agent
behaviour" when the detector has no command), episode and run counts, typical
and total elapsed time (episodes can overlap, so the total can exceed wall
time), attributed quota points, and a durable status: `open`, `reported` (with
the scenario-qa bug id) or `dismissed`, in `friction_problem_status`.
`usage friction-report <fingerprint>` files the problem through the same
report-bug writer path as the model-policy drift check
(`PublishScenarioQABug`, idempotency key `agent-manager-friction/<fingerprint>`,
honesty flags `speculative-cause` and `repro-not-attempted`) and records the
bug id; reporting twice files once. `usage friction-dismiss <fingerprint>`
sets it aside. The two actions are `EpisodesService.ReportFrictionProblem` and
`DismissFrictionProblem`, which are mutations and not part of the read-only
measures service. The recurring-friction publisher (generic text findings
sent to the meta-optimization inbox) and the findings review fields
(`operator_decision`, `target_measure`, `before_value`, `after_value`,
`effectiveness`) are removed; `run_findings` keeps only what investigations
write and read.

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

## Surfaces

- **Usage page** replaces `/stats` as the primary analytics entry. It shows the
  forecast per window, the consumers, the recommendations and coverage; the
  former Stats panels move under Details, and manual import folds into
  Coverage.
- **Dashboard strip:** one line per provider window with a bar, a status chip
  and the projected exhaustion; it opens the Usage page.
- **Run usage panel:** the run's tokens from the one accounting function,
  estimated quota points (requests, not turns) and recommendation groups that name the run.
- **Alerts:** `usageforecast.Alerter` evaluates the current windows every 5
  minutes and publishes `agent_manager.usage.notification.v1` through Vrooli
  Events. Categories are declared in `.vrooli/notifications.json`, which also
  sets `appDelivery` (Settings › Notifications renders the shared
  `AppNotificationSettings`; `main.tsx` calls `initAppNotifications()`):

  | Category | Default | When |
  |---|---|---|
  | `usage-will-exhaust` | on | A window becomes `will_exhaust`; sent again only if the projection moves earlier by more than 12 hours. The body names the window, the projected date, and the window's #1 action with its saving. |
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
- **CLI:** `agent-manager usage forecast|burners|recommendations|friction|friction-report|friction-dismiss|coverage|compare|run|reindex`
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
  `usage run <id> [--children] [--weights] [--facts]` reports a run's
  weighted non-cache tokens (and its direct children's) or its redacted
  invocation facts; `usage reindex [<run-id>] [filters] [--full]` rebuilds
  durable invocation evidence (refresh by default). They replace `run
  tokens`, `run invocation-facts` and `run replay-invocation-facts |
  refresh-invocation-facts | replay-invocation-corpus`. The same six usage
  procedures are also served by `/measures/execute` through the measures-go
  registry, from the same compute path.
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
