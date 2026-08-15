# A delivery set

What a consumer of `engraved-colonnade-vector` receives, at `web.hero` geometry.

**Reproduce:** `make integration-evidence` from `scenarios/backdrop-studio`.

`manifest.json` names every file in the set. `motion.css` is the
transform descriptor over the plates. The plate images and the composite are not
duplicated here — the stacks themselves are in
[`../plates/`](../plates/) — because this artifact exists to show the
CONTRACT, and two copies of the same PNG would only drift.

The contract itself is written up in
[`../../reference/delivery-contract.md`](../../reference/delivery-contract.md).

Two properties worth reading the CSS for:

- Every transform and keyframe sits inside
  `@media (prefers-reduced-motion: no-preference)`, so a still picture is
  what a consumer gets by default and motion is what has to be opted into.
- Each ambient keyframe restates its layer's parallax translate. A CSS
  `transform` in a keyframe REPLACES the one on the rule rather than
  composing with it, so an animation that omitted it would snap its layer back to
  zero parallax the moment the loop started.
