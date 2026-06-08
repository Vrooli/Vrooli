# How to Migrate a Domain to Connect-RPC

The mechanical recipe for moving one ecosystem-manager domain off hand-rolled
`gorilla/mux` REST + Zod onto a proto-first Connect-RPC vertical. `settings` is
the worked reference — copy its shape. Do **one** domain per change; never
bulk-migrate. The migrated domain's old REST/Zod/message-only paths are
**deleted in the same change** (greenfield — no coexistence within a migrated
domain). Un-migrated domains keeping their REST routes is the migration
mechanism, not drift. See [`COHERENCE-NOTES.md`](COHERENCE-NOTES.md).

## The foundation (already in place)

- `api/internal/module/` — the `Module` / `EndpointDescriptor` / `RESTException` contract.
- `api/internal/modules/registry.go` — `AllEndpoints()` + `MigratedDomains()`; the single registration point.
- `api/cmd/gen-endpoints/` — regenerates the Connect slice of `.vrooli/endpoints.json` while preserving un-migrated REST entries (`make endpoints`).
- `api/pkg/server/server.go` — `connectModules()` / `mountConnectModules(router)` mount Connect handlers on the root router alongside the REST `/api` subrouter.
- `ui/src/api/client.ts` — the shared `transport` (`createScenarioConnectTransport`) + `ApiError`.

## Steps

### 1. Proto (`packages/proto`)
Create `packages/proto/schemas/ecosystem-manager/v1/<domain>/<domain>.proto`:
- `package vrooli.ecosystem_manager.v1.<domain>;`
- `option go_package = ".../gen/go/ecosystem-manager/v1/<domain>;<domain>_v1";`
- one `service <Domain>Service { rpc ... }` block; move the domain's messages (with their `buf.validate` constraints) into it.
- **Delete** the legacy message-only `v1/api/<domain>.proto` + `v1/domain/<domain>.proto`.
- `cd packages/proto && buf generate`. Confirm `git diff --stat` touches only this domain's `gen/{go,typescript,python}` artifacts.

### 2. API
- `api/internal/<domain>/` — move the business/persistence logic out of `api/pkg/<domain>/`; add sentinel errors + a `ToConnectError(err)` mapper (validation → `connect.CodeInvalidArgument`, not-found → `CodeNotFound`, else `CodeInternal`).
- `api/handlers/<domain>/connect_handler.go` — implement the generated `<Domain>ServiceHandler` interface; keep it thin (translate at the edge, delegate to `internal/<domain>/`, preserve side-effects there).
- `api/handlers/<domain>/module.go` — `Module()` builds `connectPath, connectHandler := <d>connect.New<Domain>ServiceHandler(NewConnectHandler(Deps{...}))`, mounts via `connectx.RegisterServices`, and exports `Endpoints`.
- Wire it: add the live `Module(...)` to `server.go`'s `connectModules()`; add `<domain>H.Endpoints...` to `modules.AllEndpoints()` and the domain name to `modules.MigratedDomains()`.
- **Delete** the old `api/pkg/handlers/<domain>.go`, its `register<Domain>Routes`, its server field, and `api/pkg/<domain>/` (after the move).

### 3. UI
- `ui/src/api/<domain>.ts` — `export const <d>Client = createClient(<Domain>Service, transport);`.
- `ui/src/features/<domain>/` — consume `<d>Client` via react-query (`useQuery`/`useMutation`).
- **Delete** the domain's entries in `ui/src/lib/proto-contracts.ts` and the old fetch/store path; wire the feature into `App.tsx`.

### 4. CLI
- `cli/domains/<domain>/register.go` + `handlers.go` — `<d>connect.New<Domain>ServiceClient(cliapp.NewConnectHTTPClient(core))`, bind `"<Domain>Service.<Method>"` via `cliapp.LoadFromManifest`, render via `cliapp.RenderProto*`.
- Register in `cli/app.go` `Domains()`; add the CLI command(s) to `api/cmd/gen-endpoints/cli_commands_seed.json`; **delete** the old REST CLI path.
- `make endpoints` to regenerate `.vrooli/endpoints.json` (drops the domain's REST entries, adds the Connect ones).

### 5. Docs + gate
- Add the `internal/<domain>` seam to [`SEAMS.md`](SEAMS.md).
- Gate: `go build ./...` (api+cli), `npm run build` (ui) green; `<cli> <domain> ...` round-trips against the running API; `grep -r <domain> api/pkg/handlers` is empty.

## Decomposition note

Step 2's `handlers/<domain>.go` → thin `connect_handler.go` + `internal/<domain>/`
service split is also the reference for decomposing the remaining god-objects
(`api/pkg/queue/execution_manager.go`, `api/pkg/handlers/tasks.go`,
`ui/.../TaskDetailsModal.tsx`) as their domains migrate.
