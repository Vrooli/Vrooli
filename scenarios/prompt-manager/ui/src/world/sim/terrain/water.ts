import type { TerrainTuning } from '../../config'
import { heightAt, moistureAt, slopeAt, type TerrainField } from './field'

function wetHeight(field: TerrainField, tuning: TerrainTuning, x: number, z: number): number {
  return heightAt(field, x, z) - moistureAt(field, x, z) * tuning.moistureBasinDepth
}

export function isWater(field: TerrainField, tuning: TerrainTuning, x: number, z: number): boolean {
  return Math.hypot(x, z) < field.radius && wetHeight(field, tuning, x, z) < tuning.waterLevel
}

/** Approximate signed horizontal distance to shore: negative in water. */
export function shoreDistance(field: TerrainField, tuning: TerrainTuning, x: number, z: number): number {
  const vertical = wetHeight(field, tuning, x, z) - tuning.waterLevel
  const grade = Math.max(tuning.shoreMinGrade, Math.tan(slopeAt(field, x, z)))
  return vertical / grade
}

export function waterCells(field: TerrainField, tuning: TerrainTuning): { count: number; components: number } {
  const wet = new Uint8Array(field.cols * field.rows)
  let count = 0
  for (let row = 0; row < field.rows; row += 1) {
    for (let col = 0; col < field.cols; col += 1) {
      const x = field.originX + col * field.cellSize
      const z = field.originZ + row * field.cellSize
      const index = row * field.cols + col
      if (isWater(field, tuning, x, z)) {
        wet[index] = 1
        count += 1
      }
    }
  }
  let components = 0
  const stack: number[] = []
  for (let index = 0; index < wet.length; index += 1) {
    if (wet[index] !== 1) continue
    components += 1
    wet[index] = 2
    stack.push(index)
    while (stack.length > 0) {
      const current = stack.pop()
      if (current === undefined) break
      const col = current % field.cols
      const row = Math.floor(current / field.cols)
      const neighbours = [current - 1, current + 1, current - field.cols, current + field.cols]
      neighbours.forEach((next, direction) => {
        if (next < 0 || next >= wet.length || wet[next] !== 1) return
        if ((direction === 0 && col === 0) || (direction === 1 && col === field.cols - 1) || (direction === 2 && row === 0) || (direction === 3 && row === field.rows - 1)) return
        wet[next] = 2
        stack.push(next)
      })
    }
  }
  return { count, components }
}
