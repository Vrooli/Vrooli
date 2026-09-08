# World Assets

Furniture and vegetation props use locally baked CC0 models. Their runtime
loading does not fetch from a CDN. Slimes, terrain, celestial effects and
ambient wildlife use procedural geometry and shaders. The wildlife source and
animation recipes are recorded separately from the imported prop kits below.

## Sources

| Kit | Author | License | Used for |
|---|---|---|---|
| [Kenney Nature Kit](https://kenney.nl/assets/nature-kit) | Kenney | CC0 1.0 | park trees, rocks, bushes, flowers, stumps, log seats, campfire, sign |
| [Kenney Furniture Kit](https://kenney.nl/assets/furniture-kit) | Kenney | CC0 1.0 | desks, chairs, tables, lamps, benches, office decor |

The subset of source GLBs the scenes use lives under `ui/assets-src/world/<kit>/`
with each kit's `License.txt`; `ui/assets-src/world/sources.json` maps every
scene prop id to its source file and records the kit metadata. The HDRI reflection environment
(`ui/public/assets/world/env/sky_1k.hdr`) is Poly Haven's
"kloofendal 48d partly cloudy puresky" at 1K, CC0. The label font
(`ui/public/assets/world/fonts/NotoSans-Latin.ttf`) is Noto Sans subset to
Latin, SIL Open Font License 1.1.

## Procedural ambient sources

The `proceduralAmbient` section of `ui/assets-src/world/sources.json` maps each
ambient family to its geometry/shader source, animation source and construction
technique. These are original project implementations, not additions to the
Kenney kits. They use no imported wildlife textures, models, rigs or animation
clips; project source terms apply. The source files are the editable assets and
the runtime construction code is their recipe. Geometry is bounded and reused
for the presenter lifetime. Shared-clock animation remains independent of agent
state. This provenance record does not establish final visual approval.

## Pipeline

```bash
cd scenarios/prompt-manager/ui
pnpm world:assets           # bake every prop each scene names
pnpm world:assets --check   # verify outputs and the registry are current
```

`scripts/world-assets/build.mjs` reads place-bound prop ids from
`src/world/config/scenes/*.json` and ground-bound prop ids from the selected
biome records in `src/world/config/biomes.json`, then runs
`gltf-transform optimize` (join, palette, weld, prune, flatten, meshopt) on
each source, writes `public/assets/world/<scene>/<prop>.glb` and regenerates
`src/world/engine/assets/registry.generated.json` with path, bounds (after
dequantisation), size, triangle count, material count and bytes. It fails when
a prop exceeds `budgets.propTriangles` or a scene names a prop without a
source. Bounds and sizes are in kit units; each scene sets `propScale` and
`treeScale` so kit units read as metres.

At runtime `engine/assets/loader.ts` loads a prop through drei's `useGLTF`
with the meshopt decoder and returns one geometry + material part per
material. `scene/Props.tsx` batches place-bound props. `scene/Vegetation.tsx`
batches ground-bound props per visible terrain tile and material. A prop with
two materials costs two draws per visible batch; the registry test caps parts
at three.

## Adding a prop

1. Copy the source GLB into `ui/assets-src/world/<kit>/` (CC0 only; one
   vendor per scene).
2. Map a prop id to it in `sources.json`. Name a place-bound id in the scene
   prop block, or name a ground-bound id in a biome's `vegetation` or `decor`
   density table.
3. Run `pnpm world:assets`; the registry test (`registry.test.ts`) and the
   smoke tool's budgets keep it honest.

Prop ids are the contract between layout generation and rendering. Never
rename one without a layout migration (persisted overrides reference place
ids, decor additions reference prop ids).

## Scene architecture (redesign in progress)

The scene redesign retains the existing CC0 Kenney palette. Quaternius and commercial Synty packs were compared as alternatives; keeping the current kits avoids a second material style and preserves the current license policy. Original roof, wall, sign, bedroll, camper detail and landmark geometry lives in `ui/src/world/scene/SpaceDetails.tsx`; shared structural boxes and authored campsite arrangements live in `ui/src/world/sim/layout/spaces.ts`. `config/architecture.ts` owns the initial dimensions and palette. These are editable code-native assets, not imported models or AI-generated images. Repeated architectural details and roofs use instanced batches. This establishes provenance, not final visual approval.

Place-bound imported props now use their measured footprint centre as the layout anchor. `engine/assets/placement.ts` removes each model's horizontal origin offset after scale and yaw, and puts its lowest point at the requested ground height. This makes an office desk, chair, and round table share the same placement convention despite different source-kit origins.

Pitched roof surfaces are authored in `ui/src/world/scene/roofGeometry.ts`. The camera/body sweep caches convex half-spaces for each rendered triangle, preserving the tent doorway and the space beneath the roof. This uses the same immutable geometry and instance transforms as rendering; no enclosing roof bounding box blocks the interior. Shared structural walls remain the agent navigation authority.

The visible sky is original procedural geometry and shader code in
`scene/Environment.tsx`, `scene/CelestialSky.tsx`, and `scene/starField.ts`.
It includes the time/weather gradient, shaped clouds, seeded stars and a
mottled Milky Way with a dust lane. No new raster sky assets or dependencies
were added. Sky meshes disable picking and release their geometry/materials
on unmount; the reflection HDR remains in the existing asset pipeline.
