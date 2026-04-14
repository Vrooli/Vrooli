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

- `docs/concepts/`
- `docs/guides/`
- `docs/reference/`
- `docs/operations/`
- `docs/deployment/`
- `docs/strategy/`

## Scenario-System Docs

`docs/scenarios/` should explain:

- how the scenario ecosystem works
- scenario authoring norms
- scenario validation and deployment patterns

Individual scenario docs should explain the specific scenario.

## Resource-System Docs

`docs/resources/` should explain:

- how resources are modeled
- resource templates and blueprints
- resource governance and lifecycle policy

Individual resource docs should explain the specific resource.

## Plans

`docs/plans/` are not current truth by default. They are design and migration artifacts unless a canonical doc explicitly points to them as active.
