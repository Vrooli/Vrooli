# Performance — Network Manager

## Purpose Of This Document

Define performance budgets and measurement constraints for network diagnostics and UI responsiveness.

## Budgets

| Area | Target |
|---|---|
| UI initial load | Under 2 seconds in local development once implemented. |
| Snapshot status update | Progress visible within 1 second of run start. |
| DNS probe timeout | Configurable; default should avoid blocking the whole snapshot on one resolver. |
| Health snapshot run | Bounded and cancellable; long tests should report partial progress. |
| Optimization candidate | Each candidate has an explicit timeout and stabilization window. |

## Current Measurements

The snapshot domain now records read-only baseline measurements using standard-library probes: DNS lookup latency, WAN TCP reachability, IPv4/IPv6 TCP availability, TCP-based packet-loss/jitter approximation, redacted host facts, and resolver count where the host resolver configuration is available. Gateway reachability and throughput remain explicit unsupported/unavailable results until capability adapters or approved measurement backends exist.

## Known Constraints

- Network measurements are noisy and must include confidence/reliability notes.
- Throughput tests can consume bandwidth and distort other measurements.
- Router and DNS backend APIs vary in latency and rate limits.
- Continuous monitoring must avoid becoming a network load source.

## Regression Procedure

1. Run scenario tests.
2. Run deterministic snapshot tests with fake probes.
3. Run a controlled local snapshot against a known network and keep the first persisted `status=baseline` snapshot as comparison evidence.
4. Compare report shape and timing against previous baseline.
5. Record significant measurement changes in observability notes.

## Cross-References

- [`../concepts/FLOWS.md`](../concepts/FLOWS.md)
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md)
