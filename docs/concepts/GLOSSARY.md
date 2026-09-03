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

## Supervision Set

The computed set of scenarios and resources Vrooli must observe. It starts at
the operator-granted `core.seed` and follows canonical manifest edges whose
effective supervision intent is `must_start` or `try_start`. The set is an
output of `vrooli supervision-set`, not a stored roster owned by autoheal.

## Supervision Intent

The single lifecycle meaning derived for one dependency edge: `must_start`,
`try_start`, or `ignore`. It is computed from `enabled`, `required`, and
`startup_policy` under the documented precedence table; it is not a fourth
independent manifest field.

## Ownership Record

A durable control-plane record that claims a live PID for a managed resource,
resource companion, or scenario process. When PID and start-time evidence
match, this record is more authoritative than process ancestry for orphan
classification. Dead and PID-reused records do not grant ownership.

## Attribution Chain

The ordered explanation from one supervision-set member back to the
operator-granted seed that included it. Each link names the member, kind,
declaring scenario, effective supervision intent, and source; the final link is
`core.seed`.

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

## Binding Ceiling

A storage ceiling that is below current measured usage and can therefore select
bytes for enforcement.

## Measured-Value Ceiling

A ceiling copied from the usage it is meant to bound. It cannot detect excess
because the copied measurement already satisfies it.

## Policy Reconciliation

The policy-load operation that adds missing provider IDs to a persisted cleanup
policy using profile defaults while preserving deliberate existing choices.

## Owner-Delegated Provider

A cleanup provider that asks its owning scenario to choose and remove domain
objects through the shared estimate, preview, apply, and verify contract.

## Undeclared Workload

A container, process, service unit, scheduled task, or installed binary that does not match a workload Vrooli currently declares. It is classified as either unmanaged or abandoned.

## Unmanaged

An undeclared workload that matches nothing Vrooli has ever declared. It is never automatically disposed; the host workload posture controls whether it is a finding or an informational record.

## Abandoned

An undeclared workload traceable to a Vrooli manifest, scenario, resource, or agent experiment that no longer exists. It may receive a tiered-consent disposal proposal.

## Declared Workload

A live container, process, unit, or task matching an enabled Vrooli manifest, a safeguard-installed unit, or a live scenario-runtime lease.

## Crash-Looping

A declared workload whose restart rate exceeds the configured bar during its sustain window. Restart count is the signal; CPU and memory are separate signals.

## Stranded Memory

Anonymous memory held in swap while a process is idle and its resident size is far below its swapped size. It is a reclaimable condition, not automatically a leak.

## Evicted Service

A Vrooli service identified as holding stranded memory and eligible for recycling through the reclaim action.

## Unread

A sensor state meaning the operating system could not answer a field. Unread is distinct from zero and must never satisfy a numeric threshold.

## Rediscovery Gate

The acceptance phase that runs detection against captured host fixtures and independently reports the findings a human observed, without naming the signals in advance.

## Reclaim

The action that returns stranded memory to the host by recycling an evicted service. It is not `swapoff`.

## Disposal

Tiered-consent removal of an abandoned workload through a cleanup provider with preview, audit, and operator approval.

## Orphan

An unparented Vrooli-owned process as computed by the legacy process snapshot. It is not the term for an undeclared workload.
