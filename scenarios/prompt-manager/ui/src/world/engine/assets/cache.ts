import type { BufferGeometry } from 'three'
import type { PropPart } from './geometry'

export const PREPARATION_VERSION = 'canonical-float-v1'
/** Geometry reservation inside the plan's 256 MiB asset ceiling; textures are source-owned. */
export const PREPARED_GEOMETRY_BUDGET = 64 * 1024 * 1024

export interface PreparedAssetHandle {
  readonly key: string
  readonly parts: readonly PropPart[]
  readonly cpuBytes: number
  readonly estimatedGpuBytes: number
  /** Obtain an independent lease. Each lease is released at most once. */
  retain(): PreparedAssetHandle
  release(): void
}
interface Entry {
  key: string
  parts: PropPart[]
  bytes: number
  references: number
  used: number
}

/** Counts actual backing buffers once, including interleaved attributes and indices. */
export function geometryBytes(geometries: readonly BufferGeometry[]): number {
  const buffers = new Set<ArrayBufferLike>()
  for (const geometry of geometries) {
    for (const attribute of Object.values(geometry.attributes)) buffers.add(attribute.array.buffer)
    if (geometry.index) buffers.add(geometry.index.array.buffer)
  }
  return [...buffers].reduce((sum, buffer) => sum + buffer.byteLength, 0)
}

/** Owns canonical geometry only. Borrowed GLTF materials/textures are never disposed here. */
export class PreparedAssetCache {
  private entries = new Map<string, Entry>()
  private clock = 0
  private hits = 0
  private misses = 0
  private evictions = 0
  private disposals = 0

  constructor(private byteBudget: number) {
    if (!Number.isFinite(byteBudget) || byteBudget < 0) throw new Error('Prepared asset budget must be finite and nonnegative')
  }

  acquire(contentHash: string, prepare: () => PropPart[]): PreparedAssetHandle {
    const key = `${PREPARATION_VERSION}:${contentHash}`
    let entry = this.entries.get(key)
    if (entry) {
      this.hits++
    } else {
      this.misses++
      // Failed preparation is never cached; a subsequent acquisition may retry.
      const parts = prepare()
      entry = { key, parts, bytes: geometryBytes(parts.map(part => part.geometry)), references: 0, used: ++this.clock }
      this.entries.set(key, entry)
    }
    const handle = this.lease(entry)
    this.trim()
    return handle
  }

  private lease(entry: Entry): PreparedAssetHandle {
    entry.references++
    entry.used = ++this.clock
    let released = false
    return {
      key: entry.key,
      parts: entry.parts,
      cpuBytes: entry.bytes,
      estimatedGpuBytes: entry.bytes,
      retain: () => {
        if (released) throw new Error('Cannot retain a released prepared asset')
        return this.lease(entry)
      },
      release: () => {
        if (released) return
        released = true
        entry.references--
        entry.used = ++this.clock
        this.trim()
      },
    }
  }

  /** Memory pressure may reduce the resident budget; active assets remain valid. */
  setBudget(bytes: number): void {
    if (!Number.isFinite(bytes) || bytes < 0) throw new Error('Prepared asset budget must be finite and nonnegative')
    this.byteBudget = bytes
    this.trim()
  }

  /** Evict unused assets in LRU order. A zero target releases the entire idle set. */
  trim(target = this.byteBudget): void {
    let bytes = this.stats().cpuBytes
    const idle = [...this.entries.values()].filter(entry => entry.references === 0).sort((a, b) => a.used - b.used)
    for (const entry of idle) {
      if (bytes <= target) break
      this.entries.delete(entry.key)
      for (const geometry of new Set(entry.parts.map(part => part.geometry))) {
        geometry.dispose()
        this.disposals++
      }
      bytes -= entry.bytes
      this.evictions++
    }
  }

  stats() {
    const entries = [...this.entries.values()]
    const bytes = entries.reduce((sum, entry) => sum + entry.bytes, 0)
    return {
      residentAssets: entries.length,
      references: entries.reduce((sum, entry) => sum + entry.references, 0),
      cpuBytes: bytes,
      estimatedGpuBytes: bytes,
      budgetBytes: this.byteBudget,
      overBudgetBytes: Math.max(0, bytes - this.byteBudget),
      hits: this.hits, misses: this.misses, evictions: this.evictions, geometryDisposals: this.disposals,
    }
  }
}

export const preparedAssets = new PreparedAssetCache(PREPARED_GEOMETRY_BUDGET)
