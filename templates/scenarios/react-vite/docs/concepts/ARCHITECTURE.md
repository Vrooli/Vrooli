# Architecture — {{SCENARIO_DISPLAY_NAME}}

This document is the entry point for understanding what shape this
scenario has and why. Read it before [`SEAMS.md`](../internal/SEAMS.md)
or [`TESTING.md`](../internal/TESTING.md) — those documents assume you
already know where things live.

The shape below is **inherited from the `react-vite` template** and is
intentionally invariant across every scenario generated from it. If
your scenario diverges from this layout, document the deviation in
[`PROBLEMS.md`](../internal/PROBLEMS.md) so future agents know it was
deliberate.

## The three surfaces

A scenario is one product expressed through three coordinated surfaces.

```
                       ┌─────────────────────────────┐
                       │  Generated proto types      │
                       │  packages/proto/schemas/    │
                       │   {{SCENARIO_ID}}/v1/...    │
                       └──────────────┬──────────────┘
                                      │ canonical wire shape
              ┌───────────────────────┼───────────────────────┐
              │                       │                       │
              ▼                       ▼                       ▼
        ┌──────────┐            ┌──────────┐            ┌──────────┐
        │   ui/    │ Connect-JSON│  api/   │ Connect-JSON│  cli/   │
        │ React    │ ◀────────▶ │   Go     │ ◀────────▶ │   Go     │
        │ + Vite   │            │ HTTP     │            │ cli-core │
        └──────────┘            └────┬─────┘            └──────────┘
                                     │
                                     ▼
                                ┌─────────┐
                                │ SQLite  │
                                │ (local) │
                                └─────────┘
```

| Surface | Role | Owns |
|---|---|---|
| **`api/`** | The scenario's core. All business logic, persistence, and external integrations live here. | Domain rules, storage, transport edge |
| **`ui/`** | A thin React/Vite client that renders state and triggers commands. Always present (every scenario has a UI). | Components, i18n, accessibility, browser concerns |
| **`cli/`** | A thin Go wrapper over the API for scripting, agents, and operators. | Argument parsing, output formatting; **never** business logic |
| **`packages/proto/schemas/{{SCENARIO_ID}}/`** | Wire contracts in `.proto` files. The template source starts under `proto/`, but generation relocates it here automatically via `template.json::relocations`. Code is generated for Go (API + CLI), TypeScript (UI), Python, and Connect transports. | One canonical type and service descriptor per proto-typed operation |

**The load-bearing principle:** the API is the only surface that contains
business logic. UI and CLI are translation layers. Proto types flow from
one source of truth so wire-shape drift between surfaces is impossible.

## Proto as the canonical contract

Wire shapes do not live in TypeScript interfaces, Go structs, or
hand-written JSON schemas. They live in `.proto` files only. For
proto-typed API calls, the `.proto` file also declares the `service`
block that generates the Connect handler and clients.

```
packages/proto/schemas/{{SCENARIO_ID}}/v1/health/health.proto
       │
       ▼
       make generate
       │
       ├──▶  packages/proto/gen/go/{{SCENARIO_ID}}/v1/...              (messages for api/, cli/)
       ├──▶  packages/proto/gen/go/{{SCENARIO_ID}}/v1/...connect       (Connect-Go handlers/clients)
       ├──▶  packages/proto/gen/typescript/js/{{SCENARIO_ID}}/v1/...   (messages + service descriptors for ui/)
       └──▶  packages/proto/gen/python/{{SCENARIO_ID_SNAKE}}/v1/...    (Python messages)
```

**Adding a new proto-typed operation:** add request/response messages
and an RPC method to the domain's `.proto` service, regenerate, then
implement the generated Go handler interface. The UI and CLI call the
generated client. No hand-written struct, route parser, or interface
mirror exists to drift.

The codegen pipeline runs entirely on local plugins (no BSR network
calls), so it works in CI, on flight Wi-Fi, and inside firewalled
runners. See [`packages/proto/schemas/{{SCENARIO_ID}}/README.md`](../../../../packages/proto/schemas/{{SCENARIO_ID}}/README.md) and the
project-level proto pipeline guide for details.

