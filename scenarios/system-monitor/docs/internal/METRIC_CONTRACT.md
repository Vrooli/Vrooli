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

| Tier | Signal | Linux | macOS | Windows |
|---|---|---|---|---|
| 1 | paging and major-fault rates | `/proc/vmstat` | explicit unsupported in this build (native binding unavailable) | PDH or explicit unsupported |
| 1 | per-process swap and major faults | `/proc/<pid>` | explicit unsupported in this build (native binding unavailable) | native process API or explicit unsupported |
| 2 | memory pressure | PSI | memorystatus | explicit unsupported |
| 3 | fragmentation and compaction | `/proc/buddyinfo` and vmstat | unsupported | unsupported |

Unsupported capabilities carry a reason and no numeric key. A zero is valid
only when the platform actually measured zero.
