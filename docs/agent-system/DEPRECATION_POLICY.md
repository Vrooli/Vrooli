# Deprecation Policy

Rules for proposing and executing deprecation of skills, agents, and teams within Vrooli's agent system.

This file is canon (plan-of-record). Edits go through `meta-optimization` decisions. Skills and team docs cite this file rather than restating the thresholds.

---

## Thresholds

Staleness windows — below these, do not propose deprecation:

| Entity | Staleness window | Notes |
|--------|------------------|-------|
| Skill  | ≥ 90 days without being referenced by any active agent | `prompt-manager graph orphaned-skills` + temporal check |
| Agent  | ≥ 90 days without a heartbeat or invocation | Checked via agent-manager run log |
| Team   | ≥ 120 days empty (no members) OR ≥ 120 days disabled with no pending decisions | Teams have longer windows because re-enablement is cheap and losing a team design is expensive |

**Rationale:** 90 days is long enough to survive one strategic pivot cycle; short enough that accretion is bounded. 120 days for teams reflects that team designs are high-effort to produce.

---

## The mandatory roadmap check

Before filing any `skill-deprecation`, `agent-deprecation`, or `team-deprecation` decision, the proposer must check:

1. **Director-swarm initiatives.** Is this entity referenced by any active initiative? (`swarm-manager initiatives list --json` and grep.)
2. **Monetization catalog.** Is this entity load-bearing for any candidate or active SKU? (Search `docs/monetization/scenario-sku-map.json` and `catalog/`.)
3. **Cross-team relations.** Does any other team's member file reference it? (`grep` across `store/teams/*/members/*/`.)
4. **Capability coverage.** Is this entity the *only* covering of a capability the system needs? If yes, the deprecation must include a successor or be rejected.

If any of 1–4 hits, do not file deprecation. Instead, file a `skill-improvement` / `agent-improvement` / `team-structure-change` to revitalize the entity, or escalate as `capability-gap` if the entity is broken beyond repair but still needed.

---

## Archive path

Once a deprecation decision is accepted by the operator:

1. **Soft archive first.** Set `status: archived` in the entity's manifest (agent.json / team.json / skill manifest). Do NOT delete files yet.
2. **Keep for 30 heartbeats.** During this grace period, any member that hits a "this used to work" error from the archive can file a `meta-self-improvement` decision to un-archive.
3. **Hard archive after 30 heartbeats.** Move the entity's files to a `store/archived/<entity-type>/<id>/` folder (to be created on first use). Update indexes. The entity no longer appears in `prompt-manager <entity> list`.
4. **Hard delete after 180 heartbeats.** Remove the archived folder. By this point nothing should reference it, and the relation files have been cleaned up.

The grace-period and hard-archive numbers (30 / 180 heartbeats) are starting points. Edge cases that suggest tightening or relaxing these numbers are tracked as team knowledge entries under topic prefix `meta-optimization/notebook/deprecation-edges/<slug>` and reviewed at the next deprecation cadence.

---

## Who files what

- `skill-deprecation` — **skill-optimizer** files it, after running the roadmap check.
- `agent-deprecation` — **team-agent-optimizer** files it, after running the roadmap check.
- `team-deprecation` — **team-agent-optimizer** files it, after running the roadmap check.

The `debt-curator` does NOT file deprecation decisions for entities outside meta-optimization — those are the owning members' lanes. The debt-curator only retires *doc entries it owns* (the inbox/synthesis layer for the meta-optimization team itself).

---

## Known failure modes

Documented here so proposals can explicitly guard against them. Every deprecation proposal must state "guards against: [none / ...]" referencing this list.

1. **Too-fast deprecation** — entity is low-usage today but load-bearing for a roadmapped initiative. Guard: the mandatory roadmap check.
2. **Capability gap by omission** — entity is the only coverage of a capability and no successor exists. Guard: check #4 above.
3. **Usage signal lag** — entity was referenced heavily last quarter but is going through a quiet phase; deprecation would re-create wheels a month later. Guard: use 90-day windows, not 30-day.
4. **Phantom references** — entity has no active references but is referenced by archived entities (graph ghosts). Guard: graph-orphaned queries must filter out archived entity references.
