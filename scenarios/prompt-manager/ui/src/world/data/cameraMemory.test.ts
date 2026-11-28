import { afterEach, describe, expect, it, vi } from 'vitest'
import { createCameraMemory } from './cameraMemory'
import type { SavedCameraView } from '../config/navigation'

const body = { position: [3, .06, 4] as [number, number, number], yaw: .7, pitch: -.3, boom: 5.5 }
const view: SavedCameraView = { version: 1, mode: 'third-person', explore: { position: [2, 8, 12], target: [1, 0, 2], zoom: 1.2 }, body }
afterEach(() => { vi.useRealTimers(); localStorage.clear() })
describe('camera continuity storage', () => {
  it('round trips all camera fields across instances, isolates scenes/seeds, and flushes the latest pending pose', () => {
    const memory = createCameraMemory('park', 7)
    expect(memory.read()).toBeNull()
    memory.save(view); memory.save({ ...view, body: { ...body, position: [5, 0, 8] } }); memory.flush()
    expect(createCameraMemory('park', 7).read()).toEqual({ ...view, body: { ...body, position: [5, 0, 8] } })
    expect(createCameraMemory('office', 7).read()).toBeNull()
    expect(createCameraMemory('park', 8).read()).toBeNull()
  })
  it('bounds writes while moving and avoids repeated unchanged saves', () => {
    vi.useFakeTimers()
    const setItem = vi.fn(), memory = createCameraMemory('park', 1, () => ({ getItem: () => null, setItem }))
    for (let i = 0; i < 10; i++) { memory.save({ ...view, explore: { ...view.explore, position: [i, 8, 12] } }); vi.advanceTimersByTime(100) }
    expect(setItem).toHaveBeenCalledTimes(1)
    memory.flush(); expect(setItem).toHaveBeenCalledTimes(2)
    memory.save({ ...view, explore: { ...view.explore, position: [9, 8, 12] } }); vi.advanceTimersByTime(1000)
    expect(setItem).toHaveBeenCalledTimes(2)
  })
  it('rejects invalid/incomplete records and tolerates unavailable storage', () => {
    for (const raw of ['broken', JSON.stringify({ ...view, version: 3 }), JSON.stringify({ ...view, body: undefined }), JSON.stringify({ ...view, body: { ...view.body, position: [0, 'bad', 1] } })]) {
      const memory = createCameraMemory('park', 1, () => ({ getItem: () => raw, setItem: () => undefined }))
      expect(memory.read()).toBeNull()
    }
    const memory = createCameraMemory('park', 1, () => { throw new Error('Storage unavailable') })
    expect(memory.read()).toBeNull(); memory.save(view); expect(() => memory.flush()).not.toThrow()
  })
})
