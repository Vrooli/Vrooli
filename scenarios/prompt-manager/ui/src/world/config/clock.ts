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
  subscribe(listener: () => void): () => void {
    this.listeners.add(listener)
    return () => { this.listeners.delete(listener) }
  }
  private notify() { for (const listener of this.listeners) listener() }
}
