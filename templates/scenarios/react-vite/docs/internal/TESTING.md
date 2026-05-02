# Testing — {{SCENARIO_DISPLAY_NAME}}

How to write tests against this scenario's shape. Read this *before*
your first non-trivial test — the patterns below are load-bearing for
the gates documented in [`SEAMS.md`](SEAMS.md), in `eslint.config.js`,
and in `.github/workflows/test.yml`.

The shape is mature on purpose: every pattern below was already needed
in workspace-sandbox and got there by accumulating bugs. Starting here
means inheriting those lessons without repeating them.

## TL;DR — the canonical examples

Three files are the source of truth. When in doubt, copy their shape:

- **API**: `api/handlers/health/handler_test.go` — table-driven, real
  middleware via `httpx.NewLiveServer`, fake pinger from `mocks/`,
  typed-fixture JSON decode via
  `assertx.MustDecodeJSON[fixtures.HealthResponse]` (matches the
  api-core/health wire shape field-for-field — assert on typed fields,
  not `map[string]any` chains).
- **UI**: `ui/src/App.test.tsx` — `renderWithProviders` + factory data
  + inline `vi.mock` factory closure. Two describe blocks: cimode
  (copy-independent assertions) and real-locale (end-to-end i18n).
- **CLI**: `cli/app_test.go` — smoke gate (NewApp, --version, --help).
  When domain commands arrive, extend with `clitest.NewAPIServer` +
  `clitest.CaptureStdout` from `cli/internal/testutil/`.

If your test doesn't look like one of those three, ask why before
shipping.

## API testing

### Layout

```
api/
├── internal/
│   ├── clock/clock.go            # Clock interface + clock.System
│   ├── store/store.go            # Pinger interface
│   ├── middleware/logging.go     # Uses clock.Clock — no time.Now()
│   ├── server/                   # Server wires Pinger + Clock + Logger
│   └── testutil/
│       ├── assertx/              # AssertStatus, MustDecodeJSON[T]
│       ├── db/                   # NewSQLite(t) — modernc.org/sqlite
│       ├── fixtures/             # NewHealthResponse(opts...) — functional options
│       ├── httpx/                # NewLiveServer(t, *Server) over real socket
│       ├── mocks/                # FakeClock, FakePinger
│       ├── no_prod_import_test.go  # AST guardrail (see below)
│       └── testutil.go           # Package contract
└── handlers/health/
    ├── handler.go                # Production handler
    └── handler_test.go           # Canonical test
```

### The four primitives every test uses

1. **`mocks.FakeClock`** — substitutes `clock.Clock`. Construct with
   `mocks.NewFakeClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))`,
   advance with `.Advance(d)`. Tests that touch duration logging or
   timestamp output start here.
2. **`mocks.FakePinger`** — substitutes `store.Pinger`. Construct with
   `&mocks.FakePinger{PingErr: errors.New("connection refused")}` to
   exercise the unhealthy branch; default `PingErr: nil` is the happy
   path. Atomic `Calls` counter is for "the handler called Ping exactly
   once" assertions.
3. **`httpx.NewLiveServer(t, srv)`** — wraps your `*server.Server` in a
   real `httptest.Server` listening on a real socket. Returns a struct
   with a `Do(t, method, path, body) (*http.Response, []byte)` method.
   **Use this, not `httptest.NewRecorder`.** Recorder fakes `Flusher`
   and `Hijacker`, masking SSE-flush bugs that workspace-sandbox shipped
   in production on 2026-04-28. The cost of a real socket is measured
   in microseconds; the cost of the bug class it catches is measured
   in incidents.
4. **`assertx`** — `AssertStatus(t, resp, want)` for status code
   checks (dumps body on mismatch); `MustDecodeJSON[T any](t, body) T`
   for typed JSON decoding. Pair the generic decode with a typed
   fixture (`fixtures.HealthResponse`) so assertions read as
   `got.Status` / `got.Dependencies["database"].Connected` rather than
   `map[string]any` chains. Resist over-generalising; add helpers when
   the third caller appears.

### Canonical test pattern

