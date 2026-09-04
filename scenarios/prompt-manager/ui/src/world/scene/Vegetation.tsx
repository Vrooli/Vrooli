import { useFrame, useThree } from '@react-three/fiber'
import { useMemo, useRef, type RefObject } from 'react'
import { Frustum, Matrix4, Vector3 } from 'three'
import type { QualityProfile, Scene } from '../config'
import { propRecord } from '../engine/assets'
import { heightAt } from '../sim'
import { hashString } from '../sim/rng'
import { CulledPropInstances, type CulledPlacement } from './Props'
import { useWorldStore } from './WorldStoreContext'
import { visibleVegetationKeys, type VegetationCullItem } from './vegetationCull'

interface PropGroup {
  propId: string
  placements: CulledPlacement[]
}

function VegetationBudgetDriver({ items, budget, visibleKeys }: { items: VegetationCullItem[]; budget: number; visibleKeys: RefObject<ReadonlySet<string>> }) {
  const camera = useThree((state) => state.camera)
  const frustum = useMemo(() => new Frustum(), [])
  const viewProjection = useMemo(() => new Matrix4(), [])
  const cameraPosition = useMemo(() => new Vector3(), [])
  useFrame(() => {
    viewProjection.multiplyMatrices(camera.projectionMatrix, camera.matrixWorldInverse)
    frustum.setFromProjectionMatrix(viewProjection)
    camera.getWorldPosition(cameraPosition)
    visibleKeys.current = visibleVegetationKeys(items, frustum, cameraPosition, budget)
  }, -3)
  return null
}

/** One world-wide instanced group per prop id; a CPU cull owns visibility. */
export function Vegetation({ scene, profile }: { scene: Scene; profile: QualityProfile }) {
  const store = useWorldStore()
  const state = store.getState()
  const groups = useMemo(() => {
    const byProp = new Map<string, PropGroup>()
    for (const spot of state.decor) {
      if (hashString(`density:${spot.id}`) / 0xffffffff > profile.vegetationDensityScale) continue
      const propId = spot.propId
      if (!propId) continue
      const record = propRecord(scene.assetSet, propId)
      if (!record) continue
      const group = byProp.get(propId) ?? { propId, placements: [] }
      const worldScale = scene.propScale * (propId.startsWith('tree_') ? scene.treeScale : 1) * spot.scale
      group.placements.push({
        key: spot.id,
        position: spot.position,
        y: heightAt(state.terrain, spot.position[0], spot.position[1]),
        rotation: spot.rotation,
        scale: spot.scale,
        color: spot.tint,
        radius: Math.hypot(...record.size) * worldScale / 2,
      })
      byProp.set(propId, group)
    }
    return [...byProp.values()].sort((a, b) => a.propId.localeCompare(b.propId))
  }, [profile.vegetationDensityScale, scene.assetSet, scene.propScale, scene.treeScale, state.decor, state.terrain])
  const visibleKeys = useRef<ReadonlySet<string>>(new Set())
  const cullItems = useMemo<VegetationCullItem[]>(() => groups.flatMap((group) => {
    const record = propRecord(scene.assetSet, group.propId)
    if (!record) return []
    const scale = scene.propScale * (group.propId.startsWith('tree_') ? scene.treeScale : 1)
    const lift = -record.bounds.min[1] * scale
    return group.placements.map((placement) => ({
      key: placement.key,
      center: [placement.position[0], (placement.y ?? 0) + lift + record.size[1] * scale * placement.scale / 2, placement.position[1]] as const,
      radius: placement.radius,
      matrix: new Float32Array(16),
    }))
  }), [groups, scene.assetSet, scene.propScale, scene.treeScale])
  return (
    <group name="vegetation">
      <VegetationBudgetDriver items={cullItems} budget={profile.vegetationInstanceBudget} visibleKeys={visibleKeys} />
      {groups.map((group) => {
        const record = propRecord(scene.assetSet, group.propId)
        return record ? <CulledPropInstances key={group.propId} record={record} placements={group.placements} scale={scene.propScale * (group.propId.startsWith('tree_') ? scene.treeScale : 1)} budget={profile.vegetationInstanceBudget} visibleKeys={visibleKeys} /> : null
      })}
    </group>
  )
}
