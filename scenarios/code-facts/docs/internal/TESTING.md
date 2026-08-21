# Testing — Code Facts

How to write tests against this scenario's shape. Read this *before*
your first non-trivial test — the patterns below are load-bearing for
the gates documented in [`SEAMS.md`](SEAMS.md), in `eslint.config.js`,
and in `.github/workflows/test.yml`.

The shape is mature on purpose: every pattern below was already needed
in workspace-sandbox and got there by accumulating bugs. Starting here
means inheriting those lessons without repeating them.

## TL;DR — the canonical examples

These files are the source of truth. When in doubt, copy their shape:

- **API**: `api/handlers/health/handler_test.go` — table-driven, real
  middleware via `httpx.NewLiveServer`, fake pinger from `mocks/`,
  typed-proto decode via
  `assertx.MustUnmarshalProto[healthv1.Response]` (the wire shape lives
  in `packages/proto/schemas/code-facts/v1/health/health.proto`;
  assert on typed proto fields, not `map[string]any` chains). For
  endpoints whose wire shape isn't in proto yet, `MustDecodeJSON[T]`
  is the fallback — but adding the proto first is the right move.
- **UI composition**: `ui/src/App.test.tsx` — smoke-only composition
  test. App composes shell + features; feature behaviour belongs beside
  the feature.
- **UI feature**: `ui/src/features/health/HealthCard.test.tsx` —
  `renderWithProviders`, factory data, inline `vi.mock` factory
  closure, cimode assertions, and real-locale assertions.
- **UI a11y**: `ui/src/components/AppShell.a11y.test.tsx`,
  `ui/src/features/health/HealthCard.a11y.test.tsx`, and
  `ui/src/features/notes/NotesCard.a11y.test.tsx` — shell and feature
  accessibility are tested at their ownership boundary.
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
│   ├── database/pinger.go        # Pinger interface
│   ├── middleware/logging.go     # Uses clock.Clock — no time.Now()
│   ├── server/                   # Server wires cross-cutting Clock + Logger
│   └── testutil/
│       ├── assertx/              # AssertStatus, MustDecodeJSON[T]
│       ├── db/                   # NewSQLite(t) — modernc.org/sqlite
│       ├── fixtures/             # NewHealthResponse(opts...) — functional options
│       ├── httpx/                # NewLiveServer(t, *Server) over real socket
│       ├── mocks/                # FakeClock, FakePinger
│       ├── no_prod_import_test.go  # AST guardrail (see below)
│       └── testutil.go           # Package contract
└── handlers/health/
    ├── handler.go                # Production REST handler
    └── handler_test.go           # Canonical test
```

### The five primitives every test uses

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
   `packages/proto/schemas/code-facts/v1/<domain>/<file>.proto`.
   Tests import the generated Go type directly
   (`healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/health"`)
   and decode wire bodies into it via `MustUnmarshalProto`. The
   `fixtures` package re-exports the proto type as a short alias
   (`fixtures.HealthResponse = healthv1.Response`) so test code reads
   cleanly.

### Canonical test pattern

```go
package health_test

import (
    "errors"
    "io"
    "log"
    "net/http"
    "testing"

    "github.com/gorilla/mux"
    "github.com/stretchr/testify/require"
    healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/health"

    "code-facts/handlers/health"
    "code-facts/internal/clock"
    "code-facts/internal/module"
    "code-facts/internal/server"
    "code-facts/internal/testutil/assertx"
    "code-facts/internal/testutil/httpx"
    "code-facts/internal/testutil/mocks"
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
            h := health.NewHandler(health.Deps{
                Pinger:  pinger,
                Service: "code-facts",
                Version: "1.0.0",
            })
            mod := module.Module{
                Name: "health",
                Mount: func(r *mux.Router) {
                    r.HandleFunc("/health", h).Methods(http.MethodGet)
                },
            }
            srv := server.New(
                server.Deps{Clock: clock.System{}, Logger: log.New(io.Discard, "", 0)},
                mod,
            )

            live := httpx.NewLiveServer(t, srv)
            resp, body := live.Do(t, http.MethodGet, "/health", nil)

            assertx.AssertStatus(t, resp, tc.wantCode)
            got := assertx.MustUnmarshalProto[healthv1.Response](t, body)
            require.Equal(t, tc.wantStatus, got.Status)
            require.Equal(t, tc.wantConnected, got.Dependencies["database"].Connected)
            require.Equal(t, int64(1), pinger.Calls.Load())
        })
    }
}
```

