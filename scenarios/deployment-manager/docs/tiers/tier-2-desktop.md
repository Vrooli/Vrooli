# Deployment Tier 2: desktop

Deployment Tier 2 is a portable desktop application for Windows, macOS, or
Linux. It is a target, not a commercial subscription level. See the [Deployment
Hub](../../../../docs/deployment/README.md) for the distinction.

## Target contract

A desktop release must make its runtime shape explicit:

- **bundled private**: the application starts verified, app-private services;
- **external-server**: the shell calls a configured Tier 1 scenario API;
- **shared provider**: the app uses a broker-issued, scoped lease;
- **peer**: another desktop runtime, currently unsupported.

These routes have different ownership, data, secret, and failure semantics.
They must not be described as one generic “desktop deployment.”

## Current support

| Capability | Status |
| --- | --- |
| Electron generation and platform packaging | Implemented |
| Thin-client mode | Supported when the configured Tier 1 API is reachable and tested |
| Bundled private services | Available for eligible, verified target plans |
| Resource fallback | Available only when declared by the target plan |
| Shared Tier 1 provider | Broker/lease contract implemented; release evidence required |
| Tier 2 peer protocol | Unsupported |
| Linux native journey | Primary validated path when host capture tools are available |
| Windows/macOS native runtime | Requires native target evidence; compile/package is insufficient |

“Implemented” describes a code path. “Supported” describes an admitted route.
“Promotable” requires the release gate and evidence contract.

## Bundled target requirements

The dependency plan must identify every scenario and resource required by the
application. For each target OS and architecture it must state:

- deployment mode and support status;
- artifact identity, version, and checksum;
- privilege and host requirements;
- ports, health checks, and readiness signals;
- data directories and migrations;
- secret classification and provisioning route;
- fallback and functional limitations;
- evidence required before promotion.

The desktop runtime may start only verified private artifacts selected by this
plan. It must not build resources on the end-user machine, adopt an arbitrary
process, or scan arbitrary ports.

## Resource ownership

The desktop supervisor owns only private bundled service lifecycle. A shared
provider remains owned by its broker or Tier 1 host. A thin client does not own
the remote server’s resources. An external endpoint is attach-only unless an
explicit contract grants a narrower capability.

## Secret boundary

Infrastructure secrets never ship in the bundle. The target plan classifies each
credential as per-install generated, user-provided, remote-fetched, or excluded
as infrastructure. Storage and diagnosis follow the [credential authority](../../../../docs/configuration/secrets.md);
this page does not define a second secret store.

## Evidence boundary

The authoritative requirements for launch, dependency operation, communication,
fallback, peer status, redaction, and promotion are in the [desktop evidence and
communication contract](../../../../docs/reference/scenario-to-desktop-evidence-and-tier-contract.md).

## Implementation references

- [Desktop deployment workflow](../workflows/desktop-deployment.md)
- [Scenario-to-desktop overview](../../../scenario-to-desktop/docs/OVERVIEW.md)
- [Deployment resource contract](../../../../docs/resources/deployment-contract.md)
- [Bundle manifest schema](../guides/bundle-manifest-schema.md)
