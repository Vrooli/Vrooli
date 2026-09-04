import { useMemo } from 'react'
import type { QualityProfile, Scene, TerrainTuning } from '../config'
import { propRecord } from '../engine/assets'
import { heightAt } from '../sim'
import { hashString } from '../sim/rng'
import { PropInstances, type Placement } from './Props'
import { useWorldStore } from './WorldStoreContext'

interface Tile {
  key: string
  propId: string
  placements: Placement[]
}

/** Spatially chunked biome vegetation and decor; tile meshes retain frustum culling. */
export function Vegetation({ scene, tuning, profile }: { scene: Scene; tuning: TerrainTuning; profile: QualityProfile }) {
  const store = useWorldStore()
  const state = store.getState()
  const tiles = useMemo(() => {
    const groups = new Map<string, Tile>()
    for (const spot of state.decor) {
      if (hashString(`density:${spot.id}`) / 0xffffffff > profile.vegetationDensityScale) continue
      const propId = spot.propId
      if (!propId) continue
      const tileX = Math.floor(spot.position[0] / tuning.tileSize)
      const tileZ = Math.floor(spot.position[1] / tuning.tileSize)
      const key = `${tileX}:${tileZ}:${propId}`
      const tile = groups.get(key) ?? { key, propId, placements: [] }
      tile.placements.push({ key: spot.id, position: spot.position, y: heightAt(state.terrain, spot.position[0], spot.position[1]), rotation: spot.rotation, scale: spot.scale })
      groups.set(key, tile)
    }
    return [...groups.values()].sort((a, b) => a.key.localeCompare(b.key)).slice(0, profile.vegetationTileBudget)
  }, [profile.vegetationDensityScale, profile.vegetationTileBudget, state.decor, state.terrain, tuning.tileSize])
  return (
    <group name="vegetation">
      {tiles.map((tile) => {
        const record = propRecord(scene.assetSet, tile.propId)
        return record ? <PropInstances key={tile.key} record={record} placements={tile.placements} scale={scene.propScale * (tile.propId.startsWith('tree_') ? scene.treeScale : 1)} frustumCulled /> : null
      })}
    </group>
  )
}
