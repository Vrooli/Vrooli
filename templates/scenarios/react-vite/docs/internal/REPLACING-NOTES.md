# Replacing the `notes` reference

The `notes` domain ships in every scenario generated from this
template as a worked CRUD example. It demonstrates the full vertical
stack: proto contract → API service + repository → generated
Connect handler → CLI domain → UI feature card. It also includes one
deliberate REST exception for multipart file uploads. **It is meant
to be replaced** with your scenario's first real domain (`tasks`,
`users`, `orders`, …).

This guide is the delete-checklist. Following the steps in order
takes the boundary the architecture pays for: deleting six folders
plus three single-line edits leaves a clean scenario with no notes
residue.

## Before you start

Pick a domain name. The placeholders below use `tasks`; substitute
your name everywhere `tasks`/`Task` appears. Plural for the package,
PascalCase for component names.

## Steps to add your domain (alongside notes)

Build your domain first, side-by-side with notes. Once your tests
are green, delete notes (next section). Don't skip the side-by-side
phase — it's how you learn the pattern by copying.

1. **Proto contract.** Create `proto/v1/tasks/tasks.proto` (mirror
   `proto/v1/notes/notes.proto`'s message + service shape). Every
   proto-typed wire boundary needs a `service` block:
   ```proto
   service Tasks {
     rpc List(ListTasksRequest) returns (ListTasksResponse);
     rpc Create(CreateTaskRequest) returns (CreateTaskResponse);
     rpc Get(GetTaskRequest) returns (GetTaskResponse);
   }
   ```
   Use well-known types for non-string scalars, for example
   `google.protobuf.Timestamp created_at = 4;`, and add domain-owned
   relationship fields directly to the message (the notes reference
   uses `repeated string attachment_keys = 6;`). If your domain has
   opaque binary sub-resources, mirror `proto/v1/notes/attachments.proto`:
   proto describes metadata, while bytes travel over multipart REST.

   From the workspace root, run `make generate`. New proto types and
   Connect stubs appear under
   `packages/proto/gen/go/{{SCENARIO_ID}}/v1/tasks/` and
   `packages/proto/gen/typescript/js/{{SCENARIO_ID}}/v1/tasks/`.

2. **API domain package.** Create `api/internal/tasks/`:
   - `types.go` — `Task` struct, `ErrTaskNotFound`, `ErrInvalidTask`.
   - `repository.go` — `Repository` interface (Create, Get, List).
   - `sqlite.go` — `SqliteRepository` impl.
   - `service.go` — `Service` interface, `NewService(repo)`,
     validation + defaults.
   - `schema.sql` — domain-owned table DDL (`CREATE TABLE IF NOT
     EXISTS tasks (...)`).
   - `schema.go` — `//go:embed schema.sql` + `func Schema() string`.
   - `service_test.go`, `sqlite_test.go`, `schema_test.go` — tests.
   - `mocks/{repository,service,repository_test,service_test}.go` —
     co-located test fakes (`package mocks`); deleting `internal/tasks/`
     takes them along.

3. **API Connect handler + module.** Create `api/handlers/tasks/`:
   - `connect_handler.go` — `Deps`, `NewConnectHandler`, and methods
     implementing the generated `tasks_v1connect.TasksHandler`
     interface. Methods should be thin: translate request fields into
     service inputs, call the domain service, map domain errors to
     `connect.Error`, and return generated proto responses.
   - `module.go` — `Module(db, clk, blobs, logger) module.Module`
     that constructs repo + service + generated Connect handler
     internally, mounts it with `api-core/connectx.RegisterServices`,
     and re-exports `func Schema() string { return internaltasks.Schema() }`.
   - `adapter.go` or equivalent local conversion helpers only when the
     domain type and proto type differ.
   - `connect_handler_test.go`, `module_test.go` — tests.

   There is no per-domain hand-written JSON decode/encode ribbon and
   no separate `endpoints.go`. The proto service is the wire contract;
   `module.go` keeps the generated endpoint metadata local for the
   manifest.

4. **Wire into the registry + main.** Three single-line edits.

   In `api/internal/modules/registry.go`, add tasks to both lists:
   ```go
   func AllEndpoints() []module.EndpointDescriptor {
       out := make([]module.EndpointDescriptor, 0)
       out = append(out, healthH.Endpoints...)
       out = append(out, notesH.Endpoints...)
       out = append(out, tasksH.Endpoints...)  // new
       return out
   }

   func AllSchemas() []apidb.SchemaProvider {
       return []apidb.SchemaProvider{
           apidb.SchemaProviderFunc(localdb.SystemSchema),
           apidb.SchemaProviderFunc(healthH.Schema),
           apidb.SchemaProviderFunc(notesH.Schema),
           apidb.SchemaProviderFunc(tasksH.Schema),  // new
       }
   }
   ```

   In `api/main.go`'s `server.New(...)` slice, add the runtime module
   (the one place that needs live deps):
   ```go
   srv := server.New(
       server.Deps{Clock: clock.System{}, Logger: log.Default()},
       healthH.Module(db, "{{SCENARIO_ID}}-api", "1.0.0"),
       notesH.Module(db, clock.System{}, blobStore, log.Default()),
       tasksH.Module(db, clock.System{}, blobStore, log.Default()),  // new
   )
   ```

5. **CLI domain.** Create `cli/domains/tasks/`:
   - `register.go` — `Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup`
     mirroring `cli/domains/notes/register.go`. Each `cliapp.Command`
     declares its `Args cliapp.ArgSchema` (flags + positionals) and
     binds `RunCtx` instead of `Run`. The schema feeds both the parser
     and `--help`, so a flag added here automatically appears in
     `tasks <sub> --help`.
     ```go
     {
         Name: "create", Description: "Create a task",
         Args: cliapp.ArgSchema{
             Flags: []cliapp.Flag{
                 {Name: "title", Required: true, Description: "Task title"},
             },
         },
         RunCtx: h.create,
     }
     ```
   - `handlers.go` — one `func(ctx cliapp.RunContext) error` per
     subcommand. Construct the generated Connect client once with
     `cliapp.NewConnectHTTPClient(core)` plus the generated
     `tasksconnect.NewTasksClient`. Each handler calls the typed client,
     reads `ctx.Flag(...)` / `ctx.Positional(...)`, routes output via
     `cliapp.RenderProtoList` / `cliapp.RenderProtoMutation`, and wraps
     errors with `cliapp.WrapAPIError`.
     ```go
     func (h *handlers) create(ctx cliapp.RunContext) error {
         resp, err := h.client.Create(context.Background(),
             connect.NewRequest(&tasksv1.CreateTaskRequest{Title: ctx.Flag("title")}))
         if err != nil { return cliapp.WrapAPIError("create task", err, nil) }
         return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
             Result:  []string{fmt.Sprintf("Created task %s.", resp.Msg.Task.Id)},
             Changes: []string{formatTask(resp.Msg.Task)},
         })
     }
     ```
     `RenderProtoList` / `RenderProtoMutation` honor the built-in `--json`
     flag: human consumers see the report; `--json` consumers see the
     proto-typed response wire shape (no `summary` / `next_command`
     wrapper) — identical to what `curl /Tasks/Create` returns. Machine
     consumers parse the same JSON the API speaks.

     Don't reach for `flag.NewFlagSet`, hand-written HTTP calls, a
     per-domain `apiError` helper, or hand-typed `<thing>JSON` structs
     mirroring the proto shape. cli-core owns the parser, rendering,
     Connect HTTP client setup, error wrapping, and proto-typed JSON
     output.

     If a handler needs the human `ctx.RenderList` / `ctx.RenderMutation`
     path without proto-typed JSON (e.g., it aggregates multiple sources
     into one report), use those directly — `RenderProto*` is the
     proto-payload shortcut, not a hard requirement.

6. **Multipart endpoint, when the domain needs opaque bytes.** Keep
   binary upload as REST multipart and keep metadata proto-typed:
   - Add a metadata proto beside the domain service proto, as notes does
     with `attachments.proto`.
   - Add a domain service/repository for metadata and use
     `api-core/blobstore.BlobStore` for bytes.
   - Mount one REST handler under a resource path such as
     `/api/v1/tasks/{id}/attachments`.
   - In the CLI, implement `tasks attach <id> --file <path>` with
     `cliapp.UploadFile`.
   - In the UI, build `FormData` and call the shared `uploadFile()`
     helper, then parse the metadata response with the generated proto
     descriptor.
   - **Own your blob storage inside `handlers/tasks/module.go`.** Mirror
     `handlers/notes/module.go`'s `Module(db, clk, logger)` /
     `ModuleWithBlobStore(db, clk, blobs, logger)` pair: production
     `Module` calls a private `defaultBlobStore()` helper, tests pass
     an in-memory blobstore via `ModuleWithBlobStore`. This keeps
     `api/main.go` blob-storage-free, so a future delete of `tasks`
     leaves no orphan `blobstore` plumbing in `main.go`.

   You can add a multipart endpoint without breaking later deletion
   because the substrate already enforces the cleanup contract: the
   notes domain ships exactly this pattern, and `handlers/notes/module.go`
   is the worked example.

7. **Wire into CLI.** Add **one line** to `cli/domains/domains.go`'s
   `SubcommandGroups`:
   ```go
   return []cliapp.SubcommandGroup{
       notes.Register(core),
       tasks.Register(core),  // new
   }
   ```

8. **CLI commands seed.** Connect-RPC endpoints are described from the
   proto service metadata kept in `module.go`; the seed file is only
   for REST exceptions that have no proto service descriptor. Add a
   `tasks attach` entry to `api/cmd/gen-endpoints/cli_commands_seed.json`
   if you added a multipart endpoint. Run `make endpoints` to regenerate
   `.vrooli/endpoints.json`. The codegen cross-check fails if a
   `cli_mapping.command` from a REST exception isn't in the seed.

9. **UI feature.** Create `ui/src/features/tasks/`:
   - `TasksCard.tsx` — function component, mirrors
     `features/notes/NotesCard.tsx`.
   - `TasksCard.test.tsx` — tests.
   - `mocks/factories.ts` — domain-typed `makeTask` /
     `makeListTasksResponse` (proto-backed via `create(<Schema>, ...)`).
   - `mocks/tasks.ts` — `makeTasksMocks()` builder for `vi.mock(...)`.
   - `mocks/{factories,tasks}.test.ts` — self-tests.
   And `ui/src/api/tasks.ts` — exports the typed Connect client with
   `createClient(Tasks, transport)` plus any REST multipart helpers.
   Render timestamps with `timestampDate(...)` plus the locale-aware
   formatters in `ui/src/i18n/format.ts`, and surface API errors through
   `ui/src/lib/errorMessage.ts` so Connect codes map to i18n keys.
   Then add **one import + one render line** in `ui/src/App.tsx`:
   ```tsx
   import { TasksCard } from "./features/tasks/TasksCard";
   // ...
   <AppShell>
     <HealthCard />
     <NotesCard />
     <TasksCard />  // new
   </AppShell>
   ```

10. **Strings.** Add `tasks.title`, `tasks.create`, etc. to
   `ui/src/i18n/locales/en.json` (and the other locale files for
   parity). Run `pnpm strings:gen` from `ui/`.

11. **Run all tests.** API + CLI + UI. If anything breaks, fix
    your domain — the pattern is sound, your wiring isn't.

## Steps to remove the `notes` reference

Once your domain is green, delete notes. Each folder owns its own
schema, mocks, factories, tests, and helpers — folder deletion is the
fundamental: there's no central residue per Pass 3.

**1. Delete the four domain folders + the lib files.**

```bash
rm -rf api/internal/notes \
       api/handlers/notes \
       cli/domains/notes \
       proto/v1/notes \
       ui/src/features/notes \
       ui/src/api/notes.ts \
       ui/src/api/notes.test.ts
```

That single sweep takes the schema (`api/internal/notes/schema.{sql,go}`),
the API mocks (`api/internal/notes/mocks/*`), and the UI mocks +
factories (`ui/src/features/notes/mocks/*`) along with the rest of the
domain.

**2. Remove the import + Module + Schema + Endpoints registration lines.**
Three central files, three sweeps per surface:

```bash
# API runtime: drop the notesH import and the Module() call from main.go.
sed -i '/notesH "[^"]*\/handlers\/notes"/d' api/main.go
sed -i '/notesH\.Module/d' api/main.go

# Modules registry: drop the notesH.Endpoints + notesH.Schema entries.
sed -i '/notesH\.Endpoints/d' api/internal/modules/registry.go
sed -i '/notesH\.Schema/d' api/internal/modules/registry.go
# Then remove the now-orphan `notesH "..."` import line from registry.go:
sed -i '/notesH "[^"]*\/handlers\/notes"/d' api/internal/modules/registry.go

# CLI: drop the import + Register() call from domains.go.
sed -i '/\/cli\/domains\/notes/d' cli/domains/domains.go
sed -i '/notes\.Register/d' cli/domains/domains.go

# UI: drop the NotesCard import + render line from App.tsx.
sed -i '/NotesCard/d' ui/src/App.tsx
```

**3. Drop notes REST-exception entries from the codegen seed and regenerate:**

```bash
# Edit api/cmd/gen-endpoints/cli_commands_seed.json by hand: remove
# the `notes attach` entry. Connect-RPC commands are described from
# the proto service metadata, not manually seeded.
make endpoints
```

**4. Drop the notes-specific strings** from
`ui/src/i18n/locales/*.json` (search for `"notes":` blocks) and
re-run `pnpm strings:gen` from `ui/`.

**5. Drop the notes selectors block.** `ui/src/consts/selectors.ts`
groups data-testids by domain (`app`, `health`, `notes`, …). The
notes block (including its attachment-related entries
`attachmentUpload`, `attachmentFile`, `attachmentButton`,
`attachmentStatus`, `attachmentCount`) is dead code after step 1
deletes `features/notes/`. Open `ui/src/consts/selectors.ts` and
delete the entire `notes: { … },` block. Type-check stays green
either way (the selectors file is an exported object), but the
verification grep below catches the residue if you skip this step.

The remaining surface is your domain plus health. No notes residue —
including no orphan `notes` table created on every boot (schema
ownership lives in `internal/notes/`, deleted with the folder), no
orphan filesystem blobstore in `api/main.go` (the notes module owns
its own blob storage; deleting `notesH.Module(...)` takes the entire
attachments plumbing along), and no orphan `attachments_*` mocks
(co-located in `internal/notes/mocks/`, deleted with the folder).

> **Why these steps?** Step 1 is folder-deletion (schema, mocks,
> factories, tests, and the entire attachments sub-resource come
> along — `attach_handler.go`, `attachments_*.go`, `AttachmentUpload.tsx`,
> `attachments.proto`). Step 2 deletes three central registration
> lines per surface — and because `notesH.Module(db, clk, logger)`
> is the only consumer of attachments-side blob storage, removing
> that one call removes the only `blobstore` import in `api/main.go`.
> Steps 3–4 update two files codegen can't reach. Step 5 is the one
> place a domain leaks beyond its folder by design (centralised
> `data-testid` table for cross-cutting test queries).

## Verify the deletion is complete

The verification grep skips comment lines and shipped substrate
documentation. Substrate files (`api/internal/{database,module,
modules,server,httpx,httpc,testutil}/...`, `cli/internal/testutil/`,
`ui/src/test-utils/`, `docs/`) ship with educational doc-comments
that name `notes` as the canonical example — those are *intended* to
mention notes even after you delete the domain, so they're filtered
out. Anything that surfaces below is real residue you need to clean.

```bash
grep -rn 'notes\|Notes\|NOTES' \
    api/ cli/ ui/src/ proto/ .vrooli/ \
    --exclude-dir=node_modules \
    --exclude-dir=dist \
  | grep -v 'REPLACING-NOTES\|README' \
  | grep -vE '^[^:]+:[0-9]+:\s*(//|\*)' \
  | grep -vE '^api/cmd/gen-endpoints/main_test\.go:' \
  | grep -vE '^api/internal/(database|module|modules|server|httpx|httpc|testutil)/' \
  | grep -vE '^cli/internal/testutil/' \
  | grep -vE '^ui/src/test-utils/'
```

Expected: zero output. If any line surfaces, it's a residue you
need to clean. (The `gen-endpoints` test fixture exception is
because that test uses literal `notes`-prefixed strings as test
data for the codegen logic itself — they're not stale references.)

## Cross-references

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — the
  domain-module pattern this guide instantiates.
- [`SEAMS.md`](SEAMS.md) — the `module.Module` and codegen seams.
- [`TESTING.md`](TESTING.md) — what tests each layer needs.
- [`PROBLEMS.md`](PROBLEMS.md) — log any pattern divergences your
  domain forces here.
