# Start Here — {{SCENARIO_DISPLAY_NAME}}

This is the first document to read after generating the scenario from
the `react-vite` template. Use it to turn the scaffold into a specific
product without losing the template's architecture.

## First Session

1. **Confirm the scenario boots.**
   Run `make setup`, `make start`, `make status`, and `make test` from
   this scenario directory. Fix lifecycle or generation issues before
   product work.

2. **Write the charter.**
   Fill in `PRD.md` with purpose, users, deployment surfaces,
   operational targets, tech direction, dependencies, risks, and UX
   intent. Treat it as read-only after this initial pass except for
   automated checkbox updates.

3. **Seed requirements from operational targets.**
   Create `requirements/<number>-<target>/module.json` files that map to
   the P0/P1/P2 targets in `PRD.md`, then import them from
   `requirements/index.json`. Tests should later tag `[REQ:ID]`.

4. **Name the real domains.**
   Decide the scenario's bounded contexts before coding. For each
   domain, identify the data it owns, proto operations, API behavior,
   CLI commands, UI surface, storage needs, and test evidence.

5. **Decide resources and scenario dependencies.**
   Keep SQLite unless a domain truly needs a shared resource. Add
   resources or scenario dependencies in `.vrooli/service.json` only
   after documenting why in `PRD.md` or `docs/concepts/ARCHITECTURE.md`.

6. **Establish the design language.**
   Use `PRD.md`'s UX and branding section as the product-level promise.
   If the scenario needs stronger UI direction, add `docs/design.md`
   before building screens, then reflect concrete choices in
   `ui/src/styles.css`, `ui/tailwind.config.ts`, and reusable
   `ui/src/components/ui/` primitives.

7. **Build the first real vertical slice.**
   Add the first real domain beside the example `notes` domain. Mirror
   the pattern across proto, API domain, API handler module, CLI domain,
   UI API client, UI feature, i18n strings, selectors, and tests.

8. **Remove the example domain after your real slice is green.**
   Delete `notes` only after the real domain passes API, CLI, UI, and
   scenario tests.

9. **Record progress.**
   Append a concise row to `docs/internal/PROGRESS.md` after meaningful
   changes so future agents can reconstruct what happened.

## Architecture Rules

- Business logic belongs in `api/internal/<domain>/`.
- Wire contracts belong in
  `packages/proto/schemas/{{SCENARIO_ID}}/v1/<domain>/`.
- UI and CLI are translation layers over the API; they do not own
  business rules.
- Domain-owned schemas live next to the domain code.
- Generated files are regenerated, not hand-edited.
- `notes` is a worked example, not product functionality.

Read `docs/concepts/ARCHITECTURE.md` before changing structure, and
read `docs/internal/TESTING.md` before adding non-trivial tests.

## Replacing The Example Domain

Build your first real domain side-by-side with `notes`, then remove
`notes`. Use plural package/folder names such as `tasks`, `profiles`,
or `orders`; use PascalCase for components and Go exported names.

For a normal proto-backed CRUD domain:

1. Add proto messages and a service under
   `packages/proto/schemas/{{SCENARIO_ID}}/v1/<domain>/`, then run
   `make generate` from `packages/proto`.
2. Add `api/internal/<domain>/` with `types.go`, `repository.go`,
   storage implementation, `service.go`, `schema.sql`, `schema.go`,
   tests, and co-located mocks.
3. Add `api/handlers/<domain>/` with a thin generated Connect handler,
   `module.go`, conversion helpers if needed, endpoint descriptors, and
   tests.
4. Register the domain schema/endpoints in
   `api/internal/modules/registry.go` and mount the module in
   `api/main.go`.
5. Add `cli/domains/<domain>/` with declarative `cliapp.ArgSchema`
   commands that call generated Connect clients, then register the
   domain in `cli/domains/domains.go`.
6. Add endpoint-to-command seed rows in
   `api/cmd/gen-endpoints/cli_commands_seed.json`, then run
   `make endpoints`.
7. Add `ui/src/api/<domain>.ts` and `ui/src/features/<domain>/` with
   feature components, mocks, factories, selectors, i18n strings, and
   tests.
8. Run string/code generation as needed, then run `make test`.

If the domain needs opaque binary uploads, keep bytes on a REST
multipart edge and keep metadata proto-typed. The example `notes`
attachments path demonstrates that exception.

## Removing `notes`

After your real domain is green:

1. Delete `api/internal/notes`, `api/handlers/notes`,
   `cli/domains/notes`, `ui/src/features/notes`,
   `ui/src/api/notes.ts`, and `ui/src/api/notes.test.ts`.
2. Remove the `notes` imports, module registrations, schema entries, CLI
   registration, and `<NotesCard />` render.
3. Remove `notes` command rows from
   `api/cmd/gen-endpoints/cli_commands_seed.json`, then run
   `make endpoints`.
4. Remove notes-specific i18n keys and run `pnpm strings:gen` from
   `ui/`.
5. Remove the `notes` block from `ui/src/consts/selectors.ts`.
6. Verify no product residue remains with a focused search for
   `notes`, `Notes`, and `NOTES` in `api/`, `cli/`, `ui/src/`,
   `.vrooli/`, and the scenario's proto schema directory.

Expected end state: only health plus your real domains remain.
