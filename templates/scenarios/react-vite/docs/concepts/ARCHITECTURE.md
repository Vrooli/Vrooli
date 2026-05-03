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
        │   ui/    │  HTTP/JSON │   api/   │ HTTP/JSON  │   cli/   │
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
| **`proto/`** | Wire contracts in `.proto` files. At generate-time, relocated to `packages/proto/schemas/{{SCENARIO_ID}}/v1/`. Code is generated for both Go (API + CLI) and TypeScript (UI). | One canonical type per message; consumed by all three surfaces |

**The load-bearing principle:** the API is the only surface that contains
business logic. UI and CLI are translation layers. Proto types flow from
one source of truth so wire-shape drift between surfaces is impossible.

## Proto as the canonical contract

Wire shapes do not live in TypeScript interfaces, Go structs, or
hand-written JSON schemas. They live in `.proto` files only.

```
proto/v1/health/health.proto    ── source of truth
       │
       │  (vrooli scenario generate relocates to
       │   packages/proto/schemas/{{SCENARIO_ID}}/v1/health/)
       ▼
       make generate
       │
       ├──▶  packages/proto/gen/go/{{SCENARIO_ID}}/v1/health/  (used by api/, cli/)
       └──▶  packages/proto/gen/ts/{{SCENARIO_ID}}/v1/health/  (used by ui/)
```

**Adding a new endpoint:** add a `.proto` message, regenerate, import
the generated type from the API handler, the UI client, and the CLI
handler. No hand-written struct or interface mirror exists to drift.

The codegen pipeline runs entirely on local plugins (no BSR network
calls), so it works in CI, on flight Wi-Fi, and inside firewalled
runners. See [`proto/README.md`](../../proto/README.md) and the
project-level proto pipeline guide for details.

## Inside the API: layered architecture

```
HTTP request
     │
     ▼
handlers/<domain>/handler.go          ◀── transport edge: parse, validate, error-translate
     │                                    (NEVER business logic; orchestrates one service call)
     ▼
internal/<domain>/service.go          ◀── application layer: validation, defaults, policy
     │                                    (depends on Repository interface, not concrete sqlite)
     ▼
internal/<domain>/repository.go       ◀── persistence interface
internal/<domain>/sqlite.go               concrete sqlite implementation
     │
     ▼
internal/store/                       ◀── shared infrastructure: schema, Pinger
*.sql                                     embedded schema, applied at startup
```

Domain code lives in `internal/<domain>/` (e.g., `internal/notes/`),
**not** in a generic `internal/services/` directory. The notes domain
is the canonical worked example — copy its layout when adding the
first non-trivial mutation.

Cross-cutting concerns:

- `internal/clock/` — `Clock` seam for deterministic time in tests
- `internal/middleware/` — request logging (uses `Clock`, not `time.Now`)
- `internal/httpx/` — JSON decode + typed error envelope
- `internal/httpc/` — outbound HTTP `Doer` seam (ships unwired by intent)
- `internal/server/` — wires `Deps` and registers routes
- `internal/testutil/` — fakes, fixtures, harness helpers (test-only)

The `var _ Interface = (*Impl)(nil)` compile-time assertion is used
everywhere a seam exists. If an interface declared in a domain package
isn't asserted, the runtime "does this satisfy the interface" surprise
is one rename away.

See [`SEAMS.md`](../internal/SEAMS.md) for the authoritative seam
registry and [`TESTING.md`](../internal/TESTING.md) for how each layer
is tested.

## Inside the UI: feature-shaped React

```
ui/src/
  main.tsx              ◀── composition root: i18n + QueryClient + iframe-bridge + ErrorBoundary
  App.tsx               ◀── top-level shell (extract sections as the surface grows)
  components/
    ErrorBoundary.tsx   ◀── render-error catch (class component, localised fallback)
    ui/                 ◀── shadcn-style primitives (Button, Input, Textarea)
  hooks/                ◀── custom hooks (spatial nav, gamepad)
  lib/
    api.ts              ◀── network boundary: fetch + proto-typed decode
    notes.ts            ◀── canonical CRUD wrapper (ApiError + typed envelope code)
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

The CLI **never embeds business logic**. Handlers exclusively call
`core.Get(...)` / `core.Request(...)` and format the response. If a
CLI command needs to make a decision the API doesn't expose, the
correct fix is to add the API endpoint, not to compute it locally.

`cli-core` provides the scaffolding — global flags (`--api-base`,
`--auto-start`, `--json`, `--no-color`), env-var precedence, config-file
location, stale-binary detection, status/configure commands. The
template's CLI is a few hundred lines because cli-core does the heavy
lifting.

See [`reference/cli-commands.md`](../reference/cli-commands.md) for
the user-facing command set.

## Storage

The template ships with **SQLite via `modernc.org/sqlite`** — pure-Go,
CGO-clean, no external database process. The schema lives at
`api/internal/store/schema.sql` and is embedded into the binary;
`store.EnsureSchema(ctx, db)` applies it at startup (idempotent —
safe to call on every boot).

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
| HTTP transport | `httpx.NewLiveServer` over real socket | `httptest.Server` (real, not Recorder) |
| Handler → Service | `notes.Service` interface | `mocks.FakeService` |
| Service → Repository | `notes.Repository` interface | `mocks.FakeRepository` |
| Repository → DB | `*sql.DB` | `db.NewSQLite(t)` (in-memory) |
| Time | `clock.Clock` interface | `mocks.FakeClock` |
| DB reachability | `store.Pinger` interface | `mocks.FakePinger` |
| Outbound HTTP | `httpc.Doer` interface | `mocks.FakeDoer` |
| UI ↔ API | `lib/api.ts` / `lib/notes.ts` modules | inline `vi.mock` |
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
