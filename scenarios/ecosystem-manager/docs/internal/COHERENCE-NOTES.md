# Coherence Notes — Transport & Contract Drift

> Developer/agent memory for where the API's *declared* contract and its *served*
> transport disagree, and the migration that closes the gap. Read this before
> touching the API surface or the UI/CLI clients.

## Why this file exists

`ecosystem-manager` predates the proto-first Connect-RPC contract standard. The
API still serves most domains as hand-registered `gorilla/mux` REST routes with
`encoding/json`, while the protos under `packages/proto/schemas/ecosystem-manager/`
are **message-only** (no `service`, no `rpc`). The UI consumes those messages
through a hand-rolled Zod layer (`ui/src/lib/proto-contracts.ts`) rather than
generated Connect clients. That is the central coherence gap.

## Current drift (snapshot)

| Concern | Declared / intended | Actually served |
|---|---|---|
| Transport | proto-first Connect-RPC (`Service.Method`) | ~79 REST routes on one `gorilla/mux` router |
| Proto | one `service` per domain, per-domain `go_package` | flat `ecosystem_manager.v1`, shared `;api` go_package, message-only |
| UI client | generated Connect clients over a shared transport | hand-written Zod schemas + `fetch` |
| Layout | `api/internal/<domain>/`, `api/handlers/<domain>/` | business logic in `api/pkg/<domain>/`, handlers pooled in `api/pkg/handlers/` |

## Migration policy (incremental, one reference domain at a time)

The transport gap is closed **per domain**, not big-bang. Each migrated domain:

1. Gets a proto `service` under `packages/proto/schemas/ecosystem-manager/v1/<domain>/<domain>.proto`.
2. Implements a thin Connect handler in `api/handlers/<domain>/` delegating to `api/internal/<domain>/`.
3. Mounts on the existing mux via `connectx.RegisterServices` (REST and Connect coexist on one router during migration — this is the migration mechanism, **not** a compat shim).
4. Replaces the UI Zod path with a generated Connect client (`ui/src/api/<domain>.ts`).
5. **Deletes** its old REST handler, message-only protos, and Zod entries in the same change (greenfield rule — no coexistence *within* a migrated domain).

`settings` is the reference domain — copy its shape. See
[`SEAMS.md`](SEAMS.md) for the seam registry; the mechanical "how to migrate a
domain" steps are captured in `docs/internal/MIGRATION-GUIDE.md` (added with the
Connect-RPC foundation).

## Un-migrated domains

`tasks`, `queue`, `autosteer`, `discovery`, `insights`, `prompts`, `executions`
keep their REST routes until each gets its own follow-up migration. Their REST
coexistence is expected and is not drift to "fix" ad hoc — migrate the whole
domain or leave it.
