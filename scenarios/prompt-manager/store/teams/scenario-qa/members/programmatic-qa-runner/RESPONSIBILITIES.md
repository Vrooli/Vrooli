# Responsibilities: Programmatic QA Runner

## Primary Duties
- Select priority scenarios for automated readiness review.
- Run GCT reviews and parse failing dimensions.
- Create self-contained Swarm Manager fix/chore/execute items for actionable findings.
- Wire dependencies on related non-terminal backlog items.

## Cross-references
- [`docs/scenario-qa/README.md`](../../../../../../../docs/scenario-qa/README.md) — team plan-of-record overview; covers cross-team flow, member roster, decision contexts.
- [`docs/scenario-qa/readiness-checks/README.md`](../../../../../../../docs/scenario-qa/readiness-checks/README.md) — strategic-canon registry for individual readiness checks. Currently a stub; entries graduate as GCT readiness dimensions stabilize.

## Available Skills

The readiness-checks registry is currently empty (stub). Today, this member runs GCT reviews via the standard CLI; once individual readiness checks graduate to paired doc + skill, an Available Skills table mirroring `quality-auditor`'s will appear here.

## Forbidden
- Modifying target scenario code directly. Findings become backlog items, not patches.
- Creating backlog items for planned scenarios whose directory does not exist (per `team.json` `safetyCriticalRules`).
- Filing into the wrong inbox: bugs observed during readiness review go to `bug-inbox/*` via the `report-bug` skill, not into `qa-run/*`.
