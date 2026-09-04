import { useEffect, useMemo } from 'react'
import { BufferAttribute, BufferGeometry, Color } from 'three'
import { biomeSets, type QualityProfile, type TerrainTuning, type WeatherPreset } from '../config'
import { bakeVertexColour, heightAt, heightFieldAo, moistureAt } from '../sim'
import { useWorldStore } from './WorldStoreContext'

export function Terrain({ tuning, profile, weather }: { tuning: TerrainTuning; profile: QualityProfile; weather: WeatherPreset }) {
  const store = useWorldStore()
  const state = store.getState()
  const geometry = useMemo(() => {
    const field = state.terrain
    const innerRadius = Math.max(field.cellSize, Math.min(tuning.innerRadius, profile.terrainInnerRadius))
    const step = Math.max(1, Math.round(profile.terrainCellScale * tuning.innerRadius / innerRadius))
    const cols = Math.floor((field.cols - 1) / step) + 1
    const rows = Math.floor((field.rows - 1) / step) + 1
    const vertices = new Float32Array(cols * rows * 3)
    const normals = new Float32Array(vertices.length)
    const colours = new Float32Array(vertices.length)
    const indices: number[] = []
    const tint = new Color(weather.terrainTint)
    const shadowTint = new Color(weather.terrainShadowTint)
    const set = biomeSets[state.biomeSetId === 'office' ? 'office' : 'park']
    for (let row = 0; row < rows; row += 1) for (let col = 0; col < cols; col += 1) {
      const sourceRow = Math.min(field.rows - 1, row * step)
      const sourceCol = Math.min(field.cols - 1, col * step)
      const sourceIndex = sourceRow * field.cols + sourceCol
      const index = row * cols + col
      const x = field.originX + sourceCol * field.cellSize
      const z = field.originZ + sourceRow * field.cellSize
      const y = field.height[sourceIndex] ?? 0
      vertices.set([x, y, z], index * 3)
      const half = field.cellSize * 0.5
      const dx = (heightAt(field, x + half, z) - heightAt(field, x - half, z)) / (half * 2)
      const dz = (heightAt(field, x, z + half) - heightAt(field, x, z - half)) / (half * 2)
      const length = Math.hypot(dx, 1, dz)
      normals.set([-dx / length, 1 / length, -dz / length], index * 3)
      const biome = set.biomes[state.biomes[sourceIndex] ?? set.biomes.length - 1] ?? set.biomes[set.biomes.length - 1]
      if (biome) {
        const colour = bakeVertexColour({ moisture: moistureAt(field, x, z), path: state.pathMask[sourceIndex] ?? 0, ao: heightFieldAo(field, x, z, 3, 8) }, biome)
        const mix = weather.terrainTintMix
        const variation = weather.terrainTintVariation * (0.7 + Math.sin(x * 0.17 + z * 0.11) * 0.15 + Math.sin(x * 0.07 - z * 0.19) * 0.15)
        const tintR = tint.r + (shadowTint.r - tint.r) * variation
        const tintG = tint.g + (shadowTint.g - tint.g) * variation
        const tintB = tint.b + (shadowTint.b - tint.b) * variation
        colours.set([colour[0] * (1 - mix) + tintR * mix, colour[1] * (1 - mix) + tintG * mix, colour[2] * (1 - mix) + tintB * mix], index * 3)
      }
    }
    for (let row = 0; row < rows - 1; row += 1) for (let col = 0; col < cols - 1; col += 1) {
      const x = field.originX + (col + 0.5) * field.cellSize * step
      const z = field.originZ + (row + 0.5) * field.cellSize * step
      if (Math.hypot(x, z) > field.radius) continue
      const a = row * cols + col
      const b = a + 1
      const c = a + cols
      const d = c + 1
      indices.push(a, c, b, b, c, d)
    }
    const result = new BufferGeometry()
    result.setAttribute('position', new BufferAttribute(vertices, 3))
    result.setAttribute('normal', new BufferAttribute(normals, 3))
    result.setAttribute('color', new BufferAttribute(colours, 3))
    result.setIndex(indices)
    result.computeBoundingSphere()
    return result
  }, [profile.terrainCellScale, profile.terrainInnerRadius, state.biomeSetId, state.biomes, state.pathMask, state.terrain, tuning.innerRadius, weather.terrainShadowTint, weather.terrainTint, weather.terrainTintMix, weather.terrainTintVariation])
  useEffect(() => () => geometry.dispose(), [geometry])
  return (
    <mesh name="terrain" geometry={geometry} receiveShadow>
      <meshStandardMaterial vertexColors color={weather.wetness > 0 ? '#d4dce0' : '#ffffff'} roughness={Math.max(0.3, 1 - weather.wetness * 0.65)} metalness={0} />
    </mesh>
  )
}
