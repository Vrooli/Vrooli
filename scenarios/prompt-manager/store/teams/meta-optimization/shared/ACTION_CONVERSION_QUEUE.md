# Action Conversion Queue

Pipeline for moving repeated deterministic operations out of prose and into Action contracts over Vrooli-controlled CLI commands. Maintained by `skill-optimizer`.

## Candidates

| Source | Candidate Action | Target CLI | Baseline | Expected Delta | Owner | Status |
|--------|------------------|------------|----------|----------------|-------|--------|
| none | none | none | none | none | none | none |

## Blocked on CLI / Backlog

| Source | Needed CLI | Why Blocked | Next Owner |
|--------|------------|-------------|------------|
| documentation-health | A single governed documentation-health audit contract over `knowledge-observatory docs audit/health` plus any required `plan-manager` context | Current skill workflow spans two CLIs, and discovery found no exact Action. First repair stale `plan-manager author start/continue/finalize` syntax and define ownership/permissions before conversion. Baseline: 62 returns / 4 reads in 7d; graph health 0.20; command-reference 0.00. Expected delta: zero critical command findings and reduced repeated manual setup. Measure graph node, discovery, Action validate/dry-run, and post-adoption manual-vs-Action usage. | Existing backlog `documentation-health-command-drift`; owner of prompt-manager skill source |

## In Progress

| Action | Work Remaining | Owner |
|--------|----------------|-------|
| none | none | none |

## Completed

| Action | Completed At | Measurement |
|--------|--------------|-------------|
| action:scenario.status.show | 2026-05-01 | Active seed Action; dry-run validates command rendering. Post-adoption usage not yet measured. |
| action:team.swarm.work.list | 2026-05-01 | Corrected to stale-register after live registry reconciliation; absent from the live Action registry and has no post-adoption usage baseline. |

## Rejected / Retired

| Candidate | Reason | Replacement |
|-----------|--------|-------------|
| none | none | none |

## Measurement Follow-Up

- Every candidate must include current token/manual-work cost, expected delta, and a post-adoption check.
- A candidate may not move to completed until it has validation evidence and at least one discoverability reference.
