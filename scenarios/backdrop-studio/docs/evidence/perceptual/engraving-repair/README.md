# The engraved-colonnade repair

`engraved-colonnade` is the failure this plan was written around: it rendered
illegible diagonal moire while every test in both scenarios passed. This
directory records what was actually wrong with it, how that was established, and
what fixed it. Every image here was produced by a command stated below.

All renders are `--seed 7` at `web.hero` (1440x720) through the running
scenarios.

## What it looked like

`before-engraved-colonnade.png` — diagonal hatching with no colonnade in it. No
arch, no column, no statue, no horizon.

## The wrong diagnosis, and how it was ruled out

The obvious reading is "the treatment destroyed the subject", and the perceptual
gate's `subject_survival` metric exists to catch exactly that. It scored the
broken image **0.973**, essentially the same as the styles that render
correctly (`op-art-interior` 0.986, `cyanotype-arcade` 0.943). The metric was
not being fooled — it was right, and the diagnosis was wrong.

`lowfrequency-field-comparison.png` is the proof. It shows source, engraved,
op-art and cyanotype reduced to the 64x36 low-frequency field the gate measures.
The arcade is plainly present in the engraved row: arches, sea, statue, sun. The
composition survived. Something else made the image unusable.

Correlation was checked at five grid scales from 16x9 to 256x144, on both the
tonal field and the gradient field. None separated the broken style from the
working ones. The failure is not a composition failure.

## The real defect

Ink run lengths along the image rows, at 1440x720 over the same source:

| Render | Ink runs 1–2px wide | Median run |
|---|---|---|
| `op-art-interior` (works) | 1.5% | 9px |
| `cyanotype-arcade` (works) | 6.0% | 11px |
| **`engraved-colonnade` (broken)** | **31.8%** | 5px |

Nearly a third of the marks were one or two pixels across. The cause was one
expression in `image-tools`' engraving treatment:

```go
width := math.Max(0.6, (1-l)*spacing*0.55)
```

A line 0.6 pixels wide cannot be drawn. Rasterised, it becomes a dotted trail of
aliased fragments — and because 0.6 was a *floor* rather than a cutoff, those
fragments were laid over the highlights too, so every square pixel of paper in
the frame carried a broken hatch. The beat between those fragments and the hatch
period is the diagonal moire.

An engraver does not draw an infinitely fine line for a pale tone; they leave the
paper blank. The fix does the same: below one whole pixel, no mark.

```go
width := (1 - l) * spacing * 0.55
drawInk := width >= minInkWidth && ...
```

## The second half: tonal range

Fixing the marks cleaned up the texture but the colonnade still did not read.
The procedural `arcade` scene delivers a compressed tonal range — its wall sits
at L\* 0.85 and its sea at 0.45 — which maps to a narrow band of mark widths. A
screen can only express the range it is given.

Seed v3 turns on `normalize` for every screening treatment, stretching the
source's p1–p99 span onto the full ink ramp before screening.

## The result

`after-engraved-colonnade.png` — three arches, the columns between them, the
statue, the foliage canopy, and clean paper. `after-cyanotype-arcade.png` is the
same source under a blue duotone halftone, which is the reference genre this
scenario was commissioned to hit.

## Reproduce

```
# the source, as the procedural lane generates it
backdrop-studio render submit --style engraved-colonnade --placement split_panel --seed 7 --surface web.hero
backdrop-studio render submit --style cyanotype-arcade  --placement full_bleed  --seed 7 --surface web.hero
```

The mark-run measurement is not committed as a gate. It separates these three
cases cleanly, but `stipple-massif` — which renders correctly — scores 0.232 on
the same measure, because a stipple's marks are legitimately 1–2px dots. A
threshold that fails the broken engraving would also fail a working stipple, so
it is recorded here as a diagnostic rather than shipped as a check. See
`docs/internal/PROBLEMS.md`.
