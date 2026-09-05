# Scenario-to-desktop documentation

`scenario-to-desktop` is Vrooli’s deployment Tier 2 ramp. It turns a scenario
into an Electron desktop application and records the build, target decision,
preflight, native journey, and publication evidence.

This folder is the source of truth for desktop implementation. The project
Deployment Hub defines the cross-target model; deployment-manager defines
governance and release approval.

## Start with the path you need

| Need | Start here |
| --- | --- |
| Understand the two desktop modes | [Deployment modes](concepts/deployment-modes.md) |
| Build a first application | [Quickstart](QUICKSTART.md) |
| Add menus, dialogs, tray, or notifications | [Desktop integration](guides/desktop-integration.md) |
| Package or troubleshoot an artifact | [Build and packaging](guides/build-and-packaging.md) |
| Build for another operating system | [Cross-platform builds](guides/cross-platform-builds.md) |
| Configure updates | [Auto-updates](guides/AUTO_UPDATES.md) |
| Diagnose a bundled AppImage | [Bundled-app logging](guides/logging-bundled-desktop.md) |
| Validate an interactive desktop session | [Interactive desktop](guides/interactive-desktop.md) |
| Understand the runtime supervisor | [Runtime README](../runtime/README.md) |
| Inspect the API or CLI | [API contract](reference/api-contract.md) or [CLI reference](reference/cli-commands.md) |
| Understand release evidence | [Smoke-test pipeline](reference/smoke-test-pipeline.md) and the [canonical evidence contract](../../../docs/reference/scenario-to-desktop-evidence-and-tier-contract.md) |

## Agent usage and improvement

Read `prompt-manager skill read scenario-to-desktop` for usage. Read
`prompt-manager skill read scenario-to-desktop-improve` for improvement cycles.
Both skills live under this scenario's `skills/` directory and are declared in
`.vrooli/service.json`. Usage and improvement records share the
`scenario-to-desktop-usage` memory scope.

The scenario owns three read-only programs under `.vrooli/program-runtime/`:

| Program | Purpose |
|---|---|
| `scenario-to-desktop.pipeline-inspect` | Resolve one pipeline and inspect its stage state and bounded investigation sample |
| `scenario-to-desktop.evidence-inventory` | Cross-check retained capture counts and return five recent metadata references |
| `scenario-to-desktop.setpoint-read` | Read external binding condition and name missing outcome measurements |

Run a program with `program-runtime library run <program> --input key=value`.
The sibling JSON contract defines its inputs, output, effects, and fixtures.
An inspection result is not a release verdict. Capture inventories include
historical evidence and do not prove the selected artifact's readiness.

