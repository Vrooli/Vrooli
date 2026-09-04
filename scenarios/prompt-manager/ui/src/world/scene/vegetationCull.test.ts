import { describe, expect, it } from 'vitest'
import { Frustum, Matrix4, PerspectiveCamera, Vector3 } from 'three'
import { cullVegetation, matrixBufferChanged, visibleVegetationKeys, type VegetationCullItem } from './vegetationCull'

function item(key: string, x: number, z: number): VegetationCullItem {
  return { key, center: [x, 0, z], radius: 0.5, matrix: new Float32Array(new Matrix4().makeTranslation(x, 0, z).elements) }
}

function cameraFrustum(): { frustum: Frustum; position: Vector3 } {
  const camera = new PerspectiveCamera(60, 1, 0.1, 100)
  camera.position.set(0, 0, 10)
  camera.lookAt(0, 0, 0)
  camera.updateMatrixWorld(true)
  camera.updateProjectionMatrix()
  return { frustum: new Frustum().setFromProjectionMatrix(new Matrix4().multiplyMatrices(camera.projectionMatrix, camera.matrixWorldInverse)), position: camera.position }
}

describe('cullVegetation', () => {
  it('compacts the visible subset and excludes instances behind the camera', () => {
    const { frustum, position } = cameraFrustum()
    const target = new Float32Array(3 * 16)
    const count = cullVegetation([item('left', -2, 0), item('behind', 0, 20), item('right', 2, 0)], frustum, position, target, 3)
    expect(count).toBe(2)
    expect([target[12], target[28]]).toEqual([-2, 2])
  })

  it('keeps the nearest visible instances under the budget', () => {
    const { frustum, position } = cameraFrustum()
    const target = new Float32Array(2 * 16)
    const count = cullVegetation([item('far', 0, -5), item('near', 0, 5), item('middle', 0, 0)], frustum, position, target, 2)
    expect(count).toBe(2)
    expect([target[14], target[30]]).toEqual([5, 0])
  })

  it('compacts instance colours in the same order as matrices', () => {
    const { frustum, position } = cameraFrustum()
    const near = { ...item('near', 0, 5), color: [0.9, 1, 0.8] as const }
    const far = { ...item('far', 0, -5), color: [1, 0.8, 0.9] as const }
    const target = new Float32Array(16)
    const colors = new Float32Array(3)
    expect(cullVegetation([far, near], frustum, position, target, 1, undefined, colors)).toBe(1)
    expect([...colors]).toEqual([expect.closeTo(0.9), 1, expect.closeTo(0.8)])
  })

  it('applies one nearest-first budget across prop groups', () => {
    const { frustum, position } = cameraFrustum()
    const selected = visibleVegetationKeys([item('oak:far', 0, -5), item('pine:near', 0, 5), item('oak:middle', 0, 0)], frustum, position, 2)
    expect([...selected]).toEqual(['pine:near', 'oak:middle'])
    const oak = new Float32Array(2 * 16)
    const pine = new Float32Array(2 * 16)
    expect(cullVegetation([item('oak:far', 0, -5), item('oak:middle', 0, 0)], frustum, position, oak, 2, selected)).toBe(1)
    expect(cullVegetation([item('pine:near', 0, 5)], frustum, position, pine, 2, selected)).toBe(1)
  })
})

describe('matrixBufferChanged', () => {
  it('detects count and visible-matrix changes only', () => {
    const previous = new Float32Array(32)
    const next = new Float32Array(32)
    expect(matrixBufferChanged(previous, next, 2, 2)).toBe(false)
    next[31] = 1
    expect(matrixBufferChanged(previous, next, 2, 2)).toBe(true)
    expect(matrixBufferChanged(previous, previous, 1, 2)).toBe(true)
  })
})
