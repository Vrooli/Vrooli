# Bright Window Phase 0 Baseline

Captured on 2026-06-24 after adding the development-only pprof mount and
monitor self-metrics.

## Commands

```bash
make start
curl -fsS http://localhost:16914/debug/pprof/
curl -fsS 'http://localhost:16914/debug/pprof/profile?seconds=30' \
  -o scenarios/system-monitor/docs/perf/2026-06-24-bright-window-before.pprof
curl -fsS http://localhost:16914/health | jq '.metrics.self'
system-monitor metrics process-timeline --window 5m --top 20 --json
```

## Artifacts

- CPU profile: `scenarios/system-monitor/docs/perf/2026-06-24-bright-window-before.pprof`

## Self-Metrics Snapshot

```json
{
  "collector_duration_ms": {
    "cpu": 0.139,
    "disk": 5.433,
    "gpu": 67.022,
    "memory": 41.354,
    "network": 43.947,
    "process": 208.848
  },
  "collector_forks": {
    "cpu": 0,
    "disk": 2,
    "gpu": 0,
    "memory": 1,
    "network": 5,
    "process": 5
  },
  "last_cycle_duration_ms": 1507.586,
  "last_cycle_forks": 11,
  "last_proc_sample_duration_ms": 24.118,
  "recorded_at": "2026-06-24T18:02:26Z",
  "total_collector_forks": 49
}
```

## Notes

- `/debug/pprof/` is mounted only outside production.
- Development `WriteTimeout` is 75s so 30-60s pprof captures can complete;
  production keeps the existing 15s timeout.
- The live `system-monitor metrics process-timeline --window 5m --top 20 --json`
  command returned ranked owner-attributed process consumers after lifecycle
  install/restart.

## Phase 4 Native Collector Slice

Updated after the Connect/CLI migration slices:

- Process, network, CPU top-process, memory top-process, and steady disk
  collection no longer fork subprocesses. They read `/proc`, `/proc/net`, and
  `statfs` directly.
- Added zero-steady-fork regression guards for process, network, and disk
  collectors by stubbing `commandOutput`.
- Remaining `commandOutput` collector uses are explicit on-demand disk
  inspection helpers: `GetLargestDirectories` (`du`) and `GetLargestFiles`
  (`find`). They are not part of the steady collection cycle.

Verification:

```bash
rg -n "commandOutput|bash -c|netstat|ps -|pgrep|df -|du -|find " \
  scenarios/system-monitor/api/internal/collectors -g '*.go'
go test ./... # in scenarios/system-monitor/api
```

## Final Strict-Closure Evidence (2026-06-25)

After the Connect router cleanup and shared process-sampler hardening:

- API routing uses `http.ServeMux` plus generated Connect handlers for proto-owned
  domains. `github.com/gorilla/mux` is no longer present in the API module.
- The remaining REST surfaces are explicit exceptions: health probes, dev pprof,
  raw logs/forensics, and tool discovery/execution.
- The UI dispatches proto-owned calls to Connect JSON procedure paths; legacy
  `/api/v1/metrics/current` now returns `404`.
- The normal collector cycle reported zero collector forks.

Artifacts:

- CPU profile: `scenarios/system-monitor/docs/perf/2026-06-25-bright-window-after.pprof`

Live self-metrics snapshot:

```json
{
  "collector_duration_ms": {
    "cpu": 0.189,
    "disk": 0.139,
    "gpu": 75.369,
    "memory": 25.652,
    "network": 31.217,
    "process": 27.584
  },
  "collector_forks": {
    "cpu": 0,
    "disk": 0,
    "gpu": 0,
    "memory": 0,
    "network": 0,
    "process": 0
  },
  "last_cycle_duration_ms": 1267.215,
  "last_cycle_forks": 0,
  "last_proc_sample_duration_ms": 0.006,
  "recorded_at": "2026-06-25T13:38:21Z",
  "total_collector_forks": 0
}
```

Live process-timeline smoke:

```json
{
  "window_seconds": 300,
  "top": 5,
  "count": 5
}
```

Profile summary:

- 30.01s capture, 4.33s total samples.
- Collector work was small in the profile: `ProcessCollector.Collect` 0.06s,
  `NetworkCollector.Collect` 0.06s, cached process sampling 0.05s.
- The remaining hot path was SQLite health/current-metrics reads during the live
  health and profile smoke (`sqlite3VdbeExec` dominated cumulative samples).

Final managed validation:

```text
vrooli scenario test system-monitor --json
run: 20260625-135133-1c262030
result: 17 passed / 0 failed
performance: passed, L5, no findings
```