```go
package health_test

import (
    "errors"
    "net/http"
    "testing"

    "github.com/stretchr/testify/require"

    "{{SCENARIO_ID}}/internal/clock"
    "{{SCENARIO_ID}}/internal/server"
    "{{SCENARIO_ID}}/internal/testutil/assertx"
    "{{SCENARIO_ID}}/internal/testutil/fixtures"
    "{{SCENARIO_ID}}/internal/testutil/httpx"
    "{{SCENARIO_ID}}/internal/testutil/mocks"
)

func TestHealthHandler(t *testing.T) {
    cases := []struct {
        name           string
        pingErr        error
        wantCode       int
        wantStatus     string
        wantConnected  bool
    }{
        {"ok", nil, http.StatusOK, "healthy", true},
        {"db_unreachable", errors.New("connection refused"), http.StatusServiceUnavailable, "unhealthy", false},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            pinger := &mocks.FakePinger{PingErr: tc.pingErr}
            srv := server.New(server.Deps{Pinger: pinger, Clock: clock.System{}, /*…*/})

            live := httpx.NewLiveServer(t, srv)
            resp, body := live.Do(t, http.MethodGet, "/health", nil)

            assertx.AssertStatus(t, resp, tc.wantCode)
            got := assertx.MustDecodeJSON[fixtures.HealthResponse](t, body)
            require.Equal(t, tc.wantStatus, got.Status)
            require.Equal(t, tc.wantConnected, got.Dependencies["database"].Connected)
            require.Equal(t, int64(1), pinger.Calls.Load())
        })
    }
}
```

The fixture's JSON tags mirror `api-core/health.Response` exactly, so
`MustDecodeJSON` round-trips the wire shape directly into the typed
struct — no `map[string]any` chains, no per-test `interface{}` casts.

### Production-import quarantine (`no_prod_import_test.go`)

The test in `api/internal/testutil/no_prod_import_test.go` walks every
non-test `.go` file under `api/` and fails if any imports anything
starting with `<module>/internal/testutil/`. The module name is read
dynamically from `go.mod`, so it works after `{{SCENARIO_ID}}`
substitution.

This is the load-bearing rule that makes it safe to put `time.Sleep`,
process-wide globals, and `testing.T` references in the testutil
package. If the rule fires:

- ✅ **Move the helper out of testutil** into a non-test package.
- ❌ **Don't add `// nolint`** — the production code path will then
   carry the test-only dep into the binary on every build.

## UI testing

### Layout

```
ui/src/
├── test-setup.ts                # vitest setupFiles entry
├── test-utils/
│   ├── index.ts                 # re-exports
│   ├── factories.ts             # makeHealthResponse(overrides?)
│   ├── renderWithProviders.tsx  # QueryClient + i18n wrapper
│   └── mocks/
│       └── spatial.ts           # builders for @vrooli/iframe-bridge/spatial
└── *.test.tsx                   # tests live next to the code they cover
```

The `mocks/` directory holds shared mock-shape builders for external
SDKs (today: spatial-nav). Each hook test still calls
`vi.mock("@vrooli/iframe-bridge/spatial", ...)` inline (Vitest hoisting
is non-negotiable), but the factory closure invokes the builders in
`mocks/spatial.ts` so the contract for each SDK lives in one file.
Adding a new SDK: drop a `mocks/<sdk>.ts` builder beside it, add a
`mocks/<sdk>.test.ts` self-test, re-export from `test-utils/index.ts`.

### The three primitives every test uses

1. **`renderWithProviders(<Component />, opts?)`** — wraps the tree in
   `QueryClientProvider` (retries disabled — tests should fail fast,
   not paper over flakes) and `I18nextProvider` bound to the same
   singleton production uses. Returns a `RenderResult` plus the
   `queryClient` for tests that need to seed cache state.
2. **`make<Domain>(overrides?: Partial<Domain>)`** — typed factory for
   stable test data. `makeHealthResponse()` is the worked example;
   add new factories alongside it as new shapes appear. Defaults
   should make the most common test path `make<Domain>()` with no
   args.
3. **Inline `vi.mock("./lib/api", async (importOriginal) => …)`** —
   the canonical mocking shape. **Do not** wrap this in a helper
   function. Vitest hoists `vi.mock(...)` calls before any imports
   resolve; a wrapper function imported from `test-utils` would be in
   the temporal dead zone at hoist time. `make<Domain>()` calls *are*
   safe inside the factory because the closure runs after imports
   initialise.

### Canonical UI test pattern

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { makeHealthResponse, renderWithProviders } from "./test-utils";

vi.mock("./lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./lib/api")>();
  return {
    ...actual,
    fetchHealth: vi.fn().mockResolvedValue(makeHealthResponse()),
  };
});

