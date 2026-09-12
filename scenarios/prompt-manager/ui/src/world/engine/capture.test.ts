import { describe, expect, it, vi } from 'vitest'
import type { RootState } from '@react-three/fiber'
import { snapshotWorld } from './capture'

describe('world snapshot contract', () => {
  it('freezes time after live measurement and advances the real local pipeline without global timers', () => {
    const clock = { elapsedTime: 99 }
    const events: string[] = []
    const state = {
      clock,
      setFrameloop: vi.fn(() => { events.push('stop'); clock.elapsedTime = 0 }),
      advance: vi.fn((time, globalEffects) => {
        expect(clock.elapsedTime).toBe(5)
        expect(time).toBe(5)
        expect(globalEffects).toBe(false)
        events.push('render')
      }),
      gl: { shadowMap: { needsUpdate: false }, domElement: { toDataURL: vi.fn(() => { events.push('png'); return 'data:image/png;base64,world' }) } },
    }
    expect(snapshotWorld(state as unknown as RootState, 5, 8)).toBe('data:image/png;base64,world')
    expect(state.setFrameloop).toHaveBeenCalledWith('never')
    expect(state.advance).toHaveBeenCalledTimes(8)
    expect(state.gl.shadowMap.needsUpdate).toBe(true)
    expect(events).toEqual(['stop', ...Array(8).fill('render'), 'png'])
  })
  it('rejects invalid inputs before altering the running world', () => {
    const state = { setFrameloop: vi.fn() }
    const invalid: Array<[number, number]> = [[NaN, 8], [-1, 8], [5, 0], [5, 1.5]]
    for (const [seconds, frames] of invalid) expect(() => snapshotWorld(state as unknown as RootState, seconds, frames)).toThrow(/Invalid snapshot/)
    expect(state.setFrameloop).not.toHaveBeenCalled()
  })
})
