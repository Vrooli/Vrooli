import type { Vec3 } from './pose'

/** Compare with the last commanded position, so small movements accumulate. */
export function shouldFollow(last: Vec3 | null, next: Vec3, epsilon: number): boolean {
  if (!last) return true
  const dx = next[0] - last[0]
  const dy = next[1] - last[1]
  const dz = next[2] - last[2]
  return dx * dx + dy * dy + dz * dz > epsilon * epsilon
}
