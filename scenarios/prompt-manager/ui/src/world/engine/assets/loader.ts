import { useGLTF } from '@react-three/drei'
import { useMemo } from 'react'
import { Mesh, type BufferGeometry, type Material } from 'three'
import type { PropRecord } from './registry'
import { propUrl } from './registry'

export interface PropPart {
  geometry: BufferGeometry
  material: Material
}

/**
 * Load a baked prop as geometry + material parts for instancing. The bake
 * joins primitives per material, so most props are one part and a few
 * (screen + frame, wood + cushion) are two or three.
 */
export function usePropParts(record: PropRecord): PropPart[] {
  const gltf = useGLTF(propUrl(record), undefined, true)
  return useMemo(() => {
    const parts: PropPart[] = []
    gltf.scene.updateMatrixWorld(true)
    gltf.scene.traverse((object) => {
      if (!(object instanceof Mesh)) return
      const mesh = object as Mesh<BufferGeometry, Material>
      const material: Material = mesh.material
      // Bake the node transform into a geometry copy so instances share one origin.
      const geometry = mesh.geometry.clone()
      geometry.applyMatrix4(mesh.matrixWorld)
      parts.push({ geometry, material })
    })
    if (parts.length === 0) throw new Error(`prop ${record.id} has no mesh`)
    return parts
  }, [gltf, record.id])
}

export function preloadProp(record: PropRecord): void {
  useGLTF.preload(propUrl(record), undefined, true)
}
