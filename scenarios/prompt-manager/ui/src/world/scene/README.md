# scene

React Three Fiber components draw vertex-coloured terrain tiles, water,
terraced site walls and paths, place-bound props, biome vegetation, actors,
weather, labels, and editor handles. Terrain costs one draw per visible tile.
Vegetation costs one instanced draw per visible tile and prop material. Actor
poses are computed once per frame in `PoseBuffer` and shared by the actor
draws. Components read simulation state through refs in `useFrame`; they do
not own behaviour or call React state setters per frame.

The remaining `frustumCulled={false}` meshes are global batches by design.
`Places` batches every terraced room slab in one instanced mesh. `Slimes`,
`Faces`, `Extras`, and `Shadows` each batch the complete actor roster and
update instance transforms every frame. Three.js cannot derive a stable local
bounding volume for those mutable, world-spanning instance matrices, so the
camera must not reject the complete batch when its base geometry is offscreen.
Terrain and vegetation are spatial tiles and retain normal frustum culling.

Import rule: `engine`, `sim`, `config`. Never `hud` or `data`.
