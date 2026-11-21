import { BufferGeometry, Float32BufferAttribute, Mesh, SkinnedMesh, type Material, type Matrix4, type Object3D } from 'three'

export interface PropPart {
  geometry: BufferGeometry
  material: Material
}

/** One caller-owned geometry per material primitive; materials stay loader-owned. */
export function preparePropParts(root: Object3D): PropPart[] {
  const parts: PropPart[] = []
  root.updateMatrixWorld(true)
  try {
    root.traverse(object => {
      if (!(object instanceof Mesh)) return
      if (object instanceof SkinnedMesh) throw new Error('Skinned geometry cannot be baked as a static prop')
      const mesh = object as Mesh
      const geometry = bakeGeometry(mesh.geometry, mesh.matrixWorld)
      if (!Array.isArray(mesh.material)) {
        parts.push({ geometry, material: mesh.material })
        return
      }
      try {
        for (const group of geometry.groups) {
          const material = mesh.material[group.materialIndex ?? 0]
          if (!material) throw new Error('Static prop primitive references a missing material')
          const start = Math.max(group.start, geometry.drawRange.start)
          const end = Math.min(group.start + group.count, geometry.drawRange.start + geometry.drawRange.count)
          if (end <= start) continue
          const primitive = geometry.clone()
          primitive.clearGroups()
          primitive.setDrawRange(start, end - start)
          parts.push({ geometry: primitive, material })
        }
      } finally {
        geometry.dispose()
      }
    })
    if (parts.length === 0) throw new Error('Static prop has no renderable mesh primitives')
    return parts
  } catch (error) {
    for (const part of parts) part.geometry.dispose()
    throw error
  }
}

/** Bake static mesh coordinates without writing transformed values into quantized buffers.
 * The caller owns the returned geometry; the GLTF loader retains the source.
 */
export function bakeGeometry(source: BufferGeometry, matrix: Matrix4): BufferGeometry {
  if (Object.keys(source.morphAttributes).length > 0) throw new Error('Animated morph geometry cannot be baked as a static prop')
  if (!matrix.elements.every(Number.isFinite) || Math.abs(matrix.determinant()) < 1e-12) {
    throw new Error('Static prop transform must be finite and invertible')
  }
  const geometry = source.clone()
  try {
    // Attribute accessors decode normalized and interleaved data before conversion.
    for (const name of ['position', 'normal', 'tangent']) {
      if (!source.hasAttribute(name)) continue
      const attribute = source.getAttribute(name)
      const values = new Float32Array(attribute.count * attribute.itemSize)
      for (let vertex = 0; vertex < attribute.count; vertex++) {
        for (let component = 0; component < attribute.itemSize; component++) {
          values[vertex * attribute.itemSize + component] = attribute.getComponent(vertex, component)
        }
      }
      geometry.setAttribute(name, new Float32BufferAttribute(values, attribute.itemSize))
    }
    if (!geometry.hasAttribute('position')) throw new Error('Static prop has no positions')
    // Three applies inverse-transpose to normals and the linear transform to tangents.
    geometry.applyMatrix4(matrix)
    if (matrix.determinant() < 0) {
      // Mirroring changes triangle orientation and tangent-space handedness.
      // Use an index even for nonindexed sources so UVs and all vertex attributes
      // remain attached to their original vertices.
      const count = geometry.index?.count ?? geometry.getAttribute('position').count
      const indices = Array.from({ length: count }, (_, i) => geometry.index?.getX(i) ?? i)
      for (let i = 0; i + 2 < count; i += 3) {
        const second = indices[i + 1] ?? 0
        indices[i + 1] = indices[i + 2] ?? 0
        indices[i + 2] = second
      }
      geometry.setIndex(indices)
      const tangent = geometry.getAttribute('tangent')
      if (geometry.hasAttribute('tangent')) for (let i = 0; i < tangent.count; i++) tangent.setW(i, -tangent.getW(i))
    }
    geometry.computeBoundingBox()
    geometry.computeBoundingSphere()
    return geometry
  } catch (error) {
    geometry.dispose()
    throw error
  }
}
