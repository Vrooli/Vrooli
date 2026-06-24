# Collection fork-waste + per-process attribution over time — 2026-06-24

## Symptom
system-monitor sat at a steady ~7.5% CPU, partly self-inflicted, and **could not** reproduce
the manual "top consumers by scenario" forensic that found the recurring-scan storms. Each
20s collection cycle:
- forked ~10 redundant `bash -c` pipelines (zombie + high-thread queries run twice; 4 `pgrep`
  forks in the critical-process check),
- probed uncached host-inventory / `nvidia-smi` **3× per cycle** (cpu, memory, gpu collectors
  each called `hostinventory.Collect` independently),
- stored each collector's output as one **opaque JSON blob** in `metrics(...)`, capturing only
  the short `comm` name — no pid/ppid/cmdline/cwd/owner, so no attribution and no process
  timeline query.

## Fix
- **Cut fork waste** (`internal/collectors/process.go`, `collectors/hostsnapshot.go`). Zombie
  + high-thread results are computed once in `Collect` and passed to `processHealth(...)`
  (the old path re-shelled both). The 4 `pgrep -f` forks collapse to a single `ps -eo comm`
  scan + substring match. A shared `CachedSnapshotProvider` (short TTL, serves stale on
  transient probe error) is wired across cpu/memory/gpu so host inventory / nvidia-smi is
  probed **once per cycle** instead of 3×.
- **Single `/proc` walk behind a build-tagged seam** (`internal/procsampler/`). One pass over
  `/proc/<pid>/{stat,cmdline,comm,cwd}` yields pid, ppid, cmdline, cwd, utime/stime → CPU%
  (inter-sample delta, USER_HZ=100, pid-reuse guarded), rss, threads. `sampler_linux.go` is
  `//go:build linux`; non-Linux returns `ErrUnsupported` (treated as "no samples", not fatal).
  Tolerates ESRCH on short-lived/pid-reuse. Net-reduces forks (one walk vs ~10 pipelines).
- **Queryable history + downsampling** (`internal/repository/.../process_samples.go`,
  `services/retention_scheduler.go`). New additive table
  `process_samples(ts,pid,ppid,comm,cmdline,cwd,owner,cpu_pct,rss_kb,threads)` indexed on
  `(ts)` and `(owner,ts)`, written top-N per cycle (drops logged). Retention is now
  raw-then-rollup: raw rows kept for `SYSTEM_MONITOR_RAW_RETENTION`, then downsampled to
  per-owner/per-minute `process_sample_rollups`, pruned at `SYSTEM_MONITOR_ROLLUP_RETENTION`.
  The existing `metrics` blob storage is untouched.
- **Cheap `/proc` PID→scenario attribution** (`procsampler/attribution.go`). Matches
  `/proc/<pid>/cwd` against `.../scenarios/<name>/`, parses `<scenario>-api` binary names, and
  **walks PPID** (cycle-guarded) so children inherit the owning scenario — proving the
  `osv-scanner → security-health` link. Non-scenario host procs → `unknown` (first-class
  bucket). The docker/cgroup attributor is a **fallback only**.
- **Expose it** (`handlers/metrics.go`, `server/router.go`, `cli/domains/metrics/register.go`).
  `GET /api/v1/metrics/processes/timeline?window=&owner=&top=` (gorilla/mux, plain JSON per the
  scenario's existing forensics/logs handlers — no proto) returns ranked
  `{owner, comm, pid, aggregated, cpu_pct, rss_kb, sample_count, first_seen, last_seen}`. CLI:
  `system-monitor metrics process-timeline --window --owner --top [--json]`. This is the
  standing replacement for the ad-hoc `ps`/`top` investigation.

## Config knobs
| Env | Default | Effect |
|-----|---------|--------|
| `SYSTEM_MONITOR_PROC_SAMPLE_INTERVAL` | `20s` | Process-sample cadence. |
| `SYSTEM_MONITOR_PROC_SAMPLE_TOP_N` | `50` | Rows persisted per cycle (drops logged). |
| `SYSTEM_MONITOR_RAW_RETENTION` | `6h` | Raw `process_samples` window. |
| `SYSTEM_MONITOR_ROLLUP_RETENTION` | `720h` (30d) | Per-owner/minute rollup window. |

## After evidence (live, new code deployed — PROVEN)
**The headline capability works.** With the new code running, the new endpoint reproduces the
manual forensic directly:
```
GET /api/v1/metrics/processes/timeline?window=3m&top=25
  → osv-scanner  owner=security-health  cpu=329%   (top consumer)
  → aggregate by owner: security-health 329% … unknown 28% … system-monitor 17% …
```
i.e. the recurring-scan CPU is now correctly attributed to **security-health** (its spawner),
the exact answer the 2-minute manual `ps`/`top` investigation produced — now a cheap query.

**Fork reduction (the 3a CPU win) is structural, not a headline own-CPU number.** The ~10
shell-outs/cycle were *separate* short-lived `bash`/`ps`/`pgrep`/`nvidia-smi` processes, so
eliminating them reduces host process-churn and scheduler overhead, not `system-monitor-api`'s
own CPU line — the replacement single `/proc` walk runs in-process (and is cheap). The
reduction is asserted by the `no-double-fork` unit test (stubbed `commandOutput` fork counter:
`pgrep`==0, zombie/high-thread computed once, one shared host-inventory probe/cycle) rather than
by a noisy live CPU delta. (The `recurring-scan-cpu-sampler.sh` own-CPU readings for long-lived
API processes are unreliable — it substring-matches the process name in *any* command line,
including measurement commands — so they are not used here; the storm/subprocess signals and
the in-app counters are.)

Dogfood any time:
```bash
system-monitor metrics process-timeline --window 5m --top 20 --json
curl "http://localhost:${API_PORT}/api/v1/metrics/processes/timeline?window=5m&top=20"
```

## Validation
- `go build ./...` (api + cli) clean; `go test ./...` — api 12 packages + cli 2 packages pass,
  including: /proc stat parse (comm-with-spaces/parens, threads, rss-pages→KiB), CPU-delta
  across cycles + pid-reuse-drops-spike, ESRCH tolerance; attributor cwd/binary/PPID-walk/
  unknown/docker-fallback/cycle-guard; **spawner-wins-over-scanned-dir-cwd** (the
  osv-scanner→security-health precedence guard); sqlite+memory timeline ranking + rollup
  aggregation + raw prune; no-double-fork assertion; timeline endpoint ranked-window.
- `go vet`, `golangci-lint run`, `gofumpt -l` clean. `baseline diff` = `preexisting` (no new reds).

## Attribution precedence (refined during live dogfood)
First dogfood showed osv-scanner attributed to the *scanned* scenarios (agent-manager, …) via
their cwd, because the reconcile runs `osv-scanner` with cwd inside the scenario it is scanning.
Attribution was reordered to **anchor (scenario-api binary) → ancestry (nearest scenario-api
parent) → cwd-location → docker → unknown**, so a spawned tool is attributed to the scenario
that launched it. Verified live: all osv-scanner children share `ppid = security-health-api`,
and now attribute to `security-health`.