The proto schema in `packages/proto/schemas/code-facts/v1/health/health.proto`
mirrors `api-core/health.Response` field-for-field, so `MustUnmarshalProto`
round-trips the wire shape directly into the generated Go type — no
`map[string]any` chains, no per-test `interface{}` casts, no parallel
hand-written struct mirror to drift against. `DiscardUnknown:true` is
wired in `MustUnmarshalProto` so the test keeps passing when the wire
grows fields the proto hasn't caught up to.

### CRUD reference — `notes` end-to-end

The `notes` domain is the canonical CRUD reference. New scenarios add
their first non-trivial mutation by copying its layering one file at a
time. The pattern from wire to render:

| Layer | File | What it owns |
|---|---|---|
| Wire contract | `packages/proto/schemas/code-facts/v1/notes/notes.proto` | `Note`, `service NotesService`, `ListNotesResponse`, `CreateNoteRequest`, `CreateNoteResponse`, `GetNoteRequest`, `GetNoteResponse` |
| REST metadata contract | `packages/proto/schemas/code-facts/v1/notes/attachments.proto` | `Attachment` and `UploadAttachmentResponse` for the multipart upload exception |
| Connect error mapping | `internal/notes/service_error_mapping.go` | Typed sentinels become Connect codes (`invalid_argument`, `not_found`, `internal`) |
| REST error envelope | `packages/proto/schemas/code-facts/v1/shared/errors.proto` + `internal/httpx/errors.go::WriteError` | Typed body for REST exceptions, with canonical codes (`invalid_request`, `not_found`, `internal`) |
| Domain types | `internal/notes/types.go::{Note, Attachment, CreateInput, ErrInvalidNote, ErrNoteNotFound}` | Domain-pure (no proto imports); typed sentinels translate into Connect errors at the handler edge |
| Repository interface | `internal/notes/repository.go::Repository` | Persistence seam — `Create` / `Get` / `List` |
| Repository impl | `internal/notes/sqlite.go::NewSQLiteRepository` | sqlite-backed `Repository`; production wires it once in `main.go` |
| Schema | `internal/notes/schema.{sql,go}::Schema()` | Domain-owned table DDL embedded via `go:embed`; collected by `internal/modules/registry.go::AllSchemas()` and applied at boot via `apidb.EnsureSchemas` |
| Repository test | `internal/notes/sqlite_test.go` | Real handle via `db.NewSQLite(t)` + `apidb.EnsureSchemas(ctx, d, ...providers...)` over system + notes (the canonical compose pattern) |
| Service | `internal/notes/service.go::Service` (+ `NewService`) | Application layer: validation (`title` required after whitespace trim), default substitution (`defaultListLimit = 100` when caller passes 0). Handler depends on this, not the repository. |
| Service test | `internal/notes/service_test.go` | Substitutes `mocks.FakeRepository` (from co-located `internal/notes/mocks/`); pins the validation, default-substitution, and error-propagation contracts |
| Connect handler test | `handlers/notes/connect_handler_test.go` | Substitutes `mocks.FakeService` and exercises the generated Connect client/handler path |
| Multipart handler test | `handlers/notes/attachments_handler_test.go` | Uses `blobstore.MemoryBlobStore` plus test metadata repositories to exercise file-upload success and error paths |
| Mocks | `internal/notes/mocks/{repository,service}.go::{FakeRepository,FakeService}` | Co-located with the domain (Pass-3 pattern) — `FakeRepository` carries state for service tests; `FakeService` records inputs for handler tests. Both use atomic call counters + per-method error knobs. Deleting `internal/notes/` takes them along. |
| UI client | `ui/src/api/notes.ts` | `notesClient = createClient(NotesService, transport)` plus `uploadAttachment` for multipart metadata |
| UI tests | `ui/src/api/notes.test.ts` + component tests | Mock generated client methods and `uploadAttachment`; REST helper tests stub `global.fetch` |
| CLI client | `cli/domains/notes/{register,handlers,attach_handler}.go` | `Register(core)` returns a `cliapp.SubcommandGroup`; handlers use generated Connect clients or `cliapp.UploadFile` and render via cli-core reports |
| CLI test | `cli/domains/notes/handlers_test.go` | Spins a real `httptest.Server` via `testutil.NewAPIServer`, captures stdout via `testutil.CaptureStdout` |

#### Compose pattern: schema-applied repository test

