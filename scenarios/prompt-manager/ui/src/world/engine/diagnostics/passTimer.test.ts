import { describe, expect, it } from 'vitest'
import { addPassDraws, beginPassDrawFrame, passDrawsFor, PassTimer } from './passTimer'

class FakeGL {
  readonly QUERY_RESULT_AVAILABLE = 1
  readonly QUERY_RESULT = 2
  private next = 0
  private values = new Map<object, number>()
  readonly extension = {
    TIMESTAMP_EXT: 3,
    GPU_DISJOINT_EXT: 4,
    queryCounterEXT: (query: object) => this.values.set(query, (this.next += 1_000_000)),
  }
  getExtension() { return this.extension }
  createQuery() { return {} }
  deleteQuery() {}
  getParameter() { return false }
  getQueryParameter(query: object, parameter: number) {
    return parameter === this.QUERY_RESULT_AVAILABLE ? true : this.values.get(query)
  }
}

describe('PassTimer', () => {
  it('trims retained frames immediately when the configured sample window shrinks', () => {
    const timer = new PassTimer(new FakeGL() as unknown as WebGL2RenderingContext, { passSampleWindow: 4, passMaxPending: 64 })
    for (const spans of [4, 3, 0, 0]) {
      timer.beginFrame()
      for (let i = 0; i < spans; i += 1) { timer.begin('post'); timer.end('post') }
      timer.endFrame()
      timer.drain()
    }
    expect(timer.stats().total).toBe(7)
    timer.configure({ passSampleWindow: 2, passMaxPending: 1 })
    expect(timer.stats().total).toBe(1)
  })

  it('reports nested shadow and post spans with the remainder as main', () => {
    const gl = new FakeGL()
    const timer = new PassTimer(gl as unknown as WebGL2RenderingContext)
    timer.beginFrame()
    timer.begin('shadow')
    timer.end('shadow')
    timer.begin('post')
    timer.end('post')
    timer.endFrame()
    const stats = timer.stats()
    expect(stats).toMatchObject({ shadow: 1, post: 1, total: 5, main: 3, reason: '' })
  })
})

describe('pass draw attribution', () => {
  it('accumulates pass draws within a frame and resets at the next frame', () => {
    const renderer = {}
    beginPassDrawFrame(renderer)
    addPassDraws(renderer, 'post', 2, 4)
    addPassDraws(renderer, 'post', 3, 6)
    addPassDraws(renderer, 'shadow', 1, 8)
    expect(passDrawsFor(renderer)).toEqual({ shadow: { calls: 1, triangles: 8 }, post: { calls: 5, triangles: 10 } })
    beginPassDrawFrame(renderer)
    expect(passDrawsFor(renderer)).toEqual({ shadow: { calls: 0, triangles: 0 }, post: { calls: 0, triangles: 0 } })
  })
})
