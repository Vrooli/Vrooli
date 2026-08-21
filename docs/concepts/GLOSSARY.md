# Vrooli Glossary

## Control Plane

The root project-level system responsible for setup, lifecycle, diagnostics, and governance. In Vrooli this is centered on the Go-native `vrooli` CLI and supporting project internals.

## Resource

A local or connected service that provides raw capability such as storage, inference, automation, search, or secret management.

## Accelerator

A compute device other than the general-purpose CPU that a resource runs work
on. NVIDIA CUDA, Apple Metal, and AMD ROCm devices are accelerators.

## Backend

The named software path a resource uses to reach an accelerator. The set is
closed: `cuda`, `metal`, `rocm`, `vulkan`, `cpu`. A closed set is what makes
placement verification decidable and lets the manifest schema reject a typo;
adding a sixth is a deliberate contract change, not a manifest edit. A resource
declares its backends once, in the `acceleration` block of its `resource.json`.

## Declared mode

The backend a resource manifest asks the platform for — the first entry in its
`acceleration.backends` list. Surfaced as `declared_mode` on
`vrooli resource status`.

## Observed mode

The backend the control plane verifies a running resource is actually using,
read from the host and never inferred from configuration. Surfaced as
`observed_mode`. Empty means the placement could not be read, which is reported
as unknown rather than treated as agreement.

## Mode drift

Declared mode and observed mode disagree for a running resource: it is serving,
on a backend below the one it asked for. Surfaced as `mode_drift: true` with
`running: true`, `serving: true`, `healthy: false` and `status_code:
mode_drift`.

Read `serving` alongside `healthy`: a drifted resource answers requests, and a
consumer that restarts on `healthy: false` alone would loop against something
that is working — a restart cannot move a resource onto a device the host does
not have. Degraded is a state, never a secret.

## Scenario

A complete application or focused service that orchestrates resources and sometimes other scenarios to deliver business or platform value.

## Meta-Scenario

A scenario whose main job is to improve Vrooli itself by enhancing testing, deployment, governance, review, orchestration, or operator workflows.

## Capability

A reusable piece of problem-solving power that the system can invoke again later. In Vrooli, scenarios, workflows, packages, and well-structured tooling can all become capabilities.

## Operating Mode

A **generic, composable, plan-first agentic-SWE state machine**, expressed as data — the repeatable methodology loop a human runs when driving coding agents. A mode declares its **target** (unit of work: a plan-manager plan, a plan reference, or a goal), each phase's reads and emits, **classification-on-transition** (routing derived from the completed handoff), and one-level composition via **`executed_by`**. Operating modes are a core project capability because `swarm-manager` is the surface through which all agentic work runs. Canonical concept, vocabulary, and architecture: [`scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md`](../../scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md).

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

## Service Instance

One DNS-SD browse result: an instance name, service type, host, address list,
port, and TXT key map.

## Identity Claim

A hardware-grade key that permits observations to merge, such as an ADB serial,
an Android TV Remote Bluetooth key, or a Google Cast id. Addresses and names
are not identity claims.

## Observation Mode

How a transport learns that device state changed. `push` means the device sends
unsolicited status; `poll` means Device Control samples at a declared interval.

## Team Scope

A Source Ledger scope shared by one prompt-manager team. It holds durable team
observations, handoffs, and guidance with scoped recall and compaction budgets.

## Gated Work Item

A Swarm Manager backlog item whose execution waits for an operator disposition.
It carries the outcome, evidence, scope, dependencies, and completion condition
in the same work record the executing agent reads.
