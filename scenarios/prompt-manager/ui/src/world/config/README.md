# config

The world's control surface. `world.tuning.json` is the only place a behaviour
number lives; `tuning.schema.ts` bounds and documents every lever;
`scenes/*.json` describe the park and the office as data.

Import rule: this layer imports **nothing** from `sim`, `engine`, `scene`,
`hud` or `data`. It may import `zod` only.

Regenerate the documented lever table with `pnpm world:tuning-docs`; the
`tuning.test.ts` suite fails when the doc and the schema disagree.
