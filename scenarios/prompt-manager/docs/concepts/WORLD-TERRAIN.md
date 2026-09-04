# World terrain

The world uses one deterministic generator for park and office scenes. A scene
selects a biome set; it does not select a second layout implementation.

## Generation stages

1. Build seeded height and moisture arrays with value-noise FBM and radial edge
   falloff.
2. Derive water and shore distance from height, moisture, and the configured
   water level.
3. Classify every cell into exactly one ordered biome. The last biome is the
   required fallback.
4. Score seeded site candidates for level ground, shore clearance, distance to
   the commons, and separation. Terrace selected sites and connect them with
   navigation paths.
5. Scatter vegetation and decor from the selected biome's per-prop density
   table. Reject points inside place clearances or outside the terrain disc.

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
defined.

The `low` profile draws a coarser terrain mesh, omits water and weather
particles, lowers vegetation density, and limits visible vegetation tiles. It
does not change simulation geometry, site choice, navigation, or the terrain
digest.
