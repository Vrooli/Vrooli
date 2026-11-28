import type { GeneratedWorld } from '../sim/model'

/** Retained-size estimate: exact unique backing buffers plus conservative JS
 * object/slot/string allowances. This is not a measurement of browser heap.
 */
export function generatedWorldBytes(world: GeneratedWorld): number {
  const seen = new Set<object>()
  function size(value: unknown): number {
    if (typeof value === 'string') return value.length * 2
    if (value === null || typeof value !== 'object') return 8
    if (seen.has(value)) return 0
    seen.add(value)
    if (ArrayBuffer.isView(value)) return 64 + size(value.buffer)
    if (value instanceof ArrayBuffer) return value.byteLength
    return 64 + Object.entries(value).reduce((total, [key, child]) => total + 16 + key.length * 2 + size(child), 0)
  }
  return size(world)
}

/** Cache owns references, never live state or exclusive buffer ownership.
 * Eviction must not dispose or detach buffers borrowed by a visible scene.
 */
export class GeneratedWorldCache {
  private entries = new Map<string, { world: GeneratedWorld; bytes: number }>()
  private bytes = 0
  private hits = 0
  private misses = 0
  private evictions = 0
  private rejected = 0
  constructor(readonly budgetBytes = 64 * 1024 * 1024, readonly maxEntries = 4) {
    if (!Number.isSafeInteger(budgetBytes) || budgetBytes < 0 || !Number.isSafeInteger(maxEntries) || maxEntries < 0) throw new Error('Invalid generated-world cache budget')
  }
  get(identity: string): GeneratedWorld | undefined {
    const entry = this.entries.get(identity)
    if (!entry) { this.misses++; return undefined }
    this.hits++
    this.entries.delete(identity)
    this.entries.set(identity, entry)
    return entry.world
  }
  put(identity: string, world: GeneratedWorld): boolean {
    const bytes = generatedWorldBytes(world) + identity.length * 2
    if (bytes > this.budgetBytes || this.maxEntries === 0) { this.rejected++; return false }
    const previous = this.entries.get(identity)
    if (previous) { this.entries.delete(identity); this.bytes -= previous.bytes }
    while (this.entries.size >= this.maxEntries || this.bytes + bytes > this.budgetBytes) {
      const oldest = this.entries.keys().next().value
      if (oldest === undefined) break
      const entry = this.entries.get(oldest)
      this.entries.delete(oldest)
      this.bytes -= entry?.bytes ?? 0
      this.evictions++
    }
    this.entries.set(identity, { world, bytes })
    this.bytes += bytes
    return true
  }
  stats() {
    return { entries: this.entries.size, estimatedBytes: this.bytes, budgetBytes: this.budgetBytes, maxEntries: this.maxEntries, hits: this.hits, misses: this.misses, evictions: this.evictions, rejected: this.rejected }
  }
}
