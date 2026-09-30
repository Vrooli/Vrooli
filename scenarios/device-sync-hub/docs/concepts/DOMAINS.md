# Domains — Device Sync Hub

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

> The template's `notes` worked example is **not** product scope. It is
> retained only until the first real domain (`devices`) is green, then
> removed (see `docs/START-HERE.md`, gate `example-domain-removed`).

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md). Dependency contracts belong in
[`INTEGRATIONS.md`](INTEGRATIONS.md).

## Mental Model

Device Sync Hub is a **server-relayed** transfer hub for a single
owner's many trusted devices. Every transfer is `device → server →
device`; there is no peer-to-peer in v1. The product splits cleanly
into: who is allowed in (`auth`, `devices`), what moves (`transfer`),
how it appears live (`realtime`), and how the owner tunes it
(`settings`). `auth` is a thin integration boundary over the existing
`scenario-authenticator` scenario — this scenario does **not** own user
identity, JWTs, or sessions; it owns *devices* and *trust*.

## Domain Inventory

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths (target) |
|---|---|---|---|---|---|---|
| auth | Integration boundary over scenario-authenticator: validate the owner's JWT, mint/scope a per-device session, revoke a device's session. | Integration / gateway | No product data (delegates to scenario-authenticator). | API (middleware), CLI (login) | OT-P0-003 | `api/internal/auth/`, `api/internal/middleware/` |
| devices | Device registry, pairing (short-TTL single-use code/QR + request→approve fallback), trust-group membership, presence state. | Registry / lifecycle | `devices`, `pairing_codes`. | API, CLI, UI | OT-P0-002, OT-P0-003, OT-P0-008 | `api/internal/devices/`, `api/handlers/devices/`, `cli/domains/devices/`, `ui/src/features/devices/`, `packages/proto/schemas/device-sync-hub/v1/devices/` |
| transfer | Sync items (file **and** text, both first-class): create/list/get/delete, streaming + chunked/resumable upload, streaming download with original filename, image thumbnails, per-item retention (Live/Held/Pinned) + purge, delivery ACL (broadcast vs directed), quotas. | Content / blob lifecycle | `items` (metadata) + blobs via `api-core/blobstore`. | API, CLI, UI | OT-P0-001, OT-P0-004, OT-P0-007, OT-P1-001, OT-P1-002, OT-P1-003, OT-P1-004, OT-P1-007 | `api/internal/transfer/`, `api/handlers/transfer/`, `cli/domains/transfer/`, `ui/src/features/send/`, `ui/src/features/receive/`, `packages/proto/schemas/device-sync-hub/v1/transfer/` |
| realtime | WebSocket gateway: per-device presence (online/offline) and event fan-out (item-arrived, pairing-request) to the owner's trusted devices in near-real-time. | Eventing / push | Ephemeral in-memory presence (optionally Redis-backed for multi-instance). | API (WS), UI | OT-P0-005, OT-P1-005 | `api/internal/realtime/`, `ui/src/features/realtime/` |
| settings | Owner-level configuration: global retention default, storage quotas, optional at-rest encryption toggle, and the device-management surface (rename/revoke/sign-out, trust list, permissioned clear-all). | Configuration | `settings` (owner config singleton). | API, CLI, UI | OT-P0-008, OT-P1-007 | `api/internal/settings/`, `api/handlers/settings/`, `cli/domains/settings/`, `ui/src/features/settings/` |
| health | Report runtime readiness and dependency reachability (incl. scenario-authenticator). Durable template infrastructure. | Reporting / query | No product data. | API, UI | Operational (cross-cutting). | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/device-sync-hub/v1/health/` |

## Domain Details

### auth (integration boundary)

- Purpose: be the single seam through which the scenario talks to
  `scenario-authenticator`. Validate the owner's access token, refresh,
  and revoke a device's session on un-pairing.
- Owns: the authenticator HTTP client + a substitutable `AuthClient`
  seam; request-scoped owner identity in middleware.
- Does **not** own: user accounts, password/credential storage, JWT
  signing, OAuth, 2FA — all of that stays in scenario-authenticator.
- Failure mode: **fail closed** in production if the authenticator is
  unavailable (no test-mode bypass shipped). See `INTEGRATIONS.md`.
- Requirements: OT-P0-003.

### devices

- Purpose: model a *device* as a first-class entity keyed to the owner,
  and govern how a device joins or leaves the trust group.
