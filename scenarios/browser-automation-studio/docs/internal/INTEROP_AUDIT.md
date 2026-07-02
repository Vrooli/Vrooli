# Interop Audit — browser-automation-studio (BAS)

This document captures the interoperability contract between the BAS API
and the two first-class clients that consume it (the BAS UI and the BAS
CLI). It is the close-out artifact for the proto+Connect-RPC migration
tracked in `plans:bas-migration-to-proto-connect-rpc`.

The architectural rule (per `interoperability-steer`):

> **Every proto-owned operation flows through a single generated contract:
> proto schema → Connect handler (Go) → Connect-Web client (TS) →
> manifest-driven CLI (Go). No hand-written REST/JSON envelopes between
> surfaces.** Only the four `RESTReason*` exceptions in
> [`REST_EXCEPTIONS.md`](./REST_EXCEPTIONS.md) are exempt.

## Claimed Maturity Level

**Level 4 — Schema-Driven, Drift-Tested.** Per `interoperability-steer §20`:

- L1 (single source of truth): proto schemas under
  `packages/proto/schemas/browser-automation-studio/v1/` are the single
  source of truth for all proto-owned operations. ✓
- L2 (generated clients on both sides): Go Connect handlers (server) and
  Connect-Web clients (UI) and Connect Go clients (CLI) are all generated
  from the same proto. ✓
- L3 (Maturity Level 3 — server consumes the same contract): the BAS API
  registers proto services via `connectx.RegisterChi`; no parallel
  hand-rolled REST surface for proto-owned operations remains. ✓
- L4 (drift-tested in CI): `cli/<dom>/manifest_proto_test.go` per-domain
  parity tests assert every CLI binding resolves to a real Connect method
  on a real generated service descriptor; ran as part of `go test ./...`.
  The CLI dispatcher (`cli/internal/protodispatch`) is itself generic and
  cannot bind to a non-existent method without failing fast. ✓

Level 5 (semantic / behavioral drift tests across the wire) is not
claimed — the dispatcher does typed flag hydration via a single
schemaless map decode rather than per-method typed request hydration
(tracked as a Phase 10 deferral in `REST_EXCEPTIONS.md`).

## Connect services (server-registered)

`scenarios/browser-automation-studio/api/main.go` `connectMounts`
(plus the conditionally-mounted `SchemaService`):

1. `CaptureService` — handlers/capture
2. `ScenariosService` — handlers/scenarios
3. `EntitlementService` — handlers/entitlement
4. `ProjectFilesService` — handlers/project_files
5. `ProjectsService` — handlers/projects
6. `ExecutionsService` — handlers/executions
7. `ReplayConfigService` — handlers/replay_config
8. `UXMetricsService` — handlers/uxmetrics
9. `SchedulesService` — handlers/schedules
10. `SessionProfilesService` — handlers/session_profiles
11. `RecordingsService` — handlers/recordings (storage/cookies/history/tabs read-side)
12. `AIService` — handlers/ai_service
13. `WorkflowsService` — handlers/workflows
14. `ExportsService` — handlers/export (binary mounted under `exports_service`)
15. `ObservabilityService` — handlers/observability
17. `VisionNavigationService` — handlers/vision_navigation
18. `SchemaService` — handlers/schema (optional; gated by handler init)

## Manifest-driven CLI groups

`scenarios/browser-automation-studio/cli/manifest.json` declares 17
groups, each binding `kind: "connect-rpc"`:

`capture`, `scenarios`, `tools`, `schema`, `entitlement`, `projects`,
`workflows`, `project-files`, `executions`, `uxmetrics`, `replay-config`,
`schedules`, `ai`, `vision-navigation`, `session-profiles`,
`observability`, `exports`.

The CLI dispatch is a generic Connect-RPC bridge (`cli/internal/protodispatch`)
that loads the manifest at startup and forwards flag maps to the matching
generated client. There is no per-domain hand-coded API client left in
the CLI for proto-owned operations.

