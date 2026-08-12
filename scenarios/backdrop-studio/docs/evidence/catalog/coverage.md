# Tier-2 coverage, counted

Recorded 2026-08-12, against seed v5 (40 styles, 18 surfaces).

The rules are in [`../../reference/starter-catalog.md`](../../reference/starter-catalog.md).
Reproduce every number below with:

```
backdrop-studio catalog list --json | jq '.styles | length'
backdrop-studio surfaces list --json | jq '.surfaces | length'
```

and the per-axis breakdown by grouping that JSON. The build-time equivalents are
`TestNoTwoSettledStylesRenderTheSamePicture` and `TestEveryGeneratorIsReachedByAStyle`
in `api/internal/catalog/distinctness_test.go`.

## Tier 1 — the floor

| Rule | Required | Actual | Met |
|---|---|---|---|
| Every treatment family appears at least once | 8 families | all 8 | yes |
| At least two styles per `strategy` | 2 each | procedural 6, procedural-treated 28, guided 4, synthesized 2 | yes |
| At least 60% `procedural` or `procedural-treated` | 60% | 85% (34/40) | yes |
| Every page placement declared by at least two styles | 2 each | full_bleed 31, split_panel 20, corner_bleed 12, framed_inset 11 | yes |
| At least one style per surface kind | 1 each | product, store, social, email all reachable | yes |
| No style names a living artist | 0 | 0 | yes |

`procedural` had **zero** styles before this seed version, so the second rule
was unmet from the day the catalog was written and nothing reported it. That is
what the coverage count is for.

## Tier 2 — the showcase

| Rule | Required | Actual | Met |
|---|---|---|---|
| Every treatment with a landed `image-tools` operation appears in a style | 18 | 18 | yes |
| At least eight distinct lineages | 8 | 24 | yes |
| Every store placement demonstrated | 6 | 6 (device_center, three caption variants, feature_graphic, type_mask) | yes |
| Roughly 40 styles | ~40 | 40 | yes |

The four treatments closed by this version were `curve`, `defocus`,
`motion_blur` and `pixel_sort` — each had a working operation, a wire contract,
and no style, so no operator could see that the scenario could do it.

## Shape of the set — one slice is short, deliberately

| Slice | Tier-2 target | Actual | |
|---|---|---|---|
| Non-representational fields | 12–14 | 17 | over |
| Treated geometry | 8–10 | 9 | on |
| Representational scenes | 10–12 | **6** | **short** |
| Intricate subjects | 4–5 | 2 | short |
| Store-oriented | 4–6 | 5 | on |

**Why the representational and intricate slices are short.** Both are the
model-backed lanes, and local generation on the reference host runs out of
device memory at hero aspect — `ErrorOutOfDeviceMemory` from
stable-diffusion.cpp on a card already holding several gigabytes of resident
language models. Every additional `guided` or `synthesized` style is therefore a
row that cannot be rendered, cannot be scored by the perceptual gate, and cannot
be judged on a contact sheet.

Six is what the lane needs to be real: two conditioning presets, three subjects
no procedural generator draws, and both model strategies exercised. Authoring
six more to reach the suggested proportion would move a number without moving
the product, and would leave twelve unvalidated rows for the next reader to
discover. The gap closes when generation capacity does, not before.

The non-representational slice runs over its target for the opposite reason: it
is the half the procedural lane draws for free, and seven generators is a real
range rather than one generator wearing seven names —
[`../procedural/subject-collapse.md`](../procedural/subject-collapse.md) records
what that used to look like.

## What is still not covered

`cellular_automata`, `wave_function_collapse` and `l_system` are Axis 2
vocabulary with no generator. `textile_material` and `object_metaphor` are Axis 3
subjects that no generator depicts and no seeded style claims. Both lists are
backlog, not defects: the axes are open descriptions of the space, and the
catalog refuses a style naming a subject it cannot draw rather than substituting
a nearby picture.
