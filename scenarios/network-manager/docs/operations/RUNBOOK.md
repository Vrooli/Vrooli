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
2. Run the first read-only health snapshot before any resolver, policy, or optimization mutation:

```bash
network-manager snapshot run --profile home --json
```

3. Treat the first persisted snapshot with `status=baseline` as the baseline anchor for future optimization comparisons.
4. Review unavailable/unsupported probes separately from failed probes; unsupported probes are capability gaps, not failed network measurements.
5. Export the report before applying changes:

```bash
network-manager snapshot export <snapshot-id> --format markdown
```

No optimization candidate should run until at least one baseline snapshot exists.

## Privacy Defaults

1. Confirm query-log visibility and retention before broad UI/device exposure:

```bash
network-manager privacy retention --json
network-manager privacy visibility --json
```

2. Keep `home-minimal` unless the operator explicitly needs a longer small-office audit profile.
3. Treat query-log retention as disabled unless a governed query-log source is added; current sweeps record this as a no-op and prune expired non-baseline snapshots.

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

## Home Automation Integration

1. List the exported Home Automation action contract:

```bash
network-manager home actions --json
```

2. Invoke read-only health actions freely. Invoke write-intent actions only with explicit approval, and expect `manual_required` until a governed resolver/router adapter proves support:

```bash
network-manager home invoke network.health.run --json
network-manager home invoke network.adblock.pause_device --approved --json
```

3. Review recent redacted events if Home Automation publication is unavailable:

```bash
network-manager home events --json
```

## Recovery

- If DNS breaks, use the last rollback record or manual resolver instructions.
- If the resolver is unreachable, disable Network Manager policy application and inspect AdGuard Home health.
- If Home Automation events fail, Network Manager records `publish_failed` internally and keeps the core workflow running; inspect recent events and retry the consumer notification path after Home Automation is healthy.

## Escalation Notes

ISP/router issues should be reported with snapshot evidence rather than vague speed claims. Do not recommend paid plan upgrades without measurement evidence.

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md)
- [`OBSERVABILITY.md`](OBSERVABILITY.md)
- [`../concepts/FLOWS.md`](../concepts/FLOWS.md)
