import { runCooperatively } from '../sim/cooperative'
import { terrainMeshSteps, type TerrainMeshData, type TerrainMeshInput } from '../sim/terrain/mesh'
import { canonical } from '../sim/recipe'

export type TerrainMeshReply = { kind: 'complete'; mesh: TerrainMeshData } | { kind: 'failed'; message: string }

/** One request owns one worker. Abort/settlement always terminates it. Source
 * buffers are cloned, never transferred from the active scene or world cache.
 */
export async function prepareTerrainMeshAsync(input: TerrainMeshInput, options: {
  signal?: AbortSignal
  onMode?: (mode: 'worker' | 'fallback' | 'cache') => void
} = {}): Promise<TerrainMeshData> {
  options.signal?.throwIfAborted()
  let worker: Worker
  try { worker = new Worker(new URL('./terrainMesh.worker.ts', import.meta.url), { type: 'module' }) } catch {
    options.onMode?.('fallback')
    return runCooperatively(terrainMeshSteps(input), { signal: options.signal })
  }
  return new Promise((resolve, reject) => {
    let settled = false
    let workerStopped = false
    const stopWorker = () => {
      if (workerStopped) return
      workerStopped = true
      worker.onmessage = null
      worker.onerror = null
      worker.terminate()
    }
    const finish = (mesh?: TerrainMeshData, error?: unknown) => {
      if (settled) return
      settled = true
      // The abort handler is initialized before any completion callback is installed.
      // eslint-disable-next-line @typescript-eslint/no-use-before-define
      options.signal?.removeEventListener('abort', abort)
      stopWorker()
      if (mesh) resolve(mesh)
      else reject(error instanceof Error ? error : new Error(String(error)))
    }
    const abort = () => finish(undefined, options.signal?.reason ?? new DOMException('Mesh preparation cancelled', 'AbortError'))
    worker.onmessage = ({ data }: MessageEvent<TerrainMeshReply>) => {
      if (data.kind === 'complete') finish(data.mesh)
      else finish(undefined, new Error(data.message))
    }
    worker.onerror = event => {
      event.preventDefault()
      if (settled || workerStopped) return
      stopWorker()
      // A failed worker script or runtime leaves the caller's source arrays intact.
      // Recover with the same algorithm while preserving the request's abort signal.
      void Promise.resolve().then(() => {
        options.signal?.throwIfAborted()
        options.onMode?.('fallback')
        return runCooperatively(terrainMeshSteps(input), { signal: options.signal })
      }).then(mesh => finish(mesh)).catch((error: unknown) => finish(undefined, error))
    }
    options.signal?.addEventListener('abort', abort, { once: true })
    try {
      options.onMode?.('worker')
      if (options.signal?.aborted) { abort(); return }
      worker.postMessage(input)
    } catch (error) { finish(undefined, error) }
  })
}

type MeshOptions = NonNullable<Parameters<typeof prepareTerrainMeshAsync>[1]>
export interface TerrainMeshCacheStats { entries: number; estimatedBytes: number; budgetBytes: number; hits: number; misses: number; evictions: number }

/** Completed CPU meshes only. Eviction releases references without disposing or
 * detaching buffers still borrowed by visible Three.js geometry.
 */
export class TerrainMeshCache {
  private entries: Array<{ input: TerrainMeshInput; key: string; mesh: TerrainMeshData; bytes: number }> = []
  private bytes = 0
  private hits = 0
  private misses = 0
  private evictions = 0
  constructor(private readonly options: {
    budgetBytes?: number; maxEntries?: number
    prepare?: typeof prepareTerrainMeshAsync
    onStats?: (stats: TerrainMeshCacheStats) => void
  } = {}) {
    if (!Number.isSafeInteger(options.budgetBytes ?? 32 * 1024 * 1024) || (options.budgetBytes ?? 1) < 0 || !Number.isSafeInteger(options.maxEntries ?? 4) || (options.maxEntries ?? 1) < 1) throw new Error('Invalid terrain mesh cache budget')
  }
  stats = (): TerrainMeshCacheStats => ({ entries: this.entries.length, estimatedBytes: this.bytes, budgetBytes: this.options.budgetBytes ?? 32 * 1024 * 1024, hits: this.hits, misses: this.misses, evictions: this.evictions })
  prepare = async (input: TerrainMeshInput, options: MeshOptions = {}): Promise<TerrainMeshData> => {
    options.signal?.throwIfAborted()
    const { field, surface, ...settings } = input
    const key = canonical(settings)
    const index = this.entries.findIndex(entry => entry.input.field === field && entry.input.surface === surface && entry.key === key)
    if (index >= 0) {
      const entry = this.entries.splice(index, 1)[0]
      if (!entry) throw new Error('Missing terrain cache entry')
      this.entries.push(entry)
      this.hits++
      options.onMode?.('cache')
      options.signal?.throwIfAborted()
      this.options.onStats?.(this.stats())
      return entry.mesh
    }
    this.misses++
    this.options.onStats?.(this.stats())
    const mesh = await (this.options.prepare ?? prepareTerrainMeshAsync)(input, options)
    options.signal?.throwIfAborted()
    // Charge retained source buffers as well as output, conservatively counting
    // shared buffers again across entries. This is a retention estimate, not heap telemetry.
    const buffers = new Set([field.height.buffer, field.moisture.buffer, surface.normals.buffer, surface.colours.buffer, surface.wetShade.buffer, surface.aoStrength.buffer, mesh.vertices.buffer, mesh.normals.buffer, mesh.colours.buffer, mesh.indices.buffer])
    const bytes = [...buffers].reduce((sum, buffer) => sum + buffer.byteLength, key.length * 2 + 1024)
    if (bytes <= this.stats().budgetBytes) {
      while (this.entries.length >= (this.options.maxEntries ?? 4) || this.bytes + bytes > this.stats().budgetBytes) {
        const removed = this.entries.shift()
        if (!removed) break
        this.bytes -= removed.bytes
        this.evictions++
      }
      this.entries.push({ input, key, mesh, bytes })
      this.bytes += bytes
    }
    this.options.onStats?.(this.stats())
    return mesh
  }
}
