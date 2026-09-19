# Testing — Source Ledger

## Shared guidance

- [Test authoring standard](/docs/testing/UNIT-TEST-AUTHORING.md): boundaries,
  fixtures, and independently justified expectations.
- [Execution and validation scope](/docs/TESTING.md): focused checks and Test Genie.
- [Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md): API, UI, CLI,
  cancellation, workflow replay, and coverage configuration.

The recipes describe template mechanics. This guide owns local behavior, test
prerequisites, fixtures, and exceptions; local test sources and configuration
identify the helpers and gates this scenario currently uses.

## Scenario-specific testing

Existing local entry points (choose the domain test for the behavior you change):

- [api/handlers/health/handler_test.go](../../api/handlers/health/handler_test.go)
- [ui/src/App.test.tsx](../../ui/src/App.test.tsx)
- [ui/src/features/health/HealthCard.test.tsx](../../ui/src/features/health/HealthCard.test.tsx)
- [ui/src/layout/AppShell.a11y.test.tsx](../../ui/src/layout/AppShell.a11y.test.tsx)
- [cli/app_test.go](../../cli/app_test.go)

## TL;DR — the canonical examples

These files are the source of truth. When in doubt, copy their shape:

- **API**: `api/handlers/health/handler_test.go` — table-driven, real
  middleware via `httpx.NewLiveServer`, fake pinger from `mocks/`,
  typed-proto decode via
  `assertx.MustUnmarshalProto[healthv1.Response]` (the wire shape lives
  in `packages/proto/schemas/source-ledger/v1/shared/health.proto`;
  assert on typed proto fields, not `map[string]any` chains). For
  endpoints whose wire shape isn't in proto yet, `MustDecodeJSON[T]`
  is the fallback — but adding the proto first is the right move.
- **UI composition**: `ui/src/App.test.tsx` — smoke-only composition
  test. App composes shell + features; feature behaviour belongs beside
  the feature.
- **UI feature**: `ui/src/features/health/HealthCard.test.tsx` —
  `renderWithProviders`, factory data, inline `vi.mock` factory
  closure, cimode assertions, and real-locale assertions.
- **UI a11y**: `ui/src/components/AppShell.a11y.test.tsx` and
  `ui/src/features/health/HealthCard.a11y.test.tsx` — shell and feature
  accessibility are tested at their ownership boundary.
- **CLI**: `cli/app_test.go` — smoke gate (NewApp, --version, --help).
  When domain commands arrive, extend with `clitest.NewAPIServer` +
  `clitest.CaptureStdout` from `cli/internal/testutil/`.

If your test doesn't look like the documented patterns, ask why before
shipping.

## API testing

### The core primitives every test uses

1. **`mocks.FakeClock`** — substitutes `clock.Clock`. Construct with
   `mocks.NewFakeClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))`,
   advance with `.Advance(d)`. Tests that touch duration logging or
   timestamp output start here.
