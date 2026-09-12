# Troubleshooting

This page covers the current first-line troubleshooting path for project contributors.

## Start Here

```bash
vrooli status
vrooli doctor
```

These commands should be your default first checks before digging into subsystem-specific detail.

## Common Checks

### Setup Problems

```bash
vrooli setup --help
vrooli doctor
```

If setup fails:

- confirm required host tools are present
- re-run with the intended `--environment` and `--resources` values

### Development Stack Problems

```bash
vrooli develop --help
vrooli status
vrooli stop
```

If develop behaves unexpectedly:

- stop running components and retry cleanly
- inspect whether the issue is project-level, resource-level, or scenario-level

### Scenario Problems

```bash
vrooli scenario status <name>
vrooli scenario logs <name>
vrooli scenario test <name>
```

Or:

```bash
cd scenarios/<scenario-name>
make logs
make test
```

### Resource Problems

```bash
vrooli resource status
vrooli resource logs <name>
vrooli resource restart <name>
```

## Port And Process Problems

Useful commands:

```bash
vrooli locks
vrooli locks --json
vrooli orphans
vrooli diagnose-port <port>
vrooli diagnose-port <port> --json
```

These are the preferred first-line tools for registry claim hygiene,
orphaned processes, and port conflicts.

`vrooli locks --json` returns the registry claim list:

- `registry_claims` — the **authoritative** allocation state. Every active
  scenario port should appear here as a `bound` claim attached to a supervised
  runtime lease. Important fields: `listener_status`,
  `last_listener_check_at`, `last_listener_seen_at`,
  `consecutive_listener_misses`, `lease_fresh`, `authoritative`,
  `recommendation_code`, `recommendation_confidence`.

The human-readable `vrooli locks` table hides `expired` claims by default;
pass `--all` to include them (JSON always carries the full set). The legacy
`.port_<port>.lock` file layer is retired; `vrooli cleanup locks` sweeps any
stray files left by pre-registry installs.

A declared port with repeated `not_listening` observations on a `bound` claim
may be stale manifest data, but scenario source usage and health still matter.
If a claim's `authoritative` field is `false`, run `vrooli cleanup locks` to
expire stale leases and non-authoritative claims; a `bound` claim whose
`authoritative` flag is `false` after the host rebooted should be re-adopted
by restarting the scenario through lifecycle.

To feed that evidence into static scenario-auditor validation without starting
the scenario:

```bash
vrooli locks --json > /tmp/vrooli-runtime-port-evidence.json
SCENARIO_AUDITOR_RUNTIME_PORT_EVIDENCE_PATH=/tmp/vrooli-runtime-port-evidence.json \
  scenario-auditor standards scan <scenario-name> --wait
```

Current lifecycle behavior:

- scenario startup acquires a registry `PortClaim` for each declared port
  before any process starts; another instance's active claim blocks allocation
  with a typed `active registry claim already owns port <N>` error
- startup also rolls back failed runs (releases the reserved claim, marks the
  instance failed) instead of leaving tracked process records behind
- fixed-port startup will proactively terminate clearly-owned same-scenario
  managed orphan listeners before relaunch; a foreign scenario's listener
  surfaces as a typed conflict naming the owning scenario and PID

Use `vrooli diagnose-port <port>` when the conflict is still unresolved, especially for:

- a listener owned by another scenario
- a non-Vrooli process already bound to the port
- a live listener whose ownership is unclear from the startup error alone

> If the port is in `32768-60999` on Linux, the conflict may not be a listener at all — the kernel reserves that range for outbound source ports. See [../reference/port-allocation.md](../reference/port-allocation.md) for the canonical scenario bands and how to migrate off ephemeral ports.

### Runtime Registry Schema Rebuilds

The runtime registry database is greenfield internal state at
`~/.vrooli/state/runtime.db`. If a command reports that the registry schema
requires a greenfield rebuild, do not add a checked-in migration or edit the
database in place.

Use the lifecycle system to stop affected scenarios, back up the database, and
let Vrooli recreate it from the current schema:

```bash
vrooli scenario stop <scenario-name>
cp ~/.vrooli/state/runtime.db /tmp/vrooli-runtime.db.backup.$(date -u +%Y%m%dT%H%M%SZ)
rm ~/.vrooli/state/runtime.db
vrooli locks --json
```

If preserving local runtime evidence matters, write a one-off conversion script
under `/tmp`, run it against a backup, and keep it out of the repository.

## Guidance

- Prefer current CLI diagnostics over older script debugging recipes.
- Treat deployment-specific troubleshooting as tier-aware; not every old deployment guide reflects current supported behavior.
- If the issue is clearly deployment-related, cross-check the Deployment Hub before assuming the old devops pages are authoritative.

## Related

- [../guides/development-environment.md](../guides/development-environment.md)
- [logging.md](logging.md)
- [../deployment/README.md](../deployment/README.md)
