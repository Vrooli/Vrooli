# Testing — Data Backup Manager

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
- [ui/src/layout/AppShell.a11y.test.tsx](../../ui/src/layout/AppShell.a11y.test.tsx)
- [cli/app_test.go](../../cli/app_test.go)

## TL;DR — the canonical examples

These files are the source of truth. When in doubt, copy their shape:

- **API**: `api/handlers/health/handler_test.go` — table-driven, real
  middleware via `httpx.NewLiveServer`, fake pinger from `mocks/`,
  typed-proto decode via
  `assertx.MustUnmarshalProto[healthv1.Response]` (the wire shape lives
  in `packages/proto/schemas/data-backup-manager/v1/health/health.proto`;
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

If your test doesn't look like one of those three, ask why before
shipping.

## API testing

### CRUD reference — `targets` end-to-end

The `targets` domain is the canonical catalog CRUD reference. New
domains add their first non-trivial mutation by copying its layering
one file at a time. The pattern from wire to render:

| Layer | File | What it owns |
|---|---|---|
| Wire contract | `packages/proto/schemas/data-backup-manager/v1/targets/targets.proto` | `Target`, `service TargetsService`, `ListTargetsResponse`, `RegisterTargetRequest`, `RegisterTargetResponse`, `GetTargetRequest`, `GetTargetResponse` |
| REST metadata contract | Not applicable today | Data Backup Manager has no multipart REST edge. Backup/restore bytes move through source capturers and `KopiaEngine`; REST remains only for `/health`. |
| Connect error mapping | `internal/targets/service_error_mapping.go` | Typed sentinels become Connect codes (`invalid_argument`, `not_found`, `internal`) |
| REST error envelope | `packages/proto/schemas/data-backup-manager/v1/errors/errors.proto` + `internal/httpx/errors.go::WriteError` | Typed body for REST exceptions, with canonical codes (`invalid_request`, `not_found`, `internal`) |
| Domain types | `internal/targets/types.go::{Target, RegisterInput, ErrInvalidTarget, ErrTargetNotFound}` | Domain-pure (no proto imports); typed sentinels translate into Connect errors at the handler edge |
| Repository interface | `internal/targets/repository.go::Repository` | Persistence seam — register/upsert, get, list, and deregister operations |
| Repository impl | `internal/targets/sqlite.go::NewSQLiteRepository` | sqlite-backed `Repository`; production wires it once in `main.go` |
| Schema | `internal/targets/schema.{sql,go}::Schema()` | Domain-owned table DDL embedded via `go:embed`; collected by `internal/modules/registry.go::AllSchemas()` and applied at boot via `database.EnsureSchemas` |
| Repository test | `internal/targets/sqlite_test.go` | Real handle via `db.NewSQLite(t)` + `database.EnsureSchemas(ctx, d, ...providers...)` over system + targets |
| Service | `internal/targets/service.go::Service` (+ `NewService`) | Application layer: validation, idempotent owner/name upsert, and deregistration. Handler depends on this, not the repository. |
| Service test | `internal/targets/service_test.go` | Substitutes `mocks.FakeRepository` (from co-located `internal/targets/mocks/`); pins validation, idempotency, and error propagation |
| Connect handler test | `handlers/targets/connect_handler_test.go` | Substitutes `mocks.FakeService` and exercises the generated Connect client/handler path |
| Mocks | `internal/targets/mocks/{repository,service}.go::{FakeRepository,FakeService}` | Co-located with the domain; deleting `internal/targets/` takes them along. |
| UI client | `ui/src/api/targets.ts` | `targetsClient = createClient(TargetsService, transport)` |
| UI tests | Feature/component tests | Mock generated client methods; REST helper tests stub `global.fetch` only for REST exceptions. |
| CLI client | `cli/domains/targets/{register,handlers}.go` | `Register(core, manifest)` returns a manifest-backed `cliapp.SubcommandGroup`; handlers use generated Connect clients and render through cli-core reports |
| CLI test | `cli/domains/targets/handlers_test.go` | Spins a real `httptest.Server` via `clitest.NewAPIServer`, captures stdout via `clitest.CaptureStdout` |

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
        apidb.SchemaProviderFunc(targets.Schema),
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

The targets domain uses three test layers, each with a different fake:

```
HTTP → handler → Service (validates, applies defaults) → Repository (persists)
                     ↑                                       ↑
                     FakeService (handler tests)              FakeRepository (service tests)
                                                              Real sqlite (repository tests)
```

`internal/targets/service_test.go` is the reference. Service tests:

- Substitute `mocks.FakeRepository` (in-memory state) so the test can
  assert on what the repository was called with and whether the service
  filtered the call (e.g., empty owner/name/locator rejected before reaching persistence).
- Pin validation contracts (`Register` rejects missing fields with
  `ErrInvalidTarget{Field: ...}`).
- Pin idempotency contracts (`Register` updates the owner/name row instead
  of creating duplicates).
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

Data Backup Manager currently has no formal flow artifacts. When one is
added, the reference Level 5 layout is:

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

`make temporal-models` invokes `flow-verifier verify check --root "$(CURDIR)"`, which
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
domain handler tests for internal-error paths
checks the underlying error reaches operator logs).

