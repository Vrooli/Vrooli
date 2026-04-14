# Development Workflows

This page describes the preferred day-to-day development workflows for the current platform.

## Project Workflow

Use this when working on the root control plane, project-level docs, or shared platform concerns:

```bash
make setup
vrooli develop
vrooli status
make test
```

## Scenario Workflow

Use this when working on one scenario:

```bash
cd scenarios/<scenario-name>
make start
make test
make logs
make stop
```

Use the root CLI when you need cross-scenario inspection:

```bash
vrooli scenario list
vrooli scenario info <name>
vrooli scenario status <name>
```

## Resource Workflow

Use this when working on resources or resource availability:

```bash
vrooli resource list
vrooli resource status
vrooli resource start-all
vrooli resource start <name>
vrooli resource logs <name>
```

## Validation Workflow

Project-level:

```bash
make test
make validate-repo-contract
make validate-package-governance
```

Scenario-level:

```bash
vrooli scenario test <name>
```

## Workflow Rules

- Prefer canonical CLI and Makefile surfaces over ad hoc scripts.
- Prefer scenario-local lifecycle commands for scenario work.
- Use the Deployment Hub when making claims about packaging or deployment maturity.
- Do not revive outdated target matrices unless the current docs explicitly support them.
