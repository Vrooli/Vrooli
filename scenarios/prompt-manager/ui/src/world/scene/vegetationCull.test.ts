import { describe, expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import ts from 'typescript'
/* eslint-disable @typescript-eslint/no-non-null-assertion -- Test fixtures and oracle indices have known bounds. */
import { Frustum, Matrix4, PerspectiveCamera, Vector3 } from 'three'
import { VegetationBuffer, VegetationCuller, type VegetationCullItem } from './vegetationCull'

const allocations = vi.hoisted(() => ({ count: 0 }))
vi.mock('three', async (original) => {
  const three = await original<typeof import('three')>()
  const tracked = { ...three }
  for (const name of ['Vector3', 'Quaternion', 'Matrix4', 'Sphere'] as const) {
    Object.defineProperty(tracked, name, { value: new Proxy(three[name], {
      construct(target, args) {
        allocations.count += 1
        return Reflect.construct(target, args)
      },
    }) })
  }
  return tracked
})

function item(key: string, x: number, z: number, group = 0): VegetationCullItem {
  return { key, group, center: [x, 0, z], radius: 0.5, matrix: new Float32Array(new Matrix4().makeTranslation(x, 0, z).elements) }
}
function cameraAtRest() {
  const camera = new PerspectiveCamera(60, 1, 0.1, 100)
  camera.position.set(0, 0, 10)
  camera.lookAt(0, 0, 0)
  camera.updateMatrixWorld()
  return camera
}
function frustum(camera: PerspectiveCamera) {
  return new Frustum().setFromProjectionMatrix(new Matrix4().multiplyMatrices(camera.projectionMatrix, camera.matrixWorldInverse))
}

describe('global vegetation cull', () => {
  it('has no direct frame-path allocation expressions or collection transforms', () => {
    const source = ts.createSourceFile('vegetationCull.ts', readFileSync('src/world/scene/vegetationCull.ts', 'utf8'), ts.ScriptTarget.Latest, true)
    const failures: string[] = []
    const inspect = (node: ts.Node) => {
      if (ts.isNewExpression(node) || ts.isObjectLiteralExpression(node) || ts.isArrayLiteralExpression(node)
        || ts.isArrowFunction(node) || ts.isFunctionExpression(node)) failures.push(node.getText(source))
      if (ts.isCallExpression(node) && ts.isPropertyAccessExpression(node.expression)
        && ['map', 'filter', 'slice', 'subarray', 'sort', 'flatMap'].includes(node.expression.name.text)) failures.push(node.getText(source))
      ts.forEachChild(node, inspect)
    }
    for (const statement of source.statements) {
      if (!ts.isClassDeclaration(statement)) continue
      for (const member of statement.members) {
        if (ts.isMethodDeclaration(member) && member.body) inspect(member.body)
      }
    }
    expect(failures).toEqual([])
  })
  it('constructs no vectors, matrices, spheres or typed buffers during 100 moving frames', () => {
    const camera = cameraAtRest()
    const buffer = new VegetationBuffer(1)
    const culler = new VegetationCuller([item('tree', 0, 0)], [buffer], 1)
    for (const name of ['Float32Array', 'Float64Array', 'Int32Array', 'Uint8Array'] as const) {
      vi.stubGlobal(name, new Proxy(globalThis[name], {
        construct(target, args) {
          allocations.count += 1
          return Reflect.construct(target, args)
        },
      }))
    }
    const before = allocations.count
    try {
      for (let frame = 0; frame < 100; frame += 1) {
        camera.position.x = frame / 10
        camera.updateMatrixWorld()
        culler.update(camera, 0, 0)
      }
      expect(culler.runs).toBe(100)
      expect(allocations.count - before).toBe(0)
    } finally {
      vi.unstubAllGlobals()
    }
  })
  it('compacts visible matrices and excludes behind-camera items', () => {
    const buffer = new VegetationBuffer(3)
    const culler = new VegetationCuller([item('left', -2, 0), item('hidden', 0, 20), item('right', 2, 0)], [buffer], 3)
    culler.update(cameraAtRest(), 0.05, 0.002)
    expect(buffer.count).toBe(2)
    expect([buffer.matrices[12], buffer.matrices[28]]).toEqual([-2, 2])
  })
  it('applies one nearest budget across groups and copies matching colors', () => {
    const buffers = [new VegetationBuffer(2), new VegetationBuffer(2)]
    const near = { ...item('near', 0, 5), color: [0.9, 1, 0.8] as const }
    new VegetationCuller([item('far', 0, -5), near, item('middle', 0, 0, 1)], buffers, 2).update(cameraAtRest(), 0, 0)
    expect(buffers.map(b => b.count)).toEqual([1, 1])
    expect(buffers[0]!.matrices[14]).toBe(5)
    expect(buffers[0]!.colors[0]).toBeCloseTo(0.9)
    expect(buffers[1]!.colors.slice(0, 3)).toEqual(new Float32Array([1, 1, 1]))
  })
  it('retains layout order for an equal-distance budget edge', () => {
    const buffer = new VegetationBuffer(1)
    new VegetationCuller([item('first', -1, 0), item('second', 1, 0)], [buffer], 1).update(cameraAtRest(), 0, 0)
    expect(buffer.matrices[12]).toBe(-1)
  })
  it('skips static frames, accumulates sub-epsilon movement, and notices projection and rotation changes', () => {
    const camera = cameraAtRest()
    const buffer = new VegetationBuffer(1)
    const matrices = buffer.matrices
    const colors = buffer.colors
    const culler = new VegetationCuller([item('tree', 0, 0)], [buffer], 1)
    expect(culler.update(camera, 0.05, 0.002)).toBe(true)
    for (let i = 0; i < 100; i += 1) expect(culler.update(camera, 0.05, 0.002)).toBe(false)
    camera.position.x = 0.03
    camera.updateMatrixWorld()
    expect(culler.update(camera, 0.05, 0.002)).toBe(false)
    camera.position.x = 0.06
    camera.updateMatrixWorld()
    expect(culler.update(camera, 0.05, 0.002)).toBe(true)
    camera.aspect = 2
    camera.updateProjectionMatrix()
    expect(culler.update(camera, 0.05, 0.002)).toBe(true)
    camera.rotateY(0.01)
    camera.updateMatrixWorld()
    expect(culler.update(camera, 0.05, 0.002)).toBe(true)
    expect(culler.runs).toBe(4)
    expect(culler.skips).toBe(101)
    expect(buffer.matrices).toBe(matrices)
    expect(buffer.colors).toBe(colors)
  })
  it('matches an exhaustive nearest oracle for many budgets without sorting production buffers', () => {
    const camera = cameraAtRest()
    const visible = frustum(camera)
    const items = Array.from({ length: 2000 }, (_, i) => item(String(i), Math.sin(i * 3) * 30, Math.cos(i * 7) * 30))
    const candidates = items.map((entry, index) => ({ index, distance: camera.position.distanceToSquared(new Vector3(...entry.center)) }))
      .filter(({ index }) => {
        const entry = items[index]!
        // Independent plane test for bounding-sphere visibility.
        return visible.planes.every(plane => plane.distanceToPoint(new Vector3(...entry.center)) >= -entry.radius)
      }).sort((a, b) => a.distance - b.distance || a.index - b.index)
    for (const budget of [0, 1, 2, 7, 100, 1000, 2000]) {
      const buffer = new VegetationBuffer(budget)
      new VegetationCuller(items, [buffer], budget).update(camera, 0, 0)
      const expected = candidates.slice(0, budget).sort((a, b) => a.index - b.index)
      expect(buffer.count).toBe(expected.length)
      expected.forEach(({ index }, slot) => {
        expect(buffer.matrices[slot * 16 + 12]).toBe(items[index]!.matrix[12])
        expect(buffer.matrices[slot * 16 + 14]).toBe(items[index]!.matrix[14])
      })
    }
  })
})
