# Migration Handoff

- The team contract was refactored on 2026-04-09 from a leader-led analysis swarm to two independent lanes.
- `portfolio-manager` is the only active lane right now. `outcome-strategist` stays disabled until Command Center exposes metrics and gaps.
- Apply compatible accepted portfolio decisions first, record `decision-application/<decision-id>` markers, then stop early if 3 unresolved portfolio decisions are already pending.
- Treat initiative priority and dependency writes as advisory until Swarm Manager supports them directly.
