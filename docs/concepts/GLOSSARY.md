# Vrooli Glossary

## Control Plane

The root project-level system responsible for setup, lifecycle, diagnostics, and governance. In Vrooli this is centered on the Go-native `vrooli` CLI and supporting project internals.

## Resource

A local or connected service that provides raw capability such as storage, inference, automation, search, or secret management.

## Scenario

A complete application or focused service that orchestrates resources and sometimes other scenarios to deliver business or platform value.

## Meta-Scenario

A scenario whose main job is to improve Vrooli itself by enhancing testing, deployment, governance, review, orchestration, or operator workflows.

## Capability

A reusable piece of problem-solving power that the system can invoke again later. In Vrooli, scenarios, workflows, packages, and well-structured tooling can all become capabilities.

## Tier 1 Deployment

The current mature deployment path: a full Vrooli stack running locally or on a development server, typically exposed through the documented remote-access patterns.

## Tiered Deployment

The model where different targets such as local stack, desktop, mobile, SaaS, and appliance deployments are treated as distinct deployment tiers with different fitness, secret, and dependency rules.

## Repo Contract

The canonical structural contract that repo-aware tooling is allowed to depend on. It prevents scattered hard-coded assumptions about root layout, manifests, and path semantics.

## Package Governance

The policy layer that governs how shared packages under `packages/` are adopted, validated, refreshed, and kept from becoming uncontrolled cross-scenario coupling.

## Direct Scenario Execution

The model where scenarios are operated from their source locations rather than being described as generated throwaway outputs. In current docs, use this carefully and avoid overstating it as the only deployment story.

## Local Sovereignty

The principle that code, data, models, and automation should run on infrastructure you own or explicitly control whenever possible.

## Documentation Debt

Any documentation that is stale, duplicated, misleading, unowned, or disconnected from the code and workflows it claims to describe.
