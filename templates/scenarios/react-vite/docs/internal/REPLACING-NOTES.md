# Replacing the `notes` reference

The `notes` domain ships in every scenario generated from this
template as a worked CRUD example. It demonstrates the full vertical
stack: proto contract → API service + repository → handler +
endpoint descriptors → CLI domain → UI feature card. **It is meant
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
   `proto/v1/notes/notes.proto`'s message + service shape). From the
   workspace root, run `make generate`. New proto types appear under
   `packages/proto/gen/{go,ts}/{{SCENARIO_ID}}/v1/tasks/`.

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

3. **API handler + module.** Create `api/handlers/tasks/`:
   - `handler.go` — `Deps`, `NewHandler` returning a subrouter.
   - `module.go` — `Module(db, clk, logger) module.Module` that
     constructs repo + service + handler internally; also re-exports
     `func Schema() string { return internaltasks.Schema() }` so the
     registry collects all per-domain metadata via one symbol per
     handler package.
   - `endpoints.go` — `var Endpoints = []module.EndpointDescriptor{...}`
     mirroring the wire shape of each route.
   - `module_test.go`, `handler_test.go` — tests.

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
       notesH.Module(db, clock.System{}, log.Default()),
       tasksH.Module(db, clock.System{}, log.Default()),  // new
   )
   ```

5. **CLI domain.** Create `cli/domains/tasks/`:
   - `register.go` — `Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup`
     mirroring `cli/domains/notes/register.go`.
   - `handlers.go` — one function per subcommand, calling
     `core.Get(...)` / `core.Request(...)` for `/api/v1/tasks`.

6. **Wire into CLI.** Add **one line** to `cli/domains/domains.go`'s
   `SubcommandGroups`:
   ```go
   return []cliapp.SubcommandGroup{
       notes.Register(core),
       tasks.Register(core),  // new
   }
   ```

7. **CLI commands seed.** Add entries to
   `api/cmd/gen-endpoints/cli_commands_seed.json` (one per
   subcommand: `tasks list`, `tasks create`, `tasks get`). Run
   `make endpoints` to regenerate `.vrooli/endpoints.json`. The
   codegen cross-check fails if a `cli_mapping.command` from your
   new endpoints isn't in the seed.

8. **UI feature.** Create `ui/src/features/tasks/`:
   - `TasksCard.tsx` — function component, mirrors
     `features/notes/NotesCard.tsx`.
   - `TasksCard.test.tsx` — tests.
   - `mocks/factories.ts` — domain-typed `makeTask` /
     `makeListTasksResponse` (proto-backed via `create(<Schema>, ...)`).
   - `mocks/tasks.ts` — `makeTasksMocks()` builder for `vi.mock(...)`.
   - `mocks/{factories,tasks}.test.ts` — self-tests.
   And `ui/src/lib/tasks.ts` — fetcher + types.
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

9. **Strings.** Add `tasks.title`, `tasks.create`, etc. to
   `ui/src/i18n/locales/en.json` (and the other locale files for
   parity). Run `pnpm strings:gen` from `ui/`.

10. **Run all tests.** API + CLI + UI. If anything breaks, fix
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
       ui/src/lib/notes.ts \
       ui/src/lib/notes.test.ts
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

**3. Drop the notes entries from the codegen seed and regenerate:**

```bash
# Edit api/cmd/gen-endpoints/cli_commands_seed.json by hand: remove
# the three `notes list/create/get` entries.
make endpoints
```

**4. Drop the notes-specific strings** from
`ui/src/i18n/locales/*.json` (search for `"notes":` blocks) and
re-run `pnpm strings:gen` from `ui/`.

The remaining surface is your domain plus health. No notes residue —
including no orphan `notes` table created on every boot (Pass-3 moved
schema ownership to `internal/notes/`, deleted with the folder).

> **Why these steps?** Step 1 is folder-deletion (schema, mocks,
> factories, tests come along). Step 2 deletes three central
> registration lines per surface. Steps 3–4 update two files codegen
> can't reach. The cohesion gain shows up here: where prior versions
> needed nine deletions across four locations, the surface is now
> mostly `rm -rf`.

## Verify the deletion is complete

```bash
grep -rn "notes\|Notes\|NOTES" \
  api/ cli/ ui/src/ proto/ docs/ .vrooli/ \
  --exclude-dir=node_modules \
  --exclude-dir=dist \
  --exclude-dir=internal/notes-template-* \
  | grep -v "REPLACING-NOTES\|README"
```

Expected: zero output. If any line surfaces, it's a residue you
need to clean.

## Cross-references

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — the
  domain-module pattern this guide instantiates.
- [`SEAMS.md`](SEAMS.md) — the `module.Module` and codegen seams.
- [`TESTING.md`](TESTING.md) — what tests each layer needs.
- [`PROBLEMS.md`](PROBLEMS.md) — log any pattern divergences your
  domain forces here.
