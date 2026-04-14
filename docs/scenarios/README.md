# Scenario System Documentation

This section explains the scenario ecosystem at the platform level.

Scenarios are one of Vrooli's core primitives. They are how the platform turns raw capability into reusable software outcomes.

## Start Here

- [getting-started.md](getting-started.md) for creating or modifying scenarios
- [CONCEPTS.md](CONCEPTS.md) for the scenario mental model
- [VALIDATION.md](VALIDATION.md) for testing and validation expectations
- [DEPLOYMENT.md](DEPLOYMENT.md) for scenario deployment framing

## Current Truth

At the platform level, scenarios should be understood as:

- complete applications or focused services
- orchestrators of resources and sometimes other scenarios
- first-class platform assets, not throwaway demos
- inputs to testing, governance, deployment, and recursive improvement loops

Some scenarios are user-facing products. Others are meta-scenarios that improve Vrooli itself.

## Common Commands

```bash
vrooli scenario list
vrooli scenario info <name>
vrooli scenario status <name>
vrooli scenario start <name>
vrooli scenario run <name>
vrooli scenario test <name>
vrooli scenario logs <name>
vrooli scenario template list
vrooli scenario generate <template> --id <slug> --display-name <name> --description <text>
```

Preferred day-to-day workflow for one scenario:

```bash
cd scenarios/<scenario-name>
make start
make test
make logs
make stop
```

## Key References

- [../deployment/README.md](../deployment/README.md)
- [../TESTING.md](../TESTING.md)
- [../resources/README.md](../resources/README.md)
- [../reference/cli-commands.md](../reference/cli-commands.md)

## Documentation Boundary

This folder documents the scenario system as a whole.

- project-level docs explain the platform
- this folder explains how scenarios work across the platform
- each scenario's own docs explain that specific scenario
