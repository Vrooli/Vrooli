import { useMemo } from 'react'
import type { Scene } from '../config'
import { propRecord } from '../engine/assets'
import { PropInstances } from './Props'
import { useWorldStore } from './WorldStoreContext'

/** Trees and decor from the sim's deterministic scatter, one instanced draw per variant. */
export function Trees({ scene }: { scene: Scene }) {
  const store = useWorldStore()
  const decor = store.getState().decor
  const variants = useMemo(() => scene.props.trees.map((id) => propRecord(scene.assetSet, id)), [scene])
  if (variants.length === 0) return null
  return (
    <group name="trees">
      {variants.map((record, index) =>
        record ? (
          <PropInstances
            key={record.id}
            record={record}
            placements={decor.filter((d) => d.kind === 'tree' && d.variant === index).map((d) => ({ key: d.id, position: d.position, rotation: d.rotation, scale: d.scale }))}
            scale={scene.propScale * scene.treeScale}
          />
        ) : null,
      )}
    </group>
  )
}
