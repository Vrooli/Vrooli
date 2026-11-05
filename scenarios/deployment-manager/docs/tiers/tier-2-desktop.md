# Deployment Tier 2: desktop

Deployment Tier 2 is a portable desktop application for Windows, macOS, or
Linux. It is a target, not a commercial subscription level. See the [Deployment
Hub](../../../../docs/deployment/README.md) for the distinction.

The shared delivery validation contract is owned by
`packages/delivery-ramp-go`. Tier 2 documentation describes the desktop
target and its routes; the spine owns target capability, matrix, transport,
journey evidence, disposition, and reference-only verdict semantics.

## Target contract

A desktop release must make its runtime shape explicit:

- **bundled private**: the application starts verified, app-private services;
- **external-server**: the shell calls a configured Tier 1 scenario API;
- **shared provider**: the app uses a broker-issued, scoped lease;
- **peer**: another desktop runtime, currently unsupported.

These routes have different ownership, data, secret, and failure semantics.
They must not be described as one generic “desktop deployment.”

## Authentication and identity contract

Tier 2 deployment mode also determines the human identity model. The desktop
supervisor's loopback bearer token authenticates the Electron shell to the
supervisor. It is not a human account credential and it must not be accepted
by a scenario API as proof of a user identity.

| Authentication mode | Human sign-in | Identity authority | Offline behavior |
|---|---|---|---|
| `personal_local` | No by default | Current OS user plus private app boundary | Supported when all required bundle capabilities are local |
| `local_multi_user` | Required | Private local `scenario-authenticator` realm | Local sign-in works; external links may be unavailable |
| `remote_vrooli` | Required when the server requires it | Configured Tier 1 provider | Not available unless the product declares an offline lease |
| `shared_provider` | Provider-specific | Broker-issued scoped lease | Depends on the lease and provider |

Bundled applications use `personal_local` unless their product contract
requires another mode. A downloaded app must not require an LPBS website login
merely because it was downloaded. LPBS sign-in remains the authority for LPBS
customer accounts, subscriptions, downloads, and commercial entitlements.

Changing a bundle from `personal_local` to `local_multi_user` is a
security-sensitive setup operation. The application must provision the local
identity provider, create or select the first administrator, migrate or scope
existing data, and record the mode transition. It must not be an unprotected
toggle that silently exposes a local API to other users.

Website identity and local identity are linked only through an explicit,
one-time, scoped, auditable authorization-code/PKCE or device flow. Email
equality and copied website JWTs are not valid linking mechanisms.

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
