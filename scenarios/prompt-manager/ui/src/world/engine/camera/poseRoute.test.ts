import { describe, expect, it, vi } from 'vitest'
import CameraControls from 'camera-controls'
import * as THREE from 'three'
import { PoseRoute, planPoseRoute } from './poseRoute'
import { WorldCameraControls } from './input'
import { createObstacleSweep } from './obstacles'
import { nearPlaneRadius } from './pose'

CameraControls.install({ THREE })

function fixture() {
  const camera = new THREE.PerspectiveCamera(38, 1.6, 0.5, 400)
  const controls = new WorldCameraControls(camera)
  const root = new THREE.Group()
  const geometry = new THREE.BoxGeometry(0.2, 4, 10)
  geometry.userData.cameraObstacle = 'box'
  const material = new THREE.MeshBasicMaterial()
  const wall = new THREE.Mesh(geometry, material)
  root.add(wall)
  const obstacles = createObstacleSweep(root)
  controls.obstacleSweep = obstacles
  controls.obstacleRecovery = obstacles.recover
  controls.obstacleCeiling = obstacles.ceiling
  controls.obstacleBounds = obstacles.bounds
  controls.groundCeiling = () => 0
  void controls.setLookAt(-5, 1, 0, -5, 1, -5, false)
  controls.update(1 / 60)
  return { camera, controls, wall, root, dispose: () => { controls.dispose(); geometry.dispose(); material.dispose() } }
}

