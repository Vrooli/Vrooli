# Evidence Pack

## Storage and authority

Generated captures belong to Backdrop Studio's managed `test_runs` storage class,
under the logical name `evidence`. The shared storage resolver chooses the
physical path. Producers log their destination. An explicit absolute
`BACKDROP_STUDIO_EVIDENCE_DIR` can isolate a capture or a test; it is not a
source-tree default.

Keep current validation evidence separate from historical observations. A new
render cannot reproduce an old machine state. Preserve a historical capture
before retirement, including its original paths and hashes.

## Preconditions

Use the managed lifecycle to build and start Backdrop Studio and Image Tools.
Live integration checks require the API to match the working tree. Model-backed
styles additionally need an available image model and sufficient device memory.

```bash
vrooli scenario start backdrop-studio
vrooli scenario start image-tools
```

Run the commands below from the scenario root unless an API directory is named.
`make integration-evidence` captures output; it does not update the accepted
regression baseline. Review candidate measurements before changing testdata.

## Artifacts and their producing commands

Paths in this table are logical names beneath the managed output root.
Only this table declares a producer; a path mentioned in other prose does not.

| Artifact | Command | What it proves |
|---|---|---|
| `evidence/render-matrix.md` | `make integration-evidence` | Seeded styles rendered through Image Tools, with build fingerprint and named skips. |
| `evidence/treatments/*.png` | `make integration-evidence` | Treatment output at delivery geometry using resolved catalog parameters. |
| `evidence/catalog/sheet-*.png` | `make integration-evidence` | Catalog contact sheets for visual review. Model-backed sheets need generation capacity. |
| `evidence/plates/*.png` and `evidence/plates/README.md` | `make integration-evidence` | Composite output for styles that declare depth planes. |
| `evidence/delivery-set/` | `make integration-evidence` | Delivery manifest and motion CSS; see the [delivery contract](../reference/delivery-contract.md). |
| `evidence/legibility/reserved-copy.md` | `make integration-evidence` | Worst-pixel contrast inside each style's declared overlay region on live output. |
| `evidence/perceptual/corpus.json` | `make integration-evidence` | Candidate perceptual measurements for comparison with the committed regression corpus. |
| `evidence/scenes/*.png` | `make integration-evidence` | Procedural generators and palette variants before treatment. |
| `evidence/catalog/resemblance.md` | In `api`: `BACKDROP_STUDIO_WRITE_EVIDENCE=1 GOWORK=off go test ./internal/catalog/ -run TestResemblanceReportEvidence` | Nearest-neighbour structure and chain similarity; colour is deliberately excluded. |
| `evidence/phase-03/*.png` | In `api`: `UPDATE_SCAFFOLD_EVIDENCE=1 GOWORK=off go test ./internal/scaffold/ -run TestGoldenEvidence` | Scaffold seed comparisons; this explicit update also updates scaffold goldens. |

The existing `internal/evidence` checker compares artifacts against these
producer declarations. Its unit tests use isolated captures, not host history.

## Regression corpus

The accepted perceptual corpus lives in
[api/integration/testdata/perceptual-corpus.json](../../api/integration/testdata/perceptual-corpus.json).
The integration test reads that source-controlled fixture. Captures write a
candidate into managed storage, so refreshing a gallery cannot silently change
the baseline. The 0.05 metric tolerance remains unchanged. Missing models and
capacity failures are recorded skips, never passes.

## Review lessons

- Judge subject, composition, and legibility at delivery resolution. A passing
  metric or a small thumbnail does not establish that a style is worth shipping.
- A screen over a source with no composition remains texture. Repair the source
  when subject structure is absent; changing colour does not fix resemblance.
- Keep screen cells finer than the perceptual gate's sampling cells. Otherwise
  the gate measures the screen instead of the subject. Fine screens alone do
  not supply missing large-scale composition.
- Absolute spatial parameters change a treatment's appearance across delivery
  sizes. Use the relative parameter contract where scale stability is intended;
  halftone ruling must be assessed under its own semantics.
- A model-backed sheet can predate its neighbours because a capture skipped
  generation. Read its capture date and render matrix together.
- Keep one producer per observed fact. Parallel ad hoc preview writers previously
  drifted, including a preview that retained a corrected shape defect.
- Delivery CSS preserves reduced-motion behavior and includes each layer's
  parallax translation in animation keyframes because CSS transforms replace
  rather than compose automatically.

## Historical evidence

The pre-cleanup evidence tree, including original PNGs, before/after judgments,
catalog coverage qualifications, one-time baselines, and reproduction commands,
is preserved byte-for-byte outside Git beneath the protected runtime-home
`plan_artifacts` entry:

```text
docs-cleanup-20260907-followup/scenarios/backdrop-studio/docs/evidence/
```

The same capture contains the original `docs/internal/EVIDENCE.md` and detailed
progress log. The [repository preservation record](../../../../docs/internal/PROGRESS.md#documentation-cleanup-follow-up--2026-09-07)
locates hashes and recovery copies. These are historical observations, not a
current ship verdict or proof that unresolved catalog gaps have been fixed.

Current catalog obligations remain in [starter-catalog.md](../reference/starter-catalog.md),
with open issues in [PROBLEMS.md](PROBLEMS.md). Do not infer completion from
relocation of the old reports.
