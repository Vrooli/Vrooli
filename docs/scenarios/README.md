# Scenario System Documentation

This section explains the scenario ecosystem at the platform level.

Scenarios are one of Vrooli's core primitives. They are how the platform turns raw capability into reusable software outcomes.

## Start Here

- [getting-started.md](getting-started.md) for creating or modifying scenarios
- [CONCEPTS.md](CONCEPTS.md) for the scenario mental model
- [storage.md](storage.md) for canonical scenario runtime storage policy
- [VALIDATION.md](VALIDATION.md) for testing and validation expectations
- [DEPLOYMENT.md](DEPLOYMENT.md) for scenario deployment framing
- [PRODUCTION_BUNDLES.md](PRODUCTION_BUNDLES.md) for UI-bundle lifecycle expectations

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
vrooli scenario validate-env <name>
vrooli scenario start <name>
vrooli scenario run <name>
vrooli scenario test <name>
vrooli scenario logs <name>
template-manager registry list --kind scenario
template-manager generate <template> --id <slug> --display-name <name> --description <text>
vrooli scenario requirements report
vrooli scenario requirements validate
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
- [../../packages/api-core/docs/storage.md](/home/matthalloran8/Vrooli/packages/api-core/docs/storage.md)

## Documentation Boundary

This folder documents the scenario system as a whole.

- project-level docs explain the platform
- this folder explains how scenarios work across the platform
- each scenario's own docs explain that specific scenario

Keep this folder focused on canonical cross-scenario guidance. If a document only applies to one scenario, move it into that scenario instead of expanding this layer.
