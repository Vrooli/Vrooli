import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { bindAmbientWake } from './wake'
import { WorldClock } from '../../config/clock'

let hidden = false
beforeEach(() => {
  hidden = false
  vi.useFakeTimers(); vi.setSystemTime(0)
  vi.spyOn(document, 'hidden', 'get').mockImplementation(() => hidden)
})
afterEach(() => { vi.useRealTimers(); vi.restoreAllMocks() })

describe('ambient boundary wake ownership', () => {
  it('owns no frozen timer and scales progression delays when the clock changes', () => {
    const clock = new WorldClock()
    clock.fix(0)
    const invalidate = vi.fn()
    const dispose = bindAmbientWake(now => now < 10 ? 10 : null, invalidate, clock)
    expect(vi.getTimerCount()).toBe(0)
    clock.fix(0, 10)
    expect(vi.getTimerCount()).toBe(1)
    invalidate.mockClear()
    vi.advanceTimersByTime(1000)
    expect(invalidate).toHaveBeenCalledTimes(1)
    expect(vi.getTimerCount()).toBe(0)
    dispose()
  })
  it('keeps a single idle timer and invalidates only when a boundary arrives', () => {
    const invalidate = vi.fn()
    const dispose = bindAmbientWake(now => now < 90 ? 90 : null, invalidate)
    expect(vi.getTimerCount()).toBe(1)
    vi.advanceTimersByTime(89999)
    expect(invalidate).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1)
    expect(invalidate).toHaveBeenCalledTimes(1)
    expect(vi.getTimerCount()).toBe(0)
    dispose()
  })
  it('cancels hidden waits and seeks once on resume without replaying missed events', () => {
    const invalidate = vi.fn(), reads: number[] = []
    const dispose = bindAmbientWake(now => { reads.push(now); return now + 10 }, invalidate)
    hidden = true; document.dispatchEvent(new Event('visibilitychange'))
    expect(vi.getTimerCount()).toBe(0)
    vi.advanceTimersByTime(100000)
    expect(invalidate).not.toHaveBeenCalled()
    hidden = false; document.dispatchEvent(new Event('visibilitychange'))
    expect(reads).toEqual([0, 100])
    expect(invalidate).toHaveBeenCalledTimes(1)
    expect(vi.getTimerCount()).toBe(1)
    dispose(); dispose()
    window.dispatchEvent(new Event('focus'))
    expect(invalidate).toHaveBeenCalledTimes(1)
    expect(vi.getTimerCount()).toBe(0)
  })
  it('splits browser-limit waits without rendering early and owns no disabled timer', () => {
    const invalidate = vi.fn()
    const boundary = 30 * 86400
    const dispose = bindAmbientWake(now => now < boundary ? boundary : null, invalidate)
    vi.advanceTimersByTime(2147483647)
    expect(invalidate).not.toHaveBeenCalled()
    expect(vi.getTimerCount()).toBe(1)
    vi.advanceTimersByTime(boundary * 1000 - 2147483647)
    expect(invalidate).toHaveBeenCalledTimes(1)
    dispose()
    const disabled = bindAmbientWake(() => null, invalidate)
    expect(vi.getTimerCount()).toBe(0)
    disabled()
  })
})