`db.NewSQLite(t)` returns a blank handle. Repository tests apply the
production schema before the first query so the test exercises the
same shape `main.go` ships:

```go
func newSchemaDB(t *testing.T) *sql.DB {
    t.Helper()
    d := db.NewSQLite(t)
    require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
        apidb.SchemaProviderFunc(localdb.SystemSchema),
        apidb.SchemaProviderFunc(notes.Schema),
    ))
    return d
}
```

That helper is the canonical entry point for every new domain's
`*_sqlite_test.go`. Don't reach for migrations frameworks or in-test
`CREATE TABLE` literals — the per-domain `schema.sql` files (collected
by `internal/modules/registry.go::AllSchemas()` in production) are the
source of truth for both production and tests.

### Service-layer tests

The notes domain uses three test layers, each with a different fake:

```
HTTP → handler → Service (validates, applies defaults) → Repository (persists)
                     ↑                                       ↑
                     FakeService (handler tests)              FakeRepository (service tests)
                                                              Real sqlite (repository tests)
```

`internal/notes/service_test.go` is the reference. Service tests:

- Substitute `mocks.FakeRepository` (in-memory state) so the test can
  assert on what the repository was called with and whether the service
  filtered the call (e.g., empty title rejected before reaching `Create`).
- Pin validation contracts (`Create` rejects empty / whitespace-only
  title with `ErrInvalidNote{Field: "title"}`).
- Pin default-substitution contracts (`List(0)` substitutes
  `defaultListLimit`; `List(5)` passes 5 through unchanged).
- Pin error propagation (`Get` returns `ErrNoteNotFound` verbatim;
  `Create` returns repository errors verbatim).

Connect handler tests then substitute `mocks.FakeService` — they don't seed
sqlite-shaped state to assert on routing. Two-mock split keeps each
layer's tests focused on what that layer owns.

### Temporal workflow tests

The canonical workflow inventory lives in
[`FLOWS.md`](../concepts/FLOWS.md). Tests prove the state/event
contracts documented there.

Use temporal workflow tests when a domain has lifecycle states where
some events are allowed and others are forbidden. Do not use coverage
percentages as proof that the state space is complete; a suite can
touch every line while never testing "retry after success" or
"complete after cancel."

The canonical API shape is:

```
api/internal/<domain>/
  <flow>_workflow.flow.json     # hand: source of truth
  <flow>_workflow.go            # hand: wrapper
  <flow>_workflow_test.go       # hand: thin replay delegation
  generated/<foldername>/
    model.qnt
    artifact.json
    runtime.go
    replay.go
```

`workflow.go` defines:

- status and event types used by the generated topology declarations,
- a pure `Transition(state, event)` wrapper around generated
  status-transition helpers,
- `CheckInvariants(state)` for rules that must hold after every
  transition.

`model_conformance_test.go` uses
`api/internal/testutil/modeltest` to prove:

- every production status is represented,
- every production event is represented,
- every status/event pair has exactly one expected row,
- duplicate, missing, and unknown rows fail loudly,
- traces replay step-by-step against the production transition
  function.
- the generated formal artifact is fresh against the `*.flow.json`
  contract, generated `.qnt` model, generator source, and checked
  invariants.
- the generated transition-table check is present as generated-check
  metadata, not as a fake verified invariant.

The canonical UI shape is:

```
ui/src/features/<domain>/
  <Domain>Workflow.flow.json    # hand: source of truth
  <Domain>Workflow.ts           # hand: wrapper
  <Domain>Workflow.fixtures.ts  # hand: replay fixtures
  <Domain>Workflow.test.ts      # hand: thin replay delegation (~5-8 lines)
  generated/<foldername>/
    model.qnt
    artifact.json
    runtime.ts
    replay.helper.ts
```

Use TypeScript discriminated unions so impossible UI states are not
representable. For example, an upload should not be able to hold both
`{ status: "uploading" }` and a success payload through parallel
booleans. Components dispatch events to the workflow and render the
returned state; they do not duplicate transition rules in event
handlers. Generated formal replay helpers build replay transitions with
the shared `transitionFromReplayAdapter` helper plus generated fixture
map types and generated `*ReplayFixtureContract` constants, so adding a
generated status/event creates a type error until the runtime fixture
exists.

Workflow maturity is incremental:

