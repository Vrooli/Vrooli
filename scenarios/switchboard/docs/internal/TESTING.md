# Testing — Switchboard

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

## How to add a new proto

Wire shapes for new endpoints belong in proto, not in hand-written Go
structs or TS interfaces. After generation, the canonical source lives
under `packages/proto/schemas/switchboard/`.

Steps:

1. **Author the schema.** In a generated scenario, add
   `packages/proto/schemas/switchboard/v1/<domain>/<name>.proto`.
   Use snake_case in the proto package directive
   (`package vrooli.switchboard.v1.<domain>;`) and add a
   `go_package` option pointing at the per-scenario gen path:

   ```protobuf
   option go_package = "github.com/vrooli/vrooli/packages/proto/gen/go/switchboard/v1/<domain>;<domain>_v1";
   ```

2. **Regenerate.** From the repo root:

   ```bash
   cd packages/proto && make generate && make lint
   ```

   New artifacts land under the language-specific generated trees:
   `packages/proto/gen/go/switchboard/v1/<domain>/`,
   `packages/proto/gen/typescript/switchboard/v1/<domain>/`, and
   `packages/proto/gen/python/switchboard/v1/<domain>/`.
   Commit them alongside the schema — generated code is checked in so
   downstream scenarios don't have to re-run codegen.

3. **Wire it on the API side.** Import the generated Go type in your
   handler test and decode via `assertx.MustUnmarshalProto`:

   ```go
   import domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/switchboard/v1/<domain>"

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
   import { ListResponseSchema } from "@vrooli/proto-types/switchboard/v1/<domain>/<domain>_pb";

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
