# Vrooli Glossary

## Control Plane

The root project-level system responsible for setup, lifecycle, diagnostics, and governance. In Vrooli this is centered on the Go-native `vrooli` CLI and supporting project internals.

## Resource

A local or connected service that provides a dependency such as storage, inference, automation, search, or secret management. In harness declarations, `resource` names that dependency relationship; it does not mean a coding-agent harness itself.

## Commercial plan

A billing offer or subscription plan. Use **implementation plan** for a Plan Manager artifact; do not use bare “plan” when both meanings are possible.

## Optional component

A separately installable desktop dependency. Use “RCL asset” for a React Component Library component so the two meanings remain distinct.

## Component catalog

The versioned catalog of optional desktop components, their supported ranges, and installation metadata. It does not describe RCL assets. See [desktop packaging](../architecture/packaging-strategies.md).

## Supplier

A company that supplies model access or routing. In the AI provider contract, `provider` is the technical provider family/configuration; use “supplier” for the company.

## Provider rule

A data record that states when a model provider may serve a verified AI use class, including its terms and eligibility conditions. An agent can propose a rule, but only an operator can approve it. See the [AI provider contract](../resources/ai-provider-contract.md).

## Eligible route

A route that passes the shared AI-use eligibility decision. Use “HTTP route” for an API endpoint and “router” for a routing service or harness selection.

## Feature route

The API endpoint that invokes a declared paid feature contract, such as `/v1/features/<id>`. It is distinct from an eligible model route, which is selected by AI provider policy.

## Feature contract

The declared behavior and billing boundary for a paid application feature. “Capability” retains the existing Vrooli meaning of reusable problem-solving power; it is not a synonym for a paid feature.

## Route eligibility

Whether a provider route is allowed for a verified AI use class and request. This is distinct from agent-run eligibility and reclaim eligibility.

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

## Learn verb

A governed `learn.*` operation inside a Program Runtime program for task identity, step capture, recall, choice, typed notes, outcomes, inference, fragments, or delegation.

## Fragment

A normalized `def step(inputs, bindings): ...` function used to fill a `learn.act` hole through its declared binding allow-set.

## Step key

The stable identity formed from an operation, bounded input digest, and step name so learning evidence can be recalled across runs.

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

## Operation Tier

The sensitivity class of a cross-scenario operation — **read**, **cheap write**, or **costly write** — that decides how much authority it takes. Reads and cheap writes need only a verified caller; costly writes (create a run, global management, approvals) additionally need authority for the specific operation. See [Authorization](AUTHORIZATION.md).

## Operation Grant

The single representation of authority for a costly operation: a bounded operation + target + limits + expiry tied to an accountable owner, evaluated by `OperationGrant.Allows`. See [Authorization](AUTHORIZATION.md).

## Standing Grant

An operator-approved, durable Operation Grant keyed to a caller's verified identity, letting a trusted service or unattended agent perform a bounded costly operation (such as a team-heartbeat run) without a human present each time — explicit, limited, attributable, and revocable. See [Authorization](AUTHORIZATION.md).

## Operating Mode

