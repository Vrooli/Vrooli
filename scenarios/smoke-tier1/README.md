# React + Vite Scenario Template (Go API + CLI)

Use this template to bootstrap every new scenario. It mirrors the patterns from `browser-automation-studio`:

- Go API (`api/`)
- React + TypeScript + Vite UI (`ui/`)
- CLI wrapper (`cli/`)
- Lifecycle + health wiring (`.vrooli/service.json`)
- Requirements registry + progress log (`requirements/`, `docs/PROGRESS.md`)

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

## What You Get
- **Clean UI scaffold**: Vite + Tailwind + shadcn-style primitives, pnpm-based scripts, Vitest + Testing Library pre-configured, `.env.example` for API URL.
- **i18n out of the box**: i18next + react-i18next wired up with locale persistence, RTL-ready `<html lang>`/`<html dir>`, an `Intl`-based formatter helper (date/number/currency/relative-time/list), and a typed key registry generated from `en.json` at build time. ESLint forbids inline copy and string-literal `*ByText` queries, so copy edits are one-line catalog changes and tests survive them automatically. See [i18n flow](#i18n-flow).
- **Go API skeleton**: `go.mod` + `cmd/server` entrypoint ready for feature modules.
- **CLI manifest contract**: `.vrooli/service.json` declares the CLI command, adapter, install strategies, and freshness inputs.
- **Lifecycle-ready service.json**: ports aligned with the platform, no external resource required by default (storage is local SQLite), lifecycle steps that build API/UI and start dev servers.
- **Iframe-ready UI**: Automatically initializes `@vrooli/iframe-bridge` so App Monitor and other hosts can embed the scenario without extra work.
- **Smart API resolution**: UI uses `@vrooli/api-base` to resolve the correct API + WebSocket URLs across localhost/dev/proxy contexts.
- **Requirements seed**: `requirements/index.json` + `requirements/modules/foundation.json` show how operational targets trace to technical requirements.
- **Lifecycle metadata seed**: `.vrooli/service.json`, `endpoints.json`, `testing.json`, and `lighthouse.json` so status/health/testing commands work immediately after copy.
- **Progress log**: `docs/PROGRESS.md` so improvers track deltas outside PRD.md.
- **SQLite storage**: API uses `api-core/database` with the `sqlite` driver and `api-core/storage` for filesystem-safe paths — no external DB process required. The schema lives at `api/internal/store/schema.sql` (embedded into the binary, applied at startup via `store.EnsureSchema`).
- **Proto-first wire contracts**: The template ships proto sources at `proto/smoke-tier1/v1/health/health.proto`. At generation time, `vrooli scenario generate` relocates them into `packages/proto/schemas/<your-scenario>/`, runs `make generate`, and the API + UI immediately consume the generated Go and TypeScript types. No hand-written wire shape duplication; adding a new endpoint means adding a `.proto`, regenerating, and importing the typed message. The codegen pipeline runs entirely on local plugins — no BSR network calls — so it works on flight Wi-Fi or inside firewalled CI runners. See `proto/README.md`, `docs/internal/TESTING.md::How to add a new proto`, and the project-level [proto pipeline guide](../../docs/development/proto.md).
- **Notes CRUD reference**: A worked end-to-end example at `/api/v1/notes` (list / create / get-by-id) backed by sqlite, with proto-typed wire contracts (`v1/notes/`, `v1/errors/`), a typed error envelope, repository tests on a real handle, handler tests through the live HTTP harness, a UI consumer at `ui/src/lib/notes.ts` rendered by the App's Notes pane, and a CLI domain at `cli/domains/notes/`. Copy this layering when adding your first non-trivial mutation — every layer has its own self-test, so the pattern is hard to break.
- **Outbound HTTP seam**: `api/internal/httpc.Doer` declares the canonical interface every scenario uses when calling external services. Production wires `*http.Client` directly (compile-time-asserted to satisfy `Doer`); tests substitute `mocks.FakeDoer`. Ships unwired in production by intent — the seam exists so the first scenario to need an outbound call doesn't reinvent ad-hoc mocking.

## Setup Workflow
```bash
cd scenarios/<your-scenario>

# Install dependencies (Go 1.22+ + pnpm must be available)
corepack pnpm install --dir ui --ignore-workspace

# Build API + UI via lifecycle
vrooli scenario run <your-scenario> --setup

# Start dev servers (API + Vite)
make dev   # wraps `vrooli scenario run --dev`
```

Run tests with `make test` or `test-genie execute --scenario <your-scenario>`.

### UI Smoke Harness

- `vrooli scenario ui-smoke <your-scenario>` launches a Browserless session against the production UI bundle, waits for `@vrooli/iframe-bridge` to signal readiness, captures a screenshot, HTML snapshot, console logs, and network trace.
- Artifacts live under `coverage/ui-smoke/` (screenshot, console.json, network.json, dom.html, raw.json) and the latest summary is stored at `coverage/ui-smoke/latest.json`.
- Structure tests invoke the harness automatically. Disable it temporarily by toggling `structure.ui_smoke.enabled` in `.vrooli/testing.json` or extend the default timeouts via `timeout_ms` / `handshake_timeout_ms`.
- Browserless must be running (`resource-browserless status --format json`). If it is offline the harness fails early so issues surface before release.

## Required Environment Variables
The lifecycle exports everything automatically when you run `vrooli scenario run`. If you start pieces manually, set these yourself (there are no fallbacks):

| Variable | Canonical range | Purpose |
|----------|-----------------|---------|
| `API_PORT` | `15000-19999` | Port assigned to the Go API server |
| `UI_PORT`  | `20000-24999` | Port assigned to the Vite dev server / production UI |
| `WS_PORT`  | `25000-29999` | WebSocket channel for live updates |
| `SQLITE_PATH` | — | Optional override for the SQLite file path. Defaults to `api-core/storage` resolver under the scenario data dir. |
| `N8N_BASE_URL` | Base URL for workflow automation calls |
| `UI_BASE_URL` | Base URL for the Vrooli UI shell / iframe bridge |
| `API_TOKEN` | Shared secret the CLI/API uses for authentication |
| `VITE_API_BASE_URL` | — | UI → API bridge (set to `http://localhost:${API_PORT}/api/v1`) |

> All canonical bands sit below 32768 so Linux never hands the ports out as outbound source ports. See [docs/reference/port-allocation.md](../../../docs/reference/port-allocation.md) for the full policy and OS-specific ephemeral-range details.

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
- The CLI stores config in your user config directory (typically `~/.config/vrooli/smoke-tier1/config.json` or `~/.vrooli/config/smoke-tier1/config.json`).
- Run `smoke-tier1 configure api_base http://localhost:<API_PORT>/api/v1` (and optionally `smoke-tier1 configure token <token>`) to point at a remote or non-standard API.
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
2. **Append progress entries** to `docs/PROGRESS.md` whenever you land work.
3. **Add resources** in `.vrooli/service.json` only when needed; the template ships with no resource dependencies (SQLite is in-process).
4. **Keep boundaries**: only edit within `scenarios/<your-scenario>/`.

## pnpm Everywhere
The template assumes pnpm. If you run another package manager, convert lockfiles yourself before committing. Scripts use `pnpm` directly (no `npm` fallbacks) to reduce drift.

## Need Inspiration?
Open `scenarios/browser-automation-studio/` to see this template taken to completion.
