# engine

Rendering infrastructure with no knowledge of the world's meaning: the Canvas,
the lighting rig (key light, HDRI + Lightformers, sky, fog), the post chain
(N8AO, bloom, AgX), the camera rig (drei CameraControls as a diorama camera),
the quality governor, the diagnostics probe/overlay and asset URL helpers.

Import rule: `config` only. Never `sim`, `scene`, `hud` or `data`.

Everything numeric comes from `config` (tuning, scene data, the resolved
lighting period); the few constants kept here are rendering-only (env map
resolution, sky dome distance, AO/bloom shape) and are not world behaviour.