2. **`mocks.FakePinger`** — substitutes `database.Pinger` (cross-domain
   mock under `internal/testutil/mocks/`). Construct with
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
   checks (dumps body on mismatch); `MustUnmarshalProto` for proto-typed JSON
   decoding (use this whenever the
   endpoint's wire shape lives in `packages/proto/schemas/`);
   `MustDecodeJSON` for ad-hoc JSON when no proto exists yet. Resist
   over-generalising; add helpers when the third caller appears.
5. **Generated proto types** — every endpoint's wire shape lives in
   `packages/proto/schemas/source-ledger/v1/<domain>/<file>.proto`.
   Tests import the generated Go type directly
   (`healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/shared"`)
   and decode wire bodies into it via `MustUnmarshalProto`. Create
   fixture builders only when tests need reusable response inputs.

### CRUD reference — an end-to-end vertical slice

The scaffold ships one fenced worked CRUD domain (never product scope)
as the canonical reference. New scenarios add their first non-trivial
mutation by copying its layering one file at a time — wire contract,
domain types, repository, schema, service, handler, mocks, UI client,
and CLI client — then deleting the fenced example with
`template-manager detemplate <scenario>`. The fenced example below walks
the pattern from wire to render and pins the layered service-test
split; copy its shape for `api/internal/<domain>/`.

## UI testing

### The core primitives every test uses

1. **`renderWithProviders(<Component />, opts?)`** — wraps the tree in
   `QueryClientProvider` (retries disabled — tests should fail fast,
   not paper over flakes) and `I18nextProvider` bound to the same
   singleton production uses. Returns a `RenderResult` plus the
   `queryClient` for tests that need to seed cache state. The helper
   has its own self-test at `ui/src/test-utils/renderWithProviders.test.tsx`
   pinning retries-disabled, queryClient identity, custom-client
   override, and singleton i18n wiring — mirrors the API-side
   `internal/testutil/httpx/server_test.go` pattern.
2. **`make<Domain>(overrides?: Partial<Domain>)`** — typed factory for
   stable test data. `makeHealthResponse()` is the worked example;
   add new factories alongside it as new shapes appear. Defaults
   should make the most common test path `make<Domain>()` with no
   args.
3. **Inline `vi.mock("./api/health", async (importOriginal) => …)`** —
   the canonical mocking shape. **Do not** wrap this in a helper
   function. Vitest hoists `vi.mock(...)` calls before any imports
   resolve; a wrapper function imported from `test-utils` would be in
   the temporal dead zone at hoist time. `make<Domain>()` calls *are*
   safe inside the factory because the closure runs after imports
   initialise.
4. **`expectNoA11yViolations(container)`** — shared axe-core assertion
   for component-level accessibility tests. Render and wait for the
   component's stable state in the owning test file, then call this
   helper. Do not put feature-specific waits in app-composition tests.

### Mock builders for `api/*` surfaces

`vi.mock(path, factory)` is hoisted before any user import resolves;
a wrapper imported from `test-utils` would be in the temporal dead
zone at hoist time. The escape hatch is to keep the `vi.mock` call
inline at the top of each test file, but move the *factory body* into
a builder function that runs when the closure executes — which is
*after* imports initialise.

`@/test-utils` exports shared, cross-feature mock builders such as
`makeApiMocks()`. Feature-specific builders live beside the feature so
deleting the feature takes its mocks with it (import
`make<Domain>Mocks()` from `features/<domain>/mocks/<domain>`).

Canonical shape:

```tsx
import { makeApiMocks } from "@/test-utils";

vi.mock("../../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/health")>();
  return { ...actual, ...makeApiMocks() };
});
```

Defaults are picked so the most common test paths work no-args:
`makeApiMocks().fetchHealth` resolves to a healthy response. The
`...actual` spread keeps non-mocked exports (the `ApiError` class,
re-exported proto types) intact — only network-touching functions are
substituted.

A feature with its own API surface adds a second `vi.mock` for its
client and a feature-local builder; per-test overrides use vitest's
standard pattern *after* the mock is wired
(`vi.mocked(client.method).mockResolvedValueOnce(...)`). The fenced
example below shows the full mock shape.

When a third lib/* surface lands (e.g., `lib/users.ts`), follow the
same pattern: builder in `ui/src/test-utils/mocks/<surface>.ts`, self-
test alongside, re-export from `test-utils/index.ts`.

## Common patterns and anti-patterns

| ✅ DO | ❌ DON'T |
|---|---|
| `mocks.FakeClock` for time-dependent assertions | `time.Sleep(150 * time.Millisecond)` then assert on a fuzzy match |
| `httpx.NewLiveServer` for handler tests | `httptest.NewRecorder` (hides SSE-flusher bugs) |
| `getByTestId(selectors.x.y)` for stable selectors | `getByText("Save")` (breaks the moment copy changes) |
| `vi.mock("./api/health", async (importOriginal) => …)` inline at top of file | Helper-wrapped `vi.mock` (TDZ at hoist time) |
| `makeHealthResponse({ status: "degraded" })` for variants | Hardcoded literal payload repeated across different tests |
| Per-method error knob (`PingErr error`) on fakes | Single global "fail mode" boolean across the fake |
| `var _ Pinger = (*sql.DB)(nil)` to lock the contract at compile time | Runtime "does this satisfy" check in init |
| Adding new seams to [`SEAMS.md`](SEAMS.md) at the same commit | "We'll document it later" |

## Cross-references

- **Seams definition + adding new seams**: [`SEAMS.md`](SEAMS.md).
- **Skill bundle for testing-related work** (load before substantial test changes): use
  `prompt-manager skill read scenario-work-ladder`.
- **Test runner used by CI and `vrooli scenario test`**: see
  `.github/workflows/test.yml` and `packages/cli-core/cmd/scenario_test.go`.
- **Why no inline mocks in `*_test.go` files**: the testutil package
  is the single source of fake behavior. Inline mocks in tests
  fragment the contract; when the interface grows a method, every
  inline mock has to be updated. One mock in `mocks/`, one update.
