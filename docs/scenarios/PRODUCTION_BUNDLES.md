# Production Bundles

This page describes the current production-bundle expectations for UI-bearing scenarios.

Use it when you are defining or validating scenario lifecycle steps that produce deployable UI assets.

It is not a general statement that every scenario must produce a deployable front-end bundle.

## Core Rule

Scenarios with a front-end should build production assets during setup and serve the built output during lifecycle-managed runs. Do not treat ad hoc dev servers as the canonical runtime path.

## Why This Exists

Production bundles matter because they make scenario behavior predictable across:

- lifecycle restarts
- cache-busting and stale-asset detection
- iframe loading and embedded UI surfaces
- deployment-oriented packaging flows
- scenario-auditor rules and auto-fix guidance

## Expected Pattern

For a typical UI scenario:

1. install front-end dependencies in an explicit `install-ui-deps` step
2. build the UI in an explicit `build-ui` step
3. serve the built output rather than a long-running dev server in normal lifecycle operation

The exact commands can vary by toolchain, but the lifecycle intent should remain stable:

- install dependencies first
- build `ui/dist` or the equivalent production artifact
- make the runtime use the built artifact
- keep scenario-local docs explicit about where the built output lives and how it is served

## Related Docs

- [../deployment/README.md](../deployment/README.md)
- [VALIDATION.md](VALIDATION.md)
- [../operations/production-guide.md](../operations/production-guide.md)
