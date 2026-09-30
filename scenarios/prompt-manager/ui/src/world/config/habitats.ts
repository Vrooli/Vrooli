/** Stable byte IDs shared by generation, wildlife eligibility and diagnostics. */
export const HABITAT_IDS = ['none', 'meadow', 'woodland', 'rocky-slope', 'wetland', 'open-water'] as const
export type HabitatId = typeof HABITAT_IDS[number]
export const HABITAT_BY_BIOME: Readonly<Record<string, HabitatId>> = {
  meadow: 'meadow', forest: 'woodland', rocky: 'rocky-slope', wetland: 'wetland', water: 'open-water',
}
