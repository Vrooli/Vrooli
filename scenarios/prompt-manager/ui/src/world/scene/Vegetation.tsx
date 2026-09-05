import { useFrame, useThree } from '@react-three/fiber'
import { useMemo } from 'react'
import { Object3D } from 'three'
import { biomeSets, type CameraTuning, type QualityProfile, type Scene } from '../config'
import { recordVegetationCull } from '../engine/diagnostics/store'
import { propRecord } from '../engine/assets'
import { heightAt } from '../sim'
import { hashString } from '../sim/rng'
import { CulledPropInstances, type CulledPlacement } from './Props'
import { useWorldStore } from './WorldStoreContext'
import { VegetationBuffer, VegetationCuller, type VegetationCullItem } from './vegetationCull'

interface PropGroup {
  propId: string
  assetSet: string
  baseScale: number
  scaleRef: 'tree' | 'prop'
  placements: CulledPlacement[]
}

/** One world-wide instanced group per prop id; a CPU cull owns visibility. */
export function Vegetation({ scene, profile, camera: cameraTuning }: { scene: Scene; profile: QualityProfile; camera: CameraTuning }) {
  const store = useWorldStore()
  const state = store.getState()
  const groups = useMemo(() => {
    const byProp = new Map<string, PropGroup>()
    for (const spot of state.decor) {
      if (hashString(`density:${spot.id}`) / 0xffffffff > profile.vegetationDensityScale) continue
      const propId = spot.propId
      if (!propId) continue
      const source = spot.roomId ? scene : biomeSets[scene.biomeSet]
      const record = propRecord(source.assetSet, propId)
      if (!record) continue
      const key = source.assetSet + '/' + propId + '/' + spot.scaleRef
      const baseScale = source.propScale * (spot.scaleRef === 'tree' ? source.treeScale : 1)
      const group = byProp.get(key) ?? { propId, assetSet: source.assetSet, baseScale, scaleRef: spot.scaleRef, placements: [] }
      const worldScale = baseScale * spot.scale
      group.placements.push({
        key: spot.id,
        position: spot.position,
        y: heightAt(state.terrain, spot.position[0], spot.position[1]),
        rotation: spot.rotation,
        scale: spot.scale,
        color: spot.tint,
        radius: Math.hypot(...record.size) * worldScale / 2,
      })
      byProp.set(key, group)
    }
    return [...byProp.values()].sort((a, b) => a.propId.localeCompare(b.propId))
  }, [profile.vegetationDensityScale, scene, state.decor, state.terrain])
  const camera = useThree((state) => state.camera)
  const buffers = useMemo(() => groups.map((group) => new VegetationBuffer(Math.min(group.placements.length, profile.vegetationInstanceBudget))), [groups, profile.vegetationInstanceBudget])
  const culler = useMemo(() => {
    const dummy = new Object3D()
    const items: VegetationCullItem[] = []
    for (let groupIndex = 0; groupIndex < groups.length; groupIndex += 1) {
      const group = groups[groupIndex]
      if (!group) continue
      const record = propRecord(group.assetSet, group.propId)
      if (!record) continue
      const scale = group.baseScale
      const lift = -record.bounds.min[1] * scale
      for (const placement of group.placements) {
        dummy.position.set(placement.position[0], (placement.y ?? 0) + lift, placement.position[1])
        dummy.rotation.set(0, placement.rotation, 0)
        dummy.scale.setScalar(scale * placement.scale)
        dummy.updateMatrix()
        items.push({
          key: placement.key, group: groupIndex,
          center: [dummy.position.x, dummy.position.y + record.size[1] * scale * placement.scale / 2, dummy.position.z],
          radius: placement.radius, matrix: new Float32Array(dummy.matrix.elements), color: placement.color,
        })
      }
    }
    return new VegetationCuller(items, buffers, profile.vegetationInstanceBudget)
  }, [groups, buffers, profile.vegetationInstanceBudget])
  useFrame(() => {
    recordVegetationCull(culler.update(camera, cameraTuning.cullEpsilonMetres, cameraTuning.cullEpsilonRadians))
  }, -2)
  return (
    <group name="vegetation">
      {groups.map((group, index) => {
        const record = propRecord(group.assetSet, group.propId)
        const buffer = buffers[index]
        return record && buffer ? <CulledPropInstances key={group.assetSet + '/' + group.propId + '/' + group.scaleRef} record={record} buffer={buffer} /> : null
      })}
    </group>
  )
}
