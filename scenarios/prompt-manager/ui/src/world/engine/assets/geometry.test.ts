import { createHash } from 'node:crypto'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { BufferGeometry, Float32BufferAttribute, InterleavedBuffer, InterleavedBufferAttribute, Matrix3, Matrix4, Mesh, Group, MeshStandardMaterial, Vector3 } from 'three'
import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader.js'
import { MeshoptDecoder } from 'three/examples/jsm/libs/meshopt_decoder.module.js'
import registry from './registry.generated.json'
import { bakeGeometry, preparePropParts } from './geometry'
const xyz = (a: ReturnType<BufferGeometry['getAttribute']>, i: number) => new Vector3().fromBufferAttribute(a, i)

describe('canonical static geometry', () => {
  it.each(Object.values(registry.props))('preserves every shipped vertex: $path', async record => {
    const bytes = readFileSync(resolve('public/assets/world', record.path.replace(/^assets\/world\//, '')))
    expect(createHash('sha256').update(bytes).digest('hex')).toBe(record.contentHash)
    const gltf = await new GLTFLoader().setMeshoptDecoder(MeshoptDecoder).parseAsync(new Uint8Array(bytes).buffer, '')
    gltf.scene.updateMatrixWorld(true)
    let vertices = 0
    gltf.scene.traverse(object => {
      if (!(object instanceof Mesh)) return
      const mesh = object as Mesh
      const original = mesh.geometry.getAttribute('position')
      const output = bakeGeometry(mesh.geometry, mesh.matrixWorld)
      const bounds = output.boundingBox
      if (!bounds) throw new Error('Missing baked bounds')
      for (let i = 0; i < original.count; i++) {
        const expected = xyz(original, i).applyMatrix4(object.matrixWorld)
        expect(xyz(output.getAttribute('position'), i).distanceTo(expected)).toBeLessThan(0.00001)
        expect(bounds.distanceToPoint(expected)).toBeLessThan(0.00001)
        const sourceNormal = mesh.geometry.getAttribute('normal')
        if (mesh.geometry.hasAttribute('normal')) {
          const expectedNormal = xyz(sourceNormal, i).applyMatrix3(new Matrix3().getNormalMatrix(object.matrixWorld)).normalize()
          expect(xyz(output.getAttribute('normal'), i).distanceTo(expectedNormal)).toBeLessThan(0.00001)
        }
        vertices++
      }
      output.dispose()
      mesh.geometry.dispose()
    })
    expect(vertices).toBeGreaterThan(0)
  })
  it.each([false, true])('preserves mirrored interleaved geometry: indexed=%s', indexed => {
    const source = new BufferGeometry()
    const values = [0, 0, 0, 99, 32767, 0, 0, 99, 0, 32767, 0, 99]
    const data = new InterleavedBuffer(new Int16Array(values), 4)
    source.setAttribute('position', new InterleavedBufferAttribute(data, 3, 0, true))
    source.setAttribute('normal', new Float32BufferAttribute([0, 0, 1, 0, 0, 1, 0, 0, 1], 3))
    source.setAttribute('tangent', new Float32BufferAttribute([1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1], 4))
    source.setAttribute('uv', new Float32BufferAttribute([0, 0, 1, 0, 0, 1], 2))
    if (indexed) source.setIndex([0, 1, 2])
    const matrix = new Matrix4().makeRotationY(0.6).multiply(new Matrix4().makeScale(-2, 3, 4)).setPosition(5, 2, -3)
    const output = bakeGeometry(source, matrix)
    expect(Array.from(data.array)).toEqual(values)
    expect(Array.from(output.index?.array ?? [])).toEqual([0, 2, 1])
    const normal = new Vector3(0, 0, 1).applyMatrix3(new Matrix3().getNormalMatrix(matrix)).normalize()
    expect(xyz(output.getAttribute('normal'), 0).distanceTo(normal)).toBeLessThan(1e-6)
    expect(output.getAttribute('tangent').getW(0)).toBe(-1)
    const a = xyz(output.getAttribute('position'), 0)
    const b = xyz(output.getAttribute('position'), 2)
    const c = xyz(output.getAttribute('position'), 1)
    expect(b.sub(a).cross(c.sub(a)).normalize().dot(normal)).toBeGreaterThan(0.99999)
    expect(Array.from(output.getAttribute('uv').array)).toEqual([0, 0, 1, 0, 0, 1])
    output.dispose()
    source.dispose()
  })
  it('bakes nested nodes and retains each material primitive', () => {
    const root = new Group()
    root.position.set(5, 0, 0)
    const nested = new Group()
    nested.scale.set(2, 3, 4)
    root.add(nested)
    const geometry = new BufferGeometry()
    geometry.setAttribute('position', new Float32BufferAttribute([0, 0, 0, 1, 0, 0, 0, 1, 0], 3))
    geometry.setIndex([0, 1, 2, 2, 1, 0])
    geometry.addGroup(0, 3, 0)
    geometry.addGroup(3, 3, 1)
    const materials = [new MeshStandardMaterial(), new MeshStandardMaterial()]
    const mesh = new Mesh(geometry, materials)
    mesh.position.set(1, 0, 0)
    nested.add(mesh)
    const parts = preparePropParts(root)
    expect(parts.map(part => part.material)).toEqual(materials)
    expect(parts.map(part => part.geometry.drawRange)).toEqual([{ start: 0, count: 3 }, { start: 3, count: 3 }])
    expect(parts[0]?.geometry.getAttribute('position').getX(0)).toBe(7)
    expect(geometry.getAttribute('position').getX(0)).toBe(0)
    for (const part of parts) part.geometry.dispose()
    geometry.dispose()
    for (const material of materials) material.dispose()
  })
  it('rejects singular transforms', () => {
    expect(() => bakeGeometry(new BufferGeometry(), new Matrix4().makeScale(0, 1, 1))).toThrow(/invertible/)
  })
})
