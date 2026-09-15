import { useState } from 'react'
import { DEFAULT_NAVIGATION, INITIAL_NAVIGATION_STATE, NAVIGATION_STORAGE_KEY, parseNavigationPreferences, type NavigationPreferences, type NavigationState } from '../config/navigation'

/** Navigation telemetry updates its HUD without re-rendering the entire world. */
export function createNavigationTelemetry() {
  let state = INITIAL_NAVIGATION_STATE
  const listeners = new Set<() => void>()
  return {
    read: () => state,
    subscribe: (listener: () => void) => { listeners.add(listener); return () => { listeners.delete(listener) } },
    publish: (next: NavigationState) => {
      if (Object.keys(next).every(key => next[key as keyof NavigationState] === state[key as keyof NavigationState])) return
      state = next
      listeners.forEach(listener => listener())
    },
  }
}
export type NavigationTelemetry = ReturnType<typeof createNavigationTelemetry>

/** Device ergonomics belong to this browser, not the shared world or account. */
export function useNavigationPreferences() {
  const [value, setValue] = useState(() => {
    try { return parseNavigationPreferences(JSON.parse(localStorage.getItem(NAVIGATION_STORAGE_KEY) ?? 'null')) } catch { return DEFAULT_NAVIGATION }
  })
  const update = (patch: Partial<NavigationPreferences>) => setValue(previous => {
    const next = parseNavigationPreferences({ ...previous, ...patch })
    try { localStorage.setItem(NAVIGATION_STORAGE_KEY, JSON.stringify(next)) } catch { /* Session controls remain available without browser storage. */ }
    return next
  })
  return [value, update] as const
}

