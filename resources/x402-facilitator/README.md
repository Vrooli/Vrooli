# X402 Facilitator

Vrooli-managed, self-hosted x402 v2 facilitator for verification and on-chain
settlement. The resource pins Second State's Apache-2.0 Rust implementation by
multi-architecture OCI digest, extracts it into a verified managed-service
artifact, and exposes it only on loopback. Docker is not required at runtime
on Linux.

The Linux target uses the `managed-service` resource template. macOS and
Windows remain explicitly unsupported until upstream publishes signed native
facilitator artifacts.

## Intent

- Resource ID: `x402-facilitator`
- Category: `infrastructure`
- Driver: `managed-service`
- Portability tier: `conditional`

## Use Cases

- Verify x402 payment payloads before a resource is delivered.
- Settle buyer-signed x402 payments on an explicitly configured network.
- Give spending and earning scenarios one governed local facilitator endpoint.

The default runtime mounts `config/config.json`, which deliberately registers
no chain and no payment scheme. `/health` and `/supported` work, but `/verify`
and `/settle` cannot select a network or move value until a later, attended
configuration supplies an allowlisted chain, a signer, an RPC route, and the
corresponding schemes. This is a fail-closed bootstrap state, not production
payment readiness.

## Architecture

This template keeps the generated CLI thin on purpose.

- `resource.json` is the declarative authority for lifecycle, digest-pinned OCI acquisition, invoke, freshness, exports, and runtime metadata.
- `cli/` is the single binary entrypoint and command wiring surface.
- `cli/internal/` is the default home for resource-specific Go logic when the manifest and shared control plane are not enough.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add resource-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Generated placeholder packages:

- `cli/internal/install`: install/bootstrap behavior unique to the resource
- `cli/internal/runtime`: runtime config/materialization logic
- `cli/internal/status`: richer status interpretation when generic lifecycle status is insufficient
- `cli/internal/health`: resource-specific probe helpers
- `cli/internal/env`: environment export and derived configuration helpers

## Supply-chain pin

- Upstream: `second-state/x402-facilitator`
- Image source revision: `e75adda8a0e3cf23db446883ba91b88bbab2fe28` (2026-07-13)
- OCI index: `ghcr.io/x402-rs/x402-facilitator@sha256:3a71fd6e3ff5ae00c30bcfbff35e5cf858bb37b9b97de032f46545c18e953d71`
- License: Apache-2.0
- Platforms: Linux amd64 and arm64 images with provenance attestations
- Materialized artifact tree digests:
  - Linux amd64: `ea245e013e731945eef9e050b5670eb3d02e2534002af5010116c00b66c3e271`
  - Linux arm64: `257f2011bd49ec42c807e14f0d32ad3db6a0b4545956d09b7b959e05e54f7713`

The image's OCI labels identify that exact revision and the Apache-2.0 upstream;
the digest was not inferred from a floating tag. Upgrade only after reviewing the source diff, confirming the published image
provenance, exercising `/supported`, `/verify`, and `/settle` on testnet, and
then updating the digest deliberately.
