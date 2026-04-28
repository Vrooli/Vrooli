### Catalog deltas since last heartbeat
- `agent-manager` amplifier: Agent Sandbox Audit Foundation moved **0/5 → 2/5** completed (`measured`, swarm-manager overview). First non-zero motion on either of the two amplifier-gating initiatives flagged as "entirely unstarted" in the prior three snapshots.
- Protected Agent Sandboxing remains **0/3** (`measured`, unchanged).
- All other tracked initiatives unchanged (Continuous Audio Platform 0/9, GCT Pre-Commit Security 1/5, GCT Merge 0/4, GCT GitHub Integration 0/5, GCT Release Pipeline 0/2).
- operator-inputs.json still unpopulated; paying subs still 0 (`measured`, per financial-tracker knw-1777053653472379841).

### Triggered candidates
No candidate triggers fired this heartbeat.

- `lifestyle`, `property-services`, `elder-care`, `family-with-kids` — all gated on paying-subs ≥50 (`fixed`); subs=0. **No-fire.**
- Tier 2 (self-hosted, candidate) — 3 prereqs unmet (`estimate`). **No-fire.**
- Tier 3 (hosted cloud, candidate) — gated on Tier 2 (`pending-telemetry`). **No-fire.**
- Tier 4 (hardware) — north-star, no operator initiation. **No-fire.**

### Tier readiness
- **Tier 2 (self-hosted, candidate):** 3 prereqs unmet, unchanged (`estimate`).
- **Tier 3 (hosted cloud, candidate):** gated on Tier 2, unchanged (`pending-telemetry` — scenario-to-cloud readiness not yet exposed).
- **Tier 4 (hardware, north-star):** no operator initiation, unchanged.

### Headliner watch (business bundle)
- Current headliners: `web-console`, `git-control-tower` (both `in-progress`) (`fixed`, sku-map.json).
- Nearest promotion candidate: `agent-manager` → amplifier-to-future-headliner. First-ever motion this heartbeat (Agent Sandbox Audit Foundation 2/5), but still well short of deployable. **No promotion decision** — trigger has not fired; headliners are operator-promoted.

### Mapping proposals
No mapping changes this heartbeat. Agent-manager role remains `amplifier` — 2/5 on a foundation initiative is not deployability. All 6 sku-map entries unchanged.

### Current bottleneck
`agent-manager` stabilization remains the single most load-bearing block (amplifies GCT headliner AND clears swarm-manager future-headliner blockedBy), but its leading edge has shifted from "entirely unstarted" to "Agent Sandbox Audit Foundation 2/5 + Protected Agent Sandboxing 0/3" — worth re-checking next heartbeat for further motion.

### Decisions raised this heartbeat
0 decisions. Material delta exists (Agent Sandbox Audit Foundation 0/5 → 2/5) but does not cross any promotion / mapping / retirement threshold; agent-manager is not yet deployable. Team queue: 0 pending (normal mode). Own-context cap: 0/3.

### Knowledge entry written
- topic: `catalog-snapshot-2026-04-27` (id `knw-1777314656888755232`, supersedes `knw-1777055450925377798`).