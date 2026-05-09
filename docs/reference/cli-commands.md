# Vrooli CLI Commands

This document summarizes the current root CLI surface for the project-level platform.

Use `vrooli help` and subcommand `--help` output as the final authority for exact flags and any newly added commands.

## Root Commands

```bash
vrooli help
vrooli setup
vrooli develop
vrooli build
vrooli clean
vrooli status
vrooli stop
vrooli backup
vrooli restore
vrooli cleanup
vrooli doctor
vrooli orphans
vrooli locks
vrooli diagnose-port
vrooli contract
vrooli package
vrooli resource
vrooli scenario
```

These root commands are confirmed by the current CLI help surface.

## Orientation

```bash
vrooli help
vrooli status
```

- `vrooli help` shows the current command tree.
- `vrooli status` gives a health and status overview.

## Project Lifecycle

```bash
vrooli setup
vrooli develop
vrooli build
vrooli clean
vrooli stop
```

- `setup` initializes the development environment
- `develop` starts the project development workflow
- `build` builds project-level binaries
- `clean` removes build artifacts or runs the project clean lifecycle
- `stop` stops all or selected components

## Scenario Commands

Inspect the current surface with:

```bash
vrooli scenario --help
```

Common commands:

```bash
vrooli scenario list
vrooli scenario info <name>
vrooli scenario status <name>
vrooli scenario validate-env <name>
vrooli scenario start <name>
vrooli scenario start-all
vrooli scenario run <name>
vrooli scenario setup <name>
vrooli scenario restart <name>
vrooli scenario stop <name>
vrooli scenario stop-all
vrooli scenario test <name>
vrooli scenario logs <name>
vrooli scenario open <name>
vrooli scenario port <name>
vrooli scenario ui-smoke <name>
vrooli scenario requirements
vrooli scenario template
vrooli scenario generate <template> --id <slug> --display-name <name> --description <text>
vrooli scenario completeness <name>
```

Notes:

- `run` is an alias of `start`
- for day-to-day work on one scenario, prefer the scenario-local `make start|test|logs|stop` flow when available

## Resource Commands

Inspect the current surface with:

```bash
vrooli resource --help
```

Common commands:

```bash
vrooli resource list
vrooli resource info <name>
vrooli resource status
vrooli resource validate
vrooli resource install <name>
vrooli resource uninstall <name>
vrooli resource start <name>
vrooli resource restart <name>
vrooli resource start-all
vrooli resource stop <name>
vrooli resource stop-all
vrooli resource logs <name>
vrooli resource enable <name>
vrooli resource disable <name>
vrooli resource deprecate <name>
vrooli resource list-deprecated
vrooli resource archive-to-blueprint <name>
vrooli resource list-blueprint-archived
vrooli resource restore <name>
vrooli resource restore-blueprint <name>
vrooli resource archive
vrooli resource blueprint
vrooli resource template
vrooli resource schema
```

These commands are part of the current CLI help surface, though some of them are more administrative than day-to-day.

## Package Governance

Inspect the current surface with:

```bash
vrooli package --help
```

Common commands:

```bash
vrooli package list
vrooli package info <name>
vrooli package dependents <name>
vrooli package validate
vrooli package build <name>
vrooli package generate <name>
vrooli package refresh <name>
vrooli package audit
```

## Repo Contract

Inspect the current surface with:

```bash
vrooli contract --help
```

Common commands:

```bash
vrooli contract validate
vrooli contract show
vrooli contract resolve
vrooli contract match-glob
```

## Maintenance

```bash
vrooli cleanup
vrooli doctor
vrooli orphans
vrooli locks
vrooli diagnose-port <port>
```

These commands help inspect stale locks, orphaned processes, diagnostics, and port conflicts.

Practical guidance:

- `vrooli cleanup locks` removes stale lock files
- `vrooli orphans` inspects or terminates orphaned Vrooli-managed processes
- `vrooli diagnose-port <port>` is the targeted tool for a fixed-port startup failure after lifecycle has already attempted automatic cleanup

## Common Make Targets

The project root also exposes a few important Make targets:

```bash
make setup
make build
make install
make test
make hygiene
make validate-package-governance
```

These targets are confirmed in the current root `Makefile`.

For individual scenarios, the preferred lifecycle remains:

```bash
cd scenarios/<scenario-name>
make start
make test
make logs
make stop
```