| Level | Name | Validation expectation |
|---|---|---|
| 1 | Inventory | Flow listed in `docs/concepts/FLOWS.md`. |
| 2 | Workflow model | Pure transition and invariant checks exist. |
| 3 | Matrix + traces | Every state/event pair and representative trace is executable. |
| 4 | Declarative contract | A domain-local `*.flow.json` declares states, events, transitions, invariants, and named traces. |
| 5 | Checked formal model | Quint/TLA+ or equivalent is generated from the contract, checked, and replayed by production tests. |

The notes attachment upload workflow is the reference Level 5 pattern:

- The `flow-verifier` scenario CLI (`flow-verifier verify check|run`, `flows list|validate|explain`)
- `api/internal/notes/flow/flow.json`
- `api/internal/notes/flow/transition.go` (package `flow`)
- `api/internal/notes/flow/flow_test.go` (thin replay delegation, package `flow`)
- `api/internal/notes/flow/generated/{model.qnt,artifact.json,runtime.go,replay.go}` (package `generated`)
- `ui/src/features/notes/flow/flow.json`
- `ui/src/features/notes/flow/transition.ts`
- `ui/src/features/notes/flow/fixtures.ts`
- `ui/src/features/notes/flow/flow.test.ts` (thin replay delegation)
- `ui/src/features/notes/flow/generated/{model.qnt,artifact.json,runtime.ts,replay.helper.ts}`

`make temporal-models` invokes `flow-verifier verify check --root .`, which
runs `quint typecheck`, `quint test`, `quint verify`, and deterministic MBT
trace generation through the flow-verifier pipeline. It fails if the checked-in
artifacts, generated declarations, or generated replay files are stale. The
generated declarations provide state/event topology and formal freshness
expectations, including concrete hashes for the contract, model, and generator.
They also expose pure generated status-transition helpers derived from
`*.flow.json`, so production code does not maintain a second abstract
transition matrix. Generated Go and TypeScript replay tests load those
artifacts through `modeltest` and replay generated transitions/traces against
production transition functions. UI replay keeps the hand-authored runtime
fixture map in `flow/fixtures.ts`; the generated
`replay.helper.ts` owns freshness, matrix replay, and trace replay, and
the hand-authored `.test.ts` is a ~5-line module that imports the helper
and the fixtures and calls `runFormalReplay({ transition, fixtures })`
at top level. An AST-level lint in `flow-verifier verify check` rejects any
file that imports the helper without calling it.

Formal artifacts use schema v5 coverage metadata. `transitionMatrixComplete`
and `terminalTransitionsChecked` describe the generated matrix. `namedTraces`
describes required hand-authored trace coverage. `generatedTraces` reports
what Quint MBT traces visited, including `coveredPairs` and
`allPairsCovered`; that field is informational and may be false.

Schema v5 `*.flow.json` files no longer declare any output paths; the
generated subpackage location is derived from the flow ID. The contract's
`replay` block carries only `fixtureModule`, `fixtureExport`, and
`transition` metadata.
`flow-verifier flows validate`, `verify run`, and `verify check` validate each
contract against the embedded flow schema before semantic validation, so
unknown fields, missing required fields, old marker-based `replay.bindings`,
and invalid enum values fail with contract-path context before Quint runs.
`check` then compares the generated replay files byte-for-byte, which makes a
missing production replay test a generator failure instead of a later review
catch. Use `flow-verifier flows explain --flow <flow-id>` to inspect generated
files, runtime typing, fixture contracts, topology, generated replay paths,
fixture module expectations, coverage, and the exact commands to run next.

A Quint/TLA+ model is only accepted when this full loop exists.
Documentation-only formal specs are drift-prone and should not be
added. Plain CRUD should stay plain; copy the Level 5 pattern only for
flows with lifecycle states and illegal transitions.

When adding or changing a Level 5 state/event:

1. Edit the flow contract.
2. Regenerate that flow with `flow-verifier verify run --root . --flow <flow-id>`.
3. Update only runtime payload logic that the abstract model cannot own
   (file handles, attempt ids, repository side effects, user-facing
   messages).
4. Update UI replay fixture modules; missing keys should be compile-time
   failures via the generated formal replay fixture interface.
5. Run `make temporal-models` before the regular scenario tests.

### Buffer-backed logger pattern

The production `*log.Logger` shouldn't write to stderr during tests —
it pollutes the runner's output and makes failure messages harder to
read. Connect handler tests should use the shared helper:

```go
logger, logBuf := connectxtest.NewLogger(t)
client := newNotesClient(t, fakeService, logger)
```

For scenario-local helpers that do not consume `api-core/connectx`, the same
shape is a `bytes.Buffer`-backed logger:

