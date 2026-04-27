# Dev Log Craft

Patterns observed using the `x-dev-log` skill that should feed back into improving it. Primarily maintained by `oss-advertiser` during dev-log production.

**Promotion target:** `x-dev-log` skill edits — specifically the mining-strategy rubric, interestingness-scoring weights, or output-contract fields.

**Retirement signal:** the pattern has been incorporated into the skill's prose AND ≥2 subsequent dev-log runs have consumed it without needing to re-state the workaround.

**Revisit marker (file-level):** revisit after 10 heartbeats.

## Entries

### 2026-04-26 — agent-manager outage workaround for partial-data dev logs

**Written by:** oss-advertiser
**Context:** Mining window 2026-04-24 → 2026-04-26 for the swarm-manager initiative-agents p8 thread. agent-manager scenario was `stopped` per `vrooli scenario status`, so the skill's data-source matrix lost one of its four legs (no agent-run / event / cost data available).
**Pattern:** When agent-manager is unavailable, source the thread from git-control-tower (commits, diffstat) and swarm-manager (backlog, `/api/v1/stats` throughput numbers) only. Replace agent-run stories with code-shape stories: file size deltas, retry-handler test line counts, scenario→agent dispatch wiring. Flag the draft `incomplete-data:agent-manager-unavailable` in `honesty_flags`. Skip the "agent recovered from failure" interestingness signal entirely — that one requires agent-manager event data.
**Evidence:** Draft `cd-2026-04-26-swarm-manager-initiative-agents-p8` produced this heartbeat, paired with capability-gap decision filed alongside this note.
**Target skill edit:** `x-dev-log` SKILL.md section 5 (Mining Strategy) should add a "data-source-degraded mode" sub-section listing which interestingness signals are recoverable from each remaining source. Section 6 (Interestingness Scoring) should mark the "agent recovered from failure +3" row with a note that it requires agent-manager.
**Revisit marker:** revisit after 6 heartbeats, or when agent-manager stays healthy across 3 consecutive oss-advertiser runs.

Append new entries when a dev-log production pattern emerges that isn't in the skill today:

```markdown
### <YYYY-MM-DD> — <pattern title>

**Written by:** <member-id, typically oss-advertiser>
**Context:** <which period was being mined, what the pattern emerged during>
**Pattern:** <specific technique: e.g., "group multi-scenario commits into arc threads," or "flag agent-retry stories as +4 interestingness," or "always pair a feature-shipped tweet with a 'here's how agents built it' follow-up">
**Evidence:** <dev-log run refs that demonstrated the pattern>
**Target skill edit:** <specific section of x-dev-log SKILL.md this should land in>
**Revisit marker:** revisit after N heartbeats
```
