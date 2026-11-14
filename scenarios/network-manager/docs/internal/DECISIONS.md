# Decisions — Network Manager

## Purpose Of This Document

Record durable decisions and tradeoffs future agents should not relitigate without new evidence.

## Decision Log

| Date | Decision | Reasoning |
|---|---|---|
| 2026-06-23 | Build Network Manager as greenfield scenario. | The older `network-tools` scenario was retired from the live scenario tree; git history remains available for future archaeology. |
| 2026-06-23 | Use `react-vite` template with `vrooli-default` design. | The scenario needs a polished operator UI plus API/CLI surfaces. |
| 2026-06-23 | Choose AdGuard Home as first resolver backend. | It best fits integrated DNS filtering and encrypted-DNS product controls. |
| 2026-06-23 | Defer Pi-hole and Technitium adapters. | Pi-hole is valuable but narrower; Technitium is more complex and better for advanced DNS users. |
| 2026-06-23 | No router writes in P0. | Router diversity and lockout risk make preview/manual guidance safer for the first release. |
| 2026-06-23 | Home Automation consumes actions/events. | Network Manager owns network state; Home Automation surfaces home controls. |
| 2026-06-23 | Optimization defaults to reliability-first scoring. | Latency, jitter, packet loss, DNS performance, and stability better represent perceived quality than peak download speed alone. |
| 2026-06-23 | Treat household controls as P1 policy profiles. | The trust/privacy burden is higher than the core P0 diagnostic/filtering loop. |

## Superseded Decisions

None yet.

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md)
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md)
- [`PROBLEMS.md`](PROBLEMS.md)
