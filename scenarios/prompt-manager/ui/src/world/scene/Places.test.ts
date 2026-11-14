import * as THREE from 'three'
import { createObstacleSweep } from '../engine/camera/obstacles'
import { roofGeometry } from './roofGeometry'
import { describe, expect, it } from 'vitest'

describe('authored shelter geometry', () => {
  it.each([false, true])('contains only nondegenerate roof triangles (tent=%s)', tent => {
    const geometry = roofGeometry(tent), positions = geometry.getAttribute('position')
    const triangle = new THREE.Triangle()
    for (let i = 0; i < positions.count; i += 3) {
      triangle.a.fromBufferAttribute(positions, i); triangle.b.fromBufferAttribute(positions, i + 1); triangle.c.fromBufferAttribute(positions, i + 2)
      expect(triangle.getArea(), `triangle ${i / 3} can produce stable surface normals`).toBeGreaterThan(1e-6)
    }
    geometry.dispose()
  })
  it('keeps the rendered tent opening clear while blocking its slopes, rear gable and jumping through its roof', () => {
    const geometry = roofGeometry(true), material = new THREE.MeshBasicMaterial()
    for (const yaw of [0, Math.PI / 3, Math.PI]) {
      const root = new THREE.Group(), mesh = new THREE.InstancedMesh(geometry, material, 1)
      const transform = new THREE.Matrix4().makeRotationY(yaw).setPosition(8, 2, -3)
      const roof = new THREE.Matrix4().makeScale(5.3, 2.45, 5.2).setPosition(0, .65, 0)
      mesh.setMatrixAt(0, transform.clone().multiply(roof)); root.add(mesh)
      const point = (x: number, y: number, z: number) => new THREE.Vector3(x, y, z).applyMatrix4(transform)
      const sweep = createObstacleSweep(root)
      expect(sweep(point(0, 1, 4), point(0, 1, 1.5), .3, 1.2)).toBe(1)
      expect(sweep.overlaps(point(0, 1, 0), .3, 1.2)).toBe(false)
      expect(sweep(point(0, 1, 0), point(2.65, 1, 0), .3, 1.2)).toBeLessThan(.8)
      expect(sweep(point(0, 1, 0), point(0, 1, -4), .3, 1.2)).toBeLessThan(.7)
      expect(sweep(point(0, 1, 0), point(0, 3.5, 0), .3, 1.2)).toBeLessThan(.6)
      expect(sweep.overlaps(point(2, 1, 0), .3, 1.2)).toBe(true)
      const onRoof = point(1, 2.175, 0)
      const lift = sweep.recover(onRoof, .1)
      expect(lift).toBeGreaterThan(0)
      expect(sweep.overlaps(onRoof.clone().add(new THREE.Vector3(0, lift, 0)), .1)).toBe(false)
      mesh.dispose()
    }
    geometry.dispose(); material.dispose()
  })

})
