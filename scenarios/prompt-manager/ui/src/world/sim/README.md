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

Vegetation density is modulated by a seeded FBM stand mask, creating stands and
gaps. The mask samples coordinates without consuming any random-number-generator
state. Adding an RNG call there would shift the later placement stream and change
unrelated trees. Spacing uses a spatial index with exact distance checks.

`wetHeight` subtracts the moisture basin depth from terrain height. Both biome
water classification and `isWater` compare that value with the local water level.
Water biomes have empty vegetation and decor tables. Scatter rejects water and
the configured shore clearance after positional jitter. `checkVegetationDry`
independently checks the final placements, including the shore margin.

Office generation derives a centre from the floorplate and composes its level,
dry interior with the exterior landscape before navigation and scatter. Terrain
queries use the position-aware resolver, including during layout rebuilds.

Saved room transforms apply to the entire space, including doors, furniture and
seats. Campground terraces and biome classification use those final transforms.
When growth makes a saved site overlap another site or the commons, a bounded,
deterministic search finds nearby clear ground while retaining its orientation.
Older position-only campground saves derive a stable direction toward the commons.
Large unsaved plots can use alternative orientations when the preferred heading
does not fit the available land. Recovery searches spatially distinct candidates,
checks their rotated corners, and bounds its branching and candidate expansion.
Bedrolls have dedicated dimensions and standing clearance; office desk settings
do not change sleeping arrangements. Both bed rows retain an open cross aisle.

Saved office rooms reserve both their footprint and a clear doorway connection
to the authored halls. Access corridors share the normal floor/ceiling renderer.
The final rotated room and hall extents determine the building footprint and
level terrain region. The floorplan invariant checks connected flooring across
each doorway as well as a navigable route to the lounge. An impossible saved
layout produces a recoverable error; the editor offers Reset layout.
The former low-wall/nav-aisle fallback and binary office subdivision are retired.
Every generated room now uses explicit space metadata and shared structures.
