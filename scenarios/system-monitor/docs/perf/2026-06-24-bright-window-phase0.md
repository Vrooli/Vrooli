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
