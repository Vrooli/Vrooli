# Scenario Concepts

This page describes the current mental model for scenarios in Vrooli.

## What A Scenario Is

A scenario is a complete application or focused service that composes resources and sometimes other scenarios to deliver useful behavior.

A scenario may include:

- a UI
- an API
- a CLI
- tests
- requirement files
- manifests
- deployment metadata
- initialization or runtime assets

## Why Scenarios Matter

Scenarios are the main way Vrooli turns infrastructure and tooling into reusable capability.

That means a scenario can be:

- a product
- an internal tool
- an operator surface
- a validation target
- a building block for future scenarios

## Scenarios Versus Resources

- resources provide raw capability
- scenarios compose capability into usable outcomes

Examples of resources:

- databases
- vector stores
- inference services
- browser automation
- secret management

Examples of scenario outcomes:

- a web console
- a deployment planner
- a testing system
- a business workflow application

## Meta-Scenarios

Not all scenarios are customer-facing products.

Some scenarios primarily improve the platform itself through:

- testing
- review
- requirement coverage
- deployment planning
- observability
- coordination and orchestration

These meta-scenarios are part of the recursive-improvement story of the platform.

## Process Management

This section covers running a scenario, not evolving one. For the development
ladder — which layer of a scenario to change and in what order — use
`prompt-manager skill read scenario-work-ladder`.

At the operational level, scenarios are managed through:

- `vrooli scenario ...`
- scenario-local Makefiles

The preferred local workflow for one scenario is:

```bash
cd scenarios/<scenario-name>
make start
make test
make logs
make stop
```

## Documentation Rule

Do not describe scenarios as if they all share one rigid internal shape.

The right way to document them is:

- explain the scenario system at the platform level here
- document current template and generation flows in [getting-started.md](getting-started.md)
- document cross-scenario validation and deployment expectations in [VALIDATION.md](VALIDATION.md) and [DEPLOYMENT.md](DEPLOYMENT.md)
- document scenario-specific implementation details inside each scenario
