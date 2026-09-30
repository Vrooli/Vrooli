/**
 * Layout override editing: a history of overrides with undo/redo, bounded by
 * a lever, and the pure apply/invert pair the editor and the store share.
 */
import type { LayoutOverride, Vec2 } from '../model'

export interface OverrideHistory {
  /** The committed override set, one per place id. */
  current: LayoutOverride[]
  past: LayoutOverride[][]
  future: LayoutOverride[][]
}

export function emptyHistory(): OverrideHistory {
  return { current: [], past: [], future: [] }
}

/** Merge one override into a set by place id; a removed flag wins, a later position replaces. */
export function upsertOverride(set: readonly LayoutOverride[], override: LayoutOverride): LayoutOverride[] {
  const index = set.findIndex((o) => o.placeId === override.placeId)
  if (index === -1) return [...set, override]
  const merged: LayoutOverride = { ...set[index], ...override }
  return set.map((o, i) => (i === index ? merged : o))
}

export function removeOverride(set: readonly LayoutOverride[], placeId: string): LayoutOverride[] {
  return set.filter((o) => o.placeId !== placeId)
}

/** Push a new override set; drops redo history and trims the undo stack to maxHistory. */
export function commit(history: OverrideHistory, next: LayoutOverride[], maxHistory: number): OverrideHistory {
  const past = [...history.past, history.current].slice(-Math.max(1, maxHistory))
  return { current: next, past, future: [] }
}

export function undo(history: OverrideHistory): OverrideHistory {
  const previous = history.past[history.past.length - 1]
  if (!previous) return history
  return { current: previous, past: history.past.slice(0, -1), future: [history.current, ...history.future] }
}

export function redo(history: OverrideHistory): OverrideHistory {
  const next = history.future[0]
  if (!next) return history
  return { current: next, past: [...history.past, history.current], future: history.future.slice(1) }
}

export function canUndo(history: OverrideHistory): boolean {
  return history.past.length > 0
}

export function canRedo(history: OverrideHistory): boolean {
  return history.future.length > 0
}

/** Snap a position to the editor grid. */
export function snapPosition(position: Vec2, snap: number): Vec2 {
  if (snap <= 0) return position
  return [Math.round(position[0] / snap) * snap, Math.round(position[1] / snap) * snap]
}

/** The override that would restore a place to what `previous` recorded (or the generated layout when absent). */
export function invertOverride(previous: LayoutOverride | undefined, placeId: string): LayoutOverride | null {
  return previous ? { ...previous } : { placeId, position: undefined, rotation: undefined, removed: false }
}