import App from "./App";
import { selectors } from "./consts/selectors";
import { strings } from "./consts/strings";

describe("App rendering (cimode — copy-independent)", () => {
  afterEach(() => { cleanup(); });

  it("renders the title via test ID", () => {
    renderWithProviders(<App />);
    expect(screen.getByTestId(selectors.app.title)).toBeInTheDocument();
  });

  it("renders translation keys in cimode", async () => {
    renderWithProviders(<App />);
    expect(await screen.findByText(strings.app.eyebrow)).toBeInTheDocument();
  });
});
```

### Two-layer test pattern (cimode + real locale)

`test-setup.ts` puts vitest into `cimode` before every test — `t('app.title')`
returns the literal key `"app.title"`. Most tests run there: assertions
go through `selectors.*` (test IDs) and `strings.*` (the typed key
registry), so they survive any wording change in any locale.

A second `describe` block opts into real locales with
`beforeEach(async () => { await setLocale("en"); })` and asserts on the
canonical English copy via raw `en.json` references. These tests *should*
update when canonical English copy changes — that's what they verify.

See `App.test.tsx` for the full pattern and the CLDR plural variants
(`refreshCount_one`, `notifications.summary_zero` / `_one` / base).

### Test-utils quarantine

ESLint's `no-restricted-imports` rule (in `eslint.config.js`) bans
imports from `**/test-utils/*` and `@/test-utils/*` in production
files. The `*.test.{ts,tsx}` and `*.spec.{ts,tsx}` override block
turns the rule off so tests import freely.

This mirrors the Go AST guardrail. If the rule fires:

- ✅ **Confirm the importing file is a test.** If your test file is
   named `something.tsx` instead of `something.test.tsx`, ESLint
   correctly treats it as production. Rename it.
- ✅ **Move the helper out of test-utils** if it genuinely needs to
   ship in production.
- ❌ **Don't disable the rule** for a "one-off" production import. There
   is no path back from a test-utils leak — every future build carries it.

## CLI testing

### Layout

```
cli/
├── app.go                              # NewApp wires cli-core's StandardScenarioApp
├── app_test.go                         # canonical smoke test
├── domains/                            # domain-specific command groups
└── internal/
    └── testutil/
        ├── server.go                   # NewAPIServer, NewHTTPServer, WithAPIBase, CaptureStdout
        └── no_prod_import_test.go      # AST guardrail (mirrors the API side)
```

### Smoke test (always present)

`cli/app_test.go` is the canonical smoke gate every scenario inherits.
It catches regressions in cli-core wiring before any domain command
exists:

```go
func TestNewAppConstructs(t *testing.T) {
    app, err := NewApp()
    if err != nil { t.Fatalf("NewApp() error: %v", err) }
    if app == nil || app.core == nil || app.core.CLI == nil {
        t.Fatal("NewApp() returned an incomplete app")
    }
}

func TestRunVersion(t *testing.T) {
    app, _ := NewApp()
    if err := app.Run([]string{"--version"}); err != nil {
        t.Fatalf("--version: %v", err)
    }
}
```

`--version` and `--help` are NeedsAPI=false code paths in cli-core, so
they don't try to reach the configured API base. Tests for API-backed
commands need the httptest pattern below.

### Testing API-backed commands

When a scenario adds its first API-backed command (in
`cli/domains/<domain>/`), the canonical pattern is:

```go
package tasks_test

import (
    "encoding/json"
    "net/http"
    "testing"

    clitest "{{SCENARIO_ID}}/cli/internal/testutil"
)

func TestTasksList(t *testing.T) {
    server := clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/api/v1/tasks" {
            t.Fatalf("unexpected path: %s", r.URL.Path)
        }
        _ = json.NewEncoder(w).Encode([]map[string]any{
            {"id": "t-1", "title": "first"},
        })
    }))
    _ = server // server.URL is wired into API_BASE_URL by NewAPIServer

    app, err := NewApp()
    if err != nil { t.Fatalf("NewApp: %v", err) }

    out := clitest.CaptureStdout(t, func() error {
        return app.Run([]string{"tasks", "list"})
    })

    if !strings.Contains(out, "first") {
        t.Fatalf("expected task title in output, got: %s", out)
    }
}
```

Why each piece:

1. **`NewAPIServer`** wraps `httptest.NewServer` and sets `API_BASE_URL`
   to the test server's URL via `t.Setenv` — cli-core's APIBase
   resolver picks it up. Auto-restored at end-of-test.
2. **Real httptest.Server**, not a Recorder. Same reasoning as the API
   side: Recorder fakes `Flusher`/`Hijacker`; a real socket catches
   SSE/streaming bugs that have shipped before.
3. **`CaptureStdout`** captures the human output that
   `cli-core.RenderOperationalReport` / `RenderListReport` /
   `RenderMutationReport` write. Use it instead of `--json` when you
   want to assert on the contract scenario users actually see.
4. **`go test -race`** is enforced by CI (`.github/workflows/test.yml`).
   `CaptureStdout` swaps `os.Stdout` and the test server runs in a
   separate goroutine; race coverage keeps both honest.

### CLI test-utils quarantine

`cli/internal/testutil/no_prod_import_test.go` walks every non-test
`.go` file under `cli/` and fails if any imports
`<module>/cli/internal/testutil/...`. Same rule, same rationale as the
API side: the testutil package is allowed to depend on `testing`,
mutate `os.Stdout`, and run real listeners, because production code
provably can't see it. If the test fires:

- ✅ **Move the helper out of testutil** into a non-test package.
- ❌ **Don't add `// nolint`** — every future build would carry the
   test-only dep into the binary.

### When to add a CLI test

| Change | Test |
|---|---|
| New API-backed command | One success-path test + one error-path test, both via `NewAPIServer` + `CaptureStdout`. |
| New non-API command (config write, fingerprint print, etc.) | Direct `app.Run([...])` with `CaptureStdout`. No fake server needed. |
| Change to `app.go` wiring | The smoke gate (`TestNewAppConstructs`) catches most regressions automatically. Add a focused test only when the wiring touches a non-default code path. |
| New env var that affects API resolution | Extend or wrap `clitest.WithAPIBase` rather than calling `t.Setenv` inline. Keeps the env-var name in one place. |

## Coverage thresholds

Configured in `vite.config.ts` at the bottom of the `test.coverage`
block. Currently 85% across lines, branches, functions, and statements.

The `coverage.exclude` list covers test scaffolding and codegen only —
production source under `src/` is exhaustively included. The default
position is: every new `src/` file ships with its own `*.test.{ts,tsx}`
and lands inside the include set. If a scenario adds genuinely-untestable
code, prefer a narrow file exclusion with a one-line rationale comment
in `vite.config.ts` over loosening the thresholds.

The CI workflow (`.github/workflows/test.yml`) runs
`pnpm test:coverage`, which reads these thresholds. A drop below floor
fails the gate immediately. Raise thresholds (toward 90+%) when real
coverage clears the new floor for a full release.

## Common patterns and anti-patterns

| ✅ DO | ❌ DON'T |
|---|---|
| `mocks.FakeClock` for time-dependent assertions | `time.Sleep(150 * time.Millisecond)` then assert on a fuzzy match |
| `httpx.NewLiveServer` for handler tests | `httptest.NewRecorder` (hides SSE-flusher bugs) |
| `getByTestId(selectors.x.y)` for stable selectors | `getByText("Save")` (breaks the moment copy changes) |
| `vi.mock("./lib/api", async (importOriginal) => …)` inline at top of file | Helper-wrapped `vi.mock` (TDZ at hoist time) |
| `makeHealthResponse({ status: "degraded" })` for variants | Hardcoded literal payload in three different tests |
| Per-method error knob (`PingErr error`) on fakes | Single global "fail mode" boolean across the fake |
| `var _ Pinger = (*sql.DB)(nil)` to lock the contract at compile time | Runtime "does this satisfy" check in init |
| Adding new seams to [`SEAMS.md`](SEAMS.md) at the same commit | "We'll document it later" |

## Cross-references

- **Seams definition + adding new seams**: [`SEAMS.md`](SEAMS.md).
- **Skill bundle for testing-related work** (load before substantial test changes):
  ```bash
  prompt-manager skill read seam-discovery-and-enforcement test unit-testing-architecture-steer
  ```
- **Test runner used by CI and `vrooli scenario test`**: see
  `.github/workflows/test.yml` and `packages/cli-core/cmd/scenario_test.go`.
- **Why no inline mocks in `*_test.go` files**: the testutil package
  is the single source of fake behavior. Inline mocks in tests
  fragment the contract; when the interface grows a method, every
  inline mock has to be updated. One mock in `mocks/`, one update.
