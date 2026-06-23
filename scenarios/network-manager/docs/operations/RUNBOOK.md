# Runbook — Network Manager

## Purpose

Operational procedures for running, diagnosing, and safely recovering Network Manager.

## Start / Stop

Use scenario lifecycle commands:

```bash
cd scenarios/network-manager
make start
make status
make logs
make stop
```

## First Snapshot

1. Confirm Network Manager API/UI health.
2. Confirm resolver backend status.
3. Run a health snapshot.
4. Review unavailable probes separately from failed probes.
5. Export the report before applying changes.

## Applying Filtering Changes

1. Create a policy change.
2. Review preview and affected devices/groups.
3. Confirm rollback availability.
4. Approve apply.
5. Run a post-apply health snapshot.
6. Roll back if connectivity or policy behavior regresses.

## Optimization Run

1. Run baseline snapshot.
2. Generate candidate configurations only from supported adapter capabilities.
3. Run candidate measurements.
4. Review score and evidence.
5. Approve or reject persistent apply.
6. Export before/after report.

## Recovery

- If DNS breaks, use the last rollback record or manual resolver instructions.
- If the resolver is unreachable, disable Network Manager policy application and inspect AdGuard Home health.
- If Home Automation events fail, keep Network Manager running and retry or skip consumer notification.

## Escalation Notes

ISP/router issues should be reported with snapshot evidence rather than vague speed claims. Do not recommend paid plan upgrades without measurement evidence.

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md)
- [`OBSERVABILITY.md`](OBSERVABILITY.md)
- [`../concepts/FLOWS.md`](../concepts/FLOWS.md)
