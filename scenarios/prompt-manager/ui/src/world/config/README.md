# config

The world's control surface. `world.tuning.json` owns shared behaviour settings;
`tuning.schema.ts` bounds and documents those levers;
`scenes/*.json` select camera, lighting, place-bound props, and a biome set.
`biomes.json` owns terrain colour ramps and the vegetation and decor density
tables. Each vegetation entry declares its density, class (`tree`, `shrub`, or
`ground`), and scale reference (`tree` or `prop`); filenames carry no semantics.

Scene composition starts with a base biome set and terrain. An optional `centre`
derives its bounds from the floorplate plus a margin, applies local terrain and
biome overrides, and blends back to the base over its declared width. The office
uses a level, dry office centre inside a park landscape. The centre height comes
from the plate mean with water clearance; its boundary grade is constrained.
`resolveTerrain(...).at(x, z)` is the position-aware tuning authority. Config owns
the region primitives; simulation supplies the floorplate-derived region.

The optional `emissive` map belongs to rendered prop slots, not asset names.
For example, the park's hearth and lamp slots emit; the office's lamp slot emits,
but its coffee-table hearth does not. Lighting periods scale that emission.

Import rule: this layer imports **nothing** from `sim`, `engine`, `scene`,
`hud` or `data`. It may import `zod` only.

Regenerate the documented lever table with `pnpm world:tuning-docs`; the
`tuning.test.ts` suite fails when the doc and the schema disagree.
