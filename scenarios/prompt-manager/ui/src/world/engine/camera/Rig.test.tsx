import { createRef } from 'react'
import { act, render } from '@/test-utils/renderWithProviders'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { scenes, tuning } from '../../config'
import { CameraRig, type CameraRigHandle } from './Rig'
import type { WorldBounds } from '../types'

const fake = vi.hoisted(() => ({
  aspect: 1.6,
  minimumDistance: 0,
  controls: {
    camera: { fov: 45, near: .1, far: 400, updateProjectionMatrix: vi.fn() },
    enabled: true, smoothTime: 0, polarAngle: 0.7, azimuthAngle: 0.2,
    setLookAt: vi.fn(() => Promise.resolve()), setBoundary: vi.fn(), stop: vi.fn(),
    addEventListener: vi.fn(), removeEventListener: vi.fn(),
    moveTo: vi.fn(), update: vi.fn(),
    getPosition: vi.fn((out: { set(x: number, y: number, z: number): unknown }) => out.set(12, 20, 30)),
    getTarget: vi.fn((out: { set(x: number, y: number, z: number): unknown }) => out.set(1, 2, 3)),
    zoomTo: vi.fn(), setFocalOffset: vi.fn(),
  },
  frame: undefined as ((state: unknown, dt: number) => void) | undefined,
  diagnostics: vi.fn(),
  routed: vi.fn(),
  routeCancelled: vi.fn(),
}))
vi.mock('../diagnostics/store', () => ({ updateDiagnostics: fake.diagnostics }))
vi.mock('./NavigationMarkers', () => ({ NavigationMarkers: () => null }))
vi.mock('./pose', async (original) => ({
  ...await original<typeof import('./pose')>(),
  freezeCamera: (controls: { stop(): void }) => controls.stop(),
}))
vi.mock('./poseRoute', () => ({
  PoseRoute: class {
    private active = true
    private finish: () => void
    constructor(controls: typeof fake.controls, position: { toArray(): number[] }, target: { toArray(): number[] }, seconds: number, complete: () => void) {
      fake.routed(seconds)
      this.finish = () => { if (this.active) { this.active = false; complete() } }
      const setLookAt = controls.setLookAt as (...args: unknown[]) => Promise<void>
      void setLookAt(...position.toArray(), ...target.toArray(), seconds > 0).then(this.finish)
    }
    cancel() { this.active = false; fake.routeCancelled() }
    tick() { /* Route geometry and timing are tested against the real controller. */ }
    finishImmediately() { this.finish() }
  },
}))
vi.mock('@react-three/fiber', () => {
  const state = { viewport: { get aspect() { return fake.aspect } }, gl: { domElement: document.createElement('canvas') } }
  return { useFrame: (callback: typeof fake.frame, priority = 0) => { if (priority === 0) fake.frame = callback }, useThree: (select: (value: typeof state) => unknown) => select(state) }
})
vi.mock('@react-three/drei', async () => {
  const { useImperativeHandle } = await import('react')
  return {
    useProgress: () => ({ active: false }),
    CameraControls: ({ ref, minDistance }: { ref: React.Ref<typeof fake.controls>; minDistance: number }) => {
      fake.minimumDistance = minDistance
      useImperativeHandle(ref, () => fake.controls)
      return null
    },
  }
})

