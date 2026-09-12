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

## Flow-metrics change evidence

The flow-metrics implementation adds in-process `/proc/vmstat`,
`/proc/buddyinfo`, and one `/proc/<pid>/status` read per sampled process; it
does not add subprocesses. Validation used the Linux host profile, a 10-second
collector interval, and the live process table. The captured pre-final-change
sample was cycle duration 864.025ms, process-sample duration 0.247ms, and zero
native subprocess forks. The rebuilt after-sample was 356.633ms, 0.016ms, and
zero forks respectively, with `headroom_ok=true`. The gate is unchanged: cycle
duration is at most 5 seconds and `last_cycle_forks` equals the baseline.

The same SQLite database provides a before/after storage check. Before the
flow-metrics rollout (2026-08-22 13:59 UTC), the mean `metric_data` row sizes
were 858 bytes for `memory`, 437 bytes for `pressure`, and 1,923 bytes for
`process`. After the rollout (2026-08-22 16:08 UTC), they were 1,485 bytes,
3,226 bytes, and 1,923 bytes respectively. The process row is unchanged
because its paging fields are stored in the bounded `process_samples` table;
the widened pressure and memory blobs remain bounded and are included in the
existing metrics retention scheduler.
