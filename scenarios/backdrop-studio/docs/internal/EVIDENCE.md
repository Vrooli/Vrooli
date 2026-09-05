# Evidence Pack

Every artifact under `docs/evidence/` is produced by a command written here.

That rule is the whole point of this file. Twelve style previews rendered
during an earlier catalog seeding came from a throwaway probe and had to be
deleted rather than shipped, because nobody could say which command made them —
and a committed placement PNG still showed an elliptical sun from before
`drawScaled` was corrected. **Unreproducible evidence is itself the defect:** it
looks like proof, and it is a claim about a build nobody can identify.

If you add an artifact, add its command here. If an artifact has no command,
delete the artifact.

## Preconditions

Every command below needs `backdrop-studio` and `image-tools` running, and the
API built from the working tree. The lane refuses to run otherwise — a stale
binary has twice produced audit findings that were simply false.

```bash
(cd scenarios/backdrop-studio && make build) && vrooli scenario restart backdrop-studio
vrooli scenario start image-tools
```

## Artifacts and their producing commands

| Artifact | Command | What it proves |
|---|---|---|
| `docs/evidence/render-matrix.md` | `make integration-evidence` | Every seeded style rendered through a really running `image-tools`, with the API build fingerprint, the installed model set, and a named reason beside every skip. |
| `docs/evidence/baseline/pre-plan-render-matrix.md` | Recorded once, before the repair. The reproduce command is stated in the file itself. | The 4-pass / 12-fail starting state this plan repaired. Historical; it is not regenerated. |
| `docs/evidence/baseline/catalog-upgrade-proof.md` | Stated in the file: restore `data/backdrop-studio.db.preaudit`, restart, count. | A pre-plan install reaches the current catalog without losing operator-authored rows. |
| `docs/evidence/treatments/*.png` | `make integration-evidence` | Every treatment the catalog can name, rendered over one scaffold at `web.hero` geometry through a really running `image-tools`, using the scenario's own default parameters resolved by the real merge. |
| `docs/evidence/treatments/resolution-proof/` | The exact `backdrop-studio` and `image-tools` commands are stated in that directory's `README.md`. | That an absolute spatial parameter is a different picture at every size (+200.1% screen density across a 3x frame), that the relative form is not (+0.2%), and that halftone's ruling was never the absolute one the plan believed it was. |
| `docs/evidence/delivery-set/manifest.json`, `docs/evidence/delivery-set/motion.css` and `docs/evidence/delivery-set/README.md` | `make integration-evidence` | Exactly what a consumer receives for a plated style: the manifest naming every file, and the CSS transform descriptor over the plates. It exists so the delivery contract can be read against something real rather than against prose describing it. |
| `docs/evidence/legibility/reserved-copy.md` | `make integration-evidence` | Worst-pixel contrast inside each style's own declared overlay region, measured on the picture a really running `image-tools` produced. It exists because the same measurement taken through the unit suite's fake executor produced a confident catalog-wide number that was not about the catalog. |
| `docs/evidence/plates/*.png` and `docs/evidence/plates/README.md` | `make integration-evidence` | Every style whose generator separates depth planes, shown as the plates it ships and the composite assembled from them through a really running `image-tools` compositor. A stack nobody can look at is a stack nobody reviews. |
| `docs/evidence/perceptual/corpus.json` | `make integration-evidence` | Where every seeded style sits relative to its perceptual bar. The lane fails when a metric moves more than 0.05, so a style drifting toward its floor is visible before it falls through. |
| `docs/evidence/perceptual/engraving-repair/` | The `render submit` commands are stated in that directory's `README.md`. | What was actually wrong with `engraved-colonnade`, including the diagnosis that was ruled out by measurement, and the before/after. |
| `docs/evidence/catalog/sheet-*.png` | `make integration-evidence` | The whole catalog, grouped by how each treatment family fails, at a size where the failures this scenario has actually shipped are visible. A style not on a sheet is named in the run log with the reason. |
| `docs/evidence/catalog/sheet-model-backed.png` | `make integration-evidence`, **when the host can serve generation** | The model-backed lane's output. Its timestamp legitimately lags its siblings: whether a diffusion job gets device memory depends on what else is resident on the card at that moment, so a run that regenerates the other four sheets may skip this one entirely. When that happens the render matrix says `SKIP(gpu-capacity)` for every model-backed style and this file keeps the last successful render. Read its date, not its neighbours'. |
| `docs/evidence/catalog/coverage.md` | Hand-counted against the rules in `../../reference/starter-catalog.md`; every number is reproducible from `backdrop-studio catalog list --json` and `surfaces list --json`. | Which Tier-1 and Tier-2 rules are met, which slice is short, and why. |
| `docs/evidence/catalog/verdicts.md` | Written by a person reading the sheets above. | The judgement no metric makes: whether each style is worth putting on a landing page. Seven were not, and each was repaired at cause. |
| `docs/evidence/scenes/*.png` | `make integration-evidence` | Every procedural generator at `web.hero` geometry, including each palette variant, rendered in-process. This is the source layer before any treatment touches it. |
| `docs/evidence/procedural/subject-collapse.md` | The five `render submit` commands are stated in the file. | That sixteen named art directions once drew four pictures, and what the generator/subject contract replaced it with. |
| `docs/evidence/procedural/*.png` | The same five `render submit` commands, stated in `subject-collapse.md` beside them. | What each of the four new non-representational generators drew at seed v7, one style per generator. Historical, recorded 2026-08-12; not regenerated. Read the catalog sheets for a style's current state. |
| `docs/evidence/catalog/resemblance.md` | `BACKDROP_STUDIO_WRITE_EVIDENCE=1 GOWORK=off go test ./internal/catalog/ -run TestResemblanceReportEvidence` | Every style's nearest neighbour and the resemblance measured to it, so a cluster of styles that read as one picture is a number rather than an impression. Colour is excluded from the measure by design: a recolour is not a divergence. |

