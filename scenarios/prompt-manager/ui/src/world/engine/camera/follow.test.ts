import { describe, expect, it } from 'vitest'
import CameraControls from 'camera-controls'
import * as THREE from 'three'
import { shouldFollow } from './follow'
import type { Vec3 } from './pose'

CameraControls.install({ THREE })

describe('follow target gate', () => {
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
