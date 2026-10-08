# Scenario-to-desktop

`scenario-to-desktop` is the deployment Tier 2 ramp. It generates, packages,
tests, and publishes Electron applications for Vrooli scenarios. It owns
desktop execution; `deployment-manager` owns the target decision and release
record.

This page is the project-level ownership summary. The maintained technical
documentation is in [`scenarios/scenario-to-desktop/docs/`](../../../scenarios/scenario-to-desktop/docs/).

## What the ramp owns

- Electron wrapper generation and platform packaging;
- target-specific bundle and runtime configuration;
- local preflight and desktop smoke journeys;
- artifact, build, and evidence records;
- signing configuration and publication handoff.

It does not own scenario business logic, resource policy, credential authority,
release approval, or remote desktop execution on an untrusted machine.

## Supported deployment shapes

| Shape | Meaning | Current claim |
| --- | --- | --- |
| Bundled private | The desktop supervisor starts verified private scenario/resource artifacts | Available for eligible plans; promotion is evidence-gated |
| External-server | The desktop shell calls a configured Tier 1 scenario API | Supported when the server route is configured and tested |
| Shared resource | The supervisor uses a broker-issued, scoped lease to a provider | Contract and fallback behavior exist; live evidence is required per release |
| Tier 2 peer | One desktop runtime calls another desktop runtime | Unsupported until a real authenticated peer protocol exists |

These shapes must not be described interchangeably. In particular, a thin
client is not an offline bundle, and a provider candidate is not a peer
protocol.

## Evidence boundary

Compile and package results are not native runtime evidence. A desktop release
claim requires the machine assertions and reviewer-visible journey evidence
defined by the [desktop evidence and communication contract](../../reference/scenario-to-desktop-evidence-and-tier-contract.md).

Local validation runs on the current host. Remote OS validation through
`vrooli-bridge` is a separate integration and currently does not transfer a
desktop-session evidence protocol.

## Read next

- [Scenario-to-desktop documentation](../../../scenarios/scenario-to-desktop/docs/OVERVIEW.md)
- [Desktop deployment quickstart](../../../scenarios/scenario-to-desktop/docs/QUICKSTART.md)
- [Deployment modes](../../../scenarios/scenario-to-desktop/docs/concepts/deployment-modes.md)
- [Desktop evidence and communication contract](../../reference/scenario-to-desktop-evidence-and-tier-contract.md)
