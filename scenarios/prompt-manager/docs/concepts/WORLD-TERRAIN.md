# World terrain

Each scene selects its biome set, layout strategy and optional terrain override.
Overrides resolve at each world position before field, layout and navigation
generation. The office composes a level, dry floorplate-derived centre with the
park landscape. Its centre blends back to the exterior over a declared width;
the exterior retains natural terrain, water rules, and vegetation.

## Generation stages

1. Build seeded height and moisture arrays with low-frequency landform FBM,
   higher-frequency detail, moisture domain warp and radial edge falloff.
2. Derive water and shore distance from `wetHeight` (height minus moisture basin
   depth) and the local water level. Biome classification uses the same authority.
3. Classify every cell into exactly one ordered biome. The last biome is the
   required fallback.
4. Score seeded site candidates for level ground, shore clearance, distance to
   the commons, and separation. Terrace selected sites and connect them with
   navigation paths.
5. Scatter vegetation and decor from the selected biome's per-prop density
   table and seeded FBM stand mask. The mask consumes no RNG state. After jitter,
   reject water, shore-clearance violations, place clearances, and points outside
   the terrain disc. A spatial index enforces exact per-prop spacing.

The terrain digest covers the generated height and moisture arrays. The same
seed and tuning must produce the same digest, site positions, rotations, paths,
and decor.

## Tuning and invariants

Use the `terrain` block for radius, resolution, noise, falloff, water, slopes,
shore clearance, terraces, and paths. Use the `layout` block for site sampling,
spacing, scoring, place clearances, and decor jitter. Use `biomes.json` for
colour ramps and ground-bound prop density.

The simulation reports an invariant violation if a place leaves the terrain,
sites overlap, a site is not level, a team has no site, the commons is
unreachable, an actor enters water or blocked navigation, or weather is not
defined. `checkVegetationDry` also checks every final decor placement against
water and the configured shore margin.

The `low` profile draws a coarser terrain mesh, omits water and weather
particles, lowers vegetation density, and limits visible vegetation instances. It
does not change simulation geometry, site choice, navigation, or the terrain
digest.
