# Testing — Plan Manager

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
   `packages/proto/schemas/plan-manager/v1/<domain>/<file>.proto`.
   Tests import the generated Go type directly
   (`healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"`)
   and decode wire bodies into it via `MustUnmarshalProto`. The
   `fixtures` package re-exports the proto type as a short alias
   (`fixtures.HealthResponse = healthv1.Response`) so test code reads
   cleanly.

### Workflow Tests

The workflow inventory lives in [`FLOWS.md`](../concepts/FLOWS.md). This scenario
currently proves those flows through service, handler, CLI, and integration
tests for the implemented state transitions: authoring section progression,
finalize-to-plan, execution phase advancement, handoff assembly, validation, and
migration/import. There are no generated formal flow artifacts in
`api/internal/*/flow/`, so there is no `make temporal-models` gate for
plan-manager.

If a future plan adds a formal flow model, add the model, replay tests, and the
verification target in the same change. Do not leave a verifier target or docs
claim in place without checked-in artifacts for it to validate.

## How to add a new proto

Wire shapes for new endpoints belong in proto, not in hand-written Go
structs or TS interfaces. After generation, the canonical source lives
under `packages/proto/schemas/plan-manager/`.

Steps:

1. **Author the schema.** In a generated scenario, add
   `packages/proto/schemas/plan-manager/v1/<domain>/<name>.proto`.
   Use snake_case in the proto package directive
   (`package vrooli.plan_manager.v1.<domain>;`) and add a
   `go_package` option pointing at the per-scenario gen path:

   ```protobuf
   option go_package = "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/<domain>;<domain>_v1";
   ```

2. **Regenerate.** From the repo root:

   ```bash
   cd packages/proto && make generate && make lint
   ```

   New artifacts land under the language-specific generated trees:
   `packages/proto/gen/go/plan-manager/v1/<domain>/`,
   `packages/proto/gen/typescript/plan-manager/v1/<domain>/`, and
   `packages/proto/gen/python/plan_manager/v1/<domain>/`.
   Commit them alongside the schema — generated code is checked in so
   downstream scenarios don't have to re-run codegen.

3. **Wire it on the API side.** Import the generated Go type in your
   handler test and decode via `assertx.MustUnmarshalProto`:

   ```go
   import domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/<domain>"

   got := assertx.MustUnmarshalProto[domainv1.ListResponse](t, body)
   ```

   For fixtures, follow the `fixtures/health.go` pattern — re-export
   the proto type as a short alias and provide functional-options
   builders:

   ```go
   type ListResponse = domainv1.ListResponse
   func NewListResponse(opts ...ListOpt) *domainv1.ListResponse { /* ... */ }
   ```

4. **Wire it on the UI side.** Import the generated TS schema and use
   `fromJson` for decode + `create` for fixtures:

   ```ts
   import { fromJson, create } from "@bufbuild/protobuf";
   import { ListResponseSchema } from "@vrooli/proto-types/plan-manager/v1/<domain>/<domain>_pb";

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
