/**
 * Outdoor-park scene generator.
 *
 * Uses a seeded PRNG for deterministic "random" placement so the result
 * is identical on every invocation. The layout keeps a clear center area
 * (radius ~5) and places trees in a ring at radius 8–14.
 */

import type { SceneDefaults } from './types'

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

export function generateOutdoorPark(): SceneDefaults {
  const rand = mulberry32(42) // fixed seed for determinism

  const decorations: SceneDefaults['decorations'] = []
  const furniture: SceneDefaults['furniture'] = []

  // ---- Trees in a ring (radius 8–14, avoiding center) ------------------
  const treeTypes = ['oak-tree', 'pine-tree', 'birch-tree'] as const
  const treeCount = 14
  for (let i = 0; i < treeCount; i++) {
    const angle = (i / treeCount) * Math.PI * 2 + (rand() - 0.5) * 0.4
    const radius = 8 + rand() * 6 // 8–14
    const x = Math.cos(angle) * radius
    const z = Math.sin(angle) * radius
    const treeType = treeTypes[Math.floor(rand() * treeTypes.length)] ?? 'oak-tree'
    const scale = 0.8 + rand() * 0.4 // 0.8–1.2
    decorations.push({
      type: treeType,
      position: [x, 0, z],
      rotation: rand() * Math.PI * 2,
      scale,
    })
  }

  // ---- Flowers near inner edge of tree ring ----------------------------
  const flowerPositions: [number, number, number][] = [
    [5.5, 0, 2],
    [-4.5, 0, -5],
    [3, 0, -6],
    [-6, 0, 1],
  ]
  for (const pos of flowerPositions) {
    decorations.push({
      type: 'flowers',
      position: pos,
      rotation: rand() * Math.PI * 2,
      scale: 0.9 + rand() * 0.2,
    })
  }

  // ---- Benches facing center at radius ~5 ------------------------------
  const benchAngles = [Math.PI / 4, Math.PI, -Math.PI / 3]
  for (const angle of benchAngles) {
    const r = 5
    const x = Math.cos(angle) * r
    const z = Math.sin(angle) * r
    furniture.push({
      type: 'bench',
      position: [x, 0, z],
      rotation: angle + Math.PI, // face center
      occupiedBy: null,
    })
  }

  // ---- Picnic table ----------------------------------------------------
  furniture.push({
    type: 'picnic-table',
    position: [-3, 0, 3.5],
    rotation: Math.PI / 6,
    occupiedBy: null,
  })

  // ---- Floor lamps near seating ----------------------------------------
  decorations.push({
    type: 'floor-lamp',
    position: [4, 0, -4],
    rotation: 0,
    scale: 1,
  })
  decorations.push({
    type: 'floor-lamp',
    position: [-4, 0, 4.5],
    rotation: 0,
    scale: 1,
  })

  return { decorations, furniture }
}
