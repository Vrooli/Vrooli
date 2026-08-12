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
| `docs/evidence/perceptual/corpus.json` | `make integration-evidence` | Where every seeded style sits relative to its perceptual bar. The lane fails when a metric moves more than 0.05, so a style drifting toward its floor is visible before it falls through. |
| `docs/evidence/perceptual/engraving-repair/` | The `render submit` commands are stated in that directory's `README.md`. | What was actually wrong with `engraved-colonnade`, including the diagnosis that was ruled out by measurement, and the before/after. |

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
