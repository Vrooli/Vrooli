# Resource System Documentation

This section explains the resource ecosystem at the platform level.

Resources are one of Vrooli's core primitives. They provide the raw capabilities that scenarios compose into products, tools, and operator workflows.

## Start Here

- [configuration.md](configuration.md) for current configuration and dependency guidance
- [interface-standards.md](interface-standards.md) for resource-surface expectations
- [integration-cookbook.md](integration-cookbook.md) for practical integration guidance
- [resource-blueprints.md](resource-blueprints.md) for blueprint-backed future capability records
- [resource-templates.md](resource-templates.md) for canonical scaffolding
- [resource-deprecation.md](resource-deprecation.md) for leaving the active surface safely

## Current Truth

At the platform level, resources should be understood as:

- local or connected services that provide raw capability
- managed through the Go-native `vrooli resource ...` surface
- represented by resource manifests where implemented
- part of a tiered and evolving ecosystem, not one frozen inventory model

Examples include:

- databases
- vector stores
- inference systems
- browser automation
- secret management
- supporting infrastructure services

## Common Commands

```bash
vrooli resource list
vrooli resource info <name>
vrooli resource status
vrooli resource validate
vrooli resource start <name>
vrooli resource start-all
vrooli resource stop <name>
vrooli resource stop-all
vrooli resource logs <name>
```

Blueprint and template workflows:

```bash
vrooli resource blueprint list
vrooli resource blueprint info <name>
vrooli resource blueprint search <query>
vrooli resource blueprint validate
vrooli resource template list
vrooli resource template show <template>
vrooli resource template validate
vrooli resource template generate <template> --name <name>
```

Archive and schema workflows:

```bash
vrooli resource deprecate <name>
vrooli resource list-deprecated
vrooli resource archive-to-blueprint <name>
vrooli resource list-blueprint-archived
vrooli resource schema validate
vrooli resource schema sync
```

## Documentation Boundary

This folder documents the resource system as a whole.

- project-level docs explain the platform
- this folder explains how resources work across the platform
- individual resource docs under `resources/<name>/README.md` explain the specific resource

Keep this folder focused on canonical cross-resource guidance. Migration notes or one-off resource design detail should live with the resource or the owning plan.
