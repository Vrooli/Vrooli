import { describe, expect, it, vi } from 'vitest'
import { advanceShadowClock, installShadowRefresh } from './shadowRefresh'

describe('installShadowRefresh', () => {
  it('honours refresh hertz across different rendering rates', () => {
    for (const fps of [30, 60, 120]) {
      let elapsed = 0
      let refreshes = 0
      for (let frame = 0; frame < fps * 2; frame += 1) {
        const next = advanceShadowClock(elapsed, 1 / fps, 10)
        elapsed = next.elapsed
        if (next.refresh) refreshes += 1
      }
      expect(refreshes).toBe(20)
    }
    expect(advanceShadowClock(0, 1, 0)).toEqual({ elapsed: 0, refresh: false })
  })

  it('disables automatic updates, requests explicit refreshes, and restores on dispose', () => {
    const shadowMap = { autoUpdate: true, needsUpdate: false }
    const onRefresh = vi.fn()
    const controller = installShadowRefresh(shadowMap, onRefresh)
    expect(shadowMap.autoUpdate).toBe(false)
    controller.request()
    expect(shadowMap.needsUpdate).toBe(true)
    expect(onRefresh).toHaveBeenCalledOnce()
    controller.dispose()
    expect(shadowMap.autoUpdate).toBe(true)
  })
})
