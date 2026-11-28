/** Initial product rarity policy; presentation quality must not change these values. */
export const METEOR_VARIANTS = ['white', 'blue', 'green', 'violet', 'great-fireball'] as const
export type MeteorVariant = typeof METEOR_VARIANTS[number]
export const WILDLIFE_PREVIEWS = [
  { id: 'fireflies', label: 'fireflies' }, { id: 'butterflies', label: 'butterflies' }, { id: 'birds', label: 'birds' },
  { id: 'rabbit-idle', label: 'idle rabbit' }, { id: 'rabbit-travel', label: 'hopping rabbit' },
  { id: 'squirrel-forage', label: 'foraging squirrel' }, { id: 'squirrel-climb', label: 'climbing squirrel' },
  { id: 'fish-jump', label: 'fish jump' }, { id: 'fish-splash', label: 'fish splash' }, { id: 'fish-ripple', label: 'fish ripple' },
] as const
export type WildlifePreviewId = typeof WILDLIFE_PREVIEWS[number]['id']
export const ambientPolicy = {
  meteor: {
    meanEligibleSeconds: 180,
    deepNightExtraSeconds: 90,
    maximumConcurrent: 2,
    durationSeconds: [0.6, 1.8],
    weights: [8500, 1000, 350, 140, 10],
    fireballCooldownSeconds: 86400,
    fireballDurationSeconds: [2.5, 4],
  },
  comet: { opportunitySeconds: 30 * 86400, probability: 0.08, visibleDays: [3, 5], maximumConcurrent: 1 },
  fireflies: { maximumVisible: { low: 12, medium: 24, high: 48, ultra: 48 }, height: [0.3, 0.9] },
  butterflies: { maximumVisible: { low: 4, medium: 8, high: 16, ultra: 16 }, orbitCellFraction: 0.3, periodSeconds: 24 },
  birds: { maximumVisible: { low: 1, medium: 2, high: 4, ultra: 4 }, radiusCells: 5, clearance: 18, periodSeconds: 32 },
  rabbits: { maximumVisible: { low: 1, medium: 1, high: 2, ultra: 2 }, radius: .38, idleSeconds: 18, travelSeconds: 4 },
  squirrels: { maximumVisible: { low: 1, medium: 2, high: 3, ultra: 4 }, radius: .22, routeBudget: 16, periodSeconds: 30,
    // Leaf-material lower bounds in the baked park kit, before its ground lift.
    // Dense low pine tiers do not expose enough trunk for a climb.
    canopyBottom: { tree_default: .75036, tree_oak: .45040 },
    trunkRadius: { tree_default: .06, tree_oak: .052 },
    climbClearance: .6, minimumClimbHeight: .75 },
  fish: { minimumPondArea: 12, meanEligibleSeconds: 90, maximumConcurrent: 2, maximumVisible: { low: 1, medium: 1, high: 2, ultra: 2 }, jumpSeconds: .8, effectSeconds: 2.5 },
} as const