## UI testing

### Mock builders for `api/health` and domain clients

`vi.mock(path, factory)` is hoisted before any user import resolves;
a wrapper imported from `test-utils` would be in the temporal dead
zone at hoist time. The escape hatch is to keep the `vi.mock` call
inline at the top of each test file, but move the *factory body* into
a builder function that runs when the closure executes — which is
*after* imports initialise.

`@/test-utils` exports shared, cross-feature mock builders such as
`makeApiMocks()`. Feature-specific builders live beside the feature so
deleting the feature takes its mocks with it.

Canonical shape:

```tsx
import { makeApiMocks } from "@/test-utils";
vi.mock("../../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/health")>();
  return { ...actual, ...makeApiMocks() };
});
```

Defaults are picked so the most common test paths work no-args:
`makeApiMocks().fetchHealth` resolves to a healthy response. Domain
client mocks should expose the generated client methods that the
feature actually calls.
Per-test overrides use vitest's standard pattern *after* the mock is wired:

```tsx
const { targetsClient } = await import("../../api/targets");
vi.mocked(targetsClient.listTargets).mockResolvedValueOnce(
  makeListTargetsResponse({ targets: [makeTarget({ id: "t-1" })] }),
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
under `packages/proto/schemas/data-backup-manager/`.

Steps:

1. **Author the schema.** In a generated scenario, add
   `packages/proto/schemas/data-backup-manager/v1/<domain>/<name>.proto`.
   Use snake_case in the proto package directive
   (`package vrooli.data_backup_manager.v1.<domain>;`) and add a
   `go_package` option pointing at the per-scenario gen path:

   ```protobuf
   option go_package = "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/<domain>;<domain>_v1";
   ```

2. **Regenerate.** From the repo root:

   ```bash
   cd packages/proto && make generate && make lint
   ```

   New artifacts land under the language-specific generated trees:
   `packages/proto/gen/go/data-backup-manager/v1/<domain>/`,
   `packages/proto/gen/typescript/data-backup-manager/v1/<domain>/`, and
   `packages/proto/gen/python/data_backup_manager/v1/<domain>/`.
   Commit them alongside the schema — generated code is checked in so
   downstream scenarios don't have to re-run codegen.

3. **Wire it on the API side.** Import the generated Go type in your
   handler test and decode via `assertx.MustUnmarshalProto`:

   ```go
   import targetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/targets"

   got := assertx.MustUnmarshalProto[targetsv1.ListTargetsResponse](t, body)
   ```

   When tests need reusable response inputs, the template-only example
   at [the template fixture builder](/templates/scenarios/react-vite/api/internal/testutil/fixtures/health.go)
   demonstrates typed builders. Add a local builder only for actual callers:

   ```go
   type ListTargetsResponse = targetsv1.ListTargetsResponse
   func NewListTargetsResponse(opts ...ListTargetsOpt) *targetsv1.ListTargetsResponse { /* ... */ }
   ```

4. **Wire it on the UI side.** Import the generated TS schema and use
   `fromJson` for decode + `create` for fixtures:

   ```ts
   import { fromJson, create } from "@bufbuild/protobuf";
   import { ListTargetsResponseSchema } from "@vrooli/proto-types/data-backup-manager/v1/targets/targets_pb";

   // production
   return fromJson(ListTargetsResponseSchema, json, { ignoreUnknownFields: true });

   // tests
   const fixture = create(ListTargetsResponseSchema, { targets: [{ id: "t-1" }] });
   ```

5. **Tests follow.** Connect handler tests call the generated client;
   fixture tests assert on the typed shape via `proto.Equal`. UI tests
   mock the relevant `api/<domain>` module and return generated response objects from the
   factory.

Don't add a new `mocks/Fake*` interface for the proto type — the proto
isn't a seam, it's a contract. Seams are interfaces; protos are
payload shapes. See `SEAMS.md::Wire contracts live in proto, not seams`.

## Workspace recovery validation

Default tests use fake engines/commands and synthetic files. The production
command adapters reject Go test execution. Test-mode HTTP requests cannot reach
production DBM effects; compose a dedicated fixture server with fake engines,
source adapters, catalog, clock, and executor. A temporary catalog alone does
not isolate backup destinations or credential state.

`./scripts/prove-backup-restore.sh` is a fixture-only proof. Do not restore the
old live-service E2E script or enable real Kopia through an environment flag.

Focused regressions cover original index/working bytes, executable modes,
empty directories, link targets, source drift, corrupt/extra manifest content,
existing/symlink destinations, cancellation, engine isolation, temporary
storage refusal, and final materialization failure. Real backend byte transport
and hardware disaster recovery are not certified by mocked tests. A separate
operator-authorized engine qualification is required to make those claims.

Recovery journals also test the ambiguous-crash seam: a step is persisted as
`in_flight` before the host action. If the process stops before the result is
durably recorded, resume pauses for inspection instead of automatically
repeating a potentially destructive repair. This case is exercised entirely
with a fake remediator and in-memory journal.
