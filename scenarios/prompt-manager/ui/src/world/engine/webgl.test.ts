import { afterEach, describe, expect, it, vi } from 'vitest'
import { probeWebGL, resolveTwoD, retryWebGL } from './webgl'

describe('probeWebGL', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('reports a missing WebGL2 context', () => {
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null)
    expect(probeWebGL()).toEqual({
      ok: false,
      reason: 'no-context',
      detail: 'The browser did not provide a WebGL2 context.',
    })
  })

  it('reports a lost context and releases the probe context', () => {
    const loseContext = vi.fn()
    const context = {
      isContextLost: () => true,
      getExtension: (name: string) => name === 'WEBGL_lose_context' ? { loseContext } : null,
    } as unknown as WebGL2RenderingContext
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(context)

    expect(probeWebGL()).toMatchObject({ ok: false, reason: 'context-lost' })
    expect(loseContext).toHaveBeenCalledOnce()
  })

  it('releases a successful probe context and can retry', () => {
    const loseContext = vi.fn()
    const context = {
      isContextLost: () => false,
      getExtension: (name: string) => name === 'WEBGL_lose_context' ? { loseContext } : null,
    } as unknown as WebGL2RenderingContext
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(context)

    expect(probeWebGL()).toEqual({ ok: true })
    expect(retryWebGL()).toEqual({ ok: true })
    expect(loseContext).toHaveBeenCalledTimes(2)
  })

  it('supports the permanent forced-failure test lever', () => {
    expect(probeWebGL(true)).toMatchObject({ ok: false, reason: 'no-context' })
  })
})

describe('resolveTwoD', () => {
  it('lets an explicit 3D deep link outrank a stored 2D preference', () => {
    expect(resolveTwoD({ webglAvailable: true, userChoice: null, requestedTwoD: false, storedTwoD: true, narrow: true })).toBe(false)
  })

  it('forces 2D when WebGL2 is genuinely unavailable', () => {
    expect(resolveTwoD({ webglAvailable: false, userChoice: false, requestedTwoD: false, storedTwoD: false, narrow: false })).toBe(true)
  })
})
