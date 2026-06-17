# Progress — Device Sync Hub

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-06-17 | agi | done | **Phase 1 (greenfield rewrite — generate + charter + docs).** Regenerated from `react-vite` (v1.1.0) + `vrooli-default` design kit over the stale 2025 scenario (old tree backed up to `/tmp/device-sync-hub-OLD-*`). Authored `PRD.md` via prd-control-tower (8 P0 / 7 P1 / 4 P2 targets; validates clean). Generated `requirements/` (15 modules, 23 requirements covering all P0+P1; validates "all linked"). Declared dependency decisions in `.vrooli/service.json` (scenario-authenticator required+try_start+degraded_behavior; redis optional). Authored real `docs/concepts/DOMAINS.md` (domains: auth, devices, transfer, realtime, settings, health) + ARCHITECTURE/DATA/FLOWS/INTEGRATIONS. Orientation 6/8 gates green. |
| 2026-06-17 | agi | done | **Phase 2 (API foundation + auth integration + devices domain).** Authored `packages/proto/schemas/device-sync-hub/v1/devices/devices.proto` (`DevicesService`: List/Get/IssuePairingCode/RedeemPairingCode/RequestPairing/ApprovePairing/Rename/Revoke; `TrustState` enum; `Device`/`PairingCode`/`DeviceProfile`) + `buf generate`. Built `api/internal/auth/` (AuthClient seam over scenario-authenticator `GET /auth/validate` + `DELETE /sessions/{id}`; owner `Middleware` best-effort-injects Identity; `RequireOwner` gate; fail-closed, no test bypass). Built `api/internal/devices/` (types/schema/service/repository/sqlite/secrets/error-mapping + mocks): device registry, single-use conditional pairing-code claim, hub device-token issuance (SHA-256-hashed at rest), trust lifecycle pending→trusted→revoked, owner-scoped queries, best-effort authenticator session revoke. Built `api/handlers/devices/` (connect_handler/adapter/module/endpoints) — owner-gated RPCs vs open pairing RPCs. Wired registry (3 lines: endpoints/proto-files/schemas) + `main.go` (auth client + owner middleware + devices module). Regenerated `.vrooli/endpoints.json`. **All green:** `go build ./...`, `go test ./...` (incl. proto-connect parity + sqlite round-trips), `-tags e2e`, `golangci-lint` on new packages, proto module compiles. rec pending. |

## Current State & Phase 3 Handoff

**What is DONE (Phase 2):** see the 2026-06-17 Phase 2 row above. The `devices`
domain + `auth` integration are built, wired, and green. The `notes` example is
**deliberately still present** — per the plan it is removed only once the
transfer domain owns the `/measures` mount (Phase 3) so `main.go` is never left
with a dangling measures registry. So `example-domain-removed` stays open until
Phase 3.

**Key seams Phase 3 builds on:**
- `internal/devices/Repository.DeviceByToken(ctx, tokenHash)` already exists — it
  is the hook the transfer domain uses for **device-token trust enforcement**
  (resolve a presented hub token → device → require `TrustTrusted`). Add a
  device-auth middleware mirroring `auth.Middleware` but keyed on the device
  token (hash it with `devices` hashing, look up via `DeviceByToken`).
- `auth.Identity` (owner) + a future `devices.Device` (device) are the two
  request-scoped principals. Owner gates management; device gates transfer I/O.
- Storage: copy `handlers/notes/module.go::defaultBlobStore` for the transfer
  blob root (`api-core/storage` ClassData + `api-core/blobstore`).

**NEXT — Phase 3 (transfer domain + retention + presence + quotas):**
1. `transfer` proto + domain: items (file|text), streaming + chunked/resumable
   upload (REST multipart exception like notes_attach), streaming download with
   original filename, thumbnails, retention Live/Held(24h)/Pinned + purge
   scheduler, delivery ACL (broadcast default / direct), per-device+global quotas.
2. `realtime` WebSocket: presence + `item-arrived`/`pairing-request` fan-out.
3. Then **remove `notes`** (clears `example-domain-removed` + PROTO/STRUCTURE/UNIT
   residuals) and add security-headers middleware (clears STANDARDS high).

**Deferred verification:** the full `git-control-tower baseline diff --scenario
device-sync-hub --name dsh-rewrite` (plan §6a/§10) was NOT re-run this phase — it
runs the whole test-genie suite, whose DEPENDENCIES phase errors on the unhealthy
`scenario-authenticator` sibling in THIS environment (pre-existing, see below).
Phase-2 code correctness is covered by the green `go test ./...` + e2e + lint.
Run the baseline diff once the sibling is healthy (or accept the known
environmental DEPENDENCIES error when triaging the verdict).

## Superseded — original Phase 2 Handoff

**What is DONE (Phase 1):**
- Scaffold regenerated (react-vite + vrooli-default). `make setup` green.
- `PRD.md` published + validates ("No structural or linkage issues").
- `requirements/` 15 modules / 23 reqs, validates ("All requirements properly linked").
- Dependency decisions in `.vrooli/service.json`.
- `docs/concepts/DOMAINS.md` rewritten with the real domain map + ecosystem-fit.
- Concept docs (ARCHITECTURE/DATA/FLOWS/INTEGRATIONS) describe the intended design.
- Regression baseline `dsh-rewrite` captured (git-control-tower).

**Orientation gates:** 6/8 pass. The two open are **expected later-phase work**, not Phase-1 gaps:
- `example-domain-removed` — remove the template `notes` domain only AFTER the first real domain (`devices`) is green (Phase 2).
- `scaffold-health` (`make test`) — see residual failures below.

**Residual `make test` failures (all pre-existing scaffold state or environmental — none introduced by Phase 1):**
- **DEPENDENCIES (2 ERROR):** `scenario-authenticator` is unhealthy in THIS environment — missing binary `api/scenario-authenticator-api` + non-idempotent schema (`relation "idx_users_email" already exists`). Environmental; filed as an out-of-scope bug. Will pass once that sibling is built/healthy.
- **PROTO (4 ERROR):** scaffold example endpoints `health` + `notes_attach` lack proto-payload declarations/implementation proof. Resolved in Phase 2 (remove `notes`; give `health` + real domains proto-backed implementations).
- **STRUCTURE (1), UNIT (2):** fresh-scaffold/notes-example state; resolve as real domains land + `notes` is removed.
- **STANDARDS (high — "Security Headers"):** the stub API lacks security-headers middleware; add it when building the real API (Phase 2). Criticals (P0-missing-requirements) are already FIXED by the charter.

**NEXT — Phase 2 (per the plan `vrooli plans show device-sync-hub-greenfield-rewrite-cross-device-file-text-transfer`):**
1. Build the API foundation + `auth` integration (scenario-authenticator client, fail-closed) + `devices` domain (devices/pairing_codes tables, code/QR pairing, request→approve, revocation). Wire `api-core/storage` + `blobstore` seams.
2. Prove `devices` green, THEN remove the `notes` example (clears `example-domain-removed` + most PROTO/STRUCTURE/UNIT residuals).
3. Add security-headers middleware (clears STANDARDS high).

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
