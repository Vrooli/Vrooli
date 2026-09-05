# The delivery contract

Every file a consumer of Backdrop Studio receives, and what each one is for.

The short version: **you can ignore everything except the composite.** A single
PNG is a complete, finished backdrop. Everything else is additive, and a
consumer that never reads the manifest is a supported consumer rather than a
degraded one.

## The set

| File | Always present | What it is |
|---|---|---|
| `composite.png` | **yes** | The flat backdrop. The default deliverable and the only required file. |
| `manifest.json` | yes | Names every file in the set. A set whose parts a consumer has to infer is a set they will get wrong. |
| `<plate>.png` | only when the style declares a stack | One depth layer, transparent where it does not draw. |
| `motion.css` | only when the plates declare differing parallax | Transforms and keyframes over the plates. |

There is no video and no animated GIF, ever. That is a recorded boundary rather
than a gap: `image-tools` holds motion content as a non-goal, a landing page pays
a decode for something a transform does for free, and a descriptor degrades to a
still image when a viewer asks for reduced motion — which a video does not.

## Using the composite alone

```html
<div style="background-image: url('composite.png'); background-size: cover"></div>
```

That is the whole integration. It is what a consumer gets today, and nothing
below changes it.

## Using the motion descriptor

Link the stylesheet, give the container the style's class, and add one element
per layer:

```html
<link rel="stylesheet" href="motion.css">

<div class="engraved-colonnade-vector">
  <div class="engraved-colonnade-vector__layer engraved-colonnade-vector__layer--distance"></div>
  <div class="engraved-colonnade-vector__layer engraved-colonnade-vector__layer--arcade"></div>
  <div class="engraved-colonnade-vector__layer engraved-colonnade-vector__layer--canopy"></div>
</div>
```

Drive the parallax by setting `--scroll` on the container to the page's scroll
offset in pixels:

```js
addEventListener("scroll", () => {
  el.style.setProperty("--scroll", `${window.scrollY}px`);
}, { passive: true });
```

**With no `--scroll` the layers rest at zero**, which is exactly the composite's
composition. A consumer that links the stylesheet and never sets the variable
gets the still picture, not a broken one.

## Reduced motion is the default

Every transform and animation sits inside
`@media (prefers-reduced-motion: no-preference)`. A consumer who forgets to
handle the preference still gets a still picture, because there is nothing to
turn off — the motion is what has to be opted into.

`manifest.json` names the reduced-motion target explicitly. It is the composite;
it is listed separately so the contract does not depend on a consumer noticing
that.

## Where your copy goes

Every style declares an overlay region — a rectangle, in fractions of the frame,
with the text colour it is meant to carry. That is not a suggestion about
composition: the backdrop is BUILT to hold type there, and the picture is
measured against that rectangle before it is delivered.

The reserve is cut two ways. Dark copy gets a KNOCKOUT: the plate carries no ink
and the copy sits on the picture's own paper. Light copy gets a SOLID: the plate
carries full ink and the copy sits on that. Both come out in the picture's own
inks — a cream, a navy, whatever the style is printed in — rather than as white
or black patches laid over the art.

**Which of the two you get depends on the lane, and not every style is served.**
A vector style cuts both, because its generator owns its ink as well as its
paper. Every other style is served by its treatment chain, which can only take an
area toward paper — so a knockout, and dark copy. A style with LIGHT copy that is
not a vector style is currently NOT reserved, and its declared rectangle is a
statement of intent rather than a guarantee. The reason is measured and recorded
in D-022: a solid large enough to serve light copy costs more of the picture than
the quality gate allows, so the reserve gives way rather than the picture.

The honest way to know is the evidence table, which measures every style against
its own declared rectangle, colour and threshold through a really running
renderer: [`../evidence/legibility/reserved-copy.md`](../evidence/legibility/reserved-copy.md).
A style listed as passing there will carry its copy. Do not infer it from the
presence of a region.

**Where it is cut, it holds in motion, not only at rest.** The reserve extends by
the travel of the plate carrying it and is cut into every plane, so a plate
sliding under the copy brings reserved space with it rather than picture. A
reserve that held only at rest would look right in every screenshot and fail the
reader, which is worse than an obvious failure.

Nothing outside the declared rectangle is reserved, on any style. A screened
backdrop puts ink and paper everywhere else.

## What refuses, and why

**A stack whose plates all declare the same parallax emits no motion
descriptor.** Plates that move together are a flat image, and a manifest for one
would tell a consumer there is parallax to render when there is not — spending
their implementation on nothing. The manifest still ships, because they still
need to know what files exist; the refusal is about motion, not delivery.

**A style declaring a plate its generator does not draw is refused by name**, and
the error lists the layers that generator does draw. A stack with a hole in it is
never delivered.

**A style whose plates do not cover every plane its generator draws is refused.**
The mirror of the rule above, and the one that was missing: a spec naming three
plates for a generator that draws four used to deliver the picture minus its
frontmost layer, silently. Two styles shipped that way — one of them without the
sea that was its subject. Merging planes onto one plate is the accommodation
(`planes: [headland, sea]`); omitting them is not.

**A style whose copy is not legible across its parallax sweep is refused**, with
the offset and the ratio in the error. Passing at rest is not passing.

## Scoping

Every CSS rule is derived from the style id, so a page carrying two backdrops
does not have one stylesheet silently reposition the other's layers.

## Cross-references

- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — D-019 for why the flat
  composite is never optional.
- [`../evidence/plates/`](../evidence/plates/) — the plate stacks the catalog
  currently ships.
