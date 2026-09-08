# Testing — UI Health

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
- [cli/app_test.go](../../cli/app_test.go)

## TL;DR — the canonical examples

These files are the source of truth. When in doubt, copy their shape:

- **API**: `api/handlers/health/handler_test.go` — table-driven, real
  middleware via `httpx.NewLiveServer`, fake pinger from `mocks/`,
  typed-proto decode via
  `assertx.MustUnmarshalProto[healthv1.Response]` (the wire shape lives
  in `packages/proto/schemas/ui-health/v1/health/health.proto`;
  assert on typed proto fields, not `map[string]any` chains). For
  endpoints whose wire shape isn't in proto yet, `MustDecodeJSON[T]`
  is the fallback — but adding the proto first is the right move.
- **UI composition**: `ui/src/App.test.tsx` — smoke-only composition
  test. App composes shell + features; feature behaviour belongs beside
  the feature.
- **UI feature**: `ui/src/features/health/HealthCard.test.tsx` —
  `renderWithProviders`, factory data, inline `vi.mock` factory
  closure, cimode assertions, and real-locale assertions.
- **UI a11y**: `ui/src/components/ui/primitives.a11y.test.tsx`,
  `ui/src/features/validation/validation.a11y.test.tsx`,
  `ui/src/features/search/search.a11y.test.tsx`,
  `ui/src/features/inventory/inventory.a11y.test.tsx`,
  `ui/src/features/reindex/reindex.a11y.test.tsx`, and
  `ui/src/pages/DashboardPage.a11y.test.tsx` — primitive and feature
  accessibility are tested at their ownership boundary.
- **CLI**: `cli/app_test.go` — smoke gate (NewApp, --version, --help).
  When domain commands arrive, extend with `clitest.NewAPIServer` +
  `clitest.CaptureStdout` from `cli/internal/testutil/`.

If your test doesn't look like one of those three, ask why before
shipping.

## API testing

### CRUD reference

This scenario currently ships transport-only domains (health, reindex,
search, validation), so the live tree has no worked example of the
full proto → repository → service → handler → UI → CLI stack. The
canonical CRUD reference lives upstream in the react-vite template's
notes module — copy that layering when adding a persistence-backed
domain here.

The compose pattern below stays canonical for any future
persistence-backed domain in this scenario:

```go
func newSchemaDB(t *testing.T) *sql.DB {
    t.Helper()
    d := db.NewSQLite(t)
    require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
        apidb.SchemaProviderFunc(localdb.SystemSchema),
        apidb.SchemaProviderFunc(yourDomain.Schema),
    ))
    return d
}
```

Don't reach for migrations frameworks or in-test `CREATE TABLE`
literals — per-domain `schema.sql` files (collected by
`internal/modules/registry.go::AllSchemas()` in production) are the
source of truth for both production and tests.

### Service-layer tests

The canonical service-layer test pattern uses three layers, each with
a different fake:

```
HTTP → handler → Service (validates, applies defaults) → Repository (persists)
                     ↑                                       ↑
                     FakeService (handler tests)              FakeRepository (service tests)
                                                              Real sqlite (repository tests)
```

Service tests substitute `mocks.FakeRepository` (in-memory state) so
they can assert what the repository was called with and whether the
service filtered the call (e.g. empty input rejected before reaching
`Create`). Pin validation, default-substitution, and error-propagation
contracts at this layer.

Connect handler tests then substitute `mocks.FakeService` — they don't
seed sqlite-shaped state to assert on routing. Two-mock split keeps
each layer's tests focused on what that layer owns. See the upstream
react-vite notes module for a fully worked example.

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

No Level 5 workflows ship in this scenario today. The reference Level
5 pattern (declarative contract + checked formal model) is the
upstream react-vite template's notes attachment upload workflow:

- The `flow-verifier` scenario CLI (`flow-verifier verify check|run`, `flows list|validate|explain`)
- `api/internal/<domain>/flow/{flow.json,transition.go,flow_test.go,generated/...}`
- `ui/src/features/<feature>/flow/{flow.json,transition.ts,fixtures.ts,flow.test.ts,generated/...}`

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
asserts on what was logged (e.g. a 500-path test that checks the
underlying error reaches operator logs).

## UI testing

### Mock builders for `api/*`

