import { describe, expect, it, vi } from 'vitest'
import { DirectionalLight, WebGLRenderTarget } from 'three'
import { advanceShadowClock, installShadowRefresh, resizeShadowTarget } from './shadowRefresh'

describe('installShadowRefresh', () => {
  it('releases stale targets once even when React has already updated mapSize', () => {
    const shadow = new DirectionalLight().shadow
    const target = new WebGLRenderTarget(512, 512), pass = new WebGLRenderTarget(512, 512)
    shadow.map = target; shadow.mapPass = pass
    const dispose = vi.spyOn(target, 'dispose'), disposePass = vi.spyOn(pass, 'dispose')
    shadow.mapSize.set(1024, 1024)
    expect(resizeShadowTarget(shadow, 1024)).toBe(true)
    expect(shadow.map).toBeNull()
    expect(shadow.mapPass).toBeNull()
    expect(shadow.needsUpdate).toBe(true)
    expect(dispose).toHaveBeenCalledTimes(1)
    expect(disposePass).toHaveBeenCalledTimes(1)
    expect(resizeShadowTarget(shadow, 1024)).toBe(false)
    const current = new WebGLRenderTarget(1024, 1024)
    shadow.map = current
    expect(resizeShadowTarget(shadow, 1024)).toBe(false)
    expect(shadow.map).toBe(current)
    current.dispose()
  })
  it('updates the requested resolution before a target exists', () => {
    const shadow = new DirectionalLight().shadow
    expect(resizeShadowTarget(shadow, 2048)).toBe(true)
    expect(shadow.mapSize.toArray()).toEqual([2048, 2048])
    expect(shadow.map).toBeNull()
  })
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
