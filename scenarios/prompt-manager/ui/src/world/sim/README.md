# sim

The renderer-free world model. It generates height and moisture fields, water,
biomes, buildable team sites, terraces, paths, biome decor, navigation, actors,
and weather. A fixed seed, roster, signal stream, and tick sequence produce the
same state and terrain digest. Node tests require no WebGL.

Import rule: `config` only. Never `three`, `react`, `@react-three/*`, or any
other world layer. ESLint enforces both (see `__lint__`).

Home is the desk. Members rest at their desk seat and take outings to the
commons. `invariants.ts` verifies terrain bounds, dry sites, site level,
navigation reachability, seat ownership, actor separation, and weather state.
`tick.ts` uses copy-on-write actors so all mutation stays in the returned state.
