import { describe, expect, it, vi } from 'vitest'
import { BufferGeometry, InstancedMesh, MeshStandardMaterial } from 'three'
import { instanceOwner } from './instanceOwner'

describe('instance buffer ownership', () => {
  it('releases replaced and detached meshes once without disposing shared assets', () => {
    const geometry = new BufferGeometry()
    const material = new MeshStandardMaterial()
    const first = new InstancedMesh(geometry, material, 1)
    const larger = new InstancedMesh(geometry, material, 10)
    const firstDispose = vi.spyOn(first, 'dispose')
    const largerDispose = vi.spyOn(larger, 'dispose')
    const geometryDispose = vi.spyOn(geometry, 'dispose')
    const materialDispose = vi.spyOn(material, 'dispose')
    const attach = vi.fn()
    const ref = instanceOwner(attach)
    ref(first)
    ref(first)
    ref(larger)
    expect(firstDispose).toHaveBeenCalledTimes(1)
    ref(null)
    ref(null)
    expect(largerDispose).toHaveBeenCalledTimes(1)
    expect(geometryDispose).not.toHaveBeenCalled()
    expect(materialDispose).not.toHaveBeenCalled()
    expect(attach).toHaveBeenLastCalledWith(null)
  })
})
