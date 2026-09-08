import { describe, expect, it } from 'vitest'
import CameraControls from 'camera-controls'
import * as THREE from 'three'
import { createObstacleSweep } from './obstacles'
import { WorldCameraControls } from './input'
import { nearPlaneRadius } from './pose'

CameraControls.install({ THREE })

describe('room obstacle sweeps', () => {
  it('does not collide with effect cards nested beside solid furniture', () => {
    const group = new THREE.Group(); group.userData.walkObstacle = true
    const effect = new THREE.Mesh(new THREE.PlaneGeometry(2, 4), new THREE.MeshBasicMaterial())
    effect.userData.walkObstacle = false; group.add(effect)
    const sweep = createObstacleSweep(group, undefined, true)
    expect(sweep(new THREE.Vector3(0, 0, 2), new THREE.Vector3(0, 0, -2), .3, 1.2)).toBe(1)
    effect.userData.walkObstacle = true
    expect(sweep(new THREE.Vector3(0, 0, 2), new THREE.Vector3(0, 0, -2), .3, 1.2)).toBeLessThan(1)
    effect.geometry.dispose(); (effect.material as THREE.Material).dispose()
  })
  it('outlines the same visible instanced boxes expanded by camera clearance', () => {
    const root = new THREE.Group()
    const geometry = new THREE.BoxGeometry(2, 2, 2)
    geometry.userData.cameraObstacle = 'box'
    const material = new THREE.MeshBasicMaterial()
    const mesh = new THREE.InstancedMesh(geometry, material, 1)
    mesh.setMatrixAt(0, new THREE.Matrix4().makeScale(2, 1, 1).setPosition(10, 0, 0))
    root.add(mesh)
    const sweep = createObstacleSweep(root)
    const outline = sweep.outlines(0.5)
    expect(outline.count).toBe(1)
    expect(outline.positions.length).toBe(12 * 2 * 3)
    const bounds = new THREE.Box3().setFromBufferAttribute(new THREE.BufferAttribute(outline.positions, 3))
    expect(bounds.min.toArray()).toEqual([7.5, -1.5, -1.5])
    expect(bounds.max.toArray()).toEqual([12.5, 1.5, 1.5])
    expect(sweep(new THREE.Vector3(0, 0, 0), new THREE.Vector3(10, 0, 0), 0.5)).toBeCloseTo(0.75, 4)
    mesh.visible = false
    expect(sweep.outlines(0.5).count).toBe(0)
    mesh.dispose(); geometry.dispose(); material.dispose()
  })
  it('recovers through overlapping boxes to the first clear vertical gap', () => {
    const root = new THREE.Group()
    root.rotation.y = 0.7
    const geometry = new THREE.BoxGeometry(2, 2, 2)
    geometry.userData.cameraObstacle = 'box'
    const material = new THREE.MeshBasicMaterial()
    const boxes = new THREE.InstancedMesh(geometry, material, 3)
    for (const [index, y] of [0, 2, 6].entries()) boxes.setMatrixAt(index, new THREE.Matrix4().makeTranslation(0, y, 0))
    root.add(boxes)
    const sweep = createObstacleSweep(root)
    const position = new THREE.Vector3()
    const lift = sweep.recover(position, 0.5)
    expect(lift).toBeCloseTo(3.5, 4)
    position.y += lift
    expect(sweep.recover(position, 0.5)).toBe(0)
    expect(sweep.recover(new THREE.Vector3(10, 0, 0), 0.5)).toBe(0)
    boxes.dispose(); geometry.dispose(); material.dispose()
  })

  it.each([{ raisedTerrain: false, initial: false }, { raisedTerrain: true, initial: false }, { raisedTerrain: false, initial: true }])('recovers a stationary camera with raised terrain=$raisedTerrain and initial frame=$initial', ({ raisedTerrain, initial }) => {
    const camera = new THREE.PerspectiveCamera(38, 1.6, 0.5, 400)
    const controls = new WorldCameraControls(camera)
    const root = new THREE.Group()
    const geometry = new THREE.BoxGeometry(4, 4, 4)
    geometry.userData.cameraObstacle = 'box'
    const material = new THREE.MeshBasicMaterial()
    const wall = new THREE.Mesh(geometry, material)
    wall.position.set(5, raisedTerrain ? 5 : 0, 0)
    root.add(wall)
    const sweep = createObstacleSweep(root)
    controls.obstacleSweep = sweep
    controls.obstacleRecovery = sweep.recover
    let floor = -10
    controls.groundCeiling = () => floor
    void controls.setLookAt(-5, 1, 0, -5, 1, -10, false)
    if (!initial) controls.update(1 / 60)
    const before = controls.getPosition(new THREE.Vector3(), false)
    const targetBefore = controls.getTarget(new THREE.Vector3(), false)
    const direction = targetBefore.clone().sub(before).normalize()
    wall.position.x = -5
    if (raisedTerrain) floor = 3
    controls.update(1 / 60)
    const radius = nearPlaneRadius(camera.near, camera.fov, camera.aspect, camera.zoom)
    expect(camera.position.y).toBeCloseTo((raisedTerrain ? 7 : 2) + radius, 4)
    expect(camera.position.x).toBe(before.x)
    expect(camera.position.z).toBe(before.z)
    expect(camera.getWorldDirection(new THREE.Vector3()).distanceTo(direction)).toBeLessThan(1e-8)
    expect(controls.getTarget(new THREE.Vector3(), false).sub(targetBefore).distanceTo(camera.position.clone().sub(before))).toBeLessThan(1e-8)
    const recovered = camera.position.clone()
    for (let frame = 0; frame < 120; frame++) controls.update(1 / 60)
    expect(camera.position.distanceTo(recovered)).toBeLessThan(1e-8)
    controls.dispose(); geometry.dispose(); material.dispose()
  })

  it.each([0, 0.7])('blocks a thin transformed wall with clear endpoints at yaw %s', yaw => {
    const root = new THREE.Group()
    root.rotation.y = yaw
    const geometry = new THREE.BoxGeometry(1, 1, 1)
    const material = new THREE.MeshBasicMaterial()
    const wall = new THREE.InstancedMesh(geometry, material, 1)
    geometry.userData.cameraObstacle = 'box'
    wall.setMatrixAt(0, new THREE.Matrix4().makeScale(0.2, 10, 20))
    root.add(wall)
    const rotation = new THREE.Matrix4().makeRotationY(yaw)
    const from = new THREE.Vector3(-5, 0, 0).applyMatrix4(rotation)
    const to = new THREE.Vector3(5, 0, 0).applyMatrix4(rotation)
    const sweep = createObstacleSweep(root)
    const fraction = sweep(from, to, 0.5)
    expect(fraction).toBeCloseTo(0.44, 5)
    const stopped = from.clone().lerp(to, fraction)
    expect(sweep(stopped, from, 0.5)).toBe(1)
    expect(sweep(stopped, to, 0.5)).toBeLessThan(1e-5)
    const above = new THREE.Vector3(0, 7, 0)
    expect(sweep(from.clone().add(above), to.clone().add(above), 0.5)).toBe(1)
    wall.visible = false
    expect(sweep(from, to, 0.5)).toBe(1)
    wall.dispose(); geometry.dispose(); material.dispose()
  })

  it('allows escape from a newly overlapping box but rejects deeper movement', () => {
    const root = new THREE.Group()
    const geometry = new THREE.BoxGeometry(2, 2, 2)
    const material = new THREE.MeshBasicMaterial()
    const wall = new THREE.Mesh(geometry, material)
    geometry.userData.cameraObstacle = 'box'
    root.add(wall)
    const sweep = createObstacleSweep(root)
    const from = new THREE.Vector3(1.2, 0, 0)
    expect(sweep(from, new THREE.Vector3(3, 0, 0), 0.5)).toBe(1)
    expect(sweep(from, new THREE.Vector3(-3, 0, 0), 0.5)).toBe(0)
    geometry.userData.cameraObstacle = undefined
    expect(sweep(from, new THREE.Vector3(-3, 0, 0), 0.5)).toBe(1)
    geometry.dispose(); material.dispose()
  })

  it.each([{ smooth: false, mixed: false }, { smooth: true, mixed: false }, { smooth: true, mixed: true }])('protects every controller frame during a wall crossing with smoothing=$smooth and mixed input=$mixed', ({ smooth, mixed }) => {
    const camera = new THREE.PerspectiveCamera(38, 1.6, 0.5, 400)
    const controls = new WorldCameraControls(camera)
    const root = new THREE.Group()
    const geometry = new THREE.BoxGeometry(0.2, 10, 20)
    const material = new THREE.MeshBasicMaterial()
    const wall = new THREE.Mesh(geometry, material)
    geometry.userData.cameraObstacle = 'box'
    root.add(wall)
    controls.obstacleSweep = createObstacleSweep(root)
    const radius = nearPlaneRadius(camera.near, camera.fov, camera.aspect, camera.zoom)
    void controls.setLookAt(-5, 0, 0, -5, 0, -10, false)
    controls.update(1 / 60)
    void controls.moveTo(5, 0, -10, smooth)
    if (mixed) void controls.rotate(0.5, 0, true)
    for (let frame = 0; frame < 240; frame++) {
      controls.update(1 / 60)
      expect(camera.position.x).toBeLessThanOrEqual(-0.1 - radius + 1e-7)
    }
    expect(camera.position.x).toBeCloseTo(-0.1 - radius, 4)
    const yaw = controls.azimuthAngle
    void controls.rotate(0.5, 0, true)
    for (let frame = 0; frame < 120; frame++) {
      controls.update(1 / 60)
      expect(camera.position.x).toBeLessThanOrEqual(-0.1 - radius + 1e-7)
    }
    expect(Math.abs(controls.azimuthAngle - yaw)).toBeGreaterThan(0.4)
    const beforeReverse = camera.position.x
    void controls.truck(-2, 0, false)
    controls.update(1 / 60)
    expect(camera.position.x).toBeLessThan(beforeReverse)
    controls.dispose(); geometry.dispose(); material.dispose()
  })
})
