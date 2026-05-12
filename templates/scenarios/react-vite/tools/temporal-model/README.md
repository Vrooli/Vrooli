# Temporal Model Artifacts

This directory owns the template-local formal temporal-model validation loop.

## Canonical layout (schema v6)

Every flow lives in a `flow/` subdirectory next to its consumer:

```
<feature-dir>/
└── flow/
    ├── flow.json                    # hand: source of truth (schema v6)
    ├── transition.{ts,go}           # hand: payload/runtime logic
    ├── fixtures.ts                  # hand: replay fixtures (TS only)
    ├── flow.test.{ts,go}            # hand: thin replay delegation (lint-enforced)
    └── generated/
        ├── model.qnt
        ├── artifact.json
        ├── runtime.{ts,go}
        └── replay.{helper.ts,go}
```

Every flow uses the same file names. The `flow/` directory IS the unit; the
contract no longer encodes any module/path information. `flow.json` is the
source of truth; everything under any `generated/` directory is codegen
output and must not be hand-edited.

## Scaffolding a new flow

The `new` subcommand emits a runnable minimal flow into `<parent>/flow/`:

```bash
cd tools/temporal-model
GOWORK=off go run . new ui/src/features/foo --flow-id foo.workflow.ui --root ../..
GOWORK=off go run . new api/internal/bar --flow-id bar.workflow.api --root ../..
```

`--lang ts|go` is optional; it is inferred from `ui/*` / `api/*` paths.
The scaffold writes the hand-authored files (`flow.json`, `transition.*`,
`fixtures.ts` for TS, `flow.test.*`) and immediately runs `generate`, so
`check` is green from the moment the command returns. A scaffolded flow has
two states (`idle`, `ready`), two events (`start`, `reset`), and one named
trace (`smoke`).

## Run

```bash
cd tools/temporal-model && GOWORK=off go run . list --root ../..
cd tools/temporal-model && GOWORK=off go run . validate --root ../..
cd tools/temporal-model && GOWORK=off go run . generate --root ../..
cd tools/temporal-model && GOWORK=off go run . generate --root ../.. --flow notes.attachment-upload.ui
cd tools/temporal-model && GOWORK=off go run . check --root ../..
cd tools/temporal-model && GOWORK=off go run . explain --root ../.. --flow notes.attachment-upload.ui
```

The generator runs `quint typecheck`, `quint test`, `quint verify`, and
`quint run --mbt` for each discovered temporal flow contract. It renders the
canonical Quint model, normalizes ITF traces, records source/model/generator
hashes, and writes every artifact under `<flow>/generated/`. Go flows get a
`runtime.go` + `replay.go` subpackage (package `generated`) that exports
`RunReplay`; TypeScript flows get a `runtime.ts` + `replay.helper.ts` module
that exports `runFormalReplay`. The hand-authored `flow.test.{ts,go}`
delegates to that single entry point.

## Lint pass

`check` runs two stages: byte-for-byte freshness of generated files, and an
AST-level lint of the hand-authored replay test.

- Go test files are parsed with `go/ast`. The lint rejects: missing import of
  `<scenario>/.../flow/generated`, missing call to `generated.RunReplay`,
  `nil` transition argument, function-literal transition with an empty body,
  and blank-assignment-only bodies.
- TypeScript test files are parsed with a structural source reader. The lint
  rejects: missing import of `./generated/replay.helper`, missing binding of
  the wrapper's transition function from `./transition`, missing binding of
  the fixtures export from `./fixtures`, and any `runFormalReplay(...)` call
  that is not at module top level (calls nested inside `if`, `try`, function
  bodies, etc. fail).

There is no `--no-lint` flag. Lint failures are fatal.

## Migrating from schema v5

Schema v5 contracts are no longer supported. The `LoadRaw` path emits a
clear migration error pointing here. To migrate:

1. Create `<feature>/flow/` next to the consumer.
2. Move `<Name>.flow.json` → `flow/flow.json` and bump `schemaVersion` to 6.
3. Drop `replay.fixtureModule`, `replay.fixtureExport`, and
   `replay.transition.module`. Keep `replay.transition.function` (and the
   `statusAccessor` for TS / `stateType` + `statusField` for Go).
4. Move the wrapper to `flow/transition.{ts,go}` and the fixtures
   (TS only) to `flow/fixtures.ts`.
5. Move the test to `flow/flow.test.{ts,go}`. For Go, change the test's
   package declaration to `flow` and update the import from the old
   subpackage name to `<scenario>/.../flow/generated`.
6. Move `generated/<old-folder>/` → `flow/generated/`. Regenerate with
   `go run . generate --root ../..` to refresh hashes/imports.
7. Update consumers' import paths to `./flow/transition` (TS) or
   `<scenario>/.../flow` (Go).

## Workflow

To update a flow:

1. Edit `flow/flow.json`.
2. `cd tools/temporal-model && GOWORK=off go run . generate --root ../.. --flow <flow-id>`.
3. Update payload-specific wrapper logic in `flow/transition.{ts,go}`.
   Abstract status validity and next-status outcomes come from
   `flow/generated/runtime.{go,ts}` and are byte-checked.
4. Update `flow/fixtures.ts` when a new status/event needs a runtime
   fixture (TS only). The generated `<Name>FormalReplayFixtures` interface
   is the contract.
5. Review the regenerated `flow/generated/` directory.
6. `make temporal-models` (runs `check` + lint).
7. Run the scenario tests on an instantiated scenario.

Normal unit tests do not require Quint or Java:

```bash
cd tools/temporal-model && GOWORK=off go test ./...
```
