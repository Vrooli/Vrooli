# Documentation Seams

This file defines the boundaries between documentation layers.

## Project-Level Docs

Project-level docs should explain:

- what Vrooli is
- how the platform is organized
- how the root CLI and control plane work
- how scenarios and resources relate
- what deployment maturity looks like at a high level

They now primarily live under:

- `path:docs/concepts/`
- `path:docs/guides/`
- `path:docs/reference/`
- `path:docs/operations/`
- `path:docs/deployment/`
- `path:docs/strategy/`

## Scenario-System Docs

`path:docs/scenarios/` should explain:

- how the scenario ecosystem works
- scenario authoring norms
- scenario validation and deployment patterns

Individual scenario docs should explain the specific scenario.

## Resource-System Docs

`path:docs/resources/` should explain:

- how resources are modeled
- resource templates and blueprints
- resource governance and lifecycle policy

Individual resource docs should explain the specific resource.

## Plans

`path:docs/plans/` are not current truth by default. They are design and migration artifacts unless a canonical doc explicitly points to them as active.
