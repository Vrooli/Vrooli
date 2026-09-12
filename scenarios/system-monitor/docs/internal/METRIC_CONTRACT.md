# System Monitor Metric Contract

This document defines the semantic contract for a metric observation. The
protobuf schema and the API models implement this contract.

## Collection cycles

The scheduler creates one `cycle_id` and one caller-owned `observed_at` time
for each collection pass. Every enabled collector selected for that pass has
one observation in that cycle. A collector error is persisted as a failed
observation. Repositories group observations by `cycle_id`, never by timestamp
coincidence.

`observed_at` is wall-clock time for persistence and display. Rate collectors
must use monotonic elapsed time internally. A cycle is immutable after the
repository commit. Duplicate cycle IDs are rejected by the owning scheduler
contract; repository implementations must preserve all observations for a
single accepted cycle.

## Measurement states

| State | Meaning | Numeric value | Consumer behavior |
|---|---|---|---|
| `measured` | The native collector returned a valid observation. | Valid, including `0`. | Show, chart, alert, and include in statistics. |
| `failed` | The collector ran but did not produce a valid observation. | Not meaningful. | Show the reason and omit from numeric consumers. |
| `unsupported` | No backend exists for this host or platform. | Not meaningful. | Show capability state and omit from numeric consumers. |
| `stale` | The last valid observation is older than the freshness budget. | Not current. | Show age and omit from current-value decisions. |
| `not_yet_sampled` | No observation exists for the requested metric. | Not available. | Show a gap or explicit pending state. |

Legacy scalar fields remain compatibility projections. They are not the source
of truth. New consumers must use `MetricValue.state`; a missing or non-measured
state must never be inferred as numeric zero.

## Units and provenance

CPU, memory, disk, GPU utilization, and percentages use percent in the range
0–100. TCP connections are counts. Bytes and rates carry their unit in the
field name or collector metadata. Each state carries `provenance` and
`observed_at` so an operator can identify the native backend and freshness.

## Consumer rules

- Timeline rows are cycle-linked and may contain gaps for unavailable metrics.
- Charts include only `measured` values. A measured zero remains a plotted zero.
- Reports and alerts include only measured observations for the metric being
  evaluated. An unavailable CPU value cannot lower an average or suppress an
  alert.
- The UI displays the state reason, provenance, and stale age when a value is
  unavailable.

## Stocks and flows

A stock is a level at an instant, such as bytes used or swap percent. A flow
is a rate over an interval, such as pages swapped or major faults per second.
Cache-like and reclaim-like resources must expose a flow beside their stock;
the stock alone cannot distinguish cold pages parked harmlessly from active
thrashing.

## Cumulative counters

Kernel counters are monotonic totals, not gauges. Collectors convert every
cumulative counter to a per-second rate using the true monotonic interval at
collection time. The first sample after startup, a counter reset, or an invalid
interval is `not_yet_sampled`; it is never stored as a climbing level or a
numeric zero.

## Platform tiers

## CPU signal catalog

Signal keys are stable API vocabulary. A refusal is an observation with a
mechanism-specific reason and no numeric value; it is never represented by
zero. `planned` is not permitted in the shipped catalog.

| Signal key | Unit | Signal tier | Linux | macOS | Windows | Unsupported |
|---|---|---:|---|---|---|---|
| usage_percent | percent | 1 | `/proc/stat` | `kern.cp_time` | `GetSystemTimes` | no native CPU backend |
| mode_breakdown | percent by mode | 1 | `/proc/stat` user/nice/system/idle/iowait/irq/softirq/steal | `kern.cp_time` user/nice/system/idle; other modes not separately accounted | `GetSystemTimes` user/system/idle; other modes not exposed | no native CPU backend |
| load_average | load | 1 | `/proc/loadavg` | `vm.loadavg` | refuse: Windows exposes no Unix load average | no load backend |
| normalized_load_1 | load/core | 1 | derived from `/proc/loadavg` and core count | derived from `vm.loadavg` and core count | refuse: no load-average source | no load backend |
| normalized_load_5 | load/core | 1 | derived from `/proc/loadavg` and core count | derived from `vm.loadavg` and core count | refuse: no load-average source | no load backend |
| run_queue_depth | processes | 1 | `/proc/loadavg` running field | refuse: `vm.loadavg` has no runnable-process field | refuse: no native runnable-process counter | no run-queue backend |
| context_switches_per_second | per second | 1 | `/proc/stat` `ctxt` rate | refuse: no public counter in this build | refuse: PDH backend not enabled | no counter backend |
| interrupts_per_second | per second | 1 | `/proc/stat` `intr` rate | refuse: no public counter in this build | refuse: PDH backend not enabled | no counter backend |
| cpu_psi_some_avg10 | percent | 1 | `/proc/pressure/cpu` | refuse: PSI is Linux-specific | refuse: PSI is Linux-specific | no PSI backend |
| cpu_psi_full_avg10 | percent | 1 | `/proc/pressure/cpu` | refuse: PSI is Linux-specific | refuse: PSI is Linux-specific | no PSI backend |
| per_core_utilization | percent/core | 2 | `/proc/stat` `cpuN` | `host_processor_info` | PDH processor instances | no per-core backend |
| core_imbalance_index | percentage points | 2 | derived from `cpuN` | derived from processor info | derived from PDH instances | no per-core backend |
| quota_throttling | per second, percent | 2 | cgroup v2 `cpu.stat` and `cpu.max` | refuse: no cgroup backend in this build | refuse: job-object backend not enabled | no quota backend |
| frequency_derate_ratio | ratio | 2 | `scaling_cur_freq` / `cpuinfo_max_freq` | refuse: no public frequency backend in this build | refuse: performance counter backend not enabled | no frequency backend |
| thermal_throttle_evidence | count, celsius | 2 | hwmon and thermal zones | refuse: thermal join not enabled in this build | refuse: thermal backend not enabled | no thermal backend |
| fork_rate | per second | 3 | existing fork-rate collector | explicit refusal if native counter unavailable | explicit refusal if native counter unavailable | no fork counter backend |
| process_cpu_seconds | seconds | 3 | process CPU ticks | explicit refusal if sampler unavailable | explicit refusal if sampler unavailable | no process sampler backend |
| historical_process_cpu | percent | 3 | persisted process rollups | persisted process rollups | persisted process rollups | no historical store |

The collector's executable catalog and tests must stay aligned with this table.

| Tier | Signal | Linux | macOS | Windows |
|---|---|---|---|---|
| 1 | paging and major-fault rates | `/proc/vmstat` | explicit unsupported in this build (native binding unavailable) | PDH or explicit unsupported |
| 1 | per-process swap and major faults | `/proc/<pid>` | explicit unsupported in this build (native binding unavailable) | native process API or explicit unsupported |
| 2 | memory pressure | PSI | memorystatus | explicit unsupported |
| 3 | fragmentation and compaction | `/proc/buddyinfo` and vmstat | unsupported | unsupported |

Unsupported capabilities carry a reason and no numeric key. A zero is valid
only when the platform actually measured zero.