### Connect-RPC vs REST

Use Connect-RPC by default:

- UI ↔ API for proto-typed payloads: Connect-RPC.
- CLI ↔ API for proto-typed payloads: Connect-RPC.
- API ↔ API / inter-scenario calls with Vrooli-owned protos: Connect-RPC.

Use REST only when the payload shape is not Vrooli-owned proto data:

- File uploads and other opaque binary edges: REST multipart; proto
  describes metadata only.
- Webhook receivers and third-party APIs: REST/JSON in the shape the
  external system requires.
- Operational endpoints that must be reachable by lifecycle systems,
  load balancers, and curl probes without a generated client: REST.
  `/health` is the template's canonical example.

If an internal endpoint would be "REST because it is simple", add a
proto service method instead. That keeps route, request type, response
type, and error-code drift inside generated code.

## Inside the API: layered architecture

```
Connect request
     │
     ▼
handlers/<domain>/connect_handler.go  ◀── transport edge: generated request/response types, error-translate
     │                                    (NEVER business logic; orchestrates one service call)
     ▼
internal/<domain>/service.go          ◀── application layer: validation, defaults, policy
     │                                    (depends on Repository interface, not concrete sqlite)
     ▼
internal/<domain>/repository.go       ◀── persistence interface
internal/<domain>/sqlite.go               concrete sqlite implementation
internal/<domain>/schema.sql              domain-owned table DDL (Pass-3)
internal/<domain>/schema.go               //go:embed + Schema() string
internal/<domain>/mocks/                  co-located test fakes (FakeRepository, FakeService)
     │
     ▼
internal/database/                    ◀── cross-cutting infrastructure: Pinger + system schema home
  pinger.go                               connection-level reachability seam
  system.sql                              empty-by-default home for cross-cutting SQL
                                          (postgres extensions, custom types, etc.)
internal/modules/                     ◀── shared registry: AllEndpoints + AllSchemas
                                          (consumed by main.go and gen-endpoints)
```

Domain code lives in `internal/<domain>/` (e.g., `internal/notes/`),
**not** in a generic `internal/services/` directory. The notes domain
is the canonical worked example — copy its layout when adding the
first non-trivial mutation.

Cross-cutting concerns:

- `internal/clock/` — `Clock` seam for deterministic time in tests
- `internal/middleware/` — request logging (uses `Clock`, not `time.Now`)
- `internal/httpx/` — typed error envelope for REST exceptions
- `internal/httpc/` — outbound HTTP `Doer` seam (ships unwired by intent)
- `internal/module/` — `Module` data type + `EndpointDescriptor`; the seam each domain returns to be mounted
- `internal/server/` — composes a slice of modules + cross-cutting middleware (no per-domain code)
- `internal/testutil/` — fakes, fixtures, harness helpers (test-only)

The `var _ Interface = (*Impl)(nil)` compile-time assertion is used
everywhere a seam exists. If an interface declared in a domain package
isn't asserted, the runtime "does this satisfy the interface" surprise
is one rename away.

See [`SEAMS.md`](../internal/SEAMS.md) for the authoritative seam
registry and [`TESTING.md`](../internal/TESTING.md) for how each layer
is tested.

## Domain modules — adding and removing features

The horizontal axis: each domain self-describes via a `Module()`
constructor in its handler package. `main.go` lists them; the server
iterates and mounts.

```
  api/main.go
       │
       ├─ healthH.Module(db, "{{SCENARIO_ID}}-api", "1.0.0") ──┐
       ├─ notesH.Module(db, clk, logger) ───────────────────────┼─→ server.New(deps, modules...)
       └─ tasksH.Module(...)  ←─ adding a domain = one line ────┘         │
                                                                           │
                                                                           ├─ for m := range modules: m.Mount(router)
                                                                           └─ Handler() → http.Handler
```