- Owns: `devices` (id, owner, name, type/OS, capabilities, last_seen,
  trust_state), `pairing_codes` (short-TTL, single-use). Pairing-code
  issue/redeem, request→approve fallback, rename, revoke (drops row +
  asks `auth` to revoke the authenticator session).
- Surfaces: API + CLI (`devices list|pair|approve|revoke`) + UI pairing
  screen (QR + code) and device list.
- Seams: injected clock/TTL for deterministic pairing tests; `AuthClient`.
- Requirements: OT-P0-002, OT-P0-003, OT-P0-008.

### transfer

- Purpose: the actual movement of content between trusted devices.
- Owns: `items` (id, owner, origin_device, kind=file|text, name,
  mime, size, thumbnail_ref, blob_ref, retention=Live|Held|Pinned,
  expires_at, target=broadcast|device_id, created_at) and binary blobs
  via `api-core/blobstore`. Retention purge scheduler lives here.
- Behavior: streaming upload (never buffer whole files in memory),
  chunked/resumable upload, streaming download preserving original
  filename, image thumbnailing, per-item retention + global default,
  delivery ACL (only trusted devices can list/pull; broadcast default,
  optional direct-to-device), per-device/global quota enforcement.
- Seams: `BlobStore`, storage `Resolver`, injected clock for retention.
- Future seam: the "where bytes flow" boundary is isolated so a P2
  WebRTC same-LAN fast path can slot in without redesign.
- Requirements: OT-P0-001, OT-P0-004, OT-P0-007, OT-P1-001..004, OT-P1-007.

### realtime

- Purpose: make transfers feel instant and show who is online.
- Owns: the WebSocket gateway, presence registry (in-memory; optionally
  Redis for multi-instance), and event fan-out. No durable product data.
- Emits: `item-arrived`, `pairing-request`, presence changes — only to
  the owner's trusted, authenticated device sessions.
- Requirements: OT-P0-005, OT-P1-005.

### settings

- Purpose: let the owner tune the system and manage devices in one place.
- Owns: `settings` singleton (retention default, quota limits, at-rest
  encryption toggle). Hosts the device-management UI surface (delegating
  mutations to `devices`). Destructive actions (clear-all) are
  permission-gated.
- Requirements: OT-P0-008, OT-P1-007.

### health

- Purpose: expose API/DB readiness and dependency reachability (incl.
  scenario-authenticator) so the UI can show live backend state.
- Durable template infrastructure; keep.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Trusted device | A device the owner has paired into their trust group; the unit of access control. | `devices`. |
| Item | A single transferred payload (file or text) with a retention policy. | `transfer`. |
| Retention | Live (deliver-then-purge) / Held (24h default) / Pinned (until deleted). | `transfer` + `settings` (default). |
| Trust group | The owner's set of trusted devices; broadcast audience. | `devices`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| p2p-transport | WebRTC same-LAN fast path is a P2 speed optimization, not a capability gap; relay covers all cases. | Felt pain on same-LAN multi-GB transfers (OT-P2-001). Keep the `transfer` byte-flow seam ready. |
| multi-tenant trust | v1 is single-owner-many-devices by design. | A real second-owner / shared-group requirement (OT-P2-002). |
| versioning / tags | Not needed for core transfer. | OT-P2-004. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/`, `api/internal/modules/` — module descriptors / registry.
- `api/internal/database/` — cross-cutting DB infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.

## Ecosystem Fit (per `docs/concepts/ECOSYSTEM.md`)

- **Role:** product (direct-UI) + interface-enabler (programmatic delivery seam).
- **Interfaces served:** Direct UI → polished/responsive/accessible/production-ready;
  Programmatic → clean CLI + API so other scenarios/agents can deliver an artifact to
  one of the owner's devices.
- **Compound-value seam:** the `transfer` API + CLI ("deliver this file/text to device X")
  — reusable by e.g. image-tools outputs or agent-produced artifacts; discoverable via `cli-health`.
- **Self-improvement:** advances the operator/user-interaction meta-capability (artifact
  movement between the human's devices and the server).
- **Monetization/bundle:** lifestyle bundle; core capability is self-hostable → keep free
  (never gate what a self-hoster could run). Any hosted/relay-heavy metering routes through
  LPBS per `docs/concepts/PAID_FEATURES.md` (strategy deferred to canon).

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts (scenario-authenticator)
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
