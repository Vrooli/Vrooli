/**
 * Outdoor-park scene generator.
 *
 * Uses a seeded PRNG for deterministic "random" placement so the result
 * is identical for a given agent count. The layout adapts to the number of
 * agents: seats scale with agents, the clearing scales with seats, and
 * trees fill the remaining forest area via rejection sampling.
 */

import type { SceneDefaults } from './types'
import type { SceneGeneratorContext } from './types'

// ---------------------------------------------------------------------------
// Seeded PRNG (mulberry32) — deterministic, no runtime randomness
// ---------------------------------------------------------------------------
function mulberry32(seed: number) {
  let s = seed | 0
  return () => {
    s = (s + 0x6d2b79f5) | 0
    let t = Math.imul(s ^ (s >>> 15), 1 | s)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Clamp a value between min and max. */
function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

/** Squared distance between two XZ points. */
function distSq(ax: number, az: number, bx: number, bz: number): number {
  const dx = ax - bx
  const dz = az - bz
  return dx * dx + dz * dz
}

/**
 * Allocate a random mix of picnic-tables (4 seats) and campfires (6 seats)
 * until totalSeats is met. May overshoot by 1-2 seats — acceptable.
 */
function allocateSeats(
  totalSeats: number,
  rand: () => number,
): Array<'picnic-table' | 'campfire'> {
  const items: Array<'picnic-table' | 'campfire'> = []
  let remaining = totalSeats
  while (remaining > 0) {
    // Bias toward campfires when many seats remain, picnic tables when few
    if (remaining >= 6 && rand() > 0.4) {
      items.push('campfire')
      remaining -= 6
    } else {
      items.push('picnic-table')
      remaining -= 4
    }
  }
  return items
}

export function generateOutdoorPark(ctx?: SceneGeneratorContext): SceneDefaults {
  const rand = mulberry32(42) // fixed seed for determinism
  const numAgents = ctx?.numAgents ?? 0

  const decorations: SceneDefaults['decorations'] = []
  const furniture: SceneDefaults['furniture'] = []

  // ---- Seats ---------------------------------------------------------------
  const totalSeats = numAgents + 10
  const seatItems = allocateSeats(totalSeats, rand)
  const furnitureCount = seatItems.length

  // ---- Clearing radius -----------------------------------------------------
  const clearingRadius = clamp(
    Math.max(
      Math.sqrt((furnitureCount * 9) / Math.PI) + 2, // enough area for furniture
      5 + Math.sqrt(numAgents) * 1.5,                 // grows with agents
    ),
    5,  // min
    30, // max
  )

  // ---- Furniture placement (rejection sampling in clearing) ----------------
  const minFurnitureDist = 3.0
  const placed: Array<{ x: number; z: number }> = []

  for (const type of seatItems) {
    let x = 0
    let z = 0
    let ok = false
    for (let attempt = 0; attempt < 50; attempt++) {
      const angle = rand() * Math.PI * 2
      const r = rand() * (clearingRadius - 2) // leave 2-unit margin from edge
      x = Math.cos(angle) * r
      z = Math.sin(angle) * r
      ok = placed.every((p) => distSq(x, z, p.x, p.z) >= minFurnitureDist * minFurnitureDist)
      if (ok) break
    }
    placed.push({ x, z })
    furniture.push({
      type,
      position: [x, 0, z],
      rotation: rand() * Math.PI * 2,
      occupiedBy: null,
    })
  }

  // ---- Trees (rejection sampling in annular forest zone) -------------------
  const outerRadius = 40
  const forestArea = Math.PI * (outerRadius * outerRadius - clearingRadius * clearingRadius)
  const treeCount = Math.min(Math.floor(forestArea / 18), 200)
  const minTreeDist = 2.5
  const treeTypes = ['oak-tree', 'pine-tree', 'birch-tree'] as const
  const treePositions: Array<{ x: number; z: number }> = []

  for (let i = 0; i < treeCount; i++) {
    let x = 0
    let z = 0
    let ok = false
    for (let attempt = 0; attempt < 50; attempt++) {
      const angle = rand() * Math.PI * 2
      const r = clearingRadius + rand() * (outerRadius - clearingRadius)
      x = Math.cos(angle) * r
      z = Math.sin(angle) * r
      ok = treePositions.every((p) => distSq(x, z, p.x, p.z) >= minTreeDist * minTreeDist)
      if (ok) break
    }
    treePositions.push({ x, z })

    const treeType = treeTypes[Math.floor(rand() * treeTypes.length)] ?? 'oak-tree'
    const scale = 0.8 + rand() * 0.4 // 0.8–1.2

    decorations.push({
      type: treeType,
      position: [x, 0, z],
      rotation: rand() * Math.PI * 2,
      scale,
    })
  }

  return { decorations, furniture }
}
