# Collection Performance Budgets

These are initial operator budgets. Measurements are recorded from the latest
completed cycle and exposed by the self-metrics health payload.

| Budget | Standard host | Low-power host | Evidence |
|---|---:|---:|---|
| Cycle duration | <= 50% of collection interval | <= 50% of collection interval | `last_cycle_duration_ms`, `headroom_ok` |
| Native subprocess forks | bounded by collector contract | bounded by collector contract | `last_cycle_forks`, `collector_forks` |
| Process walk | one bounded walk per process cadence | one bounded walk per process cadence | `last_proc_sample_duration_ms` |
| History query | explicit repository limit | explicit repository limit | repository query tests |
| Retained in-memory samples | 1000 metric entries | 1000 metric entries | memory repository cap |

The scheduler records collector duration and fork counts. Slow persistence or
collector failures remain visible in cycle logs and state fields. A benchmark
or soak result must include host profile, collection interval, history window,
and storage mode before it can change these budgets.
