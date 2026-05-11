# Temporal Model Artifacts

This directory owns the template-local formal temporal-model validation loop.

`*.flow.json` files are the source of truth. Generated `.qnt`,
`*.formal.generated.json`, `*.generated.{go,ts}`, and generated formal replay
test/helper files are checked-in review artifacts and must not be edited by
hand.

Run:

```bash
cd tools/temporal-model && GOWORK=off go run . list --root ../..
cd tools/temporal-model && GOWORK=off go run . validate --root ../..
cd tools/temporal-model && GOWORK=off go run . generate --root ../.. --flow notes.attachment-upload.ui
cd tools/temporal-model && GOWORK=off go run . generate --root ../..
cd tools/temporal-model && GOWORK=off go run . check --root ../..
```

The generator runs `quint typecheck`, `quint test`, `quint verify`, and
`quint run --mbt` for each discovered temporal flow contract. It renders Quint
from the contract, normalizes ITF traces, records source/model/generator hashes,
and writes checked-in artifacts beside the workflow they validate. Schema v4
contracts also generate Go/TypeScript declarations for state/event topology,
pure status-transition helpers, and formal freshness expectations, so
production wrappers and replay tests do not maintain duplicate state/event
lists, transition tables, or hash constants by hand.

`validate`, `generate`, and `check` validate each `*.flow.json` against
`flow.schema.json`, then compile the raw JSON into an immutable flow model with
indexes and a first-class transition matrix. Structural mistakes such as
unknown fields, missing required sections, invalid replay kinds, duplicate or
missing transition pairs, trace drift, or stale generated replay outputs fail
before Quint runs.

Formal artifacts use schema v4. Their `coverage` block distinguishes:

- `transitionMatrixComplete`: the generated matrix contains every
  state/event pair.
- `terminalTransitionsChecked`: terminal states still have explicit
  transition rows.
- `namedTraces`: hand-authored trace state/event coverage.
- `generatedTraces`: MBT trace state/event/pair coverage. This can be
  false for `allPairsCovered` without invalidating the artifact; it is
  trace exploration metadata, not a proxy for matrix completeness.

`quint verify` only receives real invariants from `model.verify.invariants`.
The generated full transition table is still executed through
`run transitionTable`, but the JSON artifact records it under
`generatedChecks` instead of pretending it is a verified invariant.

Every Level 5 contract must declare generated replay outputs. Go flows generate
a formal replay `_test.go`; TypeScript flows generate a formal replay helper and
a Vitest file that imports the hand-authored fixture module declared by the
contract. `check` compares all generated files byte-for-byte, so a flow cannot
silently lose freshness, transition-matrix replay, or trace replay. The old
marker-based `replay.bindings` pattern is not supported.

Use `explain` as the authoring entry point before or after changing a flow:

```bash
cd tools/temporal-model && GOWORK=off go run . explain --root ../.. --flow notes.attachment-upload.ui
```

It reports generated files, runtime language/types, whether TypeScript runtime
unions and fixture contracts are generated, topology counts, generated replay
test/helper paths, fixture module expectations, named-trace coverage, exact
regenerate/check commands, and the hand-authored files that usually need
payload-specific follow-up work.

Normal unit tests do not require Quint or Java:

```bash
cd tools/temporal-model && GOWORK=off go test ./...
```

To update a flow:

1. Edit the domain-local `*.flow.json` contract.
2. Run `cd tools/temporal-model && GOWORK=off go run . generate --root ../.. --flow <flow-id>`.
3. Update only payload-specific production wrapper logic if the new
   state/event needs richer runtime data. Abstract status validity and
   next-status outcomes come from generated helpers.
4. Update the UI replay fixture module when a new status/event needs a runtime
   fixture. Use the generated formal replay fixture interface so missing
   runtime fixtures fail type-checking against the generated state/event lists.
5. Review the regenerated `.qnt`, `.formal.generated.json`, declarations, and
   formal replay files.
6. Run `make temporal-models`.
7. Run the scenario tests on an instantiated scenario.