## The rule is enforced, not merely stated

`internal/evidence.TestEveryEvidenceArtifactHasAProducingCommand` walks
`docs/evidence/` and fails with the list of artifacts the table above does not
name. Only the table counts — not this document's prose, and not the deleted-set
paths below.

That distinction is the whole reason the check works. The first version read
every backticked path in this file, which meant the sentence at the top —
"Every artifact under `docs/evidence/` is produced by a command written here" —
parsed as an entry admitting the entire tree. The rule's own statement repealed
it, and the check passed against fourteen files nobody could reproduce.

## Deleted rather than kept

Three artifact sets were removed under this file's own rule.

`docs/evidence/procedural/` held nine artifacts on 2026-08-13: four
`scene-*.png`, a `contact-sheet.png`, two `placement-preview-*.png` and two
`conditioning-input-*.png`. Every one of them had a producer — two opt-in
`EVIDENCE_DIR` writers in `internal/render/render_test.go` — which is what made
them worth deleting rather than regenerating. Each was a **second** producer for
a fact this scenario already evidences elsewhere, and the two had drifted apart:
`scene-*.png` covered the four presets that existed when it was written while
`docs/evidence/scenes/` covers all thirteen at hero geometry, and
`contact-sheet.png` duplicated the catalog sheets. Two producers for one fact
means the tree carries two answers with nothing to say which is current. The
mobile placement preview was the proof: it still showed the elliptical sun from
before `drawScaled` was corrected — the cautionary tale this file already
tells, sitting in this file's own directory. Both writers were removed with the
artifacts, so neither can return silently.

Two artifact sets were removed on 2026-08-12 under this file's own rule.

`docs/evidence/placements/` held eight placement previews with no producing
command, and the audit had already found one of them — the split-panel
desktop frame — still showing an elliptical sun from before `drawScaled` was
corrected. They were a picture of a build nobody could identify.

`docs/evidence/phase-14/store-device-center.png` was a single frame from a
phase that no longer exists by that name, also with no command.

Placement evidence returns when the server-side mockup covers every placement
rather than four of ten. `render.composePlacement` currently has cases for
`split_panel`, `framed_inset` and `corner_bleed` and a default that catches the
rest, so a committed sheet today would show `device_center` and
`feature_graphic` as full-bleed landing pages — evidence of something that is
not true. The gap is recorded in `PROBLEMS.md`.

## Reading a skip

A skip in the render matrix always names what it is waiting on:

| Skip | Meaning |
|---|---|
| `SKIP(no-image-model)` | No enabled, installed model serves `text_to_image` or `image_to_image`. The deterministic lane is unaffected. |
| `SKIP(gpu-capacity)` | The request reached the model and the host could not allocate device memory. This workstation routinely holds ~10 GB of idle language models against a 16 GB card, so a diffusion job can lose the allocation race through no fault of the catalog. |

**A skip is never a pass.** It is printed, counted separately, and carries a
reason a reader can check. A lane that quietly passed on absent capability
would recreate the exact blindness this scenario is repairing.

## What is deliberately not committed

Delivery-resolution PNGs of every style. `docs/evidence/` reached 8.8 MB across
36 PNGs at delivery resolution, and `grain.png` alone is 2.9 MB because noise
does not compress. Delivery resolution was the right call — a screen cannot be
judged at 64×48 — but the storage question is an owner decision recorded in
[`DECISIONS.md`](DECISIONS.md).
