# React + Vite Scenario Template (Go API + CLI)

Use this template to bootstrap every new scenario. It mirrors the patterns from `browser-automation-studio`:

- Go API (`api/`)
- React + TypeScript + Vite UI (`ui/`)
- CLI wrapper (`cli/`)
- Lifecycle + health wiring (`.vrooli/service.json`)
- Requirements registry + progress log (`requirements/`, `docs/internal/PROGRESS.md`)

## Copy the Template
```bash
# From the repo root
vrooli scenario generate react-vite \
  --id <your-scenario> \
  --display-name "Your Scenario" \
  --description "One sentence summary"
cd scenarios/<your-scenario>/
```

Immediately replace placeholder tokens (scenario name, description, maintainer info, etc.).

> **The `notes` domain is a worked example, not a starting feature.** It ships to demonstrate the canonical layering — proto contract → repository → service → handler → CLI domain → UI consumer, each with its own self-test. Copy the *structure* when you add your own domains, then delete the notes code once you have a real domain replacing it. Per Pass 3, deletion is mostly folder removal: `api/internal/notes/` (takes the schema, the co-located mocks, all tests with it), `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/` (takes its co-located UI mocks + factories with it), `ui/src/api/notes.ts` + test, the notes pane in `ui/src/App.tsx`, the `notes.*` keys in `ui/src/consts/strings.ts` + `ui/src/i18n/locales/*.json`, the `notesH.Endpoints` + `notesH.Schema` lines in `api/internal/modules/registry.go`, the `notes_*` entries in `.vrooli/endpoints.json`, and the proto schemas under `proto/v1/notes/` + `proto/v1/errors/`. Update the matching rows in `docs/internal/SEAMS.md` when you do. Full sequence in [`docs/internal/REPLACING-NOTES.md`](docs/internal/REPLACING-NOTES.md).