`module.Module` is a data type (not an interface): `{Name string;
Mount func(r *mux.Router); Endpoints []EndpointDescriptor}`. Each
domain owns its own constructor, its own routes, and its own slice of
endpoint descriptors. `server.Deps` shrinks to cross-cutting concerns
only (`Clock`, `Logger`); per-domain dependencies live inside the
module's constructor.

This architecture document owns the invariant: a domain adds local
files plus small registration lines in `api/main.go`,
`api/internal/modules/registry.go`, `cli/domains/domains.go`, and
`ui/src/App.tsx`.

No central schema file is edited, `.vrooli/endpoints.json` is generated
with `make endpoints`, and `gen-endpoints/main.go` does not change for
new domains.

### Domain-owned schema

Each domain owns its database schema the same way it owns its types,
repository, and service. `internal/<dom>/schema.sql` declares the
domain's tables; `internal/<dom>/schema.go` embeds it via `go:embed`
and exports `Schema() string`. Adding a column lands in the same diff
as the Go field, the repository scan, and the wire shape — single
location, single edit.

Cross-cutting infrastructure (postgres extensions, custom types,
cross-domain views) lives in `internal/database/system.sql` — empty by
default in SQLite scenarios. If you find yourself adding a `CREATE
TABLE` to the system file, the table belongs to a domain that doesn't
exist yet. Create the domain first.

Boot path:

```
main.go → apidb.EnsureSchemas(ctx, db, modules.AllSchemas()...)
          ↑                              ↑
          api-core/database helper       local shared registry
                                         (system + per-domain providers)
```

Why this shape:

- **High cohesion** — schema lives next to the code that interprets it.
- **Locality of change** — single-location edits for single logical
  changes (adds, columns, indexes).
- **Deletability** — `rm -rf internal/<dom>/` takes the schema with
  the rest of the domain. No orphan tables are created on boot after a
  domain is removed.
- **Bounded contexts** — each domain owns its data, mirroring the rest
  of the per-domain stack.

When central schema is the right answer:

- Highly relational scenarios with many cross-domain FKs. The
  bounded-context fiction breaks down; a central schema is honest
  about the coupling. (Prefer soft FKs — ID references with no
  constraint — before reaching for this.)
- Scenarios where the schema *is* the product (analytics warehouses,
  data platforms). The schema deserves its own first-class home.

The template targets bounded-context CRUD scenarios, so per-domain is
the default.

For brownfield scenarios with production data needing column drops or
renames, see the deferred versioned-migration helpers (`Migrate` /
`MigrationProvider` in `api-core/database`, landing when the first
scenario hits the pain).

**The trade-off:** one layer of indirection (the `Module` data type)
pays for the open–closed property. For a template that gets forked
many times, the cost is paid once and saved per scenario. The CLI,
API, UI, and endpoint manifest all use the same principle: domains own
their local files, while the center gets only small registration lines.

## Inside the UI: feature-shaped React

```
ui/src/
  main.tsx              ◀── composition root: i18n + QueryClient + iframe-bridge + ErrorBoundary
  App.tsx               ◀── 15-line composition: <AppShell> + per-feature cards
  components/
    AppShell.tsx        ◀── outer layout + locale switcher; takes children
    ErrorBoundary.tsx   ◀── render-error catch (class component, localised fallback)
    ui/                 ◀── shadcn-style primitives (Button, Input, Textarea)
  features/             ◀── per-domain UI (delete a folder to delete the feature)
    health/
      HealthCard.tsx    ◀── system-status reference; ships in every scenario
    notes/
      NotesCard.tsx     ◀── canonical CRUD reference; replace per scenario
  hooks/                ◀── custom hooks (spatial nav, gamepad)
  api/
    client.ts           ◀── substrate: Connect transport + REST multipart error helpers
    health.ts           ◀── health endpoint wrapper
    notes.ts            ◀── generated Connect client + attachment upload helper
  lib/
    profiler.ts         ◀── browser performance helpers
    utils.ts            ◀── pure UI utilities
  i18n/
    index.ts            ◀── i18next singleton; locale persistence; <html lang>/<html dir>
    format.ts           ◀── Intl-based date/number/currency/list helpers
    locales/            ◀── one JSON catalog per locale; en.json is canonical
  consts/
    selectors.ts        ◀── typed test-id registry (literal + dynamic)
    strings.ts          ◀── public re-export of strings.generated.ts
    strings.generated.ts ◀── codegen from en.json (DO NOT edit by hand)
  test-utils/           ◀── test scaffolding (mocks, factories, renderWithProviders)
```

