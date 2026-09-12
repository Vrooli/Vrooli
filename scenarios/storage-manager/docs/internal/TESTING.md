# Testing — Storage Manager

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

The scaffold ships one fenced worked CRUD domain (never product scope)
as the canonical reference. New scenarios add their first non-trivial
mutation by copying its layering one file at a time — wire contract,
domain types, schema, service, handler, mocks, UI client, and CLI
client. Storage Manager's current cleanup domain is provider/orchestrator
driven rather than repository driven, so tests substitute provider seams
and the CleanupService interface instead of a database repository.

### Cleanup domain test shape

The cleanup domain tests the same wire-to-render layering without a CRUD
repository:

| Layer | File | What it owns |
|---|---|---|
| Wire contract | `packages/proto/schemas/storage-manager/v1/cleanup/cleanup.proto` | CleanupService provider, policy, plan, apply, and audit messages |
| Connect error mapping | `api/handlers/cleanup/connect_handler.go` | Orchestrator errors become Connect codes (`invalid_argument`, `failed_precondition`, `permission_denied`, `internal`) |
| REST error envelope | `packages/proto/schemas/storage-manager/v1/shared/errors.proto` + `internal/httpx/errors.go::WriteError` | Typed body for REST exceptions, with canonical codes (`invalid_request`, `not_found`, `internal`) |
| Domain types | `api/internal/cleanup/types.go` | Provider metadata, plans, policy, audit, and validation rules |
| Provider seams | `api/internal/cleanup/seams.go` | Filesystem, process, Docker, journal, clock, and owner-scenario boundaries |
| Provider tests | `api/internal/providers/*_test.go` | Preview-first behavior against fake side-effect clients |
| Orchestrator service | `api/internal/orchestrator/service.go` | Policy/profile validation, deterministic plan ids, approvals, idempotency replay, and audit |
| Orchestrator tests | `api/internal/orchestrator/service_test.go` | Substitutes fake providers and clocks to prove safety gates |
| Connect handler test | `api/handlers/cleanup/connect_handler_test.go` | Substitutes fake CleanupService and exercises generated Connect handler paths |
| CLI client | `cli/domains/cleanup/{register,handlers}.go` | Cleanup command group and generated Connect calls |
| CLI tests | `cli/domains/cleanup/*_test.go` | Command registration and handler behavior |
| UI tests | `ui/src/pages/DashboardPage.test.tsx` and shell tests | Cleanup console rendering, accessibility, selectors, and navigation |

#### Service-layer tests

The cleanup domain uses layered fakes:

```
HTTP → handler → CleanupService interface → Orchestrator → Providers → typed seams
                     ↑                         ↑              ↑
                     fake service              fake clock      fake clients
```

Handler tests substitute the CleanupService interface. Orchestrator and
provider tests substitute the lower-level seams so no test performs real
filesystem, Docker, journal, process, or scenario cleanup.

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

Cleanup handler tests use the same pattern for Connect error paths in
`api/handlers/cleanup/connect_handler_test.go`.

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
(`vi.mocked(client.method).mockResolvedValueOnce(...)`).

```tsx
import { makeApiMocks } from "@/test-utils";
vi.mock("../../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/health")>();
  return { ...actual, ...makeApiMocks() };
});
```

When a third lib/* surface lands (e.g., `lib/users.ts`), follow the
same pattern: builder in `ui/src/test-utils/mocks/<surface>.ts`, self-
test alongside, re-export from `test-utils/index.ts`.

## How to add a new proto

Wire shapes for new endpoints belong in proto, not in hand-written Go
structs or TS interfaces. After generation, the canonical source lives
under `packages/proto/schemas/storage-manager/`.

Steps:

1. **Author the schema.** In a generated scenario, add
   `packages/proto/schemas/storage-manager/v1/<domain>/<name>.proto`.
   Use snake_case in the proto package directive
   (`package vrooli.cleanup_manager.v1.<domain>;`) and add a
   `go_package` option pointing at the per-scenario gen path:

   ```protobuf
   option go_package = "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/<domain>;<domain>_v1";
   ```

2. **Regenerate.** From the repo root:

   ```bash
   cd packages/proto && make generate && make lint
   ```

   New artifacts land under the language-specific generated trees:
   `packages/proto/gen/go/storage-manager/v1/<domain>/`,
   `packages/proto/gen/typescript/storage-manager/v1/<domain>/`, and
   `packages/proto/gen/python/cleanup_manager/v1/<domain>/`.
   Commit them alongside the schema — generated code is checked in so
   downstream scenarios don't have to re-run codegen.

3. **Wire it on the API side.** Import the generated Go type in your
   handler test and decode via `assertx.MustUnmarshalProto`:

   ```go
   import domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/<domain>"

   got := assertx.MustUnmarshalProto[domainv1.ListResponse](t, body)
   ```

   When tests need reusable response inputs, the template-only example
   at [the template fixture builder](/templates/scenarios/react-vite/api/internal/testutil/fixtures/health.go)
   demonstrates typed builders. Add a local builder only for actual callers:

   ```go
   type ListResponse = domainv1.ListResponse
   func NewListResponse(opts ...ListOpt) *domainv1.ListResponse { /* ... */ }
   ```

4. **Wire it on the UI side.** Import the generated TS schema and use
   `fromJson` for decode + `create` for fixtures:

   ```ts
   import { fromJson, create } from "@bufbuild/protobuf";
   import { ListResponseSchema } from "@vrooli/proto-types/storage-manager/v1/<domain>/<domain>_pb";

   // production
   return fromJson(ListResponseSchema, json, { ignoreUnknownFields: true });

   // tests
   const fixture = create(ListResponseSchema, { items: [{ id: "n-1" }] });
   ```

5. **Tests follow.** Connect handler tests call the generated client;
   fixture tests assert on the typed shape via `proto.Equal`. UI tests
   mock `api/<domain>` and return generated response objects from the
   factory.

Don't add a new `mocks/Fake*` interface for the proto type — the proto
isn't a seam, it's a contract. Seams are interfaces; protos are
payload shapes. See `SEAMS.md::Wire contracts live in proto, not seams`.