## What You Get
- **Clean UI scaffold**: Vite + Tailwind + shadcn-style primitives, pnpm-based scripts, Vitest + Testing Library pre-configured, `.env.example` for API URL.
- **i18n out of the box**: i18next + react-i18next wired up with locale persistence, RTL-ready `<html lang>`/`<html dir>`, an `Intl`-based formatter helper (date/number/currency/relative-time/list), and a typed key registry generated from `en.json` at build time. ESLint forbids inline copy and string-literal `*ByText` queries, so copy edits are one-line catalog changes and tests survive them automatically. See [i18n flow](#i18n-flow).
- **Go API skeleton**: `go.mod` + `cmd/server` entrypoint ready for feature modules.
- **CLI manifest contract**: `.vrooli/service.json` declares the CLI command, adapter, install strategies, and freshness inputs.
- **Lifecycle-ready service.json**: ports aligned with the platform, no external resource required by default (storage is local SQLite), lifecycle steps that build API/UI and start dev servers.
- **Iframe-ready UI**: Automatically initializes `@vrooli/iframe-bridge` so App Monitor and other hosts can embed the scenario without extra work.
- **Smart API resolution**: UI uses `@vrooli/api-base` to resolve the correct API URL across localhost/dev/proxy contexts.
- **Requirements seed**: `requirements/index.json` + `requirements/modules/foundation.json` show how operational targets trace to technical requirements.
- **Lifecycle metadata seed**: `.vrooli/service.json`, `endpoints.json`, `testing.json`, and `lighthouse.json` so status/health/testing commands work immediately after copy.
- **Progress log**: `docs/internal/PROGRESS.md` so improvers track deltas outside PRD.md.
- **SQLite storage with domain-owned schema**: API uses `api-core/database` with the `sqlite` driver and `api-core/storage` for filesystem-safe paths — no external DB process required. Each domain owns its schema (`api/internal/<dom>/schema.sql`, embedded via `go:embed`); cross-cutting infrastructure (postgres extensions, custom types, cross-domain views) lives in `api/internal/database/system.sql` (empty by default in SQLite scenarios). All applied at startup via `apidb.EnsureSchemas` from `api-core/database`, over the shared registry at `api/internal/modules/registry.go`. Deleting a domain folder removes its table definition with it — no orphan tables on boot.
- **Proto-first wire contracts**: The template ships proto sources at `proto/{{SCENARIO_ID}}/v1/health/health.proto`. At generation time, `vrooli scenario generate` relocates them into `packages/proto/schemas/<your-scenario>/`, runs `make generate`, and the API + UI immediately consume the generated Go and TypeScript types. No hand-written wire shape duplication; adding a new endpoint means adding a `.proto`, regenerating, and importing the typed message. The codegen pipeline runs entirely on local plugins — no BSR network calls — so it works on flight Wi-Fi or inside firewalled CI runners. See `proto/README.md`, `docs/internal/TESTING.md::How to add a new proto`, and the project-level [proto pipeline guide](../../docs/development/proto.md).
- **Notes CRUD reference**: A worked end-to-end example at `/api/v1/notes` (list / create / get-by-id) backed by sqlite, with proto-typed wire contracts (`v1/notes/`, `v1/errors/`), a typed error envelope, repository tests on a real handle, module + handler tests through the live HTTP harness, a UI feature at `ui/src/features/notes/NotesCard.tsx` (consumes `ui/src/api/notes.ts`), and a CLI domain at `cli/domains/notes/`. Copy this layering when adding your first non-trivial mutation — every layer has its own self-test, so the pattern is hard to break. The full delete-and-replace recipe is in [`docs/internal/REPLACING-NOTES.md`](docs/internal/REPLACING-NOTES.md).
- **Domain-scoped application-service layer**: The notes domain ships the canonical Vrooli layering — `internal/notes/` owns `Repository` (persistence), `Service` (validation, defaults, cross-handler policy), and the typed sentinels (`ErrInvalidNote`, `ErrNoteNotFound`) handlers translate at the transport edge. Two-mock split (`FakeRepository` for service tests, `FakeService` for handler tests) means each layer's tests stay focused on what that layer owns. New domains live in `internal/<domain>/` — never a generic `services/` directory.
- **Domain-module pattern**: Each API feature is a self-describing `module.Module` returned from `api/handlers/<dom>/module.go`, with `Endpoints` + `Schema()` re-exported for the shared registry at `api/internal/modules/registry.go` (consumed by both `main.go` and `gen-endpoints`). Adding a domain = create files + 3 lines (one in `main.go::server.New`, one in `registry.go::AllEndpoints`, one in `registry.go::AllSchemas`). No central `server.Deps` field per domain, no central `routes.go` to mutate. The `notes` reference can be removed cleanly via [`docs/internal/REPLACING-NOTES.md`](docs/internal/REPLACING-NOTES.md).
- **Generated API contract manifest**: `.vrooli/endpoints.json` is the canonical declaration of every public endpoint plus its CLI mirror — and it's **generated** from `internal/modules.AllEndpoints()` (run `make endpoints`). The shared registry means runtime (`main.go`) and codegen (`gen-endpoints`) read endpoint metadata from the same place. The CI drift check (`git diff --exit-code`) fails the build if the manifest is stale, so doc generators / Postman collections / SDK stubs always read the real wire shape.
- **Documentation manifest**: `docs/manifest.json` declares the canonical doc set with audience tags and section grouping. Manifest-driven nav tooling (web-console doc viewer, etc.) reads it. The shipped layout follows the `documentation-health` skill standard: `QUICKSTART.md` (default document), `concepts/ARCHITECTURE.md`, `guides/troubleshooting.md`, `reference/{api-endpoints,cli-commands,configuration}.md`, and `internal/{SEAMS,TESTING,PROBLEMS,PROGRESS}.md`.
- **Outbound HTTP seam**: `api/internal/httpc.Doer` declares the canonical interface every scenario uses when calling external services. Production wires `*http.Client` directly (compile-time-asserted to satisfy `Doer`); tests substitute `mocks.FakeDoer`. Ships unwired in production by intent — the seam exists so the first scenario to need an outbound call doesn't reinvent ad-hoc mocking.
- **App-level ErrorBoundary**: `ui/src/components/ErrorBoundary.tsx` is a hand-rolled React class component wrapped at `main.tsx` (inside i18n + QueryClient providers, so the localised fallback can `useTranslation`). A render-time exception in `App.tsx` falls back to a recoverable error pane with a retry button — no more silent blank-screen failures. The `onError` prop is exposed for scenarios to wire telemetry (Sentry, etc.).

## Setup Workflow
```bash
cd scenarios/<your-scenario>

# Build API + UI, install pnpm deps, install scenario CLI
make setup   # wraps `vrooli scenario setup`

# Start API + UI in the background
make start   # wraps `vrooli scenario start`
```

See [`docs/QUICKSTART.md`](docs/QUICKSTART.md) for the full clone-to-running flow.

Run tests with `make test` (which runs `vrooli scenario test`) or invoke `test-genie execute <your-scenario> --preset comprehensive` directly for finer-grained presets.

## Required Environment Variables
The lifecycle exports everything automatically when you run `make start` (`vrooli scenario start`). If you start pieces manually, set these yourself (there are no fallbacks):

| Variable | Canonical range | Purpose |
|----------|-----------------|---------|
| `API_PORT` | `15000-19999` | Port assigned to the Go API server |
| `UI_PORT`  | `20000-24999` | Port assigned to the Vite dev server / production UI |
| `SQLITE_PATH` | — | Optional override for the SQLite file path. Defaults to `api-core/storage` resolver under the scenario data dir. |
| `UI_BASE_URL` | — | Base URL for the Vrooli UI shell / iframe bridge (resolved by `@vrooli/api-base` when unset) |
| `API_TOKEN` | — | Shared secret the CLI/API uses for authentication (only enforce in deployed scenarios) |
| `VITE_API_BASE_URL` | — | UI → API bridge. Default: `http://localhost:${API_PORT}/api/v1`. |

> Canonical bands sit below 32768 so Linux never hands the ports out as outbound source ports. See [docs/reference/port-allocation.md](../../../docs/reference/port-allocation.md) for the full policy. Scenarios that add WebSocket channels declare a `websocket` port (canonical band `25000-29999`) under `.vrooli/service.json` `ports`.

> Tip: when running outside the lifecycle, fetch ports with `vrooli scenario port <name> API_PORT` (or `UI_PORT`) and then export `VITE_API_BASE_URL` accordingly:

```bash
API_PORT=$(vrooli scenario port <name> API_PORT)
UI_PORT=$(vrooli scenario port <name> UI_PORT)
cd ui && VITE_API_BASE_URL="http://localhost:${API_PORT}/api/v1" pnpm run dev -- --host --port "$UI_PORT"
```

## i18n flow

The template ships with a fully-wired i18n setup so adopters don't have to retrofit one later. The shape:

- **Catalogs** live at `ui/src/i18n/locales/<code>.json`. `en.json` is canonical; every other locale must mirror its key shape (the `locales.test.ts` parity test fails the build if it doesn't, with CLDR plural suffixes stripped before comparison).
- **Typed key registry**: `ui/src/consts/strings.generated.ts` is produced from `en.json` by `scripts/gen-strings.mjs`. Each leaf is the dotted key path passed to `t()` — `strings.app.title === "app.title"`. The bundled Vite plugin (`scripts/vite-plugin-strings-codegen.mjs`) regenerates this file on every dev start, on HMR of `en.json`, and on every build start. **Don't edit `strings.generated.ts` by hand** — your changes will be overwritten.
- **Why codegen?** Walking the catalog at module load forces the bundler to ship `en.json` twice (once as i18next's resource, once as registry input). Codegen makes the catalog bundled exactly once. See `ui/src/consts/strings.ts` for the full rationale.
- **Tests default to `cimode`** (`ui/src/test-setup.ts`), so `t('app.title')` returns the *key* `"app.title"`. Component tests assert against the typed `strings.*` registry — they're copy-independent. Locale-pipeline tests opt back in via `await setLocale("en")` and reference `en.x.y` directly to validate end-to-end behaviour.
- **ESLint enforcement**:
  1. JSX text and `{"…"}` literals containing letters are forbidden — use `{t(strings.feature.key)}`.
  2. `aria-label`, `placeholder`, `title`, `alt`, etc. with string literals are forbidden — same fix.
  3. `*ByText("literal")` and `*ByText(\`template\`)` in tests are forbidden — use `getByTestId(selectors.x.y)` or `getByText(strings.x.y)`.
  4. `eslint-plugin-jsx-a11y` (strict) catches lint-time accessibility issues to complement the runtime axe-core check in `*.a11y.test.tsx`.

### Adding a string
1. Add the key to `ui/src/i18n/locales/en.json` and every other locale in `ui/src/i18n/locales/`.
2. If `pnpm dev` (or `vitest`) is running, the registry regenerates instantly. Otherwise run `pnpm strings:gen` from `ui/`.
3. Reference it as `{t(strings.feature.key)}` in JSX. For interpolation use `{{var}}` placeholders in the JSON and `t(strings.feature.key, { var: value })`.
4. Commit `en.json`, your other-locale files, **and** `strings.generated.ts` together. CI runs `pnpm strings:check` and will fail the PR if the generated file isn't in sync.

### Adding a locale
1. Drop `ui/src/i18n/locales/<code>.json` next to `en.json` (same shape).
2. Add the code to `SUPPORTED_LOCALES` and an entry to `LOCALE_CONFIG` in `ui/src/i18n/index.ts` (native label + `dir`).
3. Import + register the catalog in the `resources` block in the same file.
4. The language switcher reads `LOCALE_CONFIG` directly — no UI changes needed. The parity test will keep the new locale honest from then on.

## Iframe Bridge + API Base
- `src/main.tsx` initializes `@vrooli/iframe-bridge` automatically whenever the UI is rendered inside App Monitor or another host.
- All API calls go through `@vrooli/api-base`, which means the UI works no matter where it’s served (localhost dev server, Cloudflare tunnel, proxied iframe, production ingress). Just keep `VITE_API_BASE_URL` pointed at `http://localhost:${API_PORT}/api/v1` during local work.

## CLI Contract
- `vrooli` resolves and installs the CLI from `.vrooli/service.json` `cli.*`; the CLI implementation is not inferred from file layout anymore.
- The template ships adapter assets at `cli/install.sh` and `cli/install.ps1`, and the manifest declares when each should be used.
- The CLI stores config in your user config directory (typically `~/.config/vrooli/{{SCENARIO_ID}}/config.json` or `~/.vrooli/config/{{SCENARIO_ID}}/config.json`).
- Run `{{SCENARIO_ID}} configure api_base http://localhost:<API_PORT>/api/v1` (and optionally `{{SCENARIO_ID}} configure token <token>`) to point at a remote or non-standard API.
- `status` and `configure` are provided by `cli-core`; `status` targets the canonical root `/health` endpoint.

### CLI Extension Model
- This is the only recommended greenfield CLI shape for new scenarios. Do not start with flat `cmd_<domain>.go` files as the intended long-term architecture.
- `cli/main.go`: entrypoint only.
- `cli/app.go`: metadata + scaffold wiring only. Avoid endpoint logic here.
- `cli/domains/domains.go`: domain registration only.
- `cli/domains/<domain>/`: default place for command handlers, request/response shaping, and output formatting.
- Use `core.Get(...)` / `core.Request(...)` for versioned API routes and `core.GetRoot(...)` / `core.RequestRoot(...)` for root paths such as `/health`.
- Mark API-backed commands with `NeedsAPI: true` so stale-checking, token validation, and `--auto-start` preflight stay connected automatically.
- Prefer `SubcommandGroup` by default for real domains (`tasks list`, `projects create`, etc.).
- Default human output should follow a command contract: operational commands use `Status -> Triage -> Next Steps`, list/read commands use `Summary -> Results -> Retrieval Hints`, and mutation commands use `Result -> What Changed -> Next Command`.
- Use `cliapp.RenderOperationalReport`, `RenderListReport`, and `RenderMutationReport` so the human output contract stays consistent across scenarios.
- When a command supports `--json`, render the same underlying structured report through `cliapp.PrintReportJSON(...)` instead of inventing a second output shape.

## Customize Safely
1. **Update PRD.md + requirements/** first. Operational targets drive code + tests.
2. **Append progress entries** to `docs/internal/PROGRESS.md` whenever you land work.
3. **Add resources** in `.vrooli/service.json` only when needed; the template ships with no resource dependencies (SQLite is in-process).
4. **Keep boundaries**: only edit within `scenarios/<your-scenario>/`.

## UI Performance Profiling
The UI ships ready for headless perf audits:

- **Profile-mode build**: `VITE_BUILD_MODE=profile` (or `pnpm run build:profile`) keeps React's profiling instrumentation in the bundle. Configured in `ui/vite.config.ts`.
- **`onProfilerRender` util**: `ui/src/lib/profiler.ts` emits a `performance.measure` per commit so component-level timing shows up in any Chrome trace as `⚛ <id>` user_timing entries.
- **Top-level `<React.Profiler id="App">`**: wired in `ui/src/main.tsx`. Inert in regular prod (the callback never fires); load-bearing only in profile builds. Add inner `<Profiler>` boundaries around heavy subtrees as you go — never remove the top-level one.
- **Capture template**: [`ui/perf/capture.template.js`](ui/perf/capture.template.js) is a starting-point Playwright + CDP tracing script. Includes `dragHorizontalOnce` and `findScrollableAncestor` helpers. See [`ui/perf/README.md`](ui/perf/README.md) for the workflow.

The full audit methodology lives in the `scenario-performance-audit` skill (`prompt-manager skill read scenario-performance-audit`). Audit results persist as `docs/perf/<date>-<slug>.md` in the scenario tree, validated by `knowledge-observatory docs audit`.

## pnpm Everywhere
The template assumes pnpm. If you run another package manager, convert lockfiles yourself before committing. Scripts use `pnpm` directly (no `npm` fallbacks) to reduce drift.

## Need Inspiration?
Open `scenarios/browser-automation-studio/` to see this template taken to completion.