`vi.mock(path, factory)` is hoisted before any user import resolves;
a wrapper imported from `test-utils` would be in the temporal dead
zone at hoist time. The escape hatch is to keep the `vi.mock` call
inline at the top of each test file, but move the *factory body* into
a builder function that runs when the closure executes — which is
*after* imports initialise.

`@/test-utils` exports shared, cross-feature mock builders such as
`makeApiMocks()` (defaults `fetchHealth` to a healthy response).
Feature-specific overrides live beside the feature so deleting the
feature takes its mocks with it.

Canonical shape — wrap a domain client and pin a single per-test
return value:

```tsx
import { makeApiMocks } from "@/test-utils";

vi.mock("../../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/health")>();
  return { ...actual, ...makeApiMocks() };
});

vi.mock("../../api/search", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/search")>();
  return {
    ...actual,
    searchSurfaces: vi.fn().mockResolvedValue({ hits: [], modeUsed: "text" }),
  };
});
```

Per-test overrides use vitest's standard pattern *after* the mock is
wired:

```tsx
const { searchSurfaces } = await import("../../api/search");
vi.mocked(searchSurfaces).mockResolvedValueOnce({ hits: [/* … */], modeUsed: "ai" });
```

The `...actual` spread keeps non-mocked exports (the `ApiError` class,
re-exported proto types, pure helpers like `filterHits` /
`reindexStateFromString`) intact — only network-touching functions are
substituted.

When a new surface lands (e.g., a future `lib/users.ts`), follow the
same pattern: builder in `ui/src/test-utils/mocks/<surface>.ts`, self-
test alongside, re-export from `test-utils/index.ts`.

## How to add a new proto

Wire shapes for new endpoints belong in proto, not in hand-written Go
structs or TS interfaces. After generation, the canonical source lives
under `packages/proto/schemas/ui-health/`.

Steps:

1. **Author the schema.** In a generated scenario, add
   `packages/proto/schemas/ui-health/v1/<domain>/<name>.proto`.
   Use snake_case in the proto package directive
   (`package vrooli.ui_health.v1.<domain>;`) and add a
   `go_package` option pointing at the per-scenario gen path:

   ```protobuf
   option go_package = "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/<domain>;<domain>_v1";
   ```

2. **Regenerate.** From the repo root:

   ```bash
   cd packages/proto && make generate && make lint
   ```

   New artifacts land under the language-specific generated trees:
   `packages/proto/gen/go/ui-health/v1/<domain>/`,
   `packages/proto/gen/typescript/ui-health/v1/<domain>/`, and
   `packages/proto/gen/python/ui_health/v1/<domain>/`.
   Commit them alongside the schema — generated code is checked in so
   downstream scenarios don't have to re-run codegen.

3. **Wire it on the API side.** Import the generated Go type in your
   handler test and decode via `assertx.MustUnmarshalProto`:

   ```go
   import searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/search"

   got := assertx.MustUnmarshalProto[searchv1.SearchResponse](t, body)
   ```

   When tests need reusable response inputs, the template-only example
   at [the template fixture builder](/templates/scenarios/react-vite/api/internal/testutil/fixtures/health.go)
   demonstrates typed builders. Add a local builder only for actual callers:

   ```go
   type SearchResponse = searchv1.SearchResponse
   func NewSearchResponse(opts ...SearchOpt) *searchv1.SearchResponse { /* ... */ }
   ```

4. **Wire it on the UI side.** Import the generated TS schema and use
   `fromJson` for decode + `create` for fixtures:

   ```ts
   import { fromJson, create } from "@bufbuild/protobuf";
   import { SearchResponseSchema } from "@vrooli/proto-types/ui-health/v1/search/search_pb";

   // production
   return fromJson(SearchResponseSchema, json, { ignoreUnknownFields: true });

   // tests
   const fixture = create(SearchResponseSchema, { results: [/* … */] });
   ```

5. **Tests follow.** Connect handler tests call the generated client;
   fixture tests assert on the typed shape via `proto.Equal`. UI tests
   mock `api/<domain>` and return generated response objects from the
   factory.

Don't add a new `mocks/Fake*` interface for the proto type — the proto
isn't a seam, it's a contract. Seams are interfaces; protos are
payload shapes. See `SEAMS.md::Wire contracts live in proto, not seams`.
