# Scenario Deployment

This page provides the scenario-system view of deployment.

## Current Truth

The canonical deployment truth for the platform lives in:

- [../deployment/README.md](../deployment/README.md)

From the scenario-system perspective, the most important current facts are:

- Tier 1 is the mature deployment path today
- scenario-local lifecycle flows matter
- portability beyond Tier 1 depends on deployment intelligence, dependency fitness, and target-specific constraints
- scenario docs should be explicit about their intended deployment tier instead of implying universal portability

## Tier 1

For current supported operation, think in terms of a full Vrooli stack running locally or on a development server.

Useful commands:

```bash
vrooli scenario start <name>
vrooli scenario test <name>
vrooli scenario logs <name>
```

Preferred local workflow:

```bash
cd scenarios/<scenario-name>
make start
make test
make logs
make stop
```

## Beyond Tier 1

Desktop, mobile, SaaS, and appliance-style deployment paths are important, but they are not all equally mature today.

Scenario docs should therefore:

- avoid implying universal portability
- point readers to the Deployment Hub
- keep scenario-specific deployment metadata honest
- distinguish current operation from target future packaging

## Who Owns What

Scenario authors do not implement deployment. Two other scenarios own it:

- `deployment-manager` decides whether a build may ship and records what shipped. It is tier-agnostic.
- A `scenario-to-*` ramp builds, packages, signs, and publishes for one target family. It asks `deployment-manager` for a gate decision before it publishes.

A scenario being deployed supplies three things and nothing else:

- a deployment profile in `deployment-manager` naming its tier and target platforms
- `bas/` workflow assets describing the user flows that prove the scenario works
- honest dependency declarations in `.vrooli/service.json`, which feed fitness scoring

See [../deployment/README.md](../deployment/README.md) for the full plane model.

## Guidance

- treat deployment as tier-aware
- do not revive older one-size-fits-all packaging stories
- when in doubt, defer to the Deployment Hub
- keep target-specific details in the owning scenario docs once they stop being cross-scenario guidance
- do not add packaging or release logic to a scenario; it belongs to a ramp
