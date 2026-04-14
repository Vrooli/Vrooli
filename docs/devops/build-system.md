# Build And Validation

This page describes the current project-level build and validation surface.

## Current Truth

At the project level, Vrooli does have real build and validation steps.

- `make build` builds the project-level Go binaries
- `make install` installs them into `~/.vrooli/bin`
- `make test` runs the retained project-level validation suite
- `vrooli build` is the root CLI build command

This is distinct from older documentation that described the entire platform as having “no build step.”

## Core Commands

```bash
make build
make install
make test
make validate-repo-contract
make validate-package-governance
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

The project-level test target runs retained validation around:

- repo contract validation
- package governance validation
- internal Go tests
- command-level tests

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

## Build Versus Deployment

Do not conflate project builds with deployment portability.

- project builds produce and validate project-level binaries
- deployment portability is governed by the Deployment Hub and target-tier maturity

See [../deployment/README.md](../deployment/README.md).
