# Logging

This page summarizes the current logging and diagnostics surfaces contributors are most likely to use.

## Project-Level Surfaces

Use:

```bash
vrooli status
vrooli doctor
```

These are the first places to inspect general platform health and environment problems.

## Scenario Logs

Preferred options:

```bash
vrooli scenario logs <name>
```

Or the scenario-local workflow:

```bash
cd scenarios/<scenario-name>
make logs
```

Scenario logs are commonly written under the Vrooli runtime area in the home directory.

## Resource Logs

Use:

```bash
vrooli resource logs <name>
vrooli resource status
```

These are the primary surfaces for resource-specific troubleshooting.

## Guidance

- Prefer Vrooli lifecycle-aware log access over jumping straight to ad hoc container assumptions.
- Treat older Docker-specific logging instructions as reference only unless the current workflow you are using explicitly depends on them.
- Do not assume one logging backend or one log file layout across all platform surfaces.

## Related

- [troubleshooting.md](troubleshooting.md)
- [development-environment.md](development-environment.md)
