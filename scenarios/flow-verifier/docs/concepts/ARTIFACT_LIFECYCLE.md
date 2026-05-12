# Artifact Lifecycle

Every flow's source of truth is its `flow.json`. From there, the
generator materialises four files under `<flow-dir>/generated/`:

| File | Purpose |
| ---- | ------- |
| `model.qnt` | Quint formal model rendered from the contract |
| `artifact.json` | Canonical machine-readable model artifact |
| `runtime.{go,ts}` | Typed runtime constants/unions for the host language |
| `replay.{go,ts}` | Replay helper consumed by the thin flow test |

These files are **derived**: a Vrooli checkout never ships them, and
every CI pipeline regenerates them from `flow.json` before checking.
The artifact lifecycle surface lets the UI and the CLI participate in
that regeneration without spawning a shell.

## States

| Status   | Meaning |
| -------- | ------- |
| `fresh`  | every expected artifact exists and matches the canonical render |
| `missing` | one or more artifacts are absent on disk |

A verification run can also fail with a typed `failureReason`:

| `failureReason`       | Pill              | UI affordance |
| --------------------- | ----------------- | ------------- |
| `missing_artifacts`   | yellow "Needs generate" | one-click "Generate" |
| `stale_artifacts`     | yellow "Stale"    | one-click "Regenerate" |
| `counterexample`      | red "Failed"      | inspector (CounterexampleDiff) |
| `lint` / `quint_failure` / `io` | red "Errored" | log inspection |

`failureReason` lives on the persisted `Run` row alongside
`missingArtifacts: string[]`; the substrate is typed, not a
string-matched error message.

## CLI ↔ UI parity

| Action | UI | CLI |
| ------ | -- | --- |
| Inspect | Flow Detail → Artifacts tab | `flow-verifier artifacts status --flow <id>` |
| Generate one | Flow Detail → Artifacts → Generate | `flow-verifier artifacts generate --flow <id>` |
| Generate scenario | Scenario detail → Generate all (Phase F) | `flow-verifier artifacts generate --scenario <id>` |
| Clear one | Flow Detail → Artifacts → Clear | `flow-verifier artifacts clear --flow <id>` |
| Clear scenario | Scenario detail → Clear all (Phase F) | `flow-verifier artifacts clear --scenario <id> --yes` |

`--yes` is required for `--scenario` and `--all` clears so a tab-complete
mistake can't wipe every flow's tree.

## HTTP surface

| Method | Path | Purpose |
| ------ | ---- | ------- |
| `GET`  | `/api/v1/flows/{id}/artifacts` | report status + per-file existence |
| `POST` | `/api/v1/flows/{id}/artifacts:generate` | generate / regenerate one flow |
| `DELETE` | `/api/v1/flows/{id}/artifacts` | clear one flow's generated/ |
| `POST` | `/api/v1/scenarios/{id}/artifacts:generate` | generate every flow in a scenario |
| `DELETE` | `/api/v1/scenarios/{id}/artifacts` | clear every flow in a scenario |

Every mutating route refuses to act on a `generatedDir` that resolves
outside the scenario root — a path-traversal guard sits in the service
layer, not the handler.

## Where the code lives

- `api/internal/pipeline/freshness.go` — typed `*FreshnessError`
  carrying `Missing` and `Stale` artifact paths; the recorder unwraps
  it to populate `RunEntry.FailureReason` and `MissingArtifacts`.
- `api/internal/artifacts/service.go` — `Status` / `Generate` / `Clear`
  for one flow or one scenario; `layout.GeneratedDirName` is the only
  path source.
- `api/handlers/artifacts/` — HTTP wire-up; `pipelineGenerator` shares
  the runs recorder with the verifications module.
- `cli/domains/artifacts/register.go` — in-process CLI; same service.
- `ui/src/api/artifacts.ts` — strict Raw → Public wire types.
- `ui/src/features/artifacts/ArtifactStatusPill.tsx` — shared visual.
- `ui/src/features/artifacts/ArtifactsPanel.tsx` — Flow Detail tab.

## In-tree flows

The built-in `flow-verifier.verification-run.api` flow lives inside
the scenario's own `api/` tree. Two substrate seams make it work the
same way an "external" scenario's flow does:

- `pipeline.detectGoModulePath` probes `<root>/api/go.mod` *and*
  `<root>/go.mod`, so the codegen finds the host module whether the
  caller passed `--root=.` or `--root=api`.
- `lint.resolvedSubpackageImportPath` substitutes the resolved
  module name into the `{{SCENARIO_ID}}` template before matching
  imports, so the lint accepts a hand-authored test that imports
  `flow-verifier/internal/runs/flow/generated` literally.

The flow ships its own `transition.go` + `flow_test.go` sidecars
under `api/internal/runs/flow/`; everything else is regenerated from
`flow.json`.
