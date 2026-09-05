import { render } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { tuning } from '../config'
import { FrameDriver } from './frameDriver'

const invalidate = vi.hoisted(() => vi.fn())
vi.mock('@react-three/fiber', () => ({ useThree: () => invalidate }))
vi.mock('@react-three/drei', () => ({ useProgress: () => ({ active: false, progress: 100 }) }))

let now = 0
let nextFrame: FrameRequestCallback | null = null
beforeEach(() => {
  now = 0
  nextFrame = null
  invalidate.mockClear()
  vi.spyOn(performance, 'now').mockImplementation(() => now)
  vi.stubGlobal('requestAnimationFrame', vi.fn((callback: FrameRequestCallback) => { nextFrame = callback; return 1 }))
  vi.stubGlobal('cancelAnimationFrame', vi.fn())
})
afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals() })

function frame(at: number) {
  now = at
  const callback = nextFrame
  nextFrame = null
  callback?.(at)
}

const store = { getState: () => ({ actors: {} }), subscribe: () => () => undefined }

describe('demand frame timing', () => {
  it('preserves the configured intro window through the initial wake and stops after it', () => {
    const settings = { ...tuning.quality.frameDriver, introMs: 2000, minimumSettleMs: 100 }
    const view = render(<FrameDriver settings={settings} store={store} intro continuous={false} weatherActive={false} diagnosticsOpen={false} settleSeconds={0} />)
    frame(1000)
    expect(nextFrame).not.toBeNull()
    frame(1999)
    expect(nextFrame).not.toBeNull()
    frame(2000)
    expect(nextFrame).toBeNull()
    view.unmount()
  })

  it('uses the configured motion threshold after the settle window expires', () => {
    let speed = 0.3
    const movingStore = { ...store, getState: () => ({ actors: { a: { speed } } }) }
    const settings = { ...tuning.quality.frameDriver, minimumSettleMs: 100, movingSpeed: 0.2 }
    const view = render(<FrameDriver settings={settings} store={movingStore} intro={false} continuous={false} weatherActive={false} diagnosticsOpen={false} settleSeconds={0} />)
    frame(200)
    expect(nextFrame).not.toBeNull()
    speed = 0.2
    frame(300)
    expect(nextFrame).toBeNull()
    view.unmount()
  })

  it('uses the configured diagnostics heartbeat and clears it on unmount', () => {
    const interval = vi.spyOn(window, 'setInterval')
    const clear = vi.spyOn(window, 'clearInterval')
    const settings = { ...tuning.quality.frameDriver, diagnosticsHeartbeatMs: 750 }
    const view = render(<FrameDriver settings={settings} store={store} intro={false} continuous={false} weatherActive={false} diagnosticsOpen settleSeconds={0} />)
    expect(interval).toHaveBeenCalledWith(invalidate, 750)
    view.unmount()
    expect(clear).toHaveBeenCalledWith(interval.mock.results[0]?.value)
  })
})
