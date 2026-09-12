# Testing — Business Health

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

## API testing

### The primitives every API test uses

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
   `packages/proto/schemas/business-health/v1/<domain>/<file>.proto`.
   Tests import the generated Go type directly
   (`healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/shared"`)
   and decode wire bodies into it via `MustUnmarshalProto`. Create
   fixture builders only when tests need reusable response inputs.

## UI testing

### The primitives every UI test uses

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

## Common patterns and anti-patterns

| ✅ DO | ❌ DON'T |
|---|---|
| `mocks.FakeClock` for time-dependent assertions | `time.Sleep(150 * time.Millisecond)` then assert on a fuzzy match |
| `httpx.NewLiveServer` for handler tests | `httptest.NewRecorder` (hides SSE-flusher bugs) |
| `getByTestId(selectors.x.y)` for stable selectors | `getByText("Save")` (breaks the moment copy changes) |
| `vi.mock("./api/health", async (importOriginal) => …)` inline at top of file | Helper-wrapped `vi.mock` (TDZ at hoist time) |
| `makeHealthResponse({ status: "degraded" })` for variants | Hardcoded literal payload duplicated across multiple tests |
| Per-method error knob (`PingErr error`) on fakes | Single global "fail mode" boolean across the fake |
| `var _ Pinger = (*sql.DB)(nil)` to lock the contract at compile time | Runtime "does this satisfy" check in init |
| Adding new seams to [`SEAMS.md`](SEAMS.md) at the same commit | "We'll document it later" |
