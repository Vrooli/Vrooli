# Kernel-signal ingestion: incremental, cursor-backed, throttled — 2026-06-24

## Symptom
Every 60s health tick, `refreshSystemEvents` → `systemevents.Service.Ingest` →
`HostCollector.collectKernelSignals` looped over **every** boot
`journalctl --list-boots` returns (~22 on this host) and ran a big-regex kernel
grep once per boot, plus re-read the full current-boot buffer (`-b 0 -n 500`).
Immutable historical boots were re-grepped every single minute, pegging one
core for the duration of the ingest. (Root-cause analysis: plan §3.)

## Root cause
The recurring job was stateless: each tick redid the entire scan from scratch
with no memory of which boots were already processed or how far into the
current boot it had read.

- No throttle on `Ingest` — it ran on every 60s tick even though host kernel
  events change far less often (the adjacent host-inventory path was already
  cached at 30s via `NewCachedCollector`).
- No journal cursor — `journal.QueryOpts`/`buildArgs` had no
  `--after-cursor`/`--show-cursor` support and `rawJournalEntry` didn't parse
  `__CURSOR`, so incremental reads were impossible.
- No per-boot "already scanned" memory — historical, immutable boots were
  re-greGped every cycle.

## Fix
Three layers, all preserving 100% of detected events (gap-free):

1. **Throttle (2a).** `Service` gained an `interval` (env
   `AUTOHEAL_SYSTEMEVENTS_INTERVAL`, default `300s`, clamped `[300s, 600s]`) and
   an `IngestIfDue` entry point used by the tick path. The expensive journal
   work now runs at most once per interval instead of every tick (immediate
   5–10× exec reduction). The explicit refresh endpoint and startup still call
   `Ingest` directly to force a fresh read.

2. **Journal cursor + scan-once (2b).** `QueryOpts` gained `AfterCursor` /
   `ShowCursor`; `buildArgs` emits `--after-cursor=<c>` / `--show-cursor`;
   `LogEntry`/`rawJournalEntry` carry `__CURSOR`. A new `CursorStore`
   abstraction (SQLite tables `journal_cursors` + `journal_scanned_boots`,
   in-memory fake for tests) persists:
   - a per-boot scanned marker, so **immutable historical boots are grepped at
     most once**; subsequent ticks skip them entirely;
   - the journald cursor of the last successfully-ingested current-boot entry,
     so the live boot is read **incrementally via `--after-cursor`** (delta
     only) instead of a full `-n 500` re-scan.

   **Gap-free semantics:** the cursor advances ONLY after a successful read. On
   cursor invalidation (journal vacuum/rotation → journalctl errors on the
   seek) or a boot change, the collector falls back to a bounded
   `-b 0 -n 500` re-scan and re-anchors the cursor — never silently skipping
   events. A historical boot whose scan fails is left unmarked so the next tick
   retries it.

   Steady state drops from ~22 greps/tick to a single delta read.

3. **Startup jitter (2c).** `Registry.SeedStartupJitter` seeds each cold-start
   check's `lastRun` to `now - rand[0, interval)`, so aligned-interval checks
   first fire spread uniformly across the first interval window instead of
   bursting together (the synchronized thundering herd). Checks restored from
   persistence keep their existing schedule.

### Observability (2e)
`HostCollector.ExecsAvoided()` counts kernel-grep invocations skipped since
process start (already-scanned historical boots + incremental current-boot
reads). Surfaced as `systemEvents.journalctlExecsAvoided` on `GET /status` and
`execsAvoided` in the ingest summary.

## Sub-phase 2d (shared kernel collection) — SKIPPED
Collapsing the three overlapping kernel-grep families into one cursor-backed
pass was assessed and intentionally skipped: they have materially different
semantics (different regexes/time windows/boot scopes; `boot_history` needs
full entries to detect *missing* clean-shutdown markers, which is incompatible
with a grep pass) and the other two are already throttled (hostinventory at 30s
via the cached collector, boot_history at 600s). Folding them would risk
detection correctness for marginal gain.

## Validation
- `go build ./...` clean; `go test ./...` — full suite passes.
- New unit tests: `--after-cursor`/`--show-cursor` argv + `__CURSOR` parse
  (`journal`); cold-start bounded re-scan, incremental delta read, cursor
  invalidation → bounded-rescan fallback (no gap), failed re-scan leaves cursor
  untouched, historical boot scanned once + not-marked-on-failure, exec-avoided
  counter (`systemevents/collector`); `IngestIfDue` interval gating + clamp +
  `Ingest` always-runs + exec-avoided surfacing (`systemevents/service`);
  cursor/scanned-boot SQLite round-trip (`persistence`); jitter offset within
  `[0, interval)` + restored checks untouched (`checks`).

## Out of scope (follow-up candidates)
- Sub-phase 2d shared kernel collection (see above).
- Retention/cleanup of `journal_scanned_boots` rows for very long-lived hosts
  (rows are tiny and keyed by boot id; unbounded growth is negligible but could
  be pruned alongside the system-events retention sweep).
