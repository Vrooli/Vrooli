# Resource Configuration

This page describes the current configuration model for resources at the platform level.

## Core Rule

Treat manifests and current CLI behavior as authoritative. Do not treat old registry-era or shell-era configuration patterns as current truth unless they are explicitly retained.

## Current Configuration Layers

### Project-Level

- `.vrooli/service.json` contains project-level configuration, lifecycle setup, and enabled dependency intent

### Scenario-Level

- `scenarios/<name>/.vrooli/service.json` contains scenario-level dependency declarations and lifecycle metadata

### Resource-Level

- implemented resources commonly expose `resources/<name>/resource.json` as manifest authority

## Dependency Guidance

At the scenario level:

- `dependencies.resources` is keyed by canonical resource name
- `required` describes functional necessity
- `startup_policy` describes startup behavior

Useful values:

- `must_start`
- `try_start`
- `ignore`

For the normalization details, see [dependency-contract-audit.md](dependency-contract-audit.md).

## Validate Configuration

```bash
vrooli resource validate
vrooli resource validate <name>
vrooli resource schema validate
```

## Guidance

- prefer canonical resource names
- keep dependency declarations honest
- avoid inventing old `resource-*` alias conventions in config
- treat scenario manifests and resource manifests as living operational truth