An improvement goal supplies scope and authority. The improve skill supplies
sensor interpretation and repair routes. `goal-loop` supplies cycles and work
tracking; an implementation executor performs the filed code work. A successful
setpoint read does not mean the targets are met. Missing measurements, unapproved
performance baselines, and required unavailable targets remain open obligations.
The [maintained inventory](internal/PROGRESS.md#skill-and-program-setup-2026-09-04)
records the initial sensor gaps, validation-binding work, and verification.

## Current support

| Capability | Status |
| --- | --- |
| Electron wrapper generation | Implemented |
| Thin client (`external-server`) | Supported when the configured Tier 1 API is reachable and validated |
| Bundled private runtime | Implemented for eligible, verified dependency plans |
| Resource fallback | Implemented where the deployment plan declares a compatible fallback |
| Shared provider selection | Implemented as a broker/lease seam; release evidence is environment dependent |
| Tier 2 peer communication | The intended route is `[node/]scenario[@variant]` through the authenticated `vrooli-bridge` relay; full peer capability remains evidence-gated |
| Linux native launch and interaction evidence | Primary validated path when the host tools are available |
| Windows/macOS native runtime evidence | Must be produced on the target; package or compile success is not runtime proof |
| Cloud API mode | Stub; do not use |

“Implemented” means the code path and contract exist. It does not mean that a
scenario is eligible for every platform or that a release is promotable.

## Desktop validation contract

Desktop validation is a matrix of immutable cells: one generated artifact,
one target, one existing BAS journey, and one environment profile. The
scenario-to-desktop target owns Electron launch, renderer selection, loopback
CDP security, lifecycle/native evidence, and cleanup. BAS remains the owner of
semantic workflow execution and workflow artifacts; workflow-health and Test
Genie remain the owners of journey classification and leased test storage.

Each cell carries an explicit disposition (`pass`, `failed`, `degraded`,
`unavailable`, `unsupported`, `refused`, or `not-run`) and linked layered
evidence. Missing isolation, target identity, artifact binding, or required
evidence cannot be represented as a pass. The durable wire contract is
`scenario-to-desktop/v1/domain/validation.proto`.

The producer-side journey sidecar and manifest use a provider-neutral
`WorkflowExecutionReference` for semantic workflow evidence. It records the
provider's asset/execution and checksummed artifact references, then requires
the same validation run, artifact digest, target, and cell identities before a
combined cell can pass. Test Genie remains generic and does not discover,
launch, or interpret a semantic workflow provider.

The Linux native smoke path now proves generated-app lifecycle and platform
behavior for both Hello Desktop and arbitrary generated scenarios. The target
package provides an executable Electron session seam: it owns
argv-safe loopback CDP launch arguments, an ephemeral port, an isolated
user-data directory, scoped validation environment variables, exact renderer
selection from `/json/list`, process-group cleanup, and the explicit
`launch-electron-validation` live-desktop endpoint. It never falls back to the
first renderer. Existing scenario BAS execution and local Linux matrix
orchestration are implemented. Remote bridge desktop transport remains
implementation work: bridge dispatch is typed, but it does not claim remote
desktop streaming or video evidence.

### Phase 1 target inventory

| Target | Current evidence | Capability disposition | Next proof |
|---|---|---|---|
| Local Linux/Xvfb | Native live-desktop/smoke evidence and real Electron CDP provider run | Eligible for platform conformance and Electron workflow execution | Combined cell evidence is persisted with the provider-neutral workflow reference |
| Linux emulator | No owned emulator adapter or runtime evidence | Unavailable, never an implicit pass | Add an adapter only when its lifecycle and renderer identity are observable |
| Bridge node | Typed `EvidenceTarget` and bridge dispatch seams exist | Unsupported for remote desktop evidence until bridge transport exists | Implement authenticated bridge-owned desktop transport and reachability evidence |
| Windows/macOS | Packaging and platform enums exist; no target runtime evidence on this host | Unavailable for release claims | Capture target-native launch, update, native-surface, and shutdown evidence on each target |

The initial representative journey remains the existing
`browser-automation-studio:bas/cases/01-foundation/01-projects/new-project-dialog-open.json`
case plus the platform lifecycle smoke journey. The real Electron run, routed
mutation proof, and provider-owned reference are complete in one matrix cell.
The provider reference is an opaque workflow execution reference; Test Genie
does not inspect or execute BAS internals.

## Choose a mode

### Bundled private runtime

Use bundled mode when the application must run without a Tier 1 server. The
bundle contains the UI, scenario services, and only the resource artifacts
selected by the immutable deployment plan.

Bundled mode can claim offline operation only when every required capability has
a verified local route and the native journey proves it. A remote API, remote
secret, cloud model, or host-only tool remains an explicit dependency.

The desktop supervisor owns only private, verified services. It allocates ports,
starts services in dependency order, applies migrations, resolves credentials,
checks native readiness, records logs, and shuts services down cleanly.

### External-server thin client

Use this mode when the application connects to an existing Tier 1 server. The
desktop shell calls the configured scenario API. Tier 1 owns the API, data,
resources, credentials, and lifecycle.

Thin client mode is not offline mode. It must show server-unavailable,
authentication, and reconnecting states instead of silently using stale local
assumptions.

### Shared providers and peers

Shared resource use requires an authenticated broker decision, user consent, a
scoped lease, and an expiry. The desktop runtime does not receive a root
credential or provider lifecycle authority.

Another desktop runtime is reached through the node-axis resolver and the
authenticated `vrooli-bridge` relay for bounded scenario calls. Do not mark the
full Tier 2 peer capability as supported until discovery, identity,
authentication, capability negotiation, scoped authority, retry/timeout,
cancellation, replay protection, failure isolation, and shutdown are defined
and evidenced on both sides. The relay route is not desktop-session evidence.

## Release truth

The release pipeline separates these claims:

- source compilation;
- package creation;
- artifact integrity and release trust;
- host/dependency eligibility;
- native runtime behavior;
- communication behavior;
- visual/user journey evidence;
- promotion eligibility.

A successful window launch or recording does not prove communication. Unsupported
and unavailable are terminal evidence states, not degraded passes. See the
[desktop evidence and communication contract](../../../docs/reference/scenario-to-desktop-evidence-and-tier-contract.md).

## Ownership boundaries

- `deployment-manager` owns profiles, fitness, approval gates, and release records.
- `scenario-dependency-analyzer` owns dependency graph and target-fitness inputs.
- `scenario-to-desktop` owns Electron generation, packaging, runtime execution,
  smoke journeys, and desktop publication handoff.
- `secrets-manager` and the credential authority own credential declaration,
  classification, storage, recovery, and diagnosis.
- `vrooli-bridge` can provide trusted remote execution, but it does not yet
  provide a desktop-session evidence-transfer protocol.

For project-level context, read the [Deployment Hub](../../../docs/deployment/README.md).
