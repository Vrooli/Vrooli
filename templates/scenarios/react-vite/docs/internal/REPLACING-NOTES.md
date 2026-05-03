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
   - `service_test.go`, `sqlite_test.go` — tests.

3. **API handler + module.** Create `api/handlers/tasks/`:
   - `handler.go` — `Deps`, `NewHandler` returning a subrouter.
   - `module.go` — `Module(db, clk, logger) module.Module` that
     constructs repo + service + handler internally.
   - `endpoints.go` — `var Endpoints = []module.EndpointDescriptor{...}`
     mirroring the wire shape of each route.
   - `module_test.go`, `handler_test.go` — tests.

4. **Wire into main.** Add **one line** to `api/main.go`'s
   `server.New(...)` slice:
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

Once your domain is green, delete notes. The full sequence:

**1. Remove the six domain folders + the lib + the lib test:**

```bash
rm -rf api/internal/notes \
       api/handlers/notes \
       cli/domains/notes \
       proto/v1/notes \
       ui/src/features/notes \
       ui/src/lib/notes.ts \
       ui/src/lib/notes.test.ts
```

**2. Remove the four notes mocks** (FakeRepository + FakeService +
their self-tests). These live under `internal/testutil/mocks/` but
are domain-specific to notes:

```bash
rm api/internal/testutil/mocks/notes_repository.go \
   api/internal/testutil/mocks/notes_service.go \
   api/internal/testutil/mocks/notes_repository_test.go \
   api/internal/testutil/mocks/notes_service_test.go
```

**3. Remove the import + Module registration lines** (one per surface):

```bash
# API: drop the import and the Module() call from main.go.
sed -i '/notesH "[^"]*\/handlers\/notes"/d' api/main.go
sed -i '/notesH\.Module/d' api/main.go

# CLI: drop the import and the Register() call from domains.go.
sed -i '/\/cli\/domains\/notes/d' cli/domains/domains.go
sed -i '/notes\.Register/d' cli/domains/domains.go

# UI: drop the NotesCard import and render line from App.tsx.
sed -i '/NotesCard/d' ui/src/App.tsx
```

**4. Drop the notes endpoints from the codegen seed and the
gen-endpoints program**:

```bash
# Edit api/cmd/gen-endpoints/cli_commands_seed.json by hand:
# remove the four `notes list/create/get` entries.

# Drop the notesH import and the notesH.Endpoints append from
# api/cmd/gen-endpoints/main.go (one import line, one append line).
sed -i '/notesH "[^"]*\/handlers\/notes"/d' api/cmd/gen-endpoints/main.go
sed -i '/notesH\.Endpoints\.\.\./d' api/cmd/gen-endpoints/main.go

# Regenerate the manifest.
make endpoints
```

**5. Drop the notes-specific strings** from
`ui/src/i18n/locales/*.json` (search for `"notes":` blocks) and
re-run `pnpm strings:gen` from `ui/`.

The remaining surface is your domain plus health. No notes residue.

> **Why so many steps?** The fundamentals are the 6 folder deletions
> + 3 single-line removals (one per surface). The mocks (#2) and
> codegen wire-up (#4) are domain-specific files that a real-world
> scenario would replace with its own equivalents — they're listed
> for completeness so the throwaway test passes cleanly.

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
