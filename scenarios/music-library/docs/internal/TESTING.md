# Testing — Music Library

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

### CRUD reference — an end-to-end vertical slice

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
        apidb.SchemaProviderFunc(<domain>.Schema),
    ))
    return d
}
```

That helper is the canonical entry point for every new domain's
`*_sqlite_test.go`. Don't reach for migrations frameworks or in-test
`CREATE TABLE` literals — the per-domain `schema.sql` files (collected
by `internal/modules/registry.go::AllSchemas()` in production) are the
source of truth for both production and tests.

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The `notes` domain is the scaffold's worked CRUD reference. Copy its
shape for your own domains, then remove it. The pattern from wire to
render:

| Layer | File | What it owns |
|---|---|---|
| Wire contract | `packages/proto/schemas/music-library/v1/notes/notes.proto` | `Note`, `service NotesService`, `ListNotesResponse`, `CreateNoteRequest`, `CreateNoteResponse`, `GetNoteRequest`, `GetNoteResponse` |
| REST metadata contract | `packages/proto/schemas/music-library/v1/notes/attachments.proto` | `Attachment` and `UploadAttachmentResponse` for the multipart upload exception |
| Connect error mapping | `internal/notes/service_error_mapping.go` | Typed sentinels become Connect codes (`invalid_argument`, `not_found`, `internal`) |
| REST error envelope | `packages/proto/schemas/music-library/v1/shared/errors.proto` + `internal/httpx/errors.go::WriteError` | Typed body for REST exceptions, with canonical codes (`invalid_request`, `not_found`, `internal`) |
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

The compose pattern's `<domain>.Schema` resolves to `notes.Schema` for
this example.

#### Service-layer tests (`notes`)

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
<!-- EXAMPLE-DOMAIN:notes END -->

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

The scaffold ships one fenced worked Level 5 flow (an attachment-upload
workflow on the example domain) as the reference; copy its file layout
for a real flow, then remove it with `template-manager detemplate`. The
generic file layout per flow is:

- The `flow-verifier` scenario CLI (`flow-verifier verify check|run`, `flows list|validate|explain`)
- `api/internal/<domain>/flow/flow.json`
- `api/internal/<domain>/flow/transition.go` (package `flow`)
- `api/internal/<domain>/flow/flow_test.go` (thin replay delegation, package `flow`)
- `api/internal/<domain>/flow/generated/{model.qnt,artifact.json,runtime.go,replay.go}` (package `generated`)
- `ui/src/features/<domain>/flow/flow.json`
- `ui/src/features/<domain>/flow/transition.ts`
- `ui/src/features/<domain>/flow/fixtures.ts`
- `ui/src/features/<domain>/flow/flow.test.ts` (thin replay delegation)
- `ui/src/features/<domain>/flow/generated/{model.qnt,artifact.json,runtime.ts,replay.helper.ts}`

<!-- EXAMPLE-DOMAIN:notes START -->
The `notes` attachment-upload workflow is the scaffold's worked Level 5
example (removed by `template-manager detemplate`). It instantiates the
layout above as:

- `api/internal/notes/flow/flow.json`
- `api/internal/notes/flow/transition.go` (package `flow`)
- `api/internal/notes/flow/flow_test.go` (thin replay delegation, package `flow`)
- `api/internal/notes/flow/generated/{model.qnt,artifact.json,runtime.go,replay.go}` (package `generated`)
- `ui/src/features/notes/flow/flow.json`
- `ui/src/features/notes/flow/transition.ts`
- `ui/src/features/notes/flow/fixtures.ts`
- `ui/src/features/notes/flow/flow.test.ts` (thin replay delegation)
- `ui/src/features/notes/flow/generated/{model.qnt,artifact.json,runtime.ts,replay.helper.ts}`
<!-- EXAMPLE-DOMAIN:notes END -->

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
client := new<Domain>Client(t, fakeService, logger)
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
asserts on what was logged — e.g. a 500-path handler test that checks
the underlying error reaches operator logs.

<!-- EXAMPLE-DOMAIN:notes START -->
The example domain's reference for the buffer-logger 500-path assertion
is `handlers/notes/connect_handler_test.go::TestConnectHandler_GetInternalError`
(removed by `template-manager detemplate`).
<!-- EXAMPLE-DOMAIN:notes END -->

## UI testing

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
example below shows the full two-mock shape.

<!-- EXAMPLE-DOMAIN:notes START -->
#### Example domain — `notes` mock builders (removed by `template-manager detemplate`)

For notes, import `makeNotesMocks()` from `features/notes/mocks/notes`
and wire it alongside the shared `makeApiMocks()`:

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

Defaults: `makeNotesMocks().notesClient.listNotes` resolves to an empty
list; `notesClient.createNote({ title })` echoes the title back as a
Note. Per-test overrides:

```tsx
const { notesClient } = await import("../../api/notes");
vi.mocked(notesClient.listNotes).mockResolvedValueOnce(
  makeListNotesResponse({ notes: [makeNote({ id: "a" })] }),
);
```
<!-- EXAMPLE-DOMAIN:notes END -->

When a third lib/* surface lands (e.g., `lib/users.ts`), follow the
same pattern: builder in `ui/src/test-utils/mocks/<surface>.ts`, self-
test alongside, re-export from `test-utils/index.ts`.
