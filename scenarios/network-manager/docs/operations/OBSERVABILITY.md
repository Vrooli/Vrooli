# Observability — Network Manager

## Purpose

Define health, product, and business signals needed to operate Network Manager and validate its value.

## Health Signals

| Signal | Source | Purpose |
|---|---|---|
| API health | Scenario API | Lifecycle readiness. |
| UI health | Scenario UI | Operator access. |
| Resolver health | AdGuard Home adapter | DNS filtering availability. |
| Gateway reachability | Snapshot probes | LAN/router health. |
| WAN reachability | Snapshot probes | Internet availability. |
| DNS latency | Snapshot probes | Resolver performance. |
| Packet loss/jitter | Snapshot probes | Quality and reliability. |

## Product Signals

- Number of snapshots run.
- Number of filtering previews and applies.
- Rollbacks performed.
- Optimization runs completed.
- Candidate score deltas.
- Devices discovered and assigned to groups.
- Home Automation actions/events emitted.

## Privacy-Sensitive Signals

DNS query-level telemetry is sensitive. Default observability should use aggregate counts and health status rather than raw query logs unless the operator explicitly enables detailed visibility.

## Business Validation Signals

- Time from setup to first useful report.
- Before/after improvement evidence.
- Number of manual networking tasks avoided.
- Small-office reports exported.
- Router adapter requests by platform.

## Alerting Candidates

- Resolver down.
- DNS latency regression.
- WAN outage.
- IPv6 resolver bypass detected.
- Filtering policy failed to apply.
- Optimization applied but post-check regressed.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md)
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md)
- [`../concepts/DATA.md`](../concepts/DATA.md)
