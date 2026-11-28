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
- `enabled` controls whether the dependency participates at all
- `required` describes semantic importance
- `startup_policy` describes orchestration behavior
- `degraded_behavior` explains the fallback mode when intentional degradation is allowed

Useful values:

- `must_start`
- `try_start`
- `ignore`

Current defaults:

- if `enabled` is omitted, declared dependencies are treated as enabled
- if `startup_policy` is omitted and `required=true`, it normalizes to `must_start`
- if `startup_policy` is omitted and `required=false`, it normalizes to `ignore`

Current guidance:

- use `required=true` plus `startup_policy=must_start` for hard dependencies
- use `required=false` plus `startup_policy=try_start` for optional-but-useful dependencies
- avoid `required=true` plus `startup_policy=ignore`
- if you intentionally use `required=true` plus `startup_policy=try_start`, also declare `degraded_behavior`

The normalization work described by older audit docs has been folded into the current resource configuration and migration guidance.

## Validate Configuration

```bash
vrooli resource validate
vrooli resource validate "<name>"
vrooli resource schema validate
```

## Guidance

- prefer canonical resource names
- keep dependency declarations honest
- avoid inventing old `resource-*` alias conventions in config
- treat scenario manifests and resource manifests as living operational truth
- keep project, scenario, and resource configuration roles distinct even when they interact
