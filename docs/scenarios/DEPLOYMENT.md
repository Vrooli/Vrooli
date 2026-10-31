# Scenario deployment contract

This page defines what a scenario author must declare. It does not describe
how a packager implements a target.

The project-level deployment model and maturity status live in the
[Deployment Hub](../deployment/README.md). Target-specific implementation
guides live with the owning `scenario-to-*` scenario.

## Current baseline

The supported reference path is **deployment Tier 1: the local Vrooli stack**.
Scenarios run through the lifecycle-managed control plane and may use the
resources available to that installation.

Portability beyond that baseline is target-specific. A scenario is not desktop,
mobile, cloud, or appliance-ready merely because its Tier 1 process starts or
its UI builds.

## Author responsibilities

A scenario that wants a target evaluated must provide:

- honest dependencies in `.vrooli/service.json`;
- a deployment profile or target declaration appropriate to the target;
- target-specific credentials and secret strategies;
- migrations or compatibility notes for any dependency swap;
- `bas/` workflows for the user journeys that prove the scenario works;
- documented limitations when a capability is conditional, degraded, or remote.

The author must not silently omit dependencies to improve a fitness score or
describe a remote capability as offline.

## Ownership

| Concern | Owner |
| --- | --- |
| Scenario behavior and dependency declarations | Scenario author |
| Dependency graph and target fitness | `scenario-dependency-analyzer` |
| Approval, release gate, and release record | `deployment-manager` |
| Target packaging and native runtime | `scenario-to-*` ramp |
| Credential classification and secure storage | `secrets-manager` and the credential authority |
| Journey and target evidence | Test Genie, evidence producers, and the ramp |

Scenario code should consume these contracts. It should not grow a second
packaging or release system.

## Target references

- [Deployment Hub](../deployment/README.md)
- [Resource deployment contract](../resources/deployment-contract.md)
- [Credential configuration](../configuration/secrets.md)
- [Desktop evidence contract](../reference/scenario-to-desktop-evidence-and-tier-contract.md)
- [Production UI bundle expectations](PRODUCTION_BUNDLES.md)