```go
logBuf := &bytes.Buffer{}
srv := server.New(server.Deps{
    Logger: log.New(logBuf, "", 0),
    // …other deps
})
```

Discard-only sinks (`log.New(io.Discard, "", 0)`) work for tests that
don't need to inspect log output; reach for the buffer when the test
asserts on what was logged (e.g., the 500-path test in
`handlers/notes/connect_handler_test.go::TestConnectHandler_GetInternalError`
checks the underlying error reaches operator logs).

### Testing context cancellation

The template ships no streaming endpoints today, but the test
infrastructure supports them. When a scenario adds an SSE / long-poll
/ background-work endpoint, the canonical cancellation test is:

```go
live := httpx.NewLiveServer(t, srv)
ctx, cancel := context.WithCancel(context.Background())
req, _ := http.NewRequestWithContext(ctx, http.MethodGet, live.URL+"/stream", nil)
resp, err := live.Client.Do(req)
require.NoError(t, err)
defer resp.Body.Close()

// Read enough bytes to confirm the handler started writing.
buf := make([]byte, 64)
_, _ = resp.Body.Read(buf)

cancel()
// Expect the handler to observe r.Context().Done() and abort cleanly.
// The fake (Pinger / FakeService / future StreamProducer) records the
// cancellation; assert on the recorded state.
```

`httpx.NewLiveServer` runs over a real socket (not `httptest.NewRecorder`),
so `http.Flusher` and the request `Context()` plumbing match production
behavior. Recorder-based tests would silently pass while production
hangs on a never-cancelled handler — the same class of bug
workspace-sandbox shipped on 2026-04-28.

### Production-import quarantine (`no_prod_import_test.go`)

The test in `api/internal/testutil/no_prod_import_test.go` walks every
non-test `.go` file under `api/` and fails if any imports anything
starting with `<module>/internal/testutil/`. The module name is read
dynamically from `go.mod`, so it works after `code-facts`
substitution.

This is the load-bearing rule that makes it safe to put `time.Sleep`,
process-wide globals, and `testing.T` references in the testutil
package. If the rule fires:

- ✅ **Move the helper out of testutil** into a non-test package.
- ❌ **Don't add `// nolint`** — the production code path will then
   carry the test-only dep into the binary on every build.

### Outbound HTTP — `httpc.Doer`

Production callers consuming external services depend on the `Doer`
interface declared at `internal/httpc/doer.go`. `*http.Client` satisfies
it directly (compile-time-asserted in the same file); tests substitute
`mocks.FakeDoer`. Reference test:
`internal/httpc/doer_test.go::TestDoer_TestPath` exercises both the
production-side and test-side wiring through one tiny inline caller —
the canonical substitution shape.

`mocks.FakeDoer` queues canned `*http.Response` (or errors) via
`AddResponse(status, body) []byte` and records every inbound request
into `.Requests` for after-the-fact assertions. The test fake is the
same shape every scenario reaches for; resist hand-rolling per-feature
HTTP fakes when this surface fits.

The seam ships *unwired* in production by intent — there's no
`server.Deps.Doer` field until the first scenario actually needs one.
When you wire it, follow the canonical pattern:

```go
// main.go
deps := server.Deps{
    Doer: &http.Client{Timeout: 10 * time.Second},
    // …other deps
}
```

## UI testing

### Layout

```
ui/src/
├── test-setup.ts                # vitest setupFiles entry
├── test-utils/
│   ├── index.ts                 # re-exports
│   ├── a11y.ts                  # expectNoA11yViolations(container)
│   ├── factories.ts             # makeHealthResponse(overrides?)
│   ├── renderWithProviders.tsx  # QueryClient + i18n wrapper
│   └── mocks/
│       └── spatial.ts           # builders for @vrooli/iframe-bridge/spatial
├── components/
│   ├── AppShell.test.tsx
│   └── AppShell.a11y.test.tsx
└── features/<name>/             # feature tests and feature a11y live here
```

The `mocks/` directory holds shared mock-shape builders for external
SDKs (today: spatial-nav). Each hook test still calls
`vi.mock("@vrooli/iframe-bridge/spatial", ...)` inline (Vitest hoisting
is non-negotiable), but the factory closure invokes the builders in
`mocks/spatial.ts` so the contract for each SDK lives in one file.
Adding a new SDK: drop a `mocks/<sdk>.ts` builder beside it, add a
`mocks/<sdk>.test.ts` self-test, re-export from `test-utils/index.ts`.

