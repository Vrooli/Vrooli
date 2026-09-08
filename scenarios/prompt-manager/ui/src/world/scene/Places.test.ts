import * as THREE from 'three'
import { createObstacleSweep } from '../engine/camera/obstacles'
import { roofGeometry } from './roofGeometry'
import { describe, expect, it } from 'vitest'
import { tuning } from '../config'
import type { Place } from '../sim'
import { buildRoomSlabs } from './Places'

describe('buildRoomSlabs', () => {
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

  it('preserves default slab dimensions and responds to configured surface dimensions', () => {
    const room: Place = { id: 'room:a', kind: 'room', teamId: 'a', position: [2, 3], rotation: 0, size: [8, 6], seats: [], label: 'A' }
    const original = buildRoomSlabs([room], [], tuning.layout, () => 4, true)
    expect(original.floors[0]).toMatchObject({ position: [2, 4.012, 3], scale: [8, 0.02, 6] })
    expect(original.walls[0]?.scale).toEqual([8.18, tuning.layout.wallHeight, 0.18])
    const layout = { ...tuning.layout, surfaces: { ...tuning.layout.surfaces, wallThickness: 0.3, floorLift: 0.05, floorThickness: 0.08, doorFrameScale: 2 } }
    const changed = buildRoomSlabs([room], [], layout, () => 4, true)
    expect(changed.floors[0]).toMatchObject({ position: [2, 4.05, 3], scale: [8, 0.08, 6] })
    expect(changed.walls[0]?.scale).toEqual([8.3, layout.wallHeight, 0.3])
    expect(changed.walls.find((wall) => wall.key.includes(':door-jamb:'))?.scale).toEqual([0.6, layout.wallHeight, 0.6])
  })

  it('encloses an indoor room around the configured doorway gap and frame', () => {
    const room: Place = { id: 'room:a', kind: 'room', teamId: 'a', position: [0, 0], rotation: 0, size: [8, 6], seats: [], label: 'A' }
    const door: Place = { id: 'door:a', kind: 'door', teamId: 'a', parentId: room.id, position: [0, 3], rotation: 0, size: [tuning.layout.floorplan.doorWidth, tuning.layout.cellSize], seats: [], label: 'A door' }
    const { walls, floors } = buildRoomSlabs([room], [door], tuning.layout, () => 0, true)
    expect(floors).toHaveLength(1)
    expect(walls.filter((wall) => wall.key.includes(':front:'))).toHaveLength(2)
    expect(walls.filter((wall) => wall.key.includes(':door-jamb:'))).toHaveLength(2)
    expect(walls.some((wall) => wall.key.endsWith(':door-lintel'))).toBe(true)
    const frontWidth = walls.filter((wall) => wall.key.includes(':front:')).reduce((sum, wall) => sum + wall.scale[0], 0)
    expect(room.size[0] - frontWidth).toBeCloseTo(door.size[0])
  })
})
