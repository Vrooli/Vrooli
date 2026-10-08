# Testing — Portal

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

### Domain vertical slices

Portal domain tests follow the same wire-to-render layering as production:
proto contract, domain service/repository, handler module, CLI handler, UI API
client, and UI feature. Add focused tests at the layer where a behavior lives;
avoid asserting API business rules from the CLI or UI.

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

Use this generic file layout when Portal adds a formal workflow contract:

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

## How to add a new proto

Wire shapes for new endpoints belong in proto, not in hand-written Go
structs or TS interfaces. After generation, the canonical source lives
under `packages/proto/schemas/portal/`.

Steps:

1. **Author the schema.** In a generated scenario, add
   `packages/proto/schemas/portal/v1/<domain>/<name>.proto`.
   Use snake_case in the proto package directive
   (`package vrooli.portal.v1.<domain>;`) and add a
   `go_package` option pointing at the per-scenario gen path:

   ```protobuf
   option go_package = "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/<domain>;<domain>_v1";
   ```

2. **Regenerate.** From the repo root:

   ```bash
   cd packages/proto && make generate && make lint
   ```

   New artifacts land under the language-specific generated trees:
   `packages/proto/gen/go/portal/v1/<domain>/`,
   `packages/proto/gen/typescript/portal/v1/<domain>/`, and
   `packages/proto/gen/python/portal/v1/<domain>/`.
   Commit them alongside the schema — generated code is checked in so
   downstream scenarios don't have to re-run codegen.

3. **Wire it on the API side.** Import the generated Go type in your
   handler test and decode via `assertx.MustUnmarshalProto`:

   ```go
   import domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/<domain>"

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
   import { ListResponseSchema } from "@vrooli/proto-types/portal/v1/<domain>/<domain>_pb";

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
