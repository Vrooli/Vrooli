import { ambientPolicy } from '../../config/ambient'
import { HABITAT_IDS } from '../../config/habitats'
import type { DecorSpot } from '../model'
import { hashString } from '../rng'
import { heightAt, type TerrainField } from '../terrain/field'

export interface ButterflySite {
  id: string
  sample: number
  vegetationId: string
  x: number
  z: number
  radius: number
  phase: number
  rank: number
}

/** Meadow vegetation anchors a small loop. All nine neighboring habitat
 * samples must be meadow, keeping the whole loop away from protected edges.
 * Bounded ranking does not depend on decoration enumeration or visual quality.
 */
export function* butterflySiteSteps(seed: number, field: TerrainField, habitats: Uint8Array, decor: readonly DecorSpot[]) {
  if (habitats.length !== field.cols * field.rows) throw new Error('Invalid butterfly habitat inputs')
  const meadow = HABITAT_IDS.indexOf('meadow')
  const sites: ButterflySite[] = []
  const maximum = ambientPolicy.butterflies.maximumVisible.ultra
  for (const [offset, spot] of decor.entries()) {
    if (offset % 128 === 0) yield { completed: offset, total: decor.length }
    if (spot.roomId || spot.kind !== 'ground' && spot.kind !== 'shrub') continue
    const col = Math.round((spot.position[0] - field.originX) / field.cellSize)
    const row = Math.round((spot.position[1] - field.originZ) / field.cellSize)
    if (col < 1 || col >= field.cols - 1 || row < 1 || row >= field.rows - 1) continue
    let eligible = true
    for (let dz = -1; dz <= 1; dz++) for (let dx = -1; dx <= 1; dx++) {
      if (habitats[(row + dz) * field.cols + col + dx] !== meadow) eligible = false
    }
    if (!eligible) continue
    const sample = row * field.cols + col
    const rank = hashString(`butterfly-site-v1:${seed}:${sample}`)
    const existing = sites.find(site => site.sample === sample)
    if (existing) {
      if (spot.id < existing.vegetationId) existing.vegetationId = spot.id
      continue
    }
    const site: ButterflySite = {
      id: `butterfly:${seed}:${sample}`, sample, vegetationId: spot.id, rank,
      x: field.originX + col * field.cellSize, z: field.originZ + row * field.cellSize,
      radius: field.cellSize * ambientPolicy.butterflies.orbitCellFraction,
      phase: hashString(`butterfly-phase-v1:${seed}:${sample}`) / 4294967296 * Math.PI * 2,
    }
    const insertion = sites.findIndex(other => other.rank > rank || other.rank === rank && other.sample > sample)
    if (insertion >= 0) sites.splice(insertion, 0, site)
    else if (sites.length < maximum) sites.push(site)
    if (sites.length > maximum) sites.pop()
  }
  return sites
}

export function butterflyPose(site: ButterflySite, field: TerrainField, seconds: number) {
  const angle = seconds % ambientPolicy.butterflies.periodSeconds / ambientPolicy.butterflies.periodSeconds * Math.PI * 2 + site.phase
  const x = site.x + Math.sin(angle) * site.radius
  const z = site.z + Math.sin(angle * 2) * site.radius * .5
  return {
    position: [x, heightAt(field, x, z) + .45 + .12 * Math.sin(angle * 3), z] as const,
    yaw: Math.atan2(Math.cos(angle), Math.cos(angle * 2)),
  }
}
