import type { ActorState } from '../sim'

export const STATE_LABEL: Record<ActorState, string> = {
  idle: 'Idle',
  walkingToDesk: 'Heading to desk',
  working: 'Working',
  failed: 'Failed',
  walkingToTable: 'Heading to table',
  gathered: 'Gathered',
  socializing: 'Chatting',
}

/** Compact duration: 42s, 3m 05s, 1h 12m. */
export function formatDuration(seconds: number): string {
  const s = Math.max(0, Math.round(seconds))
  if (s < 60) return `${s}s`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ${String(s % 60).padStart(2, '0')}s`
  const h = Math.floor(m / 60)
  return `${h}h ${String(m % 60).padStart(2, '0')}m`
}

/** T-mm:ss countdown, or "now" once due. */
export function formatCountdown(untilSeconds: number, nowSeconds: number): string {
  const delta = untilSeconds - nowSeconds
  if (delta <= 0) return 'now'
  const m = Math.floor(delta / 60)
  const s = Math.floor(delta % 60)
  return `T-${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

export function formatClock(seconds: number): string {
  const date = new Date(seconds * 1000)
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}
