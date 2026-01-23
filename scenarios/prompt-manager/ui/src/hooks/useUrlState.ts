/**
 * useUrlState - Hook for syncing UI state with URL search parameters.
 *
 * This hook enables:
 * - URL-backed state persistence across page refreshes
 * - Direct navigation to skills via URL (/?skill=<skillId>)
 * - Direct navigation to settings modal (/?settings=true)
 * - Browser back/forward navigation support
 *
 * URL Structure:
 * - /?skill=<skillId> - Select and open skill for editing
 * - /?settings=true - Open settings modal
 * - /?skill=<id>&settings=true - Both simultaneously
 */

import { useEffect, useCallback, useRef } from 'react'

export interface UrlState {
  skillId: string | null
  settingsOpen: boolean
}

export interface UseUrlStateOptions {
  /** Called when skill ID changes from URL navigation */
  onSkillIdChange: (id: string | null) => void
  /** Called when settings open state changes from URL navigation */
  onSettingsOpenChange: (open: boolean) => void
  /** Whether there are unsaved changes */
  isDirty: boolean
  /** Function to store current changes before navigation */
  storeCurrentChanges: () => void
}

export interface UseUrlStateReturn {
  /** Update URL with new state (without page reload) */
  updateUrl: (state: Partial<UrlState>) => void
  /** Get initial state from current URL */
  getInitialState: () => UrlState
}

/** URL parameter names */
const URL_PARAMS = {
  SKILL: 'skill',
  SETTINGS: 'settings',
} as const

/**
 * Parse URL search parameters into UrlState
 */
function parseUrlState(search: string): UrlState {
  const params = new URLSearchParams(search)
  return {
    skillId: params.get(URL_PARAMS.SKILL),
    settingsOpen: params.get(URL_PARAMS.SETTINGS) === 'true',
  }
}

/**
 * Build URL search string from UrlState
 */
function buildUrlSearch(state: UrlState): string {
  const params = new URLSearchParams()

  if (state.skillId) {
    params.set(URL_PARAMS.SKILL, state.skillId)
  }

  if (state.settingsOpen) {
    params.set(URL_PARAMS.SETTINGS, 'true')
  }

  const search = params.toString()
  return search ? `?${search}` : ''
}

/**
 * Custom hook for syncing UI state with URL search parameters.
 *
 * Uses the browser History API to update the URL without page reloads.
 * Handles browser back/forward navigation via the popstate event.
 */
export function useUrlState(options: UseUrlStateOptions): UseUrlStateReturn {
  const { onSkillIdChange, onSettingsOpenChange } = options

  // Store current URL state to detect changes
  const currentStateRef = useRef<UrlState>({
    skillId: null,
    settingsOpen: false,
  })

  // Store options in ref to avoid stale closures
  const optionsRef = useRef(options)
  useEffect(() => {
    optionsRef.current = options
  }, [options])

  /**
   * Get initial state from current URL
   */
  const getInitialState = useCallback((): UrlState => {
    if (typeof window === 'undefined') {
      return { skillId: null, settingsOpen: false }
    }
    return parseUrlState(window.location.search)
  }, [])

  /**
   * Update URL with new state without triggering page reload
   */
  const updateUrl = useCallback((partialState: Partial<UrlState>) => {
    if (typeof window === 'undefined') return

    // Merge with current state
    const newState: UrlState = {
      ...currentStateRef.current,
      ...partialState,
    }

    // Skip if state hasn't changed
    if (
      newState.skillId === currentStateRef.current.skillId &&
      newState.settingsOpen === currentStateRef.current.settingsOpen
    ) {
      return
    }

    // Update ref
    currentStateRef.current = newState

    // Build new URL
    const newSearch = buildUrlSearch(newState)
    const newUrl = `${window.location.pathname}${newSearch}`

    // Update URL without triggering navigation
    window.history.replaceState({ ...newState }, '', newUrl)
  }, [])

  /**
   * Handle popstate event (browser back/forward)
   */
  const handlePopState = useCallback((event: PopStateEvent) => {
    const opts = optionsRef.current
    const newState = event.state as UrlState | null

    // If no state in history, parse from URL
    const urlState = newState ?? parseUrlState(window.location.search)

    // Store changes before navigation if dirty
    if (opts.isDirty) {
      opts.storeCurrentChanges()
    }

    // Update ref
    currentStateRef.current = urlState

    // Notify about changes
    opts.onSkillIdChange(urlState.skillId)
    opts.onSettingsOpenChange(urlState.settingsOpen)
  }, [])

  // Set up popstate listener
  useEffect(() => {
    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
  }, [handlePopState])

  // Initialize state from URL on mount
  useEffect(() => {
    const initialState = getInitialState()
    currentStateRef.current = initialState

    // Apply initial state (defer to allow component to mount)
    if (initialState.skillId || initialState.settingsOpen) {
      // Use setTimeout to ensure this runs after initial render
      setTimeout(() => {
        if (initialState.skillId) {
          onSkillIdChange(initialState.skillId)
        }
        if (initialState.settingsOpen) {
          onSettingsOpenChange(initialState.settingsOpen)
        }
      }, 0)
    }
    // Only run on mount
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return {
    updateUrl,
    getInitialState,
  }
}