Historical name for Swarm Manager's declarative agent-work state machines.
The legacy operating-mode runtime has been removed. Agent Manager now executes
declared workflows, while Swarm Manager owns work authorization, acceptance,
and terminal state. See [Swarm Manager architecture](../../scenarios/swarm-manager/docs/concepts/ARCHITECTURE.md#typed-workflow-pilots)
for the current boundary; do not author new work against the retired mode model.

## Projection

One of three measurable views of how ready the project is for self-driven software engineering: **Answer** (can the project be understood? — owned by `search-hub`), **Validate** (can a change be verified and auto-fixed? — owned by `test-genie`), and **Guide** (is there a skill to guide each task? — owned by `prompt-manager`). Projections partition the *questions* an engineer asks, not the tools that answer them. Each is measured against a **space** — its denominator, the enumerable set of everything it should eventually cover — and a capability matures along the gradient Guide → Validate → Answer. See [`RECURSIVE_SELF_IMPROVEMENT.md`](RECURSIVE_SELF_IMPROVEMENT.md).

## Tier 1 Deployment

The current mature deployment path: a full Vrooli stack running locally or on a development server, typically exposed through the documented remote-access patterns.

## Tiered Deployment

The model where different targets such as local stack, desktop, mobile, SaaS, and appliance deployments are treated as distinct deployment tiers with different fitness, secret, and dependency rules.

## Coding-agent harness

A user-operated coding agent, such as OpenCode, Codex, Claude Code, Grok, or Antigravity, that can be discovered and launched by a Vrooli app. A harness is distinct from its model provider: the harness owns its interaction and coding workflow, while provider configuration determines where model requests go. See [Coding agents](../configuration/coding-agents.md).

## Model source

The origin used to serve a model request: the user's account, a user-supplied key, local compute, or Vrooli credits where the use is eligible. A model source is distinct from the coding-agent harness that sends the request. See [connections](../configuration/integrations/connections.md) and the [agent policy runtime](../architecture/agent-policy-runtime.md).

## Found copy

A detected installation of a coding-agent harness. Detection does not mean Vrooli manages, can update, or owns that installation.

## Managed copy

A coding-agent harness installation for which Vrooli has an identified lifecycle owner and can report supported management actions. A found copy becomes managed only when its ownership and capabilities are established; do not infer management from detection alone. See [host tools](../configuration/host/tools.md#harness-state).

## Vrooli credits

The prepaid balance used for eligible AI requests routed through Vrooli's billing authority. Credits are not a coding-agent subscription and do not imply that every harness or provider is eligible. See [paid features](PAID_FEATURES.md) and the [AI provider contract](../resources/ai-provider-contract.md).

## Billing authority

The service boundary that decides whether a paid feature use is allowed, reserves and settles its charge, and records its usage receipt. This is the precise term for the billing concept; do not use “money boundary.” See [paid features](PAID_FEATURES.md).

## AI use class

A policy category describing why an AI request happens, as far as the server can verify. The classes are `own_account`, `bounded_feature`, `in_app`, and `harness_access`; they are distinct from deployment tiers and from a coding-agent harness. A request cannot choose its own class. See the [AI provider contract](../resources/ai-provider-contract.md).

## Usage receipt

The usage and cost record returned for an AI operation. This is distinct from a payment receipt, a signed entitlement lease, or an update verification receipt; qualify the kind of receipt in technical documentation.

## Deployment tier

A deployment target category. Prefer the specific names “full local deployment”, “desktop deployment”, “hosted deployment”, or “mobile deployment”; do not use bare “tier” for a deployment target, subscription rank, operation sensitivity class, or model-routing preference. Use “operation tier” only for the read/cheap-write/costly-write authority classes.

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

## Epoch

One large, verifiable unit of an orchestrated delivery goal. See the
`large-effort-orchestration` skill; no other work shape uses epochs.

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

## Mark

The figure of a logo without any container: for Aquila, the white eagle outline plus the cyan constellation stars, on a transparent background. Stored as SVG and owned by brand-manager.

## Container style

The deterministic frame a mark is composed into: tile shape and corner-radius ratio, background fill (solid or gradient stops), mark padding ratio, glow settings, and safe-zone scales. A record that several brands can share.

## Product line

A set of brands that share one container style and a naming theme, each with its own mark and accent colour. The first is the star-named line: Aquila, Vega, Rigel, Mizar, Pan, Messier.

## Logo candidate

A durable brand-manager record for one proposed mark: its asset, concept label, prompt, role or model, seed, origin (generated, uploaded, edited, vectorized), parent candidate, and status (proposed, picked, rejected, superseded).

## Target profile

A named, versioned list of icon targets defined once in brand-manager, for example `web-public-v1` (the `/public/*` PWA set) and `electron-v1` (the desktop set). Sizes and wiring live in `scenarios/brand-manager/docs/concepts/BRAND-ASSET-PIPELINE.md`.

## Branding declaration

The `branding` block in a scenario's `.vrooli/service.json` that names the brand and the target profiles the scenario ships.
