# engine

Rendering infrastructure with no knowledge of the world's meaning: the Canvas,
the lighting rig (key light, HDRI + Lightformers, sky, fog), the post chain
(N8AO, bloom, AgX), the camera rig (drei CameraControls as a diorama camera),
the quality governor, the diagnostics probe/overlay and asset URL helpers.

Home, stored poses, and focus use the same footprint framing solver.
Focus preserves the current viewing angles and accounts for elevated boxes.
The input map assigns every mouse and touch slot explicitly; no library input
defaults are inherited. Follow commands are gated by target displacement and
translate immediately, preserving an in-progress user orbit or dolly. Gesture
smoothing has its own follow lever and does not restart target transitions.

`frameDistance` is the single distance solver; focus supplies its target bounds
through `poseForBox`. The former library box-fitting path was removed.
`camera.input` declares the complete gesture map:

| Input | Action |
| --- | --- |
| Left mouse drag | Rotate |
| Middle mouse drag | Truck (pan) |
| Right mouse drag | Truck (pan) |
| Mouse wheel | Dolly toward cursor |
| One-finger touch | Rotate |
| Two-finger touch | Dolly and truck |
| Three-finger touch | Truck |

Capture mode exposes a fixed-time canvas snapshot after live performance
measurement. It runs the normal local frame callbacks and post chain, freezes
animation time, and excludes DOM chrome from golden pixels. The smoke tool also
retains a full-page screenshot and separately gates runtime and loading errors.

Environment owns the outdoor sky background. Period effects set exposure and
indoor background colors; they must not overwrite the outdoor cube texture.
The diagnostic attribution pass counts the background once, excludes it from
isolated child renders, and restores it even when measurement fails.

Import rule: `config` only. Never `sim`, `scene`, `hud` or `data`.

Adjustable values come from `config`, including material, post-processing,
lighting, camera, and diagnostics settings. The literal gate permits only
documented structural constants such as unit conversions and hash mathematics.
