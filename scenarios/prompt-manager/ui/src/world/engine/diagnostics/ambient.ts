export const AMBIENT_FAMILIES = ['sky', 'fireflies', 'butterflies', 'birds', 'rabbits', 'fish'] as const
export type AmbientFamily = typeof AMBIENT_FAMILIES[number]
const WINDOW = 256

/** CPU callback measurements only: excludes draw submission, GPU and generation.
 * Recording mutates fixed buffers; sorting/allocation happens only on inspection.
 */
export class AmbientMeasurements {
  private rows = AMBIENT_FAMILIES.map(family => ({ family, values: new Float64Array(WINDOW), samples: 0,
    latestMs: 0, active: 0, capacity: 0, limit: 0, mounted: false }))
  record(family: AmbientFamily, elapsedMs: number, active: number, capacity: number, limit = capacity) {
    const row = this.rows[AMBIENT_FAMILIES.indexOf(family)]
    if (!row || !Number.isFinite(elapsedMs) || elapsedMs < 0 || !Number.isInteger(active) || !Number.isInteger(capacity) || !Number.isInteger(limit) || active < 0 || active > limit || limit > capacity || capacity > 256) throw new Error('Invalid ambient measurement')
    row.values[row.samples % WINDOW] = elapsedMs
    row.samples++; row.latestMs = elapsedMs; row.active = active; row.capacity = capacity; row.limit = limit; row.mounted = true
  }
  clear(family: AmbientFamily) {
    const row = this.rows[AMBIENT_FAMILIES.indexOf(family)]
    if (row) { row.active = 0; row.capacity = 0; row.limit = 0; row.latestMs = 0; row.mounted = false }
  }
  reset() {
    for (const row of this.rows) { row.values.fill(0); row.samples = 0; this.clear(row.family) }
  }
  snapshot() {
    const families = this.rows.map(row => {
      const count = Math.min(WINDOW, row.samples)
      const sorted = [...row.values.subarray(0, count)].sort((a, b) => a - b)
      return { family: row.family, mounted: row.mounted, active: row.active, capacity: row.capacity, limit: row.limit,
        samples: row.samples, windowSamples: count, latestMs: row.latestMs,
        meanMs: count ? sorted.reduce((sum, value) => sum + value, 0) / count : 0,
        p95Ms: sorted[Math.floor(Math.max(0, count - 1) * .95)] ?? 0, maxMs: sorted[count - 1] ?? 0 }
    })
    return { windowSize: WINDOW, bufferBytes: this.rows.length * WINDOW * 8,
      latestTotalMs: families.reduce((sum, row) => sum + row.latestMs, 0), families }
  }
}

export const ambientMeasurements = new AmbientMeasurements()
