# scene

React Three Fiber components draw vertex-coloured terrain tiles, water,
terraced site walls and paths, place-bound props, biome vegetation, actors,
weather, labels, and editor handles. Terrain costs one draw per visible tile.
Vegetation costs one instanced draw per prop id and material. Actor
poses are computed once per frame in `PoseBuffer` and shared by the actor
draws. Components read simulation state through refs in `useFrame`; they do
not own behaviour or call React state setters per frame.

The shared actor pose writer blends a focused resting actor toward the camera.
Body, face, and accessories consume the same rendered yaw. Simulation facing
remains unchanged; walking uses its simulation heading without a viewer blend.
Gameplay and navigation read simulation facing, never this presentation yaw.

Lamp emission is scene-role data scaled by the lighting period. The place-prop
owner passes its existing lamp placements to a fixed point-light pool, avoiding
separate placement calculations. Only the nearest profile-budgeted lamps cast
light. Camera changes reuse the pool objects; daylight mounts no lamp lights.
Both this selection and vegetation use the shared camera-motion gate.

The remaining `frustumCulled={false}` meshes are global batches by design.
`Places` batches every terraced room slab in one instanced mesh. `Slimes`,
`Faces`, `Extras`, and `Shadows` each batch the complete actor roster and
update instance transforms every frame. Three.js cannot derive a stable local
bounding volume for those mutable, world-spanning instance matrices, so the
camera must not reject the complete batch when its base geometry is offscreen.
Terrain retains normal frustum culling. Vegetation's world-wide meshes set
`frustumCulled={false}` because one global CPU sphere test compacts visible
instance matrices into driver-owned buffers. A nearest-K heap enforces the
world-wide budget only when visibility exceeds it; equal-distance ties retain
layout order. Camera position, orientation, and projection gate both selection
and uploads. Resting frames reuse all buffers and increment the diagnostics
skip counter without allocating a new diagnostic snapshot.

Import rule: `engine`, `sim`, `config`. Never `hud` or `data`.
