# World Assets

Every prop in the world is a CC0 model baked by a checked-in script. There
are no runtime procedural textures, no CDN fetches and no hand-made meshes in
JSX. Slimes are the one exception: they stay procedural (a sphere and the
wobble shader).

## Sources

| Kit | Author | License | Used for |
|---|---|---|---|
| [Kenney Nature Kit](https://kenney.nl/assets/nature-kit) | Kenney | CC0 1.0 | park trees, rocks, bushes, flowers, stumps, log seats, campfire, sign |
| [Kenney Furniture Kit](https://kenney.nl/assets/furniture-kit) | Kenney | CC0 1.0 | desks, chairs, tables, lamps, benches, office decor |

The subset of source GLBs the scenes use lives under `ui/assets-src/world/<kit>/`
with each kit's `License.txt`; `ui/assets-src/world/sources.json` maps every
scene prop id to its source file and records the kit metadata. The HDRI sky
(`ui/public/assets/world/env/sky_1k.hdr`) is Poly Haven's
"kloofendal 48d partly cloudy puresky" at 1K, CC0. The label font
(`ui/public/assets/world/fonts/NotoSans-Latin.ttf`) is Noto Sans subset to
Latin, SIL Open Font License 1.1.

## Pipeline

```bash
cd scenarios/prompt-manager/ui
pnpm world:assets           # bake every prop each scene names
pnpm world:assets --check   # verify outputs and the registry are current
```

`scripts/world-assets/build.mjs` reads `src/world/config/scenes/*.json`, runs
`gltf-transform optimize` (join, palette, weld, prune, flatten, meshopt) on
each source, writes `public/assets/world/<scene>/<prop>.glb` and regenerates
`src/world/engine/assets/registry.generated.json` with path, bounds (after
dequantisation), size, triangle count, material count and bytes. It fails when
a prop exceeds `budgets.propTriangles` or a scene names a prop without a
source. Bounds and sizes are in kit units; each scene sets `propScale` and
`treeScale` so kit units read as metres.

At runtime `engine/assets/loader.ts` loads a prop through drei's `useGLTF`
with the meshopt decoder and returns one geometry + material part per
material; `scene/Props.tsx` draws each part as one `Instances` batch across
every place of that kind. A prop with two materials costs two draws; the
registry test caps parts at three.

## Adding a prop

1. Copy the source GLB into `ui/assets-src/world/<kit>/` (CC0 only; one
   vendor per scene).
2. Map a prop id to it in `sources.json`, and name that id in the scene file
   (`props.decor`, `props.trees`, or one of the place kinds).
3. Run `pnpm world:assets`; the registry test (`registry.test.ts`) and the
   smoke tool's budgets keep it honest.

Prop ids are the contract between layout generation and rendering. Never
rename one without a layout migration (persisted overrides reference place
ids, decor additions reference prop ids).
