import { describe, expect, it } from 'vitest'
import CameraControls from 'camera-controls'
import * as THREE from 'three'
import { shouldFollow } from './follow'
import type { Vec3 } from './pose'
import { freezeCamera, nearPlaneRadius, orbitClamps } from './pose'
import { tuning } from '../../config'
import { focusVisibility, createZoomSurfaceQuery, WorldCameraControls } from './input'

CameraControls.install({ THREE })

/** Drive the same controller entry point used by wheel and touch dolly. */
class DollyReplayControls extends WorldCameraControls {
  pulse(delta: number, x: number, y: number, interrupt = true) {
    if (interrupt) freezeCamera(this)
    this._dollyInternal(delta, x, y)
  }
}

describe('cursor zoom over empty sky', () => {
  it.each([
    { angle: 0, interrupt: true, cursor: true },
    { angle: 0.7, interrupt: true, cursor: true },
    { angle: 0.7, interrupt: false, cursor: true },
    { angle: 0.7, interrupt: true, cursor: false },
    { angle: -0.7, interrupt: true, cursor: true },
    { angle: -0.7, interrupt: true, cursor: true, boundary: true },
    { angle: -0.7, pitch: 0.4, interrupt: false, cursor: true, boundary: true, boundaryMin: -0.5 },
  ])('stops zoom before a visible surface at $angle with interruption=$interrupt cursor=$cursor boundary=$boundary and permits reversal', ({ angle, pitch = 0, interrupt, cursor, boundary = false, boundaryMin = 0 }) => {
    const camera = new THREE.PerspectiveCamera(38, 1.6, 0.5, 400)
    const controls = new DollyReplayControls(camera)
    controls.dollyToCursor = cursor
    controls.minDistance = 0.1
    controls.maxDistance = 140
    void controls.setLookAt(0, 3, 10, 0, 3, 0, false)
    if (boundary) controls.setBoundary(new THREE.Box3(new THREE.Vector3(boundaryMin, -100, -100), new THREE.Vector3(100, 3.2, 100)))
    controls.update(1 / 60)
    const root = new THREE.Group()
    const geometry = new THREE.PlaneGeometry(100, 100)
    const material = new THREE.MeshBasicMaterial()
    const wall = new THREE.Mesh(geometry, material)
    wall.position.set(0, 3, 5)
    wall.rotation.set(pitch, angle, 0)
    wall.userData.cameraSurface = true
    root.add(wall)
    const hidden = new THREE.Mesh(geometry, material)
    hidden.position.set(0, 3, 9)
    hidden.userData.cameraSurface = true
    hidden.visible = false
    root.add(hidden)
    const cloud = new THREE.Mesh(geometry, material)
    cloud.position.set(0, 3, 9)
    root.add(cloud)
    controls.zoomSurfaceTravel = createZoomSurfaceQuery(root, camera)
    const normal = new THREE.Vector3(0, 0, 1).applyEuler(wall.rotation)
    const radius = nearPlaneRadius(camera.near, camera.fov, camera.aspect, camera.zoom)
    const clearance = () => camera.position.clone().sub(wall.position).dot(normal)
    const start = camera.position.clone()
    for (let step = 0; step < 20; step++) {
      controls.pulse(-100, -0.6, 0.4, interrupt)
      for (let frame = 0; frame < 30; frame++) {
        controls.update(1 / 60)
        expect(clearance()).toBeGreaterThanOrEqual(radius - 1e-6)
      }
    }
    expect(camera.position.distanceTo(start)).toBeGreaterThan(1)
    expect(clearance()).toBeCloseTo(radius, 4)
    const stopped = camera.position.clone()
    controls.pulse(4, -0.6, 0.4)
    for (let frame = 0; frame < 120; frame++) controls.update(1 / 60)
    expect(clearance()).toBeGreaterThan(radius + 0.1)
    expect(camera.position.distanceTo(stopped)).toBeGreaterThan(0.1)
    controls.dispose()
    geometry.dispose()
    material.dispose()
  })

  it('uses instance transforms and excludes misses from surface guards', () => {
    const camera = new THREE.PerspectiveCamera(38, 1.6, 0.5, 400)
    camera.position.set(0, 0, 10)
    camera.lookAt(0, 0, 0)
    const root = new THREE.Group()
    const geometry = new THREE.BoxGeometry(2, 2, 1)
    const material = new THREE.MeshBasicMaterial()
    const wall = new THREE.InstancedMesh(geometry, material, 1)
    wall.userData.cameraSurface = true
    wall.setMatrixAt(0, new THREE.Matrix4().makeTranslation(0, 0, 5))
    // Drei delegates pointer hits to child proxies; physical queries must
    // still inspect the instanced geometry and follow changed transforms.
    wall.raycast = () => undefined
    root.add(wall)
    const query = createZoomSurfaceQuery(root, camera)
    expect(query(0, 0, 0.5)).toBeCloseTo(4, 8)
    expect(query(0.9, 0.9, 0.5)).toBeNull()
    wall.setMatrixAt(0, new THREE.Matrix4().makeTranslation(4, 0, 5))
    expect(query(0, 0, 0.5)).toBeNull()
    wall.setMatrixAt(0, new THREE.Matrix4().makeTranslation(0, 0, 2))
    expect(query(0, 0, 0.5)).toBeCloseTo(7, 8)
    wall.dispose()
    geometry.dispose()
    material.dispose()
  })

  it('keeps a finite focal-plane anchor under an off-center cursor while looking upward', () => {
    const camera = new THREE.PerspectiveCamera(38, 1.6, 0.5, 400)
    const controls = new DollyReplayControls(camera)
    controls.dollyToCursor = true
    controls.minDistance = 1
    controls.maxDistance = 140
    void controls.setLookAt(0, 12, 10, 0, 16, 0, false)
    controls.update(1 / 60)
    camera.updateMatrixWorld()
    controls.zoomSurfaceTravel = createZoomSurfaceQuery(new THREE.Group(), camera)
    const cursor = new THREE.Vector2(-0.6, 0.4)
    const ray = new THREE.Raycaster()
    ray.setFromCamera(cursor, camera)
    expect(ray.ray.direction.y).toBeGreaterThan(0)
    const plane = new THREE.Plane().setFromNormalAndCoplanarPoint(
      camera.getWorldDirection(new THREE.Vector3()), controls.getTarget(new THREE.Vector3(), false))
    const anchor = ray.ray.intersectPlane(plane, new THREE.Vector3())
    if (!anchor) throw new Error('Cursor must intersect the focal plane in front of the camera')
    expect(anchor.toArray().every(Number.isFinite)).toBe(true)
    const before = controls.distance
    controls.pulse(-4, cursor.x, cursor.y)
    for (let frame = 0; frame < 180; frame++) {
      controls.update(1 / 60)
      camera.updateMatrixWorld()
      const projected = anchor.clone().project(camera)
      expect(projected.x).toBeCloseTo(cursor.x, 5)
      expect(projected.y).toBeCloseTo(cursor.y, 5)
      expect(camera.position.toArray().every(Number.isFinite)).toBe(true)
    }
    expect(controls.distance).toBeLessThan(before)
    controls.dispose()
  })

  it.each([1, 140])('reverses repeated off-center zoom at distance limit %s without escaping bounds', limit => {
    const camera = new THREE.PerspectiveCamera(38, 1.6, 0.5, 400)
    const controls = new DollyReplayControls(camera)
    controls.dollyToCursor = true
    controls.minDistance = 1
    controls.maxDistance = 140
    void controls.setLookAt(0, 20, limit, 0, 20, 0, false)
    controls.update(1 / 60)
    const initialTarget = controls.getTarget(new THREE.Vector3(), false)
    const outward = limit === 1 ? -4 : 4
    for (let step = 0; step < 20; step++) {
      controls.pulse(outward, -0.6, 0.4)
      controls.update(1 / 60)
    }
    expect(controls.distance).toBeCloseTo(limit, 8)
    expect(controls.getTarget(new THREE.Vector3(), false).distanceTo(initialTarget)).toBeLessThan(1e-8)
    controls.pulse(-outward, -0.6, 0.4)
    for (let frame = 0; frame < 8; frame++) controls.update(1 / 60)
    expect(limit === 1 ? controls.distance > limit : controls.distance < limit).toBe(true)
    for (let step = 0; step < 100; step++) {
      controls.pulse(step % 2 === 0 ? 4 : -4, -0.6, 0.4)
      controls.update(1 / 60)
      expect(controls.distance).toBeGreaterThanOrEqual(1 - 1e-8)
      expect(controls.distance).toBeLessThanOrEqual(140 + 1e-8)
      expect(camera.position.toArray().every(Number.isFinite)).toBe(true)
    }
    controls.dispose()
  })
})

