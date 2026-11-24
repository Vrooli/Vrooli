import type { GeneratedWorld } from '../sim/model'
import type { GenerationProgress } from '../sim/world'
import { createGenerationService, validateGenerationJob, type GenerationJob, type GenerationReply, type GenerationRequest } from './generation'

export interface GenerationWorker {
  postMessage(request: GenerationRequest): void
  terminate(): void
  onmessage: ((event: MessageEvent<GenerationReply>) => void) | null
  onerror: ((event: ErrorEvent) => void) | null
}

/** Latest-request ownership. Existing rendered arrays are never sent to a worker. */
export class GenerationClient {
  private epoch = 0
  private disposed = false
  private pending?: {
    epoch: number
    resolve: (world: GeneratedWorld) => void
    reject: (error: Error) => void
    progress?: (progress: GenerationProgress) => void
    cleanup: () => void
  }
  constructor(private readonly worker: GenerationWorker, readonly mode: 'worker' | 'fallback' = 'worker', readonly fallbackReason?: string) {
    worker.onmessage = ({ data }) => {
      const pending = this.pending
      if (!pending || data.epoch !== pending.epoch) return
      if (data.kind === 'progress') { pending.progress?.(data.progress); return }
      this.pending = undefined
      pending.cleanup()
      if (data.kind === 'complete') pending.resolve(data.world)
      else pending.reject(data.kind === 'cancelled' ? new DOMException('Generation cancelled', 'AbortError') : new Error(`${data.code}: ${data.detail}`))
    }
    worker.onerror = event => {
      const pending = this.pending
      this.pending = undefined
      pending?.cleanup()
      pending?.reject(new Error(`Generation worker failed: ${event.message}`))
      this.dispose()
    }
  }
  generate(job: GenerationJob, options: { signal?: AbortSignal; onProgress?: (progress: GenerationProgress) => void } = {}): Promise<GeneratedWorld> {
    if (this.disposed) return Promise.reject(new Error('Generation client is disposed'))
    try { job = validateGenerationJob(job); options.signal?.throwIfAborted() } catch (error) { return Promise.reject(error instanceof Error ? error : new Error(String(error))) }
    this.cancel()
    const epoch = ++this.epoch
    return new Promise((resolve, reject) => {
      const abort = () => { if (this.pending?.epoch === epoch) this.cancel() }
      const cleanup = () => options.signal?.removeEventListener('abort', abort)
      this.pending = { epoch, resolve, reject, cleanup, progress: options.onProgress }
      options.signal?.addEventListener('abort', abort, { once: true })
      try { this.worker.postMessage({ kind: 'generate', epoch, job }) } catch (error) {
        this.pending = undefined
        cleanup()
        reject(error instanceof Error ? error : new Error(String(error)))
      }
    })
  }
  cancel(): void {
    const pending = this.pending
    if (!pending) return
    this.pending = undefined
    pending.cleanup()
    pending.reject(new DOMException('Generation cancelled', 'AbortError'))
    this.worker.postMessage({ kind: 'cancel', epoch: pending.epoch })
  }
  dispose(): void {
    if (this.disposed) return
    this.disposed = true
    try { this.cancel() } finally {
      this.worker.onmessage = null
      this.worker.onerror = null
      this.worker.terminate()
    }
  }
}

export function createGenerationClient(): GenerationClient {
  try {
    return new GenerationClient(new Worker(new URL('./generation.worker.ts', import.meta.url), { type: 'module' }))
  } catch (error) {
    return new GenerationClient(new CooperativeGenerationTransport(), 'fallback', String(error))
  }
}

/** Main-thread transport shares the worker's algorithms, scheduler and epoch ownership.
 * No buffers are transferred or cloned: only completed generation-owned output is delivered.
 */
export class CooperativeGenerationTransport implements GenerationWorker {
  onmessage: GenerationWorker['onmessage'] = null
  onerror: GenerationWorker['onerror'] = null
  private disposed = false
  private readonly service = createGenerationService(reply => {
    if (!this.disposed) this.onmessage?.({ data: reply } as MessageEvent<GenerationReply>)
  }, { transfer: false })

  postMessage(request: GenerationRequest): void {
    if (this.disposed) throw new Error('Generation transport is disposed')
    // Match asynchronous worker delivery; the service yields between generation slices.
    void Promise.resolve().then(() => {
      if (!this.disposed) return this.service.receive(request)
    }).catch((error: unknown) => {
      if (!this.disposed) this.onerror?.({ message: String(error) } as ErrorEvent)
    })
  }
  terminate(): void {
    if (this.disposed) return
    this.disposed = true
    this.service.dispose()
    this.onmessage = null
    this.onerror = null
  }
}
