import { useEffect, useMemo } from 'react'
import { BufferAttribute, BufferGeometry, Color } from 'three'
import { type QualityProfile, type Scene, type TerrainResolver, type TerrainVisualTuning, type WeatherPreset } from '../config'
import { sceneBiomes } from '../config/biomes'
import { bakeVertexColour, heightAt, heightFieldAo, moistureAt, shoreDistance } from '../sim'
import { useWorldStore } from './WorldStoreContext'
import { terrainMaterialSettings, terrainTintVariation } from './terrainAppearance'

export function Terrain({ scene, tuning, profile, weather, visual }: { scene: Scene; tuning: TerrainResolver; profile: QualityProfile; weather: WeatherPreset; visual: TerrainVisualTuning }) {
  const store = useWorldStore()
  const state = store.getState()
  const geometry = useMemo(() => {
    const field = state.terrain
    if (scene.environment === 'indoor' && !scene.centre) {
      const half = field.radius
      const result = new BufferGeometry()
      result.setAttribute('position', new BufferAttribute(new Float32Array([-half, 0, -half, -half, 0, half, half, 0, -half, half, 0, half]), 3))
      result.setAttribute('normal', new BufferAttribute(new Float32Array([0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0]), 3))
      const ground = new Color(scene.palette.ground)
      result.setAttribute('color', new BufferAttribute(new Float32Array([ground.r, ground.g, ground.b, ground.r, ground.g, ground.b, ground.r, ground.g, ground.b, ground.r, ground.g, ground.b]), 3))
      result.setIndex([0, 1, 2, 2, 1, 3])
      result.computeBoundingSphere()
      return result
    }
    const innerRadius = Math.max(field.cellSize, Math.min(tuning.base().innerRadius, profile.terrainInnerRadius))
    const step = Math.max(1, Math.round(profile.terrainCellScale * tuning.base().innerRadius / innerRadius))
    const cols = Math.floor((field.cols - 1) / step) + 1
    const rows = Math.floor((field.rows - 1) / step) + 1
    const vertices = new Float32Array(cols * rows * 3)
    const normals = new Float32Array(vertices.length)
    const colours = new Float32Array(vertices.length)
    const indices: number[] = []
    const tint = new Color(weather.terrainTint)
    const shadowTint = new Color(weather.terrainShadowTint)
    const set = sceneBiomes(scene)
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
        const local = tuning.at(x, z)
        const shore = shoreDistance(field, tuning, x, z)
        const colour = bakeVertexColour({ moisture: moistureAt(field, x, z), path: state.pathMask[sourceIndex] ?? 0, ao: heightFieldAo(field, x, z, visual.aoRadius, visual.aoSamples), wetShore: shore < 0 ? Math.max(0, 1 + shore / local.wetShoreWidth) : 0, wetShoreDarkening: local.wetShoreDarkening }, biome)
        const mix = weather.terrainTintMix
        const variation = terrainTintVariation(x, z, weather.terrainTintVariation, visual)
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
  }, [profile.terrainCellScale, profile.terrainInnerRadius, scene, state.biomes, state.pathMask, state.terrain, tuning, visual, weather.terrainShadowTint, weather.terrainTint, weather.terrainTintMix, weather.terrainTintVariation])
  useEffect(() => () => geometry.dispose(), [geometry])
  return (
    <mesh name="terrain" geometry={geometry} receiveShadow>
      <meshStandardMaterial vertexColors {...terrainMaterialSettings(weather.wetness, visual)} metalness={0} />
    </mesh>
  )
}
