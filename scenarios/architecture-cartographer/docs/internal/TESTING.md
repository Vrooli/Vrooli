# Testing — Architecture Cartographer

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

## Cartographer-specific test patterns

These patterns build on the template's canonical patterns above and
are non-negotiable for the cartographer's pluggable architecture.

### Signal tests are pure-function tests

Every `Signal` implementation has a unit test file under
`internal/signals/<name>/<name>_test.go` that covers:

1. **Reproducibility** — same `(chunk, domain, ctx)` inputs produce
   byte-identical `Score`. Run the signal three times in a row and
   compare.
2. **Bounded output** — `Score.Value` always in `[0.0, 1.0]`. Property
   test with random fixtures; refuse to merge if any out-of-bounds
   value can be produced.
3. **Evidence non-emptiness** — every `Score` carries a non-empty
   `Reason` and at least one `Evidence` entry. A signal that returns
   `Score{}` is broken by definition.
4. **Documented failure modes** — every failure mode listed in
   [`../concepts/SIGNAL_LADDER.md`](../concepts/SIGNAL_LADDER.md) for
   this signal must have a corresponding test case that exercises it
   and asserts the signal degrades gracefully (returns low score, not
   panic or error).
5. **No mutation** — assert the input graph snapshot is unchanged after
   scoring. Use a deep-equality check against a pre-scoring clone.

### Detector tests use deterministic fixture graphs

Every `Detector` implementation has a unit test file under
`internal/conflicts/detectors/<name>_test.go` that covers:

1. **Empty input** — no graph, no manifest → empty `[]Conflict`.
2. **Known positive cases** — fixture graph + manifest that should
   produce specific conflicts; assert ids, types, severities, and at
   minimum the locations field.
3. **Known negative cases** — graphs that look like they might trigger
   the detector but should not.
4. **Envelope stability** — the `Conflict` struct shape is checked by
   a golden-file test against `testdata/conflict-envelope.json`.

### Aggregator tests use fake signals

`internal/signals/aggregator_test.go` uses `mocks.FakeSignal` to feed
the aggregator deterministic scores and assert:

1. Weighted-sum math is correct (compare to hand-calculated expected
   verdict values).
2. Tier thresholds dispatch correctly (`auto_place` vs `suggest` vs
   `conflict`).
3. Tie-breaking — two candidate domains within `0.10` of each other
   correctly produce `conflict` rather than picking one arbitrarily.
4. Explainability — verdict output includes every signal's reason
   and evidence.

### Graph adapter tests use Connect-RPC fake servers

`internal/graph/gocodegraph/client_test.go` (and the TS counterpart)
use `connectxtest.StartTestServer` to spin up a fake `GoCodeGraphService`
implementation that returns canned `Graph` responses. The cartographer
client under test does not know it is talking to a fake. Coverage:

1. Successful extraction → normalized graph matches expected shape.
2. Transport failure → adapter retries per policy, then surfaces a
   typed `IntegrationError`.
3. URL re-resolution after `CodeUnavailable` — same pattern as
   ui-health's react-component-library client.

### BuildGuard tests substitute the shell

`internal/apply/buildguard_test.go` uses a `FakeBuildGuard` that
returns programmed `BuildStatus` values. Real `go build` invocation
is tested separately in an integration test that targets a tiny
fixture Go module under `bas/fixtures/go-build-baseline/`.

### Fixture scenarios live under `bas/fixtures/`

Cartographer's integration tests need known-bad scenarios to detect
against. Curated fixtures:

- `bas/fixtures/go-cycles/` — small Go module with a deliberate
  cross-package import cycle.
- `bas/fixtures/go-mislocated/` — Go module where one file is in the
  wrong package per the bundled manifest.
- `bas/fixtures/ts-junk-drawer/` — TS package with a `utils.ts`
  imported by everything, importing back.
- `bas/fixtures/medium-realistic/` — ~200-file mixed scenario
  exercising path-token, glossary, importer-voting, and test-coupling
  signals all at once.

Each fixture has a hand-curated `expected-graph.json` and
`expected-conflicts.json`. The integration test loads the fixture,
runs cartographer, and compares output to expected files byte-for-byte
(or with documented semantic-equality helpers when ordering is
non-deterministic).

### Override-recording tests guard the calibration loop

`internal/analytics/overrides_test.go` asserts that every recorded
auto-placement verdict can be paired with its corresponding override
event, and that the calibration query produces consistent signal-weight
adjustment suggestions for the same input data across runs. Override
tracking is the highest-value analytics signal — its correctness is
load-bearing for the entire signal ladder.
