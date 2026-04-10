import { describe, it, expect } from 'vitest'
import { RequestTimer, NOOP_TIMER, ProxyMetrics, PHASE } from '../../server/perf.js'

describe('PHASE constants', () => {
  it('exposes all expected phase names', () => {
    expect(PHASE.TOTAL).toBe('total')
    expect(PHASE.CTX).toBe('ctx')
    expect(PHASE.HTML_STREAM).toBe('html_stream')
    expect(PHASE.HTML_CACHE).toBe('html_cache')
    expect(PHASE.FWD_HTTP).toBe('fwd_http')
    expect(PHASE.WS_UPGRADE).toBe('ws_upgrade')
    expect(PHASE.INJECT).toBe('inject')
  })
})

describe('RequestTimer', () => {
  it('start/stop records duration', async () => {
    const timer = new RequestTimer()
    const stop = timer.start('test_phase')
    await sleep(5)
    stop()

    const entries = timer.getEntries()
    expect(entries).toHaveLength(1)
    expect(entries[0].name).toBe('test_phase')
    expect(entries[0].dur).toBeGreaterThan(0)
  })

  it('start with desc includes description in entry', () => {
    const timer = new RequestTimer()
    const stop = timer.start('phase', 'my description')
    stop()

    const entries = timer.getEntries()
    expect(entries[0].desc).toBe('my description')
  })

  it('measure times an async function', async () => {
    const timer = new RequestTimer()
    const result = await timer.measure('async_phase', async () => {
      await sleep(5)
      return 42
    })

    expect(result).toBe(42)
    const entries = timer.getEntries()
    expect(entries).toHaveLength(1)
    expect(entries[0].name).toBe('async_phase')
    expect(entries[0].dur).toBeGreaterThan(0)
  })

  it('measure records duration even if function throws', async () => {
    const timer = new RequestTimer()
    await expect(
      timer.measure('fail_phase', async () => {
        throw new Error('boom')
      }),
    ).rejects.toThrow('boom')

    const entries = timer.getEntries()
    expect(entries).toHaveLength(1)
    expect(entries[0].name).toBe('fail_phase')
  })

  it('record stores a pre-computed duration', () => {
    const timer = new RequestTimer()
    timer.record('precomputed', 123.456, 'already measured')

    const entries = timer.getEntries()
    expect(entries).toHaveLength(1)
    expect(entries[0]).toEqual({ name: 'precomputed', dur: 123.456, desc: 'already measured' })
  })

  it('toHeaderValue serializes to Server-Timing format', () => {
    const timer = new RequestTimer()
    timer.record('ctx', 2.1)
    timer.record('html_stream', 186.8, 'streaming')

    const header = timer.toHeaderValue()
    expect(header).toBe('ctx;dur=2.1, html_stream;dur=186.8;desc="streaming"')
  })

  it('toHeaderValue returns empty string when no entries', () => {
    const timer = new RequestTimer()
    expect(timer.toHeaderValue()).toBe('')
  })

  it('accumulates multiple entries in order', () => {
    const timer = new RequestTimer()
    timer.record('a', 1)
    timer.record('b', 2)
    timer.record('c', 3)

    const entries = timer.getEntries()
    expect(entries.map((e) => e.name)).toEqual(['a', 'b', 'c'])
  })
})

describe('NOOP_TIMER', () => {
  it('start returns a no-op stop function', () => {
    const stop = NOOP_TIMER.start('ignored')
    expect(stop).toBeTypeOf('function')
    stop() // should not throw
  })

  it('measure passes through the function result', async () => {
    const result = await NOOP_TIMER.measure('ignored', async () => 99)
    expect(result).toBe(99)
  })

  it('record does nothing', () => {
    NOOP_TIMER.record('ignored', 100)
    // no error
  })

  it('toHeaderValue returns empty string', () => {
    expect(NOOP_TIMER.toHeaderValue()).toBe('')
  })

  it('getEntries returns empty array', () => {
    expect(NOOP_TIMER.getEntries()).toHaveLength(0)
  })
})

describe('ProxyMetrics', () => {
  it('recordPhase tracks count, total, min, max', () => {
    const m = new ProxyMetrics()
    m.recordPhase('ctx', 10)
    m.recordPhase('ctx', 20)
    m.recordPhase('ctx', 5)

    const snapshot = m.toJSON() as any
    expect(snapshot.phases.ctx.count).toBe(3)
    expect(snapshot.phases.ctx.avg_ms).toBeCloseTo(11.67, 1)
    expect(snapshot.phases.ctx.min_ms).toBe(5)
    expect(snapshot.phases.ctx.max_ms).toBe(20)
  })

  it('recordAll increments total_requests and records all entries', () => {
    const m = new ProxyMetrics()
    const timer = new RequestTimer()
    timer.record('total', 100)
    timer.record('ctx', 5)
    m.recordAll(timer)

    const snapshot = m.toJSON() as any
    expect(snapshot.total_requests).toBe(1)
    expect(snapshot.phases.total.count).toBe(1)
    expect(snapshot.phases.ctx.count).toBe(1)
  })

  it('computes percentiles from ring buffer', () => {
    const m = new ProxyMetrics(100)
    for (let i = 1; i <= 100; i++) {
      m.recordPhase('latency', i)
    }

    const snapshot = m.toJSON() as any
    expect(snapshot.phases.latency.p50_ms).toBe(50)
    expect(snapshot.phases.latency.p95_ms).toBe(95)
    expect(snapshot.phases.latency.p99_ms).toBe(99)
  })

  it('ring buffer wraps around when exceeding sample size', () => {
    const m = new ProxyMetrics(10)
    // Push 20 values — only last 10 should remain
    for (let i = 1; i <= 20; i++) {
      m.recordPhase('test', i)
    }

    const snapshot = m.toJSON() as any
    // Last 10 values are 11..20, sorted: 11,12,...20
    expect(snapshot.phases.test.count).toBe(20) // count tracks all
    expect(snapshot.phases.test.p50_ms).toBe(15)
  })

  it('recordCacheEvent tracks hits and misses', () => {
    const m = new ProxyMetrics()
    m.recordCacheEvent('html', true)
    m.recordCacheEvent('html', true)
    m.recordCacheEvent('html', false)
    m.recordCacheEvent('metadata', true)

    const snapshot = m.toJSON() as any
    expect(snapshot.cache.html).toEqual({ hits: 2, misses: 1, hit_rate: 0.67 })
    expect(snapshot.cache.metadata).toEqual({ hits: 1, misses: 0, hit_rate: 1 })
  })

  it('toJSON includes uptime_s', () => {
    const m = new ProxyMetrics()
    const snapshot = m.toJSON() as any
    expect(snapshot.uptime_s).toBeGreaterThanOrEqual(0)
    expect(typeof snapshot.uptime_s).toBe('number')
  })

  it('reset clears all counters', () => {
    const m = new ProxyMetrics()
    m.recordPhase('ctx', 10)
    m.recordCacheEvent('html', true)
    const timer = new RequestTimer()
    timer.record('total', 50)
    m.recordAll(timer)

    m.reset()
    const snapshot = m.toJSON() as any
    expect(snapshot.total_requests).toBe(0)
    expect(snapshot.phases).toEqual({})
    expect(snapshot.cache).toEqual({})
  })

  it('toJSON returns zero values for empty metrics', () => {
    const m = new ProxyMetrics()
    const snapshot = m.toJSON() as any
    expect(snapshot.total_requests).toBe(0)
    expect(snapshot.phases).toEqual({})
    expect(snapshot.cache).toEqual({})
  })
})

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}
