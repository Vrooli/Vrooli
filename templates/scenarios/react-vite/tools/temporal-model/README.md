# Temporal Model Artifacts

This directory owns the template-local formal temporal-model validation loop.

## Canonical layout (schema v5)

Every flow lands as:

```
<feature-dir>/
├── <Name>.flow.json            # hand: source of truth
├── <wrapper>.{go,ts}           # hand: payload/runtime logic
├── <Name>.fixtures.ts          # hand: replay fixtures (TS only)
├── <test>_test.go OR <Name>.test.ts   # hand: thin replay delegation
└── generated/<foldername>/
    ├── model.qnt
    ├── artifact.json
    ├── runtime.{go,ts}
    └── replay.{go,helper.ts}
```

The folder name under `generated/` is derived mechanically from the flow ID
(middle dotted segments, lowercased, dashes stripped). The contract no longer
declares any output paths — there is exactly one convention.

`*.flow.json` files are the source of truth. Everything under any `generated/`
directory is codegen output and must not be hand-edited.

## Run

```bash
cd tools/temporal-model && GOWORK=off go run . list --root ../..
cd tools/temporal-model && GOWORK=off go run . validate --root ../..
cd tools/temporal-model && GOWORK=off go run . generate --root ../.. --flow notes.attachment-upload.ui
cd tools/temporal-model && GOWORK=off go run . generate --root ../..
cd tools/temporal-model && GOWORK=off go run . check --root ../..
cd tools/temporal-model && GOWORK=off go run . explain --root ../.. --flow notes.attachment-upload.ui
```

The generator runs `quint typecheck`, `quint test`, `quint verify`, and
`quint run --mbt` for each discovered temporal flow contract. It renders the
canonical Quint model, normalizes ITF traces, records source/model/generator
hashes, and writes every artifact under `generated/<foldername>/`. Go flows
get a `runtime.go` + `replay.go` subpackage that exports `RunReplay`;
TypeScript flows get a `runtime.ts` + `replay.helper.ts` module that exports
`runFormalReplay`. The hand-authored test at the wrapper-directory top
delegates to that single entry point.

## Lint pass

`check` runs two stages: byte-for-byte freshness of generated files, and an
AST-level lint of the hand-authored replay test.

- Go test files are parsed with `go/ast`. The lint rejects: missing import of
  the generated subpackage, missing call to `<subpkg>.RunReplay`, `nil`
  transition argument, function-literal transition with an empty body, and
  blank-assignment-only bodies.
- TypeScript test files are parsed with a structural source reader. The lint
  rejects: missing import of `./generated/<folder>/replay.helper`, missing
  binding of the wrapper's transition function, missing binding of the
  fixtures export, and any `runFormalReplay(...)` call that is not at module
  top level (calls nested inside `if`, `try`, function bodies, etc. fail).

There is no `--no-lint` flag. Lint failures are fatal.

> **TS lint mechanism note.** The plan recommends shelling out to Node's
> TypeScript Compiler API for the TS lint. The current implementation is a
> pure-Go structural scanner that achieves the same invariants without the
> Node dependency. It is intentionally strict: every required binding is
> verified against the import section, and `hasTopLevelCall` walks the
> source at brace depth zero. If a future change requires field-level type
> inspection beyond what the scanner covers, swap in a Node helper —
> `tools/temporal-model/internal/lint/tslint.go` is the single seam.

## Formal artifacts

Use schema v5. The `coverage` block distinguishes:

- `transitionMatrixComplete`: the generated matrix contains every
  state/event pair.
- `terminalTransitionsChecked`: terminal states still have explicit
  transition rows.
- `namedTraces`: hand-authored trace state/event coverage.
- `generatedTraces`: MBT trace state/event/pair coverage.

`quint verify` only receives real invariants from `model.verify.invariants`.
The generated full transition table is still executed through
`run transitionTable`, but the JSON artifact records it under
`generatedChecks` instead of pretending it is a verified invariant.

## Workflow

To update a flow:

1. Edit the `*.flow.json` contract.
2. `cd tools/temporal-model && GOWORK=off go run . generate --root ../.. --flow <flow-id>`.
3. Update payload-specific wrapper logic if a new state or event needs richer
   runtime data. Abstract status validity and next-status outcomes come from
   `generated/<folder>/runtime.{go,ts}` and are byte-checked.
4. Update the UI replay fixture file when a new status/event needs a runtime
   fixture. The generated `AttachmentUploadFormalReplayFixtures` (or
   equivalent) interface is the contract.
5. Review the regenerated subdirectory under `generated/`.
6. `make temporal-models` (runs `check` + lint).
7. Run the scenario tests on an instantiated scenario.

Normal unit tests do not require Quint or Java:

```bash
cd tools/temporal-model && GOWORK=off go test ./...
```
