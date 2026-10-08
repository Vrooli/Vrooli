export interface PoolEvent { readonly id: string; readonly start: number; readonly end: number }
export interface PresentationSlot<T> { readonly index: number; event: T | null }

/** Fixed renderer slots. Dropped candidates never queue or alter scheduled identity. */
export class PresentationPool<T extends PoolEvent> {
  readonly slots: readonly PresentationSlot<T>[]
  private acquired = 0
  private released = 0
  private rejected = 0
  constructor(readonly capacity: number) {
    if (!Number.isInteger(capacity) || capacity < 1 || capacity > 256) throw new Error('Invalid ambient pool capacity')
    this.slots = Array.from({ length: capacity }, (_, index) => ({ index, event: null }))
  }
  /** Priority is start then ID, independent of caller iteration order. Existing
   * selected events retain slots; quality may reduce admission without rescheduling.
   */
  sync(events: readonly T[], now: number, limit = this.capacity): void {
    if (!Number.isFinite(now) || !Number.isInteger(limit) || limit < 0 || limit > this.capacity || events.length > 1024) throw new Error('Invalid ambient pool update')
    const unique = new Map<string, T>()
    for (const event of events) {
      if (!event.id || !Number.isFinite(event.start) || !Number.isFinite(event.end) || event.end <= event.start || unique.has(event.id)) throw new Error('Invalid ambient pool event')
      unique.set(event.id, event)
    }
    const active = [...unique.values()].filter(event => event.start <= now && now < event.end)
      .sort((a, b) => a.start - b.start || (a.id < b.id ? -1 : a.id > b.id ? 1 : 0))
    const selected = new Map(active.slice(0, limit).map(event => [event.id, event]))
    // Count current rejected candidates, not a frame-rate-dependent cumulative total.
    this.rejected = Math.max(0, active.length - limit)
    for (const slot of this.slots) {
      if (!slot.event) continue
      const retained = selected.get(slot.event.id)
      if (retained) { slot.event = retained; selected.delete(retained.id) }
      else { slot.event = null; this.released++ }
    }
    for (const event of selected.values()) {
      const slot = this.slots.find(candidate => candidate.event === null)
      if (slot) { slot.event = event; this.acquired++ }
    }
  }
  clear(): void {
    for (const slot of this.slots) if (slot.event) { slot.event = null; this.released++ }
    this.rejected = 0
  }
  stats() {
    return { capacity: this.capacity, active: this.slots.filter(slot => slot.event !== null).length,
      acquired: this.acquired, released: this.released, rejected: this.rejected }
  }
}
