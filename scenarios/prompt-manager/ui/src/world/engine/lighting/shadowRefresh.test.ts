import { describe, expect, it, vi } from 'vitest'
import { installShadowRefresh } from './shadowRefresh'

describe('installShadowRefresh', () => {
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