The `recordings` storage/cookies sub-resources are consumed by the UI
directly via the generated `RecordingsService` Connect-Web client and
have no CLI verb today; they are intentionally absent from the manifest
(the CLI surface for "recordings" historically meant ingestion + asset
serving, both of which stay REST as multipart/byte-streaming exceptions).

## Manifest ↔ proto parity

Enforced at build time by per-domain parity tests:

```
cli/capture/manifest_proto_test.go
cli/scenarios/manifest_proto_test.go
cli/tools/manifest_proto_test.go
cli/schema/manifest_proto_test.go
cli/entitlement/manifest_proto_test.go
cli/projects/manifest_proto_test.go
cli/workflows/manifest_proto_test.go
cli/project_files/manifest_proto_test.go
cli/executions/manifest_proto_test.go
cli/uxmetrics/manifest_proto_test.go
cli/replay_config/manifest_proto_test.go
cli/schedules/manifest_proto_test.go
cli/ai/manifest_proto_test.go
cli/vision_navigation/manifest_proto_test.go
cli/session_profiles/manifest_proto_test.go
cli/observability/manifest_proto_test.go
cli/exports/manifest_proto_test.go
```

Every assertion resolves a `binding.{service,method}` to a real method
on a generated service descriptor under `packages/proto/gen/go/...`.

## Generated client coverage

| Surface | Generated client root | Status |
|---|---|---|
| Go server (handler) | `packages/proto/gen/go/browser-automation-studio/v1/<dom>/<dom>connect/` | 100% — every Connect service has a registered handler |
| Go client (CLI) | same | 100% — dispatched via `protodispatch` |
| TS client (UI) | `@vrooli/proto-types` (`packages/proto/gen/ts/...`) — Connect-Web | Proto-owned domains all import generated clients under `ui/src/api/<dom>.ts`; residual `fetch(`/`axios.` calls only target the `RESTReason*`-tagged endpoints in `REST_EXCEPTIONS.md` |

Enum display names in the UI are derived from protobuf-es enum descriptors at
runtime. The executions mapper uses the descriptor `sharedPrefix` and enum value
metadata instead of importing a hand-authored `packages/proto/gen/.../enum-names`
side artifact, keeping `packages/proto/gen/` limited to buf output.

## REST exceptions

See [`REST_EXCEPTIONS.md`](./REST_EXCEPTIONS.md) for the full table.
Summary by reason:

- `ops_probe` — `/health` (+ `/api/v1/health`)
- `third_party_shape` — WebSocket frames (`/ws*`), byte-serve
  (`/screenshots/*`, `/screenshots/thumbnail/*`, `/artifacts/*`,
  `/recordings/assets/...`, `/projects/{id}/files/*`),
  `/executions/{id}/export`, the Record Mode session lifecycle
  (`/recordings/live/...`).
- `webhook_receiver` — playwright-driver callbacks
  (`/internal/history-callback`, `/internal/ai-navigate/callback`,
  `/executions/{executionId}/frames`, plus the `/recordings/live/*`
  driver-callback POSTs for actions/frames/page-events).
- `multipart_upload` — `/recordings/import` (zip archive ingestion).

Every entry is tagged inline in `api/main.go` with both a
`// RESTException` comment and a `// RESTReason: <reason>` comment.

## Verification commands

```bash
cd packages/proto && buf lint --path schemas/browser-automation-studio
cd scenarios/browser-automation-studio/api && go build ./... && go test ./... -timeout 300s
cd scenarios/browser-automation-studio/cli && go build ./... && go test ./... -timeout 300s
cd scenarios/browser-automation-studio/ui && pnpm type-check && pnpm test --run
golangci-lint run scenarios/browser-automation-studio/api/... scenarios/browser-automation-studio/cli/...
```

All currently green on this branch (2026-05-20).