describe('follow target gate', () => {
  it.each([false, true])('reaches an upward view with smoothing=%s while keeping the near plane clear', smooth => {
    const camera = new THREE.PerspectiveCamera(38, 1.6, 0.5, 400)
    const controls = new WorldCameraControls(camera)
    const clamps = orbitClamps(tuning.camera, 0)
    controls.minPolarAngle = clamps.minPolar
    controls.maxPolarAngle = clamps.maxPolar
    controls.groundCeiling = () => 2
    void controls.setLookAt(0, 10, 10, 0, 2, 0, false)
    controls.update(1 / 60)
    const radius = Math.hypot(0.5, 0.5 * Math.tan(38 * Math.PI / 360), 0.8 * Math.tan(38 * Math.PI / 360))
    for (let frame = 0; frame < 240; frame++) {
      void controls.rotate(0, Math.PI / 180, smooth)
      controls.update(1 / 60)
      expect(camera.position.y).toBeGreaterThanOrEqual(2 + radius - 1e-8)
    }
    expect(camera.getWorldDirection(new THREE.Vector3()).y).toBeGreaterThan(0.4)
    expect(controls.polarAngle).toBeCloseTo(115 * Math.PI / 180, 3)
    controls.dispose()
  })
  it('rejects a ridge crossing with clear endpoints and permits crossing after ascending', () => {
    const camera = new THREE.PerspectiveCamera(60, 1, 0.5, 400)
    const controls = new WorldCameraControls(camera)
    controls.groundCeiling = (minX, _minZ, maxX) => minX <= 1 && maxX >= -1 ? 8 : 0
    void controls.setLookAt(-5, 3, 10, -5, 3, 0, false)
    controls.update(1 / 60)
    void controls.setLookAt(5, 3, 10, 5, 3, 0, false)
    controls.update(1 / 60)
    expect(camera.position.x).toBeCloseTo(-5, 10)
    expect(camera.position.y).toBeCloseTo(3, 10)
    void controls.setLookAt(-5, 12, 10, -5, 12, 0, false)
    controls.update(1 / 60)
    void controls.setLookAt(5, 12, 10, 5, 12, 0, false)
    controls.update(1 / 60)
    expect(camera.position.x).toBeCloseTo(5, 10)
    expect(camera.position.y).toBeCloseTo(12, 10)
    controls.dispose()
  })
  it('keeps the near plane above a terrain ceiling while preserving view direction', () => {
    const camera = new THREE.PerspectiveCamera(60, 2, 0.5, 400)
    const controls = new WorldCameraControls(camera)
    const boxes: number[][] = []
    controls.groundCeiling = (...box) => { boxes.push(box); return 3 }
    void controls.setLookAt(5, 2, 10, 5, 4, 0, false)
    controls.update(1 / 60)
    const radius = Math.hypot(0.5, 0.5 * Math.tan(Math.PI / 6), Math.tan(Math.PI / 6))
    expect(camera.position.y).toBeCloseTo(3 + radius, 10)
    expect(controls.getTarget(new THREE.Vector3(), false).y - camera.position.y).toBeCloseTo(2, 10)
    for (const [index, value] of [5 - radius, 10 - radius, 5 + radius, 10 + radius].entries()) expect(boxes[0]?.[index]).toBeCloseTo(value, 10)
    const position = camera.position.clone()
    for (let i = 0; i < 60; i++) controls.update(1 / 60)
    expect(camera.position.distanceTo(position)).toBeLessThan(1e-8)
    controls.dispose()
  })
  it('integrates a resumed automatic move as one bounded frame instead of accumulated tab time', () => {
    const resumed = new WorldCameraControls(new THREE.PerspectiveCamera(38, 1.6, 0.5, 400))
    const bounded = new CameraControls(new THREE.PerspectiveCamera(38, 1.6, 0.5, 400))
    for (const controls of [resumed, bounded]) {
      void controls.setLookAt(5, 8, 10, 0, 0, 0, false)
      void controls.setLookAt(30, 20, 40, 10, 0, 10, true)
    }
    resumed.update(120)
    bounded.update(0.05)
    const actual = resumed.getPosition(new THREE.Vector3(), false)
    expect(actual.distanceTo(bounded.getPosition(new THREE.Vector3(), false))).toBeLessThan(1e-8)
    expect(actual.distanceTo(resumed.getPosition(new THREE.Vector3(), true))).toBeGreaterThan(1)
    for (const delta of [NaN, Infinity, -1]) resumed.update(delta)
    expect(resumed.getPosition(new THREE.Vector3(), false).toArray().every(Number.isFinite)).toBe(true)
    resumed.dispose()
    bounded.dispose()
  })
  it('allows repeated full yaw in both directions without a seam clamp', () => {
    const controls = new CameraControls(new THREE.PerspectiveCamera(38, 1.6, 0.5, 400))
    const clamps = orbitClamps(tuning.camera, 20)
    expect(clamps.minAzimuth).toBe(-Infinity)
    expect(clamps.maxAzimuth).toBe(Infinity)
    controls.minAzimuthAngle = clamps.minAzimuth
    controls.maxAzimuthAngle = clamps.maxAzimuth
    void controls.setLookAt(5, 8, 10, 0, 0, 0, false)
    const initial = controls.azimuthAngle
    for (const direction of [1, -1]) {
      for (let step = 0; step < 80; step++) {
        const before = controls.azimuthAngle
        void controls.rotate(direction * Math.PI / 10, 0, false)
        controls.update(1 / 60)
        expect(controls.azimuthAngle - before).toBeCloseTo(direction * Math.PI / 10, 10)
      }
    }
    expect(controls.azimuthAngle).toBeCloseTo(initial, 10)
    controls.dispose()
  })
  it('freezes an interrupted transition at its current pose instead of its destination', () => {
    const camera = new THREE.PerspectiveCamera(38, 1.6, 0.5, 400)
    const controls = new CameraControls(camera)
    void controls.setLookAt(5, 8, 10, 0, 0, 0, false)
    void controls.setLookAt(30, 20, 40, 10, 0, 10, true)
    void controls.setFocalOffset(2, 1, 0, true)
    void controls.zoomTo(2, true)
    for (let frame = 0; frame < 10; frame++) controls.update(1 / 60)
    const position = controls.getPosition(new THREE.Vector3(), false)
    const target = controls.getTarget(new THREE.Vector3(), false)
    const offset = controls.getFocalOffset(new THREE.Vector3(), false)
    const zoom = camera.zoom
    expect(position.distanceTo(controls.getPosition(new THREE.Vector3(), true))).toBeGreaterThan(1)
    freezeCamera(controls)
    for (let frame = 0; frame < 120; frame++) {
      controls.update(1 / 60)
      expect(controls.getPosition(new THREE.Vector3(), false).distanceTo(position)).toBeLessThan(1e-8)
      expect(controls.getTarget(new THREE.Vector3(), false).distanceTo(target)).toBeLessThan(1e-8)
      expect(controls.getFocalOffset(new THREE.Vector3(), false).distanceTo(offset)).toBeLessThan(1e-8)
      expect(camera.zoom).toBeCloseTo(zoom, 10)
    }
    controls.dispose()
  })
  it('uses a strict epsilon and accumulates movement from the last command', () => {
    expect(shouldFollow([0, 0, 0], [0.1, 0, 0], 0.15)).toBe(false)
    expect(shouldFollow([0, 0, 0], [0.15, 0, 0], 0.15)).toBe(false)
    expect(shouldFollow([0, 0, 0], [0.16, 0, 0], 0.15)).toBe(true)
    expect(shouldFollow(null, [0, 0, 0], 0.15)).toBe(true)
  })
  it('issues zero commands for 100 resting frames and bounded commands while walking', () => {
    let last: Vec3 = [0, 0, 0]
    let commands = 0
    for (let frame = 0; frame < 100; frame += 1) if (shouldFollow(last, [0, 0, 0], 0.15)) commands += 1
    expect(commands).toBe(0)
    for (let frame = 1; frame <= 100; frame += 1) {
      const next: Vec3 = [frame * 0.01, 0, 0]
      if (shouldFollow(last, next, 0.15)) {
        commands += 1
        last = next
      }
      expect(commands).toBeLessThanOrEqual(frame)
    }
    expect(commands).toBeGreaterThan(0)
    expect(commands).toBeLessThan(10)
  })
  it('immediate translation follows motion without resetting an active user orbit or dolly', () => {
    const controls = new CameraControls(new THREE.PerspectiveCamera(38, 1.6, 0.5, 400))
    const orbitOnly = new CameraControls(new THREE.PerspectiveCamera(38, 1.6, 0.5, 400))
    for (const rig of [controls, orbitOnly]) {
      void rig.setLookAt(5, 8, 10, 0, 0, 0, false)
      void rig.rotate(0.3, 0.1, true)
      void rig.dolly(1, true)
    }
    for (let frame = 1; frame <= 100; frame += 1) {
      void controls.moveTo(frame * 0.2, 0, 0, false)
      controls.update(1 / 60)
      orbitOnly.update(1 / 60)
      expect(controls.azimuthAngle).toBeCloseTo(orbitOnly.azimuthAngle, 10)
      expect(controls.polarAngle).toBeCloseTo(orbitOnly.polarAngle, 10)
      expect(controls.distance).toBeCloseTo(orbitOnly.distance, 10)
      expect(controls.getTarget(new THREE.Vector3(), false).x).toBeCloseTo(frame * 0.2)
    }
    controls.dispose()
    orbitOnly.dispose()
  })
})


it('focus visibility considers rendered instanced furniture even when picking is disabled', () => {
  const root = new THREE.Group()
  root.userData.cameraOccluder = true
  const mesh = new THREE.InstancedMesh(new THREE.BoxGeometry(1, 2, 1), new THREE.MeshBasicMaterial(), 1)
  mesh.setMatrixAt(0, new THREE.Matrix4().makeTranslation(0, 1, 2))
  mesh.raycast = () => undefined
  root.add(mesh)
  const target = new THREE.Vector3(0, 1, 0)
  expect(focusVisibility(root, new THREE.Vector3(0, 1, 4), target)).toBe(false)
  expect(focusVisibility(root, new THREE.Vector3(4, 1, 0), target)).toBe(true)
  mesh.visible = false
  expect(focusVisibility(root, new THREE.Vector3(0, 1, 4), target)).toBe(true)
  mesh.geometry.dispose()
  mesh.material.dispose()
})
