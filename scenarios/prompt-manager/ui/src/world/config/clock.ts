export interface WorldClockSnapshot {
  utcMilliseconds: number
  localMinutes: number
  timeScale: number
  mode: 'clock' | 'fixed'
  timeZone: string
}

export function isTimeZone(value: string): boolean {
  try { new Intl.DateTimeFormat('en-GB', { timeZone: value }); return true }
  catch { return false }
}

/** Shared presentation clock. Authoritative agent/feed timestamps remain independent. */
export class WorldClock {
  private fixed: { utc: number; origin: number; scale: number } | null = null
  private listeners = new Set<() => void>()
  private formatter: Intl.DateTimeFormat | null
  private timeZone: string
  constructor(private readonly wallNow: () => number = Date.now, timeZone?: string) {
    this.formatter = timeZone ? new Intl.DateTimeFormat('en-GB', { timeZone, hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23' }) : null
    this.timeZone = (this.formatter ?? new Intl.DateTimeFormat()).resolvedOptions().timeZone
  }
  /** Called on focus/time refresh; avoid constructing an Intl formatter per frame. */
  refreshTimeZone() { if (!this.formatter) this.timeZone = new Intl.DateTimeFormat().resolvedOptions().timeZone }
  snapshot(): WorldClockSnapshot {
    const utcMilliseconds = this.fixed ? this.fixed.utc + (this.wallNow() - this.fixed.origin) * this.fixed.scale : this.wallNow()
    const date = new Date(utcMilliseconds)
    let localMinutes = date.getHours() * 60 + date.getMinutes() + date.getSeconds() / 60
    if (this.formatter) {
      const parts = this.formatter.formatToParts(date)
      const number = (type: string) => Number(parts.find(part => part.type === type)?.value ?? 0)
      localMinutes = number('hour') * 60 + number('minute') + number('second') / 60
    }
    return { utcMilliseconds, localMinutes, timeScale: this.fixed?.scale ?? 1, mode: this.fixed ? 'fixed' : 'clock', timeZone: this.timeZone }
  }
  fix(utcMilliseconds: number, timeScale = 0) {
    if (!Number.isFinite(utcMilliseconds) || Math.abs(utcMilliseconds) > 1e15 || !Number.isFinite(timeScale) || timeScale < 0 || timeScale > 1000) throw new Error('Invalid world clock instant or scale')
    this.fixed = { utc: utcMilliseconds, origin: this.wallNow(), scale: timeScale }
    this.notify()
  }
  live() { this.fixed = null; this.notify() }
  /** Seek to a civil time in this clock's zone. Offset changes are resolved from
   * the candidate date; nonexistent DST minutes advance to the next valid hour.
   */
  fixLocalTime(localMinutes: number) {
    if (!Number.isInteger(localMinutes) || localMinutes < 0 || localMinutes >= 1440) throw new Error('Invalid local time')
    const snapshot = this.snapshot()
    let utc = snapshot.utcMilliseconds - snapshot.utcMilliseconds % 1000 + (localMinutes - snapshot.localMinutes) * 60000
    const readMinutes = (instant: number) => {
      const date = new Date(instant)
      if (!this.formatter) return date.getHours() * 60 + date.getMinutes()
      const parts = this.formatter.formatToParts(date)
      return Number(parts.find(p => p.type === 'hour')?.value) * 60 + Number(parts.find(p => p.type === 'minute')?.value)
    }
    const first = utc
    for (let attempt = 0; attempt < 3; attempt++) {
      const delta = localMinutes - readMinutes(utc)
      if (delta === 0) break
      const next = utc + delta * 60000
      if (next === first) { utc = Math.max(first, utc); break }
      utc = next
    }
    this.fix(utc)
  }
  subscribe(listener: () => void): () => void {
    this.listeners.add(listener)
    return () => { this.listeners.delete(listener) }
  }
  private notify() { for (const listener of this.listeners) listener() }
}
