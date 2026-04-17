# Build And Validation

This page describes the current project-level build and validation surface.

## Current Truth

At the project level, Vrooli does have real build and validation steps.

- `make build` builds the project-level Go binaries
- `make install` installs them into `~/.vrooli/bin`
- `make test` runs the project-level Go test surface
- `vrooli build` is the root CLI build command

This is distinct from older documentation that described the entire platform as having “no build step.”

## Core Commands

```bash
make build
make install
make test
```

You can also use the root CLI:

```bash
vrooli build
```

## What `make build` Does

The project Makefile builds:

- `vrooli`
- `vrooli-api`

These are the project-level Go entrypoints under `cmd/`.

## What `make test` Does

The project-level test target is the canonical project test entrypoint.

It covers project-level Go test surfaces. Validation that has its own dedicated policy target should still be run explicitly when your change touches that area.

For scenario-level testing, use:

```bash
vrooli scenario test <name>
```

Or the preferred scenario-local flow:

```bash
cd scenarios/<scenario-name>
make test
```

## What `vrooli build` Means

`vrooli build` is the root CLI build lifecycle command. Treat the CLI help and current lifecycle definitions as the final authority for its exact behavior.

Use:

```bash
vrooli build --help
```

## CLI Freshness Versus Runtime Freshness

Installed scenario CLI freshness and scenario runtime freshness are intentionally separate.

- Installed scenario CLIs are owned by `internal/cliinstall`.
- `vrooli scenario ...` command entrypoints ensure the scenario CLI is installed and current before use.
- Lifecycle runtime setup checks (`lifecycle.setup.condition.checks`) should only describe artifacts that affect the running scenario itself, such as API binaries, UI bundles, data directories, or required files.

`type: "cli"` is therefore not treated as a runtime setup input during lifecycle start/restart decisions. That boundary prevents dependency restart loops where lifecycle marks a dependency stale because its installed CLI changed, but the dependency setup phase only rebuilds API/UI artifacts and never refreshes the CLI.

## Build Versus Deployment

Do not conflate project builds with deployment portability.

- project builds produce and validate project-level binaries
- deployment portability is governed by the Deployment Hub and target-tier maturity

See [../deployment/README.md](../deployment/README.md).