**i18n flow:** every JSX literal goes through `t(strings.x.y)`.
ESLint forbids inline copy and `*ByText("literal")` queries in tests,
so copy edits are one-line catalog changes and tests survive them
automatically. The locale-parity test in `i18n/locales/locales.test.ts`
fails the build if other locales drift from `en.json`'s key shape.

**Selector strategy:** tests query DOM with `getByTestId(selectors.x.y)`
or `getByText(strings.x.y)` — never with literal copy. The
`selectors.ts` map supports both static IDs (`selectors.app.title`)
and parameterized IDs (`selectors.locale.toggle({ code: "ja" })`)
with full type safety.

**Vrooli-specific wiring:** `main.tsx` initializes
`@vrooli/iframe-bridge` so the UI works embedded in App Monitor or
proxied through Cloudflare. All API calls resolve via `@vrooli/api-base`
so no scenario hardcodes a localhost URL.

See [`reference/api-endpoints.md`](../reference/api-endpoints.md) for
endpoint shapes and [`SEAMS.md`](../internal/SEAMS.md#ui-side-seams)
for UI seam patterns.

## Inside the CLI: thin wrapper, domain-organized

```
cli/
  main.go               ◀── entrypoint; calls app.Run
  app.go                ◀── NewStandardScenarioApp; metadata + cli-core wiring
  install.sh            ◀── POSIX installer (Linux/macOS)
  install.ps1           ◀── Windows installer (PowerShell)
  domains/
    domains.go          ◀── aggregator; registers each domain's SubcommandGroup
    notes/
      register.go       ◀── Subcommands: list, create, get
      handlers.go       ◀── calls core.Get / core.Request; renders via cliapp.Render*Report
  internal/testutil/    ◀── parallel of api/internal/testutil/
```

The CLI **never embeds business logic**. Handlers construct generated
Connect clients with `cliapp.NewConnectHTTPClient` and format typed
responses. REST-only commands, such as multipart uploads, use
`cliapp.UploadFile` and decode the proto metadata response. If a CLI
command needs to make a decision the API doesn't expose, the correct
fix is to add the API operation, not to compute it locally.

`cli-core` provides the scaffolding — global flags (`--api-base`,
`--auto-start`, `--json`, `--no-color`), env-var precedence, config-file
location, stale-binary detection, status/configure commands. The
template's CLI is a few hundred lines because cli-core does the heavy
lifting.

**Declarative argument schemas.** Each `cliapp.Command` declares its
flags and positionals as `Args cliapp.ArgSchema` and exposes a
`RunCtx func(ctx cliapp.RunContext) error` handler. The schema is one
source of truth: the parser (`--flag value`, `-f value`, `--flag=value`,
positionals, `--`, `--help`) reads from it; `--help` output is
generated from it; the runtime `RunContext` exposes typed accessors
(`ctx.Flag("title")`, `ctx.Positional("id")`, `ctx.JSON()`) keyed by
the same names. Adding a flag means adding a row to the schema —
`flag.NewFlagSet`, manual `--help` strings, and per-handler HTTP
serialization are all gone. The notes domain
(`cli/domains/notes/{register,handlers}.go`) is the canonical worked
example; mirror its shape when adding a second domain.

See [`reference/cli-commands.md`](../reference/cli-commands.md) for
the user-facing command set.

## Storage

The template ships with **SQLite via `modernc.org/sqlite`** — pure-Go,
CGO-clean, no external database process. Each domain owns its schema
(`api/internal/<dom>/schema.sql`, embedded via `go:embed`); cross-cutting
infrastructure lives in `api/internal/database/system.sql` (empty by
default in SQLite scenarios). The shared registry at
`api/internal/modules/registry.go::AllSchemas()` collects them, and
`apidb.EnsureSchemas(ctx, db, modules.AllSchemas()...)` applies them at
startup — idempotent (`CREATE TABLE IF NOT EXISTS` everywhere), safe to
call on every boot.

This is the right default for most scenarios. Scenarios that need
Postgres, Redis, Qdrant, or other resources declare them in
`.vrooli/service.json` and connect through `api-core/database`. See
the storage-steer skill for the full hierarchy.

## Cross-platform contract

Every binary built from this template runs on Linux, macOS, and
Windows without modification:

- `CGO_ENABLED=0` is enforced in CI — no C dependencies sneak in
- `modernc.org/sqlite` is pure-Go (the reason CGO can stay disabled)
- `install.sh` + `install.ps1` are paired installers; the manifest
  selects per-OS in `.vrooli/service.json::cli.install`
- File paths are resolved through `api-core/storage`, not hardcoded
- Ports come from env vars assigned by the lifecycle, not literals

See the cross-platform-readiness skill if your scenario adds a new
dependency or a new install path.

## Testing aligns with architecture

The test layout mirrors the production layout. Each layer has its own
test seam and its own fake. This is not coincidence — it falls out of
clean responsibility boundaries.

| Production layer | Test seam | Fake/harness |
|---|---|---|
| Connect transport | generated Connect client over real handler | `httptest.Server` or in-process handler client |
| Handler → Service | `notes.Service` interface | `notes/mocks.FakeService` (co-located) |
| Service → Repository | `notes.Repository` interface | `notes/mocks.FakeRepository` (co-located) |
| Repository → DB | `*sql.DB` | `db.NewSQLite(t)` (in-memory) |
| Time | `clock.Clock` interface | `mocks.FakeClock` (cross-domain) |
| DB reachability | `database.Pinger` interface | `mocks.FakePinger` (cross-domain) |
| Outbound HTTP | `httpc.Doer` interface | `mocks.FakeDoer` |
| UI ↔ API | `api/client.ts` transport / `api/notes.ts` typed client | inline `vi.mock` |
| Render-error catch | `ErrorBoundary` (system under test) | controlled-throw fixture |

The full register lives in [`SEAMS.md`](../internal/SEAMS.md). Test
patterns and primitives live in [`TESTING.md`](../internal/TESTING.md).

## What this scenario is NOT

Boundary-setting matters as much as architecture. This template
deliberately does **not** ship:

- A router (the App is single-page until the scenario needs more)
- An auth provider (scenarios add one when they have something to protect)
- A state library beyond React Query + `useState` (Redux/Zustand only when warranted)
- A form-validation framework (add when forms appear)
- A theme system (Tailwind config is bare; extend per scenario)
- A monorepo workspace (`pnpm install --ignore-workspace` keeps the UI standalone)

These are decisions deferred to the scenario, not gaps in the template.

## Cross-references

- [`QUICKSTART.md`](../QUICKSTART.md) — clone-to-running in 5 minutes
- [`reference/configuration.md`](../reference/configuration.md) — env vars, service.json, CLI config
- [`reference/api-endpoints.md`](../reference/api-endpoints.md) — endpoint reference
- [`reference/cli-commands.md`](../reference/cli-commands.md) — CLI command reference
- [`guides/troubleshooting.md`](../guides/troubleshooting.md) — common issues and fixes
- [`internal/SEAMS.md`](../internal/SEAMS.md) — seam registry
- [`internal/TESTING.md`](../internal/TESTING.md) — test patterns
- [`internal/PROBLEMS.md`](../internal/PROBLEMS.md) — known issues / tech debt
- [`internal/PROGRESS.md`](../internal/PROGRESS.md) — lifecycle log
