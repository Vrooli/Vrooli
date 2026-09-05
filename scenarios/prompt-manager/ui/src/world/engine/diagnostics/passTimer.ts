import { tuning, type QualityTuning } from '../../config'

export type PassLabel = 'shadow' | 'post' | 'total'
type PassSettings = Pick<QualityTuning['diagnostics'], 'passSampleWindow' | 'passMaxPending'>

interface TimerQueryExtension {
  TIMESTAMP_EXT: number
  GPU_DISJOINT_EXT: number
  queryCounterEXT(query: WebGLQuery, target: number): void
}

interface PendingSpan {
  label: PassLabel
  frame: number
  start: WebGLQuery
  end: WebGLQuery
}

/** Non-blocking timestamp-query spans that may nest inside the total frame span. */
export class PassTimer {
  private readonly extension: TimerQueryExtension | null
  private readonly pending: PendingSpan[] = []
  private readonly pool: WebGLQuery[] = []
  private readonly open = new Map<PassLabel, { frame: number; query: WebGLQuery }>()
  private readonly frameSamples: Record<PassLabel, Map<number, number>> = {
    shadow: new Map(),
    post: new Map(),
    total: new Map(),
  }
  private frame = 0
  private failureReason = ''

  constructor(private readonly gl: WebGL2RenderingContext, private settings: PassSettings = tuning.quality.diagnostics) {
    const extension = gl.getExtension('EXT_disjoint_timer_query_webgl2') as TimerQueryExtension | null
    this.extension = extension?.queryCounterEXT ? extension : null
    if (!this.extension) this.failureReason = 'GPU timestamp queries unavailable'
  }

  configure(settings: PassSettings): void {
    this.settings = settings
    for (const samples of Object.values(this.frameSamples)) {
      while (samples.size > settings.passSampleWindow) samples.delete(samples.keys().next().value as number)
    }
  }

  beginFrame(): void {
    this.frame += 1
    this.drain()
    this.markBegin('total')
  }

  endFrame(): void {
    this.markEnd('total')
  }

  begin(label: Exclude<PassLabel, 'total'>): void {
    this.markBegin(label)
  }

  end(label: Exclude<PassLabel, 'total'>): void {
    this.markEnd(label)
  }

  stats(): { shadow: number; main: number; post: number; total: number; reason: string } {
    this.drain()
    const shadow = percentile([...this.frameSamples.shadow.values()].sort((a, b) => a - b), 0.95)
    const post = percentile([...this.frameSamples.post.values()].sort((a, b) => a - b), 0.95)
    const total = percentile([...this.frameSamples.total.values()].sort((a, b) => a - b), 0.95)
    return { shadow, main: Math.max(0, total - shadow - post), post, total, reason: this.failureReason }
  }

  drain(): void {
    if (!this.extension) return
    if (this.gl.getParameter(this.extension.GPU_DISJOINT_EXT)) {
      this.failureReason = 'GPU reported disjoint pass timing data'
      this.clearPending()
      for (const samples of Object.values(this.frameSamples)) samples.clear()
      return
    }
    while (this.pending.length > 0) {
      const span = this.pending[0]
      if (!span || !this.available(span.start) || !this.available(span.end)) break
      this.pending.shift()
      const start = Number(this.gl.getQueryParameter(span.start, this.gl.QUERY_RESULT))
      const end = Number(this.gl.getQueryParameter(span.end, this.gl.QUERY_RESULT))
      this.release(span.start)
      this.release(span.end)
      const milliseconds = (end - start) / 1_000_000
      if (!Number.isFinite(milliseconds) || milliseconds < 0) {
        this.failureReason = 'GPU pass timer returned an invalid timestamp pair'
        continue
      }
      const samples = this.frameSamples[span.label]
      samples.set(span.frame, (samples.get(span.frame) ?? 0) + milliseconds)
      while (samples.size > this.settings.passSampleWindow) samples.delete(samples.keys().next().value as number)
      this.failureReason = ''
    }
  }

  dispose(): void {
    for (const { query } of this.open.values()) this.gl.deleteQuery(query)
    this.open.clear()
    this.clearPending()
    for (const query of this.pool) this.gl.deleteQuery(query)
    this.pool.length = 0
  }

  private markBegin(label: PassLabel): void {
    if (!this.extension || this.open.has(label) || this.pending.length >= this.settings.passMaxPending) return
    const query = this.take()
    this.extension.queryCounterEXT(query, this.extension.TIMESTAMP_EXT)
    this.open.set(label, { frame: this.frame, query })
  }

  private markEnd(label: PassLabel): void {
    if (!this.extension) return
    const active = this.open.get(label)
    if (!active) return
    const end = this.take()
    this.extension.queryCounterEXT(end, this.extension.TIMESTAMP_EXT)
    this.open.delete(label)
    this.pending.push({ label, frame: active.frame, start: active.query, end })
  }

  private take(): WebGLQuery {
    return this.pool.pop() ?? this.gl.createQuery()
  }

  private release(query: WebGLQuery): void {
    this.pool.push(query)
  }

  private available(query: WebGLQuery): boolean {
    return Boolean(this.gl.getQueryParameter(query, this.gl.QUERY_RESULT_AVAILABLE))
  }

  private clearPending(): void {
    for (const span of this.pending) {
      this.gl.deleteQuery(span.start)
      this.gl.deleteQuery(span.end)
    }
    this.pending.length = 0
  }
}

const timers = new WeakMap<object, PassTimer>()

export interface PassDrawCounts {
  shadow: { calls: number; triangles: number }
  post: { calls: number; triangles: number }
}

const drawCounts = new WeakMap<object, PassDrawCounts>()

export function beginPassDrawFrame(renderer: object): void {
  drawCounts.set(renderer, { shadow: { calls: 0, triangles: 0 }, post: { calls: 0, triangles: 0 } })
}

export function addPassDraws(renderer: object, pass: keyof PassDrawCounts, calls: number, triangles: number): void {
  const counts = drawCounts.get(renderer) ?? { shadow: { calls: 0, triangles: 0 }, post: { calls: 0, triangles: 0 } }
  counts[pass].calls += Math.max(0, calls)
  counts[pass].triangles += Math.max(0, triangles)
  drawCounts.set(renderer, counts)
}

export function passDrawsFor(renderer: object): PassDrawCounts {
  return drawCounts.get(renderer) ?? { shadow: { calls: 0, triangles: 0 }, post: { calls: 0, triangles: 0 } }
}

export function passTimerFor(renderer: object, gl: WebGL2RenderingContext, settings?: PassSettings): PassTimer {
  const existing = timers.get(renderer)
  if (existing) {
    if (settings) existing.configure(settings)
    return existing
  }
  const timer = new PassTimer(gl, settings)
  timers.set(renderer, timer)
  return timer
}

export function disposePassTimer(renderer: object): void {
  timers.get(renderer)?.dispose()
  timers.delete(renderer)
  drawCounts.delete(renderer)
}

function percentile(sorted: number[], ratio: number): number {
  if (sorted.length === 0) return 0
  return sorted[Math.min(sorted.length - 1, Math.floor(ratio * (sorted.length - 1)))] ?? 0
}
