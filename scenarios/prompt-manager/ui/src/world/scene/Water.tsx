import { useEffect, useMemo } from 'react'
import { BufferAttribute, BufferGeometry } from 'three'
import type { QualityProfile, TerrainTuning } from '../config'
import { isWater } from '../sim'
import { useWorldStore } from './WorldStoreContext'

export function Water({ tuning, profile }: { tuning: TerrainTuning; profile: QualityProfile }) {
  const store = useWorldStore()
  const state = store.getState()
  const geometry = useMemo(() => {
    const field = state.terrain
    const positions: number[] = []
    const indices: number[] = []
    for (let row = 0; row < field.rows - 1; row += 1) for (let col = 0; col < field.cols - 1; col += 1) {
      const x = field.originX + col * field.cellSize
      const z = field.originZ + row * field.cellSize
      const centerX = x + field.cellSize * 0.5
      const centerZ = z + field.cellSize * 0.5
      if (!isWater(field, tuning, centerX, centerZ)) continue
      const base = positions.length / 3
      const y = tuning.waterLevel + 0.015
      positions.push(x, y, z, x, y, z + field.cellSize, x + field.cellSize, y, z, x + field.cellSize, y, z + field.cellSize)
      indices.push(base, base + 1, base + 2, base + 2, base + 1, base + 3)
    }
    const result = new BufferGeometry()
    result.setAttribute('position', new BufferAttribute(new Float32Array(positions), 3))
    result.setIndex(indices)
    result.computeVertexNormals()
    result.computeBoundingSphere()
    return result
  }, [state.terrain, tuning])
  useEffect(() => () => geometry.dispose(), [geometry])
  if (!profile.waterEnabled || geometry.getAttribute('position').count === 0) return null
  return (
    <mesh name="water" geometry={geometry} renderOrder={2}>
      <meshStandardMaterial color="#4f9db8" transparent opacity={0.72} roughness={0.25} metalness={0.05} depthWrite={false} />
    </mesh>
  )
}
