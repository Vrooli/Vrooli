/**
 * Performance instrumentation for scenario proxy hosts
 *
 * Provides per-request timing (Server-Timing header) and optional
 * aggregate metrics (/__perf endpoint).
 */

import { performance } from 'node:perf_hooks'

/** Standardized phase names for proxy request processing */
export const PHASE = {
  TOTAL: 'total',
  CTX: 'ctx',
  HTML_STREAM: 'html_stream',
  HTML_CACHE: 'html_cache',
  FWD_HTTP: 'fwd_http',
  WS_UPGRADE: 'ws_upgrade',
  INJECT: 'inject',
} as const

export type PhaseName = (typeof PHASE)[keyof typeof PHASE]

interface TimingEntry {
  name: string
  dur: number
  desc?: string
}

/**
 * Per-request timer that accumulates named timing spans and serializes
 * them to the standard Server-Timing header format.
 */
export class RequestTimer {
  private entries: TimingEntry[] = []

  /** Start a named span. Returns a stop() function that records the duration. */
  start(name: string, desc?: string): () => void {
    const t0 = performance.now()
    return () => {
      this.entries.push({ name, dur: performance.now() - t0, desc })
    }
  }

  /** Time an async function and record the duration under the given name. */
  async measure<T>(name: string, fn: () => Promise<T>, desc?: string): Promise<T> {
    const t0 = performance.now()
    try {
      return await fn()
    } finally {
      this.entries.push({ name, dur: performance.now() - t0, desc })
    }
  }

  /** Record a pre-computed duration. */
  record(name: string, durationMs: number, desc?: string): void {
    this.entries.push({ name, dur: durationMs, desc })
  }

  /** Serialize entries to Server-Timing header value. */
  toHeaderValue(): string {
    return this.entries
      .map((e) => {
        const dur = `dur=${e.dur.toFixed(1)}`
        return e.desc ? `${e.name};${dur};desc="${e.desc}"` : `${e.name};${dur}`
      })
      .join(', ')
  }

  /** Return raw entries for aggregate collection. */
  getEntries(): readonly TimingEntry[] {
    return this.entries
  }
}

/** No-op timer singleton — all methods are zero-cost. */
class NoopTimer extends RequestTimer {
  override start(_name: string, _desc?: string): () => void {
    return noopFn
  }
  override async measure<T>(_name: string, fn: () => Promise<T>, _desc?: string): Promise<T> {
    return fn()
  }
  override record(_name: string, _durationMs: number, _desc?: string): void {}
  override toHeaderValue(): string {
    return ''
  }
  override getEntries(): readonly TimingEntry[] {
    return EMPTY_ENTRIES
  }
}

const noopFn = () => {}
const EMPTY_ENTRIES: readonly TimingEntry[] = Object.freeze([])

export const NOOP_TIMER: RequestTimer = new NoopTimer()

/** Ring buffer for recent samples used for percentile calculation. */
class RingBuffer {
  private buf: Float64Array
  private pos = 0
  private full = false

  constructor(size: number) {
    this.buf = new Float64Array(size)
  }

  push(value: number): void {
    this.buf[this.pos] = value
    this.pos += 1
    if (this.pos >= this.buf.length) {
      this.pos = 0
      this.full = true
    }
  }

  /** Return a sorted copy of the stored samples. */
  sorted(): Float64Array {
    const len = this.full ? this.buf.length : this.pos
    const copy = this.buf.slice(0, len)
    copy.sort()
    return copy
  }

  get length(): number {
    return this.full ? this.buf.length : this.pos
  }
}

interface PhaseStats {
  count: number
  total: number
  min: number
  max: number
  samples: RingBuffer
}

interface CacheStats {
  hits: number
  misses: number
}

/**
 * Aggregate metrics accumulator.
 *
 * Collects per-phase latency distributions and cache hit/miss counters.
 * Exposed via the /__perf endpoint when enableMetrics is true.
 */
export class ProxyMetrics {
  private phases = new Map<string, PhaseStats>()
  private cacheCounters = new Map<string, CacheStats>()
  private startedAt = Date.now()
  private requestCount = 0
  private sampleSize: number

  constructor(sampleSize = 1000) {
    this.sampleSize = sampleSize
  }

  /** Record a timing entry from a RequestTimer. */
  recordPhase(name: string, durationMs: number): void {
    let stats = this.phases.get(name)
    if (!stats) {
      stats = {
        count: 0,
        total: 0,
        min: Infinity,
        max: -Infinity,
        samples: new RingBuffer(this.sampleSize),
      }
      this.phases.set(name, stats)
    }
    stats.count += 1
    stats.total += durationMs
    if (durationMs < stats.min) stats.min = durationMs
    if (durationMs > stats.max) stats.max = durationMs
    stats.samples.push(durationMs)
  }

  /** Record all entries from a finished RequestTimer. */
  recordAll(timer: RequestTimer): void {
    this.requestCount += 1
    for (const entry of timer.getEntries()) {
      this.recordPhase(entry.name, entry.dur)
    }
  }

  /** Record a cache event (hit or miss). */
  recordCacheEvent(category: string, hit: boolean): void {
    let stats = this.cacheCounters.get(category)
    if (!stats) {
      stats = { hits: 0, misses: 0 }
      this.cacheCounters.set(category, stats)
    }
    if (hit) {
      stats.hits += 1
    } else {
      stats.misses += 1
    }
  }

  /** Produce a JSON-serializable snapshot. */
  toJSON(): Record<string, unknown> {
    const phases: Record<string, unknown> = {}
    for (const [name, stats] of this.phases) {
      const sorted = stats.samples.sorted()
      phases[name] = {
        count: stats.count,
        avg_ms: stats.count > 0 ? round2(stats.total / stats.count) : 0,
        min_ms: stats.count > 0 ? round2(stats.min) : 0,
        max_ms: stats.count > 0 ? round2(stats.max) : 0,
        p50_ms: percentile(sorted, 0.5),
        p95_ms: percentile(sorted, 0.95),
        p99_ms: percentile(sorted, 0.99),
      }
    }

    const cache: Record<string, unknown> = {}
    for (const [category, stats] of this.cacheCounters) {
      const total = stats.hits + stats.misses
      cache[category] = {
        hits: stats.hits,
        misses: stats.misses,
        hit_rate: total > 0 ? round2(stats.hits / total) : 0,
      }
    }

    return {
      uptime_s: round2((Date.now() - this.startedAt) / 1000),
      total_requests: this.requestCount,
      cache,
      phases,
    }
  }

  /** Clear all counters and restart uptime. */
  reset(): void {
    this.phases.clear()
    this.cacheCounters.clear()
    this.requestCount = 0
    this.startedAt = Date.now()
  }
}

function round2(n: number): number {
  return Math.round(n * 100) / 100
}

function percentile(sorted: Float64Array, p: number): number {
  if (sorted.length === 0) return 0
  const idx = Math.ceil(p * sorted.length) - 1
  return round2(sorted[Math.max(0, idx)])
}
