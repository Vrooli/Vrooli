# Resource System Documentation

This section explains the resource ecosystem at the platform level.

Resources are one of Vrooli's core primitives. They provide the raw capabilities that scenarios compose into products, tools, and operator workflows.

Resource CLI behavior is manifest-driven. Implemented resources declare their CLI contract explicitly in `resources/<name>/resource.json` rather than relying on `path:resources/<name>/cli` layout folklore.

## Start Here

- [configuration.md](configuration.md) for current configuration and dependency guidance
- [storage.md](storage.md) for target resource runtime storage policy
- [architecture.md](architecture.md) for target native-Go resource implementation structure
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
vrooli resource restart <name>
vrooli resource start-all
vrooli resource stop <name>
vrooli resource stop-all
vrooli resource logs <name>
vrooli resource enable <name>
vrooli resource disable <name>
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

Canonical resource templates emit the same shared CLI manifest shape used by scenarios:

- explicit `cli.command`
- explicit adapter metadata
- explicit install steps
- explicit invocation policy
- explicit freshness inputs

Most generated resource CLIs remain thin control-plane delegates, not scenario-style API clients. The explicit exception is the `native-cli` archetype for repo-owned Go resource binaries with richer operator surfaces.

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
