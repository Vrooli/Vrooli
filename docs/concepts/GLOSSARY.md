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

## Operating Mode

A reusable, inspectable, testable **methodology loop for agentic software engineering** — the repeatable state-machine a human runs when driving coding agents (what unit of work an operator-and-agent pair operates on, in what phases, with what checkpoints, and how the loop reacts when a phase doesn't converge). Operating modes are a core project capability because `swarm-manager` is the surface through which all agentic work runs, so *how* that work is driven must be first-class: a mode is **data** (a folder validated by a JSON Schema) interpreted by **one generic engine**, not hardcoded logic — legible at a glance, simulable before use, and robust against unreliable model output via a resolution ladder. Three modes exist today: `item-level`, `holistic-loop`, `phased-plan-drain`. See [`scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md`](../../scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md).

## Projection

One of three measurable views of how ready the project is for self-driven software engineering: **Answer** (can the project be understood? — owned by `search-hub`), **Validate** (can a change be verified and auto-fixed? — owned by `test-genie`), and **Guide** (is there a skill to guide each task? — owned by `prompt-manager`). Projections partition the *questions* an engineer asks, not the tools that answer them. Each is measured against a **space** — its denominator, the enumerable set of everything it should eventually cover — and a capability matures along the gradient Guide → Validate → Answer. See [`RECURSIVE_SELF_IMPROVEMENT.md`](RECURSIVE_SELF_IMPROVEMENT.md).

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
