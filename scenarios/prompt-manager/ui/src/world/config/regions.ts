/** Pure geometry shared by configuration composition and simulation terracing. */
export interface CentreRegion {
  x: number
  z: number
  width: number
  depth: number
  blend: number
}

export function smoothstep(edge0: number, edge1: number, value: number): number {
  const t = Math.max(0, Math.min(1, (value - edge0) / Math.max(Number.EPSILON, edge1 - edge0)))
  return t * t * (3 - 2 * t)
}

export function centreWeight(region: CentreRegion, x: number, z: number): number {
  const dx = Math.max(0, Math.abs(x - region.x) - region.width / 2)
  const dz = Math.max(0, Math.abs(z - region.z) - region.depth / 2)
  const distance = Math.hypot(dx, dz)
  return distance === 0 ? 1 : region.blend === 0 ? 0 : 1 - smoothstep(0, region.blend, distance)
}

export function blendHeight(original: number, target: number, weight: number): number {
  return original * (1 - weight) + target * weight
}
