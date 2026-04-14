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
vrooli orphans
vrooli diagnose-port
```

These are the preferred first-line tools for stale lock files, orphaned processes, and port conflicts.

## Guidance

- Prefer current CLI diagnostics over older script debugging recipes.
- Treat deployment-specific troubleshooting as tier-aware; not every old deployment guide reflects current supported behavior.
- If the issue is clearly deployment-related, cross-check the Deployment Hub before assuming the old devops pages are authoritative.

## Related

- [../guides/development-environment.md](../guides/development-environment.md)
- [logging.md](logging.md)
- [../deployment/README.md](../deployment/README.md)