describe('camera continuity across world commits', () => {
  it('restores the remembered Explore pose without an intro and flushes its view on page exit', () => {
    const bounds: WorldBounds = { width: 100, depth: 100, center: [0, 0], footprint: { width: 80, depth: 80, center: [0, 0] }, outline: [[-40, -40], [40, 40]] }
    const memory = { read: vi.fn(() => ({ version: 1 as const, mode: 'explore' as const, explore: { position: [12, 20, 30] as [number, number, number], target: [1, 2, 3] as [number, number, number], zoom: 1.3 } })), save: vi.fn(), flush: vi.fn() }
    const view = render(<CameraRig epoch={1} scene={scenes.park} camera={tuning.camera} bounds={bounds} intro reducedMotion={false} memory={memory} />)
    expect(fake.controls.setLookAt).toHaveBeenLastCalledWith(12, 20, 30, 1, 2, 3, false)
    expect(fake.controls.zoomTo).toHaveBeenCalledWith(1.3, false)
    expect(fake.routed).not.toHaveBeenCalled()
    window.dispatchEvent(new Event('pagehide'))
    expect(memory.save).toHaveBeenCalledWith(expect.objectContaining({ mode: 'explore', explore: expect.objectContaining({ position: [12, 20, 30], target: [1, 2, 3] }) }))
    expect(memory.flush).toHaveBeenCalledOnce()
    view.unmount()
  })
  it('updates the existing lens without restarting the camera pose', () => {
    const bounds: WorldBounds = { width: 40, depth: 40, center: [0, 0], footprint: { width: 30, depth: 30, center: [0, 0] }, outline: [[-15, -15], [15, 15]] }
    const props = { epoch: 1, scene: scenes.park, bounds, intro: false, reducedMotion: true }
    const view = render(<CameraRig {...props} camera={tuning.camera} />)
    const moves = fake.controls.setLookAt.mock.calls.length
    fake.controls.camera.updateProjectionMatrix.mockClear()
    const camera = { ...tuning.camera, fov: 63, near: .23 }
    view.rerender(<CameraRig {...props} camera={camera} />)
    expect(fake.controls.camera.fov).toBe(63)
    expect(fake.controls.camera.near).toBe(.23)
    expect(fake.controls.camera.updateProjectionMatrix).toHaveBeenCalledTimes(1)
    expect(fake.controls.setLookAt).toHaveBeenCalledTimes(moves)
    view.rerender(<CameraRig {...props} camera={{ ...camera, truckSpeed: 3 }} />)
    expect(fake.controls.camera.updateProjectionMatrix).toHaveBeenCalledTimes(1)
    view.unmount()
  })
  beforeEach(() => { vi.clearAllMocks(); fake.aspect = 1.6; fake.controls.setLookAt.mockImplementation(() => Promise.resolve()) })
  it('restores an absolute imported camera through the route owner and respects reduced motion', () => {
    const bounds: WorldBounds = { width: 40, depth: 40, center: [0, 0], footprint: { width: 30, depth: 30, center: [0, 0] }, outline: [[-15, -15], [15, 15]] }
    const view = render(<CameraRig epoch={1} scene={scenes.park} camera={tuning.camera} bounds={bounds} intro reducedMotion
      initialPose={{ position: [12, 20, 30], target: [1, 2, 3] }} />)
    expect(fake.controls.setLookAt).toHaveBeenLastCalledWith(12, 20, 30, 1, 2, 3, false)
    expect(fake.routed).toHaveBeenCalledWith(0)
    view.unmount()
  })
  it('stops an in-flight intro when reduced motion is enabled and ignores its late completion', async () => {
    let finish: (() => void) | undefined
    fake.controls.setLookAt.mockImplementation(() => new Promise<void>(resolve => { finish = resolve }))
    const bounds: WorldBounds = { width: 40, depth: 40, center: [0, 0], footprint: { width: 30, depth: 30, center: [0, 0] }, outline: [[-15, -15], [15, 15]] }
    const props = { epoch: 1, scene: scenes.park, camera: tuning.camera, bounds, intro: true }
    const view = render(<CameraRig {...props} reducedMotion={false} />)
    const commands = fake.controls.setLookAt.mock.calls.length
    const stops = fake.controls.stop.mock.calls.length
    view.rerender(<CameraRig {...props} reducedMotion />)
    expect(fake.controls.stop.mock.calls.length).toBeGreaterThan(stops)
    expect(fake.routeCancelled).toHaveBeenCalled()
    expect(fake.controls.setLookAt).toHaveBeenCalledTimes(commands)
    expect(fake.diagnostics).toHaveBeenLastCalledWith({ introDone: true })
    fake.diagnostics.mockClear()
    await act(async () => { finish?.(); await Promise.resolve() })
    expect(fake.diagnostics).not.toHaveBeenCalled()
    view.rerender(<CameraRig {...props} reducedMotion={false} />)
    expect(fake.controls.setLookAt).toHaveBeenCalledTimes(commands)
    view.unmount()
  })
  it('routes both intro and home through the guarded pose runner with their declared duration', async () => {
    const ref = createRef<CameraRigHandle>()
    const bounds: WorldBounds = { width: 40, depth: 40, center: [0, 0], footprint: { width: 30, depth: 30, center: [0, 0] }, outline: [[-15, -15], [15, 15]] }
    const view = render(<CameraRig ref={ref} epoch={1} scene={scenes.park} camera={tuning.camera} bounds={bounds} intro reducedMotion={false} />)
    expect(fake.routed).toHaveBeenCalledWith(tuning.camera.introSeconds)
    await act(async () => { ref.current?.home(); await Promise.resolve() })
    expect(fake.routed).toHaveBeenLastCalledWith(Math.max(0.15, tuning.camera.smoothTime * 3))
    expect(fake.controls.addEventListener).not.toHaveBeenCalled()
    view.unmount()
  })
  it('detaches a vanished follow target without translating to a fallback point or restarting when it reappears', () => {
    const ref = createRef<CameraRigHandle>()
    const bounds: WorldBounds = { width: 40, depth: 40, center: [0, 0], footprint: { width: 30, depth: 30, center: [0, 0] }, outline: [[-15, -15], [15, 15]] }
    const view = render(<CameraRig ref={ref} epoch={1} scene={scenes.park} camera={tuning.camera} bounds={bounds} intro={false} reducedMotion={false} />)
    let target: [number, number, number] | null = [10, 2, 12]
    act(() => { ref.current?.follow(() => target) })
    target = [11, 2, 12]
    act(() => { fake.frame?.({}, 1 / 60) })
    expect(fake.controls.moveTo).toHaveBeenCalledWith(11, 2, 12, false)
    target = null
    act(() => { fake.frame?.({}, 1 / 60) })
    expect(fake.diagnostics).toHaveBeenLastCalledWith(expect.objectContaining({ cameraOwnership: expect.objectContaining({ owner: 'explore', moving: false }) }))
    target = [15, 2, 12]
    act(() => { fake.frame?.({}, 1 / 60) })
    expect(fake.controls.moveTo).toHaveBeenCalledTimes(1)
    expect(fake.controls.smoothTime).toBe(tuning.camera.smoothTime)
    view.unmount()
  })
  it('updates boundaries and readiness without issuing a new pose or reapplying selection', async () => {
    const ref = createRef<CameraRigHandle>()
    const onReady = vi.fn()
    const bounds: WorldBounds = { width: 40, depth: 40, center: [0, 0], footprint: { width: 30, depth: 30, center: [0, 0] }, outline: [[-15, -15], [15, 15]] }
    const props = { ref, scene: scenes.park, camera: tuning.camera, bounds, intro: false, initialPresentation: false, reducedMotion: false, onReady }
    const view = render(<CameraRig {...props} epoch={1} />)
    await act(async () => {
      ref.current?.setPose({ ...scenes.park.camera.hero, azimuthDeg: 75 }, false)
      await Promise.resolve()
    })
    const before = fake.controls.setLookAt.mock.calls.length
    fake.diagnostics.mockClear()
    const larger = { ...bounds, width: 80 }
    view.rerender(<CameraRig {...props} bounds={larger} epoch={2} />)
    expect(fake.controls.setLookAt).toHaveBeenCalledTimes(before)
    expect(fake.controls.setBoundary.mock.lastCall?.[0].max.x).toBe(40)
    expect(fake.diagnostics).toHaveBeenCalledWith({ introDone: true })
    expect(onReady).toHaveBeenCalledTimes(1)
    expect(fake.controls.camera.far).toBeGreaterThanOrEqual(tuning.camera.far)
    fake.aspect = 20
    view.rerender(<CameraRig {...props} bounds={larger} epoch={2} />)
    expect(fake.minimumDistance).toBeGreaterThan(tuning.camera.minDistance)
    expect(fake.controls.setLookAt).toHaveBeenCalledTimes(before)
    expect(onReady).toHaveBeenCalledTimes(1)
    view.unmount()
  })
})
