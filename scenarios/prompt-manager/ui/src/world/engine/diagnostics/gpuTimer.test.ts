import { describe, expect, it, vi } from 'vitest'
import { GpuTimer } from './gpuTimer'

function fakeContext(options: { extension?: boolean; disjoint?: boolean; results?: number[] } = {}) {
  const queries: object[] = []
  const results = [...(options.results ?? [])]
  const gl = {
    QUERY_RESULT_AVAILABLE: 1,
    QUERY_RESULT: 2,
    getExtension: () => options.extension === false ? null : { TIME_ELAPSED_EXT: 3, GPU_DISJOINT_EXT: 4 },
    createQuery: () => {
      const query = {}
      queries.push(query)
      return query
    },
    beginQuery: vi.fn(),
    endQuery: vi.fn(),
    deleteQuery: vi.fn(),
    getParameter: () => options.disjoint ?? false,
    getQueryParameter: (_query: object, key: number) => key === 1 ? true : results.shift() ?? 1_000_000,
  } as unknown as WebGL2RenderingContext
  return { gl, queries }
}

describe('GpuTimer', () => {
  it('bounds pending queries and retains only the configured recent samples', () => {
    const { gl, queries } = fakeContext({ results: [1_000_000, 2_000_000, 3_000_000] })
    const timer = new GpuTimer(gl, { gpuMaxInFlight: 1, gpuSampleWindow: 2 })
    timer.begin()
    timer.end()
    timer.begin()
    expect(queries).toHaveLength(1)
    timer.drain()
    for (let i = 0; i < 2; i += 1) { timer.begin(); timer.end(); timer.drain() }
    expect(timer.stats()).toMatchObject({ samples: 2, p50: 2, p95: 2 })
    timer.dispose()
    expect(gl.deleteQuery).toHaveBeenCalledTimes(3)
  })

  it('labels an unavailable extension', () => {
    const timer = new GpuTimer(fakeContext({ extension: false }).gl)
    expect(timer.stats()).toMatchObject({ available: false, samples: 0, reason: 'EXT_disjoint_timer_query_webgl2 unavailable' })
  })

  it('drains completed queries into GPU millisecond percentiles', () => {
    const { gl } = fakeContext({ results: [1_000_000, 4_000_000, 2_000_000] })
    const timer = new GpuTimer(gl)
    for (let index = 0; index < 3; index += 1) {
      timer.begin()
      timer.end()
    }
    timer.drain()
    expect(timer.stats()).toMatchObject({ available: true, samples: 3, p50: 2, p95: 2 })
  })

  it('discards pending samples when the GPU reports disjoint data', () => {
    const { gl } = fakeContext({ disjoint: true })
    const timer = new GpuTimer(gl)
    timer.begin()
    timer.end()
    timer.drain()
    expect(timer.stats()).toMatchObject({ available: false, samples: 0, reason: 'GPU reported disjoint timer data' })
  })
})