describe('automatic pose routes', () => {
  it('uses one total duration across a detour while turning to a different view direction', () => {
    const f = fixture()
    const position = new THREE.Vector3(5, 1, 0)
    const target = new THREE.Vector3(5, 3, 5)
    const complete = vi.fn()
    const blocked = vi.fn()
    const route = new PoseRoute(f.controls, position, target, 0.6, complete, blocked)
    let frames = 0
    while (complete.mock.calls.length === 0 && frames < 180) {
      route.tick(1 / 60)
      f.controls.update(1 / 60)
      frames++
    }
    expect(complete).toHaveBeenCalledTimes(1)
    expect(blocked).not.toHaveBeenCalled()
    expect(frames / 60).toBeGreaterThanOrEqual(0.6)
    expect(frames / 60).toBeLessThanOrEqual(0.68)
    expect(f.camera.position.distanceTo(position)).toBeLessThan(1e-4)
    expect(f.controls.getTarget(new THREE.Vector3(), false).distanceTo(target)).toBeLessThan(1e-4)
    f.dispose()
  })
  it('reports an unavailable route without completing or writing an unsafe pose', () => {
    const f = fixture()
    f.controls.obstacleSweep = () => 0
    const complete = vi.fn()
    const blocked = vi.fn()
    const before = f.camera.position.clone()
    const route = new PoseRoute(f.controls, new THREE.Vector3(5, 1, 0), new THREE.Vector3(5, 1, -5), 0.3, complete, blocked)
    for (let frame = 0; frame < 120; frame++) { route.tick(1 / 60); f.controls.update(1 / 60) }
    expect(blocked).toHaveBeenCalledTimes(1)
    expect(complete).not.toHaveBeenCalled()
    expect(f.camera.position.distanceTo(before)).toBeLessThan(1e-8)
    f.dispose()
  })
  it('recovers a newly enclosed origin before planning focus', () => {
    const f = fixture()
    f.wall.position.x = -5
    const complete = vi.fn()
    const blocked = vi.fn()
    const position = new THREE.Vector3(5, 1, 0)
    const route = new PoseRoute(f.controls, position, new THREE.Vector3(5, 1, -5), 0.3, complete, blocked)
    for (let frame = 0; frame < 180; frame++) { route.tick(1 / 60); f.controls.update(1 / 60) }
    expect(complete).toHaveBeenCalledTimes(1)
    expect(blocked).not.toHaveBeenCalled()
    expect(f.camera.position.distanceTo(position)).toBeLessThan(1e-4)
    f.dispose()
  })
  it('routes around the obstruction when an overhead slab blocks ascent', () => {
    const f = fixture()
    const geometry = new THREE.BoxGeometry(20, 0.2, 20)
    geometry.userData.cameraObstacle = 'box'
    const roof = new THREE.Mesh(geometry, f.wall.material)
    roof.position.y = 3
    f.root.add(roof)
    const position = new THREE.Vector3(5, 1, 0)
    const target = new THREE.Vector3(5, 1, -5)
    const path = planPoseRoute(f.controls, position, target)
    expect(path).toHaveLength(3)
    expect(Math.abs(path?.[0]?.position.z ?? 0)).toBeGreaterThan(10)
    const complete = vi.fn()
    const blocked = vi.fn()
    const route = new PoseRoute(f.controls, position, target, 0.3, complete, blocked)
    for (let frame = 0; frame < 180; frame++) { route.tick(1 / 60); f.controls.update(1 / 60) }
    expect(complete).toHaveBeenCalledTimes(1)
    expect(blocked).not.toHaveBeenCalled()
    expect(f.camera.position.distanceTo(position)).toBeLessThan(1e-4)
    geometry.dispose(); f.dispose()
  })
  it('reaches the requested framing over a wall without accepting intermediate rest as completion', () => {
    const f = fixture()
    const position = new THREE.Vector3(5, 1, 0)
    const target = new THREE.Vector3(5, 1, -5)
    expect(planPoseRoute(f.controls, position, target)).toHaveLength(3)
    const complete = vi.fn()
    const blocked = vi.fn()
    const route = new PoseRoute(f.controls, position, target, 0.3, complete, blocked)
    const radius = nearPlaneRadius(f.camera.near, f.camera.fov, f.camera.aspect)
    f.controls.dispatchEvent({ type: 'rest' })
    expect(complete).not.toHaveBeenCalled()
    for (let frame = 0; frame < 180; frame++) {
      route.tick(1 / 60)
      f.controls.update(1 / 60)
      if (Math.abs(f.camera.position.x) <= 0.1 + radius) expect(f.camera.position.y).toBeGreaterThanOrEqual(2 + radius)
      expect(f.camera.position.y).toBeGreaterThanOrEqual(radius)
    }
    expect(complete).toHaveBeenCalledTimes(1)
    expect(blocked).not.toHaveBeenCalled()
    expect(f.camera.position.distanceTo(position)).toBeLessThan(1e-4)
    expect(f.controls.getTarget(new THREE.Vector3(), false).distanceTo(target)).toBeLessThan(1e-4)
    f.dispose()
  })

  it('cancels a route without a late pose write or completion', () => {
    const f = fixture()
    const complete = vi.fn()
    const route = new PoseRoute(f.controls, new THREE.Vector3(5, 1, 0), new THREE.Vector3(5, 1, -5), 0.3, complete, vi.fn())
    for (let frame = 0; frame < 5; frame++) { route.tick(1 / 60); f.controls.update(1 / 60) }
    route.cancel()
    const stopped = f.camera.position.clone()
    for (let frame = 0; frame < 180; frame++) { route.tick(1 / 60); f.controls.update(1 / 60) }
    expect(f.camera.position.distanceTo(stopped)).toBeLessThan(1e-8)
    expect(complete).not.toHaveBeenCalled()
    f.dispose()
  })

  it('uses a direct clear route and completes reduced-motion detours at the guarded destination', () => {
    const f = fixture()
    expect(planPoseRoute(f.controls, new THREE.Vector3(-3, 1, 0), new THREE.Vector3(-3, 1, -5))).toHaveLength(1)
    const complete = vi.fn()
    const blocked = vi.fn()
    const position = new THREE.Vector3(5, 1, 0)
    const route = new PoseRoute(f.controls, position, new THREE.Vector3(5, 1, -5), 0, complete, blocked)
    route.finishImmediately()
    expect(complete).toHaveBeenCalledTimes(1)
    expect(blocked).not.toHaveBeenCalled()
    expect(f.camera.position.distanceTo(position)).toBeLessThan(1e-4)
    f.dispose()
  })

  it('replans from actual clearance after geometry changes instead of claiming a clipped endpoint', () => {
    const f = fixture()
    f.wall.visible = false
    const complete = vi.fn()
    const blocked = vi.fn()
    const statuses: number[] = []
    const position = new THREE.Vector3(5, 1, 0)
    const route = new PoseRoute(f.controls, position, new THREE.Vector3(5, 1, -5), 0.3, complete, blocked, s => statuses.push(s.replans))
    f.wall.visible = true
    for (let frame = 0; frame < 240; frame++) { route.tick(1 / 60); f.controls.update(1 / 60) }
    expect(Math.max(...statuses)).toBeGreaterThan(0)
    expect(complete).toHaveBeenCalledTimes(1)
    expect(blocked).not.toHaveBeenCalled()
    expect(f.camera.position.distanceTo(position)).toBeLessThan(1e-4)
    f.dispose()
  })
})
