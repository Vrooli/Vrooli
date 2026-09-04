export interface GpuFrameStats {
  p50: number
  p95: number
  samples: number
  available: boolean
  reason: string
}

interface TimerQueryExtension {
  TIME_ELAPSED_EXT: number
  GPU_DISJOINT_EXT: number
}

const MAX_IN_FLIGHT = 4
const SAMPLE_WINDOW = 120

/** Non-blocking WebGL2 timer-query ring. Completed queries are drained on later frames. */
export class GpuTimer {
  private readonly extension: TimerQueryExtension | null
  private readonly pending: WebGLQuery[] = []
  private readonly samples: number[] = []
  private active: WebGLQuery | null = null
  private failureReason = ''

  constructor(private readonly gl: WebGL2RenderingContext) {
    this.extension = gl.getExtension('EXT_disjoint_timer_query_webgl2') as TimerQueryExtension | null
    if (!this.extension) this.failureReason = 'EXT_disjoint_timer_query_webgl2 unavailable'
  }

  begin(): void {
    if (!this.extension || this.active || this.pending.length >= MAX_IN_FLIGHT) return
    const query = this.gl.createQuery()
    try {
      this.gl.beginQuery(this.extension.TIME_ELAPSED_EXT, query)
      this.active = query
    } catch (error) {
      this.gl.deleteQuery(query)
      this.failureReason = error instanceof Error ? error.message : String(error)
    }
  }

  end(): void {
    if (!this.extension || !this.active) return
    try {
      this.gl.endQuery(this.extension.TIME_ELAPSED_EXT)
      this.pending.push(this.active)
    } catch (error) {
      this.gl.deleteQuery(this.active)
      this.failureReason = error instanceof Error ? error.message : String(error)
    } finally {
      this.active = null
    }
  }

  drain(): void {
    if (!this.extension) return
    if (this.gl.getParameter(this.extension.GPU_DISJOINT_EXT)) {
      this.samples.length = 0
      this.failureReason = 'GPU reported disjoint timer data'
      this.clearPending()
      return
    }
    while (this.pending.length > 0) {
      const query = this.pending[0]
      if (!query || !this.gl.getQueryParameter(query, this.gl.QUERY_RESULT_AVAILABLE)) break
      this.pending.shift()
      const nanoseconds = Number(this.gl.getQueryParameter(query, this.gl.QUERY_RESULT))
      this.gl.deleteQuery(query)
      if (!Number.isFinite(nanoseconds) || nanoseconds <= 0) {
        this.failureReason = 'GPU timer returned a zero or invalid sample'
        continue
      }
      this.samples.push(nanoseconds / 1_000_000)
      if (this.samples.length > SAMPLE_WINDOW) this.samples.shift()
      this.failureReason = ''
    }
  }

  stats(): GpuFrameStats {
    const sorted = [...this.samples].sort((a, b) => a - b)
    return {
      p50: percentile(sorted, 0.5),
      p95: percentile(sorted, 0.95),
      samples: sorted.length,
      available: Boolean(this.extension) && this.failureReason === '',
      reason: this.failureReason,
    }
  }

  dispose(): void {
    if (this.active) this.gl.deleteQuery(this.active)
    this.active = null
    this.clearPending()
  }

  private clearPending(): void {
    for (const query of this.pending) this.gl.deleteQuery(query)
    this.pending.length = 0
  }
}

function percentile(sorted: number[], ratio: number): number {
  if (sorted.length === 0) return 0
  return sorted[Math.min(sorted.length - 1, Math.floor(ratio * (sorted.length - 1)))] ?? 0
}
