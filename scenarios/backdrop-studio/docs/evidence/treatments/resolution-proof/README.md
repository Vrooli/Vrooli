# Resolution proof — why spatial treatment parameters must be relative

Reproduced 2026-08-12 against the running `image-tools` scenario. Every image in
this directory is regenerable with the commands below; nothing here was
hand-edited.

## The source

One deterministic scaffold rendered at two geometries with the same seed, so the
two frames are the same picture at two sizes and nothing but scale differs:

```
backdrop-studio scaffold render --preset horizon --width 768  --height 448  --seed 7 --json | jq -r .image_png | base64 -d > source-768x448.png
backdrop-studio scaffold render --preset horizon --width 2304 --height 1344 --seed 7 --json | jq -r .image_png | base64 -d > source-2304x1344.png
```

## Proof 1 — an absolute parameter is a different picture at every size

```
image-tools ops line_screen source-768x448.png   --out linescreen-spacing8-768x448.png   --spacing 8
image-tools ops line_screen source-2304x1344.png --out linescreen-spacing8-2304x1344.png --spacing 8
```

`spacing: 8` means eight pixels between lines at both sizes, so the 768px frame
carries ~96 lines across the width and the 2304px frame carries ~288. Same
style, same parameter, three times the screen density. The coarse frame reads as
a bold graphic screen; the fine frame reads as flat tone with texture. A style
tuned at one delivery surface is mistuned at every other one.

This is the defect Phase 4 removes. After the fix, `spacing_rel` is a fraction of
the short edge, so the same declared value produces the same visual density at
both sizes and the resolved pixel value appears in the operation result.

## Proof 2 — screen ruling is a tuning knob, not a resolution knob

```
image-tools ops halftone source-768x448.png --out halftone-lpi26-768x448.png  --lpi 26
image-tools ops halftone source-768x448.png --out halftone-lpi130-768x448.png --lpi 130
```

At `lpi=26` the screen cell is ~30px and the subject is destroyed; at `lpi=130`
the cell is ~6px and sun, horizon and water all read through the screen. Both
values are legal. This is the pair the plan cites as its Phase 4 motivation.

**Correction to the plan.** The plan's finding B3 lists halftone ruling among the
absolute parameters. It is not one. `treatments.go` computes the screen step as
`imageWidth / lpi`, so `lpi` is a count of screen lines across the image width
and is already resolution-independent — confirmed by
`halftone-lpi130-2304x1344.png`, which carries the same 130 lines across the
frame as its 768px counterpart and reads identically. The lpi pair above proves
a *tuning* defect (the shipped backdrop-studio default was coarser than the
value that destroys the subject), not a *scaling* defect.

The genuinely absolute spatial parameters, all fixed by this phase, are:
line-screen spacing, stipple spacing, engraving spacing, ASCII block size,
displacement spacing, aberration distance, bloom radius, defocus radius, and
motion-blur distance.

A second defect fell out of writing this up: `image-tools ops halftone --help`
described `--lpi` as "Screen cell size in pixels", which is neither the unit nor
the direction of the real parameter. Fixed in the same phase.

## Proof 3 — the fix, measured end to end

`op-art-interior` was retuned from `spacing: 6` (pixels) to `spacing_rel:
0.0181` (a fraction of the short edge) in seed v2, then rendered through the
live scenario at two real delivery surfaces:

```
backdrop-studio render submit --style op-art-interior --placement full_bleed --seed 7 --surface web.hero
backdrop-studio render submit --style op-art-interior --placement full_bleed --seed 7 --surface web.hero-mobile
```

Screen density here is ink transitions counted along the middle rows,
normalised to the short edge — a measure that is constant when two frames carry
the same screen at different sizes, and that scales with the frame when they do
not.

| Case | Small frame | Large frame | Drift |
|---|---|---|---|
| `spacing: 8` absolute, 768x448 → 2304x1344 | 111.96 | 335.96 | **+200.1%** |
| `spacing_rel: 0.0181`, 390x844 → 1440x720 | 78.62 | 78.43 | **+0.2%** |

Three times the frame used to mean three times the screen. It now means the same
picture. The residual 0.2% is whole-pixel quantisation of the line pitch.

`style-op-art-web.hero.png` and `style-op-art-web.hero-mobile.png` are those two
renders; the arcade reads at both sizes with the same graphic weight.

## What this phase did not fix

`op-art-interior`'s source is still the `arcade` scaffold — flat vector art with
no tonal depth — so the screen has little to modulate and the result reads as
pattern over a diagram rather than as a screened photograph. That is the source
layer, and it is Phase 7's subject. This phase's claim is narrower and fully
met: whatever the source, the treatment now looks the same at every size.
