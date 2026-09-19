# Testing — Scenario Completeness Scoring

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
  in `packages/proto/schemas/scenario-completeness-scoring/v1/health/health.proto`;
  assert on typed proto fields, not `map[string]any` chains). For
  endpoints whose wire shape isn't in proto yet, `MustDecodeJSON[T]`
  is the fallback — but adding the proto first is the right move.
- **UI composition**: `ui/src/App.test.tsx` — smoke-only composition
  test. App composes shell + features; feature behaviour belongs beside
  the feature.
- **UI feature**: `ui/src/features/health/HealthCard.test.tsx` —
  `renderWithProviders`, factory data, inline `vi.mock` factory
  closure, cimode assertions, and real-locale assertions.
- **UI a11y**: `ui/src/layout/AppShell.a11y.test.tsx`,
  `ui/src/features/health/HealthCard.a11y.test.tsx`, and
  `ui/src/features/scoring/ScoreDashboard.a11y.test.tsx` — shell and
  feature accessibility are tested at their ownership boundary.
- **CLI**: `cli/app_test.go` — smoke gate (NewApp, --version, --help).
  When domain commands arrive, extend with `clitest.NewAPIServer` +
  `clitest.CaptureStdout` from `cli/internal/testutil/`.

If your test doesn't look like one of those three, ask why before
shipping.

## API testing

### Vertical reference — `scoring` end-to-end

The `scoring` domain is the canonical vertical-slice reference. New
domains copy its layering one file at a time. The pattern from wire to
render:

| Layer | File | What it owns |
|---|---|---|
| Wire contract | `packages/proto/schemas/scenario-completeness-scoring/v1/scoring/scoring.proto` | `service ScoreService`, `GetScoreRequest/Response`, and every payload sub-message (maturity, composite, freshness, recommendations, degradations) |
| REST error envelope | `packages/proto/schemas/scenario-completeness-scoring/v1/errors/errors.proto` + `internal/httpx/errors.go::WriteError` | Typed body for any REST exception, with canonical codes (`invalid_request`, `not_found`, `internal`) |
| Domain types | `internal/scoring/service.go::{Result, ErrUnknownScenario}` (+ `internal/signals`, `internal/freshness` sub-results) | Domain-pure (no proto imports); the typed sentinel translates into a Connect error at the handler edge |
| Signal collectors | `internal/signals/` (`Collector` interface, circuit-breaker registry) | Per-source artifact decoding; failures degrade, never crash |
| Freshness | `internal/freshness/` (thin over `packages/freshness-go`) | Digest + per-phase verdicts; semantics owned by the shared package |
| Score assembly | `internal/scoring/` (ladder rung, composite, recommendations, action plan) | Score math and classification bands |
| Service tests | `internal/scoring/service_test.go` (incl. `TestGetScoreStalenessLoop`), per-collector tests in `internal/signals/` | Deterministic fixture scenario trees under `t.TempDir()` via `WithScenariosRoot` |
| Connect handler | `handlers/scoring/connect_handler.go` (+ `module.go`) | Domain→proto conversion and error mapping (`ErrUnknownScenario` → `NOT_FOUND`) behind the `Scorer` seam |
| Connect handler test | `handlers/scoring/connect_handler_test.go` | Substitutes the in-file `stubScorer` and exercises the generated Connect client/handler path |
| UI client | `ui/src/api/scoring.ts` | `scoringClient = createClient(ScoreService, transport)` + `fetchScore(scenario)` |
| UI feature tests | `ui/src/features/scoring/ScoreDashboard{,.a11y}.test.tsx` + `ui/src/api/scoring.test.ts` | Mock the API module via `makeScoringMocks()`; factories build generated proto types |
| CLI client | `cli/domains/scores/{register,handlers,format}.go` | `Register(core, manifest)` returns a `cliapp.SubcommandGroup`; the handler calls the generated Connect client and renders the report |
| CLI test | `cli/domains/scores/handlers_test.go` + `format_test.go` | Spins a real `httptest.Server` via `testutil.NewAPIServer`, captures stdout via `testutil.CaptureStdout`; golden-output assertions on the report |

#### Compose pattern: schema-applied repository test

The scoring domain persists nothing, but the pattern for the first
persistence-owning domain is pinned by the testutil helper docs:
`db.NewSQLite(t)` returns a blank handle, and repository tests apply
the production schema before the first query so the test exercises the
same shape `main.go` ships:

```go
func newSchemaDB(t *testing.T) *sql.DB {
    t.Helper()
    d := db.NewSQLite(t)
    require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
        apidb.SchemaProviderFunc(localdb.SystemSchema),
        apidb.SchemaProviderFunc(mydomain.Schema),
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

A persistence-owning domain uses three test layers, each with a
different fake:

```
HTTP → handler → Service (validates, applies defaults) → Repository (persists)
                     ↑                                       ↑
                     FakeService (handler tests)              FakeRepository (service tests)
                                                              Real sqlite (repository tests)