### The four primitives every test uses

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

### Canonical UI test pattern

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { makeApiMocks, renderWithProviders } from "../../test-utils";

vi.mock("../../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/health")>();
  return { ...actual, ...makeApiMocks() };
});

import { HealthCard } from "./HealthCard";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

describe("HealthCard rendering (cimode — copy-independent)", () => {
  afterEach(() => { cleanup(); });

  it("renders the card via test ID", () => {
    renderWithProviders(<HealthCard />);
    expect(screen.getByTestId(selectors.health.card)).toBeInTheDocument();
  });

  it("renders translation keys in cimode", async () => {
    renderWithProviders(<HealthCard />);
    expect(await screen.findByText(strings.health.title)).toBeInTheDocument();
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

See `features/health/HealthCard.test.tsx` for the full pattern and the
CLDR plural variants (`refreshCount_one`,
`notifications.summary_zero` / `_one` / base). Keep `App.test.tsx`
smoke-only so deleting a feature does not require rewriting the app
composition test.

### Mock builders for `api/health` and `api/notes`

`vi.mock(path, factory)` is hoisted before any user import resolves;
a wrapper imported from `test-utils` would be in the temporal dead
zone at hoist time. The escape hatch is to keep the `vi.mock` call
inline at the top of each test file, but move the *factory body* into
a builder function that runs when the closure executes — which is
*after* imports initialise.

`@/test-utils` exports shared, cross-feature mock builders such as
`makeApiMocks()`. Feature-specific builders live beside the feature so
deleting the feature takes its mocks with it; for notes, import
`makeNotesMocks()` from `features/notes/mocks/notes`.

Canonical shape:

```tsx
import { makeApiMocks } from "@/test-utils";
import { makeNotesMocks } from "./mocks/notes";

vi.mock("../../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/health")>();
  return { ...actual, ...makeApiMocks() };
});

vi.mock("../../api/notes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/notes")>();
  return { ...actual, ...makeNotesMocks() };
});
```

Defaults are picked so the most common test paths work no-args:
`makeApiMocks().fetchHealth` resolves to a healthy response;
`makeNotesMocks().notesClient.listNotes` resolves to an empty list;
`notesClient.createNote({ title })` echoes the title back as a Note.
Per-test overrides use vitest's standard pattern *after* the mock is wired:

```tsx
const { notesClient } = await import("../../api/notes");
vi.mocked(notesClient.listNotes).mockResolvedValueOnce(
  makeListNotesResponse({ notes: [makeNote({ id: "a" })] }),
);
```

The `...actual` spread keeps non-mocked exports (the `ApiError` class,
re-exported proto types) intact — only network-touching functions are
substituted.

When a third lib/* surface lands (e.g., `lib/users.ts`), follow the
same pattern: builder in `ui/src/test-utils/mocks/<surface>.ts`, self-
test alongside, re-export from `test-utils/index.ts`.

### Accessibility tests

Accessibility tests follow the same ownership rule as production UI:

- **Shell**: `components/AppShell.a11y.test.tsx` renders
  `<AppShell>` with stable placeholder children. It covers page layout,
  headings, locale controls, and shell-level semantics.
- **Feature**: `features/<name>/<Name>Card.a11y.test.tsx` renders the
  feature directly, owns its API mocks, waits for each state it scans,
  and calls `expectNoA11yViolations(container)`.
- **App**: `App.test.tsx` stays composition smoke. Do not make it the
  default a11y gate; a full-`App` a11y test couples shell coverage to
  every async feature query and becomes fragile as features change.

Before running axe, wait for the state the test owns. For example,
`HealthCard.a11y.test.tsx` waits for `selectors.health.statusValue`
for the success state and `selectors.health.error` for the error state.
This keeps React Query updates inside the awaited test boundary and
prevents `act(...)` warnings.

`test-setup.ts` fails tests that write unexpected `console.error` or
`console.warn` output. If a test intentionally exercises a noisy React
path, suppress that warning locally and assert the user-visible
contract. `ErrorBoundary.test.tsx` is the reference: it suppresses
React's intentional boundary logging while asserting `onError` and the
fallback UI.

### ErrorBoundary tests

`ui/src/components/ErrorBoundary.test.tsx` is the reference for testing
React error boundaries. The patterns it pins:

- **Controlled-throw fixture** (`Throw({ when, message })`). One
  component shared across cases keeps the surface narrow — the
  boundary's contract is "if a child throws, swap to the fallback,"
  and that's what each case exercises.
- **`console.error` suppression** in `beforeEach` / `afterEach`. React
  intentionally logs the caught error; without the spy, the suite's
  output drowns real failure messages. The `onError` test still
  asserts the prop fired, so coverage of the error path is preserved.
- **Mutable-control recovery test**. To prove `setState` re-renders
  children, the test flips a shared object's `value` field rather
  than re-mounting the boundary — boundary identity has to survive
  the retry click for `setState` to take effect.
- **cimode key assertions**. The default test setup runs i18next in
  cimode, so `t(strings.errorBoundary.title)` returns the literal
  key path. Asserting on the key (not the English copy) proves the
  fallback consulted the registry — the test stays green when copy
  changes and breaks loudly if a key is renamed.

App-level wrap point lives in `ui/src/main.tsx`; the boundary nests
inside `<QueryClientProvider>` (and after the `./i18n` side-effect
init) so `useTranslation` works inside the localised fallback.

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

    clitest "code-facts/cli/internal/testutil"
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
3. **`CaptureStdout`** captures the human output written by
   `RenderProtoList`, `RenderProtoMutation`, or the `RunContext`
   report helpers. Use it instead of `--json` when you want to assert
   on the contract scenario users actually see.
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

## How to add a new proto

Wire shapes for new endpoints belong in proto, not in hand-written Go
structs or TS interfaces. After generation, the canonical source lives
under `packages/proto/schemas/code-facts/`.

Steps:

1. **Author the schema.** In a generated scenario, add
   `packages/proto/schemas/code-facts/v1/<domain>/<name>.proto`.
   Use snake_case in the proto package directive
   (`package vrooli.code_facts.v1.<domain>;`) and add a
   `go_package` option pointing at the per-scenario gen path:

   ```protobuf
   option go_package = "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/<domain>;<domain>_v1";
   ```

2. **Regenerate.** From the repo root:

   ```bash
   cd packages/proto && make generate && make lint
   ```

   New artifacts land under the language-specific generated trees:
   `packages/proto/gen/go/code-facts/v1/<domain>/`,
   `packages/proto/gen/typescript/js/code-facts/v1/<domain>/`, and
   `packages/proto/gen/python/code_facts/v1/<domain>/`.
   Commit them alongside the schema — generated code is checked in so
   downstream scenarios don't have to re-run codegen.

3. **Wire it on the API side.** Import the generated Go type in your
   handler test and decode via `assertx.MustUnmarshalProto`:

   ```go
   import notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/notes"

   got := assertx.MustUnmarshalProto[notesv1.ListResponse](t, body)
   ```

   For fixtures, follow the `fixtures/health.go` pattern — re-export
   the proto type as a short alias and provide functional-options
   builders:

   ```go
   type ListResponse = notesv1.ListResponse
   func NewListResponse(opts ...ListOpt) *notesv1.ListResponse { /* ... */ }
   ```

4. **Wire it on the UI side.** Import the generated TS schema and use
   `fromJson` for decode + `create` for fixtures:

   ```ts
   import { fromJson, create } from "@bufbuild/protobuf";
   import { ListResponseSchema } from "@vrooli/proto-types/code-facts/v1/notes/notes_pb";

   // production
   return fromJson(ListResponseSchema, json, { ignoreUnknownFields: true });

   // tests
   const fixture = create(ListResponseSchema, { items: [{ id: "n-1" }] });
   ```

5. **Tests follow.** Connect handler tests call the generated client;
   fixture tests assert on the typed shape via `proto.Equal`. UI tests
   mock `api/notes` and return generated response objects from the
   factory.

Don't add a new `mocks/Fake*` interface for the proto type — the proto
isn't a seam, it's a contract. Seams are interfaces; protos are
payload shapes. See `SEAMS.md::Wire contracts live in proto, not seams`.

## E2E binary smoke gate

`api/main_e2e_test.go` is a build-tag-isolated test (`//go:build e2e`)
that:

1. `go build -o <tmp> .` from the api directory
2. Boots the binary with `API_PORT` and
   `VROOLI_LIFECYCLE_MANAGED=true` set
3. Polls `/health` over a real socket
4. Sends `SIGTERM` and asserts clean exit within 5s

It catches regressions handler tests can't see:

- main.go forgets to wire a new field into `server.Deps` (handler
  tests construct `server.New` directly with a hand-built Deps;
  main.go's wiring is unverified)
- `preflight.Run` order changes break the boot path
- `apiserver.Run` listener config drifts (SIGTERM cleanup hook,
  port resolution from env)
- `storage.NewResolver(ProfileAuto)` chooses an unwritable path on
  the host

Default `go test ./...` skips it (no `e2e` tag). The CI workflow
runs it as a dedicated step via `go test -tags=e2e -run TestE2E .`.
Local invocation: `cd api && go test -tags=e2e .`.

This is the only test in the suite that exercises the actual binary
entry point. As scenarios add streaming endpoints, websocket upgrades,
or background workers, extend this file with matching `TestE2E_*`
cases — one per top-level boot-path concern. Resist using it for
feature-level coverage; that's what handler tests are for.

## Coverage thresholds

| Module | Floor | Where it's enforced |
|---|---|---|
| UI (`ui/`) | 85% lines / branches / functions / statements | `ui/vite.config.ts` `test.coverage`; CI runs `pnpm test:coverage` |
| API (`api/`) | 75% total | `.github/workflows/test.yml` `api` job |
| CLI (`cli/`) | 75% total | `.github/workflows/test.yml` `cli` job |

### UI

Configured in `vite.config.ts` at the bottom of the `test.coverage`
block. The `coverage.exclude` list covers test scaffolding and codegen
only — production source under `src/` is exhaustively included. The
default position is: every new `src/` file ships with its own
`*.test.{ts,tsx}` and lands inside the include set. If a scenario adds
genuinely-untestable code, prefer a narrow file exclusion with a
one-line rationale comment in `vite.config.ts` over loosening the
thresholds.

### Go (API + CLI)

Both Go modules gate on a 75% total floor in CI. The threshold is
intentionally lower than the UI's 85% because Go has more
declaration-only surface (interfaces, generated proto types, struct
types) that doesn't carry executable lines — 75% in Go is roughly
equivalent to 85% in TypeScript by lines-per-meaningful-coverage.

`internal/testutil/...` is excluded from the denominator. Those
packages exist to support tests; including them would create the wrong
incentive (writing tests *of* test helpers to inflate the gate). Each
test-utils package has its own self-test (see
`internal/testutil/db/sqlite_test.go`,
`internal/testutil/httpx/server_test.go`, etc.) so the substrate
itself is still verified — it's just not what the production-coverage
number tracks.

Raise toward 80%/85% as scenarios stabilise. Tighten the threshold
rather than loosening it when a new file lands without tests; that's
the signal that drives the test-first habit.

### CI failure mode

A drop below floor fails the relevant CI job immediately with the
actual percentage in the error message (`::error::API coverage 71.4%
< 75%`). The fix is to raise coverage in the missing file, not to
lower the gate.

## Common patterns and anti-patterns

| ✅ DO | ❌ DON'T |
|---|---|
| `mocks.FakeClock` for time-dependent assertions | `time.Sleep(150 * time.Millisecond)` then assert on a fuzzy match |
| `httpx.NewLiveServer` for handler tests | `httptest.NewRecorder` (hides SSE-flusher bugs) |
| `getByTestId(selectors.x.y)` for stable selectors | `getByText("Save")` (breaks the moment copy changes) |
| `vi.mock("./api/health", async (importOriginal) => …)` inline at top of file | Helper-wrapped `vi.mock` (TDZ at hoist time) |
| `makeHealthResponse({ status: "degraded" })` for variants | Hardcoded literal payload in three different tests |
| Per-method error knob (`PingErr error`) on fakes | Single global "fail mode" boolean across the fake |
| `var _ Pinger = (*sql.DB)(nil)` to lock the contract at compile time | Runtime "does this satisfy" check in init |
| Adding new seams to [`SEAMS.md`](SEAMS.md) at the same commit | "We'll document it later" |

## Cross-references

- **Seams definition + adding new seams**: [`SEAMS.md`](SEAMS.md).
- **Skill bundle for testing-related work** (load before substantial test changes):
  ```bash
  prompt-manager skill read seam-discovery-and-enforcement
  prompt-manager skill read test
  prompt-manager skill read unit-testing-architecture-steer
  ```
- **Test runner used by CI and `vrooli scenario test`**: see
  `.github/workflows/test.yml` and `packages/cli-core/cmd/scenario_test.go`.
- **Why no inline mocks in `*_test.go` files**: the testutil package
  is the single source of fake behavior. Inline mocks in tests
  fragment the contract; when the interface grows a method, every
  inline mock has to be updated. One mock in `mocks/`, one update.
