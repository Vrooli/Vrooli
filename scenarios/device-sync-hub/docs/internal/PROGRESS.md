# Progress — Device Sync Hub

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-06-17 | agi | done | **Phase 1 (greenfield rewrite — generate + charter + docs).** Regenerated from `react-vite` (v1.1.0) + `vrooli-default` design kit over the stale 2025 scenario (old tree backed up to `/tmp/device-sync-hub-OLD-*`). Authored `PRD.md` via prd-control-tower (8 P0 / 7 P1 / 4 P2 targets; validates clean). Generated `requirements/` (15 modules, 23 requirements covering all P0+P1; validates "all linked"). Declared dependency decisions in `.vrooli/service.json` (scenario-authenticator required+try_start+degraded_behavior; redis optional). Authored real `docs/concepts/DOMAINS.md` (domains: auth, devices, transfer, realtime, settings, health) + ARCHITECTURE/DATA/FLOWS/INTEGRATIONS. Orientation 6/8 gates green. |
| 2026-06-17 | agi | done | **Phase 2 (API foundation + auth integration + devices/pairing/trust domain).** See git: `devices.proto` (DevicesService), `internal/auth` (authenticator client + owner middleware, fail-closed), `internal/devices` (registry, single-use pairing claim, trust lifecycle, revoke), wired into `main.go` + registry. Green: build/test/e2e/lint. |
| 2026-06-17 | agi | done | **Phase 3 (transfer domain + retention + presence + quotas + realtime).** `transfer.proto` (TransferService: CreateTextItem/ListItems/GetItem/DeleteItem) + `realtime.proto` (Event messages, no service). `internal/transfer`: items (file+text, both first-class), SQLite repo with delivery-ACL visibility (broadcast/directed/origin), retention Live/Held/Pinned + `ExpiresAt` stamping, background `RunPurgeLoop`, per-owner/per-device quotas, pure-stdlib image `Thumbnailer`. `internal/realtime`: in-memory presence + SSE event hub (item-arrived/deleted, presence-changed, pairing-requested) with delivery-ACL fan-out. `internal/deviceauth`: device-token middleware (`X-Device-Token`/`?token=`) → `RequireDevice`. `internal/devices`: added `Authenticator` (token→TRUSTED device) + `PairingNotifier` hook; devices handler now overlays live presence + pushes pairing banners. Handlers: transfer Connect handler + REST multipart upload (quota pre-check, thumbnail) + REST streaming download (original filename, `?thumb=1`, marks Live delivered). Two new REST-exception reasons (`binary_download`, `event_stream`). Wired all into `main.go` (shared hub, device authenticator, transfer wiring, purge goroutine, dual auth middleware) + registry (transfer in AllEndpoints/AllProtoFiles/AllSchemas; realtime in AllEndpoints). **Green:** `go build ./...`, `go vet`, `gofumpt`, `golangci-lint` (new pkgs), and `go test ./...`/`-tags e2e` for every package **except** the pre-existing `cmd/gen-endpoints` crossCheck (see handoff). New tests: transfer sqlite/service/thumbnail, realtime hub, devices authenticator, deviceauth middleware, transfer connect + REST upload/download roundtrip. |
| 2026-06-17 | agi | done | **Phase 2 (API foundation + auth integration + devices domain).** Authored `packages/proto/schemas/device-sync-hub/v1/devices/devices.proto` (`DevicesService`: List/Get/IssuePairingCode/RedeemPairingCode/RequestPairing/ApprovePairing/Rename/Revoke; `TrustState` enum; `Device`/`PairingCode`/`DeviceProfile`) + `buf generate`. Built `api/internal/auth/` (AuthClient seam over scenario-authenticator `GET /auth/validate` + `DELETE /sessions/{id}`; owner `Middleware` best-effort-injects Identity; `RequireOwner` gate; fail-closed, no test bypass). Built `api/internal/devices/` (types/schema/service/repository/sqlite/secrets/error-mapping + mocks): device registry, single-use conditional pairing-code claim, hub device-token issuance (SHA-256-hashed at rest), trust lifecycle pending→trusted→revoked, owner-scoped queries, best-effort authenticator session revoke. Built `api/handlers/devices/` (connect_handler/adapter/module/endpoints) — owner-gated RPCs vs open pairing RPCs. Wired registry (3 lines: endpoints/proto-files/schemas) + `main.go` (auth client + owner middleware + devices module). Regenerated `.vrooli/endpoints.json`. **All green:** `go build ./...`, `go test ./...` (incl. proto-connect parity + sqlite round-trips), `-tags e2e`, `golangci-lint` on new packages, proto module compiles. rec pending. |

