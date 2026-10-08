import type { InstancedMesh } from 'three'

/** Own the mesh's instance buffers while geometry and materials belong to the asset cache. */
export function instanceOwner(attach?: (mesh: InstancedMesh | null) => void): (mesh: InstancedMesh | null) => void {
  let current: InstancedMesh | null = null
  return next => {
    if (current && current !== next) current.dispose()
    current = next
    attach?.(next)
  }
}