```

The scoring domain follows the same split without persistence: handler
tests substitute the `Scorer` seam (`stubScorer`); service tests drive
the real assembly against fixture scenario trees; collector tests pin
each artifact decoder in isolation. Whatever the domain, the rule is
identical — each layer's tests substitute the seam *below* them and
own only that layer's contracts.

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

No domain in this scenario owns lifecycle state today, so there are no
checked-in flow models. When one arrives, the Level 5 file shape per
flow is (scaffold with `flow-verifier flows new`):

- The `flow-verifier` scenario CLI (`flow-verifier verify check|run`, `flows list|validate|explain`)
- `api/internal/<domain>/flow/flow.json`
- `api/internal/<domain>/flow/transition.go` (package `flow`)
- `api/internal/<domain>/flow/flow_test.go` (thin replay delegation, package `flow`)
- `api/internal/<domain>/flow/generated/{model.qnt,artifact.json,runtime.go,replay.go}` (package `generated`)
- `ui/src/features/<feature>/flow/flow.json`
- `ui/src/features/<feature>/flow/transition.ts`
- `ui/src/features/<feature>/flow/fixtures.ts`
- `ui/src/features/<feature>/flow/flow.test.ts` (thin replay delegation)
- `ui/src/features/<feature>/flow/generated/{model.qnt,artifact.json,runtime.ts,replay.helper.ts}`

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
client := newDomainClient(t, fakeService, logger)
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
asserts that an underlying error reaches operator logs on a 500 path.

## UI testing

### Mock builders for `api/health` and `api/scoring`

`vi.mock(path, factory)` is hoisted before any user import resolves;
a wrapper imported from `test-utils` would be in the temporal dead
zone at hoist time. The escape hatch is to keep the `vi.mock` call
inline at the top of each test file, but move the *factory body* into
a builder function that runs when the closure executes — which is
*after* imports initialise.

`@/test-utils` exports shared, cross-feature mock builders such as
`makeApiMocks()`. Feature-specific builders live beside the feature so
deleting the feature takes its mocks with it; for scoring, import
`makeScoringMocks()` from `features/scoring/mocks/scoring`.

Canonical shape:

```tsx
import { makeApiMocks } from "@/test-utils";
import { makeScoringMocks } from "./mocks/scoring";

vi.mock("../../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/health")>();
  return { ...actual, ...makeApiMocks() };
});

vi.mock("../../api/scoring", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/scoring")>();
  return { ...actual, ...makeScoringMocks() };
});
```

Defaults are picked so the most common test paths work no-args:
`makeApiMocks().fetchHealth` resolves to a healthy response;
`makeScoringMocks().fetchScore` resolves to the default
`makeGetScoreResponse()` payload for any scenario name.
Per-test overrides use vitest's standard pattern *after* the mock is wired:

```tsx
const { fetchScore } = await import("../../api/scoring");
vi.mocked(fetchScore).mockResolvedValueOnce(
  makeGetScoreResponse({ scenario: "cli-health" }),
);
```

The `...actual` spread keeps non-mocked exports (the `ApiError` class,
re-exported proto types) intact — only network-touching functions are
substituted.

When a third lib/* surface lands (e.g., `lib/users.ts`), follow the
same pattern: builder in `ui/src/test-utils/mocks/<surface>.ts`, self-
test alongside, re-export from `test-utils/index.ts`.

## How to add a new proto

Wire shapes for new endpoints belong in proto, not in hand-written Go
structs or TS interfaces. After generation, the canonical source lives
under `packages/proto/schemas/scenario-completeness-scoring/`.

Steps:

1. **Author the schema.** In a generated scenario, add
   `packages/proto/schemas/scenario-completeness-scoring/v1/<domain>/<name>.proto`.
   Use snake_case in the proto package directive
   (`package vrooli.scenario_completeness_scoring.v1.<domain>;`) and add a
   `go_package` option pointing at the per-scenario gen path:

   ```protobuf
   option go_package = "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/<domain>;<domain>_v1";
   ```

2. **Regenerate.** From the repo root:

   ```bash
   cd packages/proto && make generate && make lint
   ```

   New artifacts land under the language-specific generated trees:
   `packages/proto/gen/go/scenario-completeness-scoring/v1/<domain>/`,
   `packages/proto/gen/typescript/scenario-completeness-scoring/v1/<domain>/`, and
   `packages/proto/gen/python/scenario_completeness_scoring/v1/<domain>/`.
   Commit them alongside the schema — generated code is checked in so
   downstream scenarios don't have to re-run codegen.

3. **Wire it on the API side.** Import the generated Go type in your
   handler test and decode via `assertx.MustUnmarshalProto`:

   ```go
   import scoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring"

   got := assertx.MustUnmarshalProto[scoringv1.GetScoreResponse](t, body)
   ```

   When tests need reusable response inputs, the template-only example
   at [the template fixture builder](/templates/scenarios/react-vite/api/internal/testutil/fixtures/health.go)
   demonstrates typed builders. Add a local builder only for actual callers:

   ```go
   type GetScoreResponse = scoringv1.GetScoreResponse
   func NewGetScoreResponse(opts ...ScoreOpt) *scoringv1.GetScoreResponse { /* ... */ }
   ```

4. **Wire it on the UI side.** Import the generated TS schema and use
   `fromJson` for decode + `create` for fixtures:

   ```ts
   import { fromJson, create } from "@bufbuild/protobuf";
   import { GetScoreResponseSchema } from "@vrooli/proto-types/scenario-completeness-scoring/v1/scoring/scoring_pb";

   // production
   return fromJson(GetScoreResponseSchema, json, { ignoreUnknownFields: true });

   // tests
   const fixture = create(GetScoreResponseSchema, { scenario: "web-search" });
   ```

5. **Tests follow.** Connect handler tests call the generated client;
   fixture tests assert on the typed shape via `proto.Equal`. UI tests
   mock `api/scoring` and return generated response objects from the
   factory.

Don't add a new `mocks/Fake*` interface for the proto type — the proto
isn't a seam, it's a contract. Seams are interfaces; protos are
payload shapes. See `SEAMS.md::Wire contracts live in proto, not seams`.
