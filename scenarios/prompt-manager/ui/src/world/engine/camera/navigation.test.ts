import { afterEach, describe, expect, it } from 'vitest'
import CameraControls from 'camera-controls'
import * as THREE from 'three'
import { tuning } from '../../config'
import { WorldCameraControls } from './input'
import { bindExploreInput, DEFAULT_NAVIGATION, parseNavigationPreferences } from './navigation'

CameraControls.install({ THREE })
const cleanup: Array<() => void> = []
afterEach(() => { cleanup.splice(0).forEach(fn => fn()); document.body.replaceChildren() })
function fixture(device: 'mouse' | 'trackpad' = 'mouse') {
  const canvas = document.createElement('canvas')
  canvas.getBoundingClientRect = () => ({ left: 0, top: 0, x: 0, y: 0, width: 1000, height: 800 } as DOMRect)
  document.body.append(canvas)
  const camera = new THREE.PerspectiveCamera(38, 1.25, .5, 400)
  const c = new WorldCameraControls(camera, canvas)
  void c.setLookAt(0, 10, 40, 0, 10, 0, false)
  c.update(0)
  cleanup.push(bindExploreInput(canvas, c, tuning.camera, { ...DEFAULT_NAVIGATION, device }, 'orbit', false, () => {}), () => c.dispose())
  const wheel = (dy: number, ctrlKey = false, dx = 0) => canvas.dispatchEvent(new WheelEvent('wheel', { deltaY: dy, deltaX: dx, ctrlKey, clientX: 500, clientY: 400, cancelable: true }))
  return { c, camera, wheel }
}
describe('operator navigation contract', () => {
  it('pinch and wheel dolly equally without changing the lens', () => {
    const mouse = fixture(), trackpad = fixture('trackpad')
    mouse.wheel(-120)
    trackpad.wheel(-120, true)
    expect(mouse.c.distance).toBeLessThan(40)
    expect(trackpad.c.distance).toBeCloseTo(mouse.c.distance, 6)
    expect(trackpad.camera.zoom).toBe(1)
  })
  it('retains equal-total input across event rates and stops after the final event', () => {
    const single = fixture('trackpad'), burst = fixture('trackpad')
    single.wheel(-120, true)
    for (let i = 0; i < 60; i++) burst.wheel(-2, true)
    expect(burst.c.distance).toBeCloseTo(single.c.distance, 6)
    const stopped = burst.camera.position.clone()
    for (let i = 0; i < 120; i++) burst.c.update(1 / 60)
    expect(burst.camera.position.distanceTo(stopped)).toBeLessThan(.00001)
    burst.wheel(10, true)
    expect(burst.c.distance).toBeGreaterThan(single.c.distance)
  })
  it('two-finger scrolling pans both axes without changing distance or magnification', () => {
    const { c, camera, wheel } = fixture('trackpad')
    const before = camera.position.clone()
    wheel(40, false, 30)
    expect(camera.position.x).not.toBe(before.x)
    expect(camera.position.y).not.toBe(before.y)
    expect(c.distance).toBeCloseTo(40)
    expect(camera.zoom).toBe(1)
  })
  it('integrates ordinary low-FPS frames at the same rate as 60 FPS', () => {
    const fast = fixture().c, slow = fixture().c
    fast.smoothTime = slow.smoothTime = .35
    void fast.rotate(1, 0, true); void slow.rotate(1, 0, true)
    for (let i = 0; i < 60; i++) fast.update(1 / 60)
    for (let i = 0; i < 5; i++) slow.update(.2)
    expect(slow.azimuthAngle).toBeCloseTo(fast.azimuthAngle, 2)
  })
  it('bounds stored preferences and ignores corrupt values', () => {
    expect(parseNavigationPreferences(null)).toEqual(DEFAULT_NAVIGATION)
    expect(parseNavigationPreferences({ sensitivity: Infinity }).sensitivity).toBe(1)
    expect(parseNavigationPreferences({ device: 'unknown', sensitivity: 100 }).sensitivity).toBe(3)
  })
})
