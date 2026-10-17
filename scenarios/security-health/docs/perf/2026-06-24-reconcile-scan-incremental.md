# Reconcile-loop osv-scanner CPU storm — incremental, bounded, cached — 2026-06-24

## Symptom
A live 2-minute process sample on the host (`scripts/perf/recurring-scan-cpu-sampler.sh`)
found `osv-scanner` among the top CPU consumers in **every** sample, peaking at
**203% CPU**, effectively near-continuous. The cause was the 5-minute reconcile loop
(`api/main.go` `runReconcileLoop` → `RunReconcileOnce` → `Annotator.Annotate`) running
`osv-scanner --format json -r .` **once per scenario (~110), sequentially, every 5 minutes**,
with:
- no result caching (every scenario re-scanned every cycle regardless of change),
- no offline OSV DB (each invocation could re-resolve the vuln DB over the network),
- no overlap lock between the timer and the on-demand `Reindex` RPC (2× scan-storm risk),
- a hardcoded `const` interval with no jitter (synchronized with the rest of the fleet).

## Root cause
The reconcile job was a stateless full recompute: it redid the entire expensive fleet
scan from scratch every tick with no memory of unchanged inputs — the same anti-pattern as
the previously-fixed AI re-indexing and architecture-cartographer validation incidents
(`swarm-manager` rec-9cbab63365931119, rec-aa6c3f877e075ed7).

## Fix (detection coverage unchanged — only *when/how* the work runs)
- **Lockfile-content-hash result cache** (`internal/dependencies/annotate.go`,
  `internal/dependencies/store.go`, `schema.sql`). Each scenario's scan is keyed by a
  SHA-256 over **every resolved-version lockfile's content** (`go.mod`, `go.sum`,
  `pnpm-lock.yaml`, `package-lock.json`, `yarn.lock`, `npm-shrinkwrap.json`) plus the
  osv-scanner version and a UTC **day epoch**. A cache hit skips the subprocess.
  The key folds in everything that can change the result, so any real change forces a
  re-scan — **no false skips**. A walk error yields an empty key → always scan (fail-safe).
  > NOTE: the npm lockfile set is deliberately a superset. The repo has 55+
  > `package-lock.json` files that osv-scanner reads under `-r` and whose npm vulns the
  > annotator surfaces; hashing only `pnpm-lock.yaml` would have been a false-skip bug.
- **Daily-fresh detection (online scans + day epoch)** (`internal/validation/scan_osv.go`,
  `internal/dependencies/annotate.go`). Per-scenario scans run **online** (osv-scanner resolves
  the live OSV database). The day epoch in the cache key makes a scenario with unchanged
  dependencies re-scan **at most once per day**, picking up newly-published CVEs without ever
  serving a result more than a day stale.
  > **Offline mode evaluated and rejected (measured regression).** The plan called for an
  > offline local OSV DB. Live measurement on this host killed it: osv-scanner loads its
  > **full per-ecosystem DB into memory on every invocation**, and the npm DB alone is
  > **~208 MB** — `/usr/bin/time -v` measured **~102 s and ~2.66 GB RSS per scan**. At 110
  > scenarios × concurrency 4 that turns the cold pass into a **~45-minute, ~10 GB-RAM storm**
  > (observed: the cold reconcile ran 14+ min with osv-scanner pegged). A compounding bug: the
  > refresh scanned an empty temp dir → **no DB ever downloaded** → offline scans found **0
  > vulns vs 125 online** (silent detection loss). The result cache already removes
  > steady-state scanning, so online keeps the unavoidable cold/daily scans cheap and
  > detection-complete. Daily online re-scanning is strictly better than the original
  > every-5-minutes online scanning.
- **Bounded-parallel fleet annotation.** The sequential per-scenario loop is now an
  `errgroup` capped by `SECURITY_HEALTH_SCAN_CONCURRENCY` (default 4). Caps peak CPU and
  shortens the unavoidable changed-scenario pass.
- **Overlap lock + interval/jitter** (`internal/dependencies/service.go`, `main.go`). A
  shared `reconcileMu` serializes `RunReconcileOnce` and `runReindexJob` so the timer and
  the Reindex RPC cannot scan-storm concurrently. The interval is env-tunable
  (`SECURITY_HEALTH_RECONCILE_INTERVAL`, default 5m) with `[0, interval/4)` jitter on reset.
- **Observability.** `Annotator.LastScanStats()` reports `scans_run` vs `scans_skipped_cache`
  per reconcile.

## Config knobs
| Env | Default | Effect |
|-----|---------|--------|
| `SECURITY_HEALTH_RECONCILE_INTERVAL` | `5m` | Base reconcile cadence (+ jitter). |
| `SECURITY_HEALTH_SCAN_CONCURRENCY` | `4` | Max scenarios scanned in parallel. |
| (cache) | on | Correctness-keyed; no flag. Cold start / changed scenarios still scan. |

## Before evidence (live host, old code, `recurring-scan-cpu-sampler.sh`, 132s/22 samples)
`osv-scanner`: sighted in **22/22** samples, 13 distinct pids, peak **217%** CPU, sum 1836.9
across the window — i.e. near-continuous. (Full JSON: `/tmp/recurring-scan-perf/before-live-*.json`.)
Concurrency cap verified live: the cold reconcile ran exactly **4** concurrent osv-scanner procs.

## After evidence (live, new code deployed — PROVEN)
Read from `DependencyService/Status` across two consecutive reconciles after restart:
- **Cold reconcile** (cache cold): `reconciled 55141 records (scans_run=110 scans_skipped_cache=0)`
  — one online pass over the fleet, fast and low-memory (no 208 MB DB load).
- **Warm reconcile** (~5.5 min later): `reconciled 55141 records (scans_run=0
  scans_skipped_cache=110)` — **all 110 scenarios served from cache, zero osv-scanner
  subprocesses.**

Steady-state sampler (90s) between ticks: `osv-scanner` **peak 0% / sum 0** CPU
(vs before: 22/22 samples, peak 217%, sum 1836.9) — i.e. ~100% reduction in steady-state
osv-scanner CPU. Re-run any time:
```bash
scripts/perf/recurring-scan-cpu-sampler.sh --window 120 --interval 6 --label after --out /tmp/recurring-scan-perf
```

## Validation
- `go build ./...` clean; `go test ./...` — 23 packages pass (cache-hit-skips-subprocess,
  per-input-change-forces-rescan incl. `package-lock.json`, **day-epoch rollover** forces
  rescan + same-day hits, concurrency cap, overlap-lock serialization). The lone
  `internal/modules` `TestProtoConnectParity` failure (validation `PreviewFix` endpoint
  descriptor) is **pre-existing** and unrelated to this work.
- `gofumpt -l` clean. Live: concurrency cap held at exactly 4 concurrent osv-scanner during the
  cold pass; `baseline diff` = `preexisting` (no new reds).