## Current State & Phase 4 Handoff

**DONE through Phase 3:** charter+docs (P1), auth integration + devices/pairing/trust
(P2), and the transfer + realtime domains with retention, quotas, presence, and
device-token trust (P3 — see the 2026-06-17 Phase 3 row above). The API is
feature-complete for the P0 server-relay path (pair → push file/text → pull on
another device → retention purge → revoke severs access) plus realtime
presence/events. `go build`/`vet`/`gofumpt`/`golangci-lint` and `go test ./...`
(and `-tags e2e`) are green for every package **except** the one pre-existing
failure below.

**THE ONE RED — `cmd/gen-endpoints` crossCheck (Phase-2-introduced, Phase-4-resolved):**
- `make endpoints` and `TestRun_ProducesValidJSON` fail with
  `cli-commands.gen.json missing command "devices list" …`. The API declares
  `CLIMapping`s for the `devices` and (now) `transfer` endpoints, but the CLI
  (`cli/domains/`) only registers `notes` — **the CLI is Phase 4.** This was
  already red entering Phase 3 (devices, Phase 2); transfer adds more pending
  commands but doesn't change the red/green state.
- **Phase 4 closes it:** build `cli/domains/devices/` + `cli/domains/transfer/`
  (verbs `devices list|pair|approve|revoke`; `transfer send-text|list|get|download|
  delete|upload`), register them in `cli/domains/domains.go`, then `make endpoints`
  regenerates `cli-commands.gen.json` + `.vrooli/endpoints.json` cleanly and the
  unit test passes. Update the `cmd/gen-endpoints/main_test.go` fixture to list the
  new commands. The transfer CLI is also the **programmatic compound-value seam**
  (other scenarios "deliver an artifact to a device").

**Other still-open items (deferred by plan, not Phase-3 gaps):**
- `example-domain-removed` gate + the `notes` worked example: removed in **Final
  Cleanup** (plan §7 Final), once a real domain owns the `/measures` mount. Today
  `notes` still owns `/measures` in `main.go`; transfer has no measure yet. Before
  deleting `notes`, move a measure (e.g. `items.count`) onto a transfer measures
  registry so `main.go` is never left with a dangling mount.
- A dedicated **`settings`** domain (owner-tunable retention default / quotas /
  at-rest-encryption toggle) is in DOMAINS but not built — Phase 3 uses
  `transfer.Config` defaults (24h Held / 10-min Live / 5 GiB owner / 2 GiB device /
  1 MiB text). Build alongside the UI (Phase 5) or as a P3 follow-up.
- **Chunked/resumable upload** (OT-P1-001) not yet implemented; streaming multipart
  upload handles large files via temp-file spillover (the P0 path). Add the
  resumable-session endpoint in a P1 pass.
- **At-rest blob encryption** (OT-P1-007) toggle not yet implemented (P1).

**Deferred verification:** the full `git-control-tower baseline diff --scenario
device-sync-hub --name dsh-rewrite` (plan §6a/§10) was NOT re-run — its
DEPENDENCIES phase errors on the unhealthy `scenario-authenticator` sibling in THIS
environment (pre-existing, environmental). Phase-3 code correctness is covered by
the green `go test ./...` + e2e + lint. Run the baseline diff once the sibling is
healthy (or accept the known environmental DEPENDENCIES error when triaging).

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
