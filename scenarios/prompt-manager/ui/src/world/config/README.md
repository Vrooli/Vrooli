# config

The world's control surface. `world.tuning.json` is the only place a behaviour
number lives; `tuning.schema.ts` bounds and documents every lever;
`scenes/*.json` select camera, lighting, place-bound props, and a biome set.
`biomes.json` owns terrain colour ramps and the vegetation and decor density
tables. Park and office use the same terrain generator; the office selects one
flat `floor` biome with no ground-bound props.

Import rule: this layer imports **nothing** from `sim`, `engine`, `scene`,
`hud` or `data`. It may import `zod` only.

Regenerate the documented lever table with `pnpm world:tuning-docs`; the
`tuning.test.ts` suite fails when the doc and the schema disagree.
