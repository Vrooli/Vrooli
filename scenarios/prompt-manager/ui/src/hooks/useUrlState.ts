/**
 * useUrlState - Hook for syncing UI state with URL search parameters.
 *
 * This hook enables:
 * - URL-backed state persistence across page refreshes
 * - Direct navigation to skills via URL (/?skill=<skillId>)
 * - Direct navigation to agents via URL (/?agent=<agentId>)
 * - Direct navigation to teams via URL (/?team=<teamId>)
 * - Direct navigation to settings modal (/?settings=true)
 * - Cross-reference highlight navigation (/?hlFile=...&hlLine=...&hlText=...)
 * - Browser back/forward navigation support
 *
 * URL Structure:
 * - /?skill=<skillId> - Select and open skill for editing
 * - /?agent=<agentId> - Select and open agent for editing
 * - /?team=<teamId> - Select and open team for editing
 * - /?settings=true - Open settings modal
 * - /?hlFile=<path>&hlLine=<n>&hlText=<text> - Highlight a reference
 * - Combinations: /?skill=<id>&settings=true, etc.
 */

import { useEffect, useCallback, useRef } from 'react'
import type { HighlightRequest } from '@/lib/highlight'

export interface UrlState {
  skillId: string | null
  agentId: string | null
  teamId: string | null
  settingsOpen: boolean
  hlFile: string | null
  hlLine: number | null
  hlText: string | null
}

export interface UseUrlStateOptions {
  /** Called when skill ID changes from URL navigation */
  onSkillIdChange: (id: string | null) => void
  /** Called when agent ID changes from URL navigation */
  onAgentIdChange: (id: string | null) => void
  /** Called when team ID changes from URL navigation */
  onTeamIdChange: (id: string | null) => void
  /** Called when settings open state changes from URL navigation */
  onSettingsOpenChange: (open: boolean) => void
  /** Called when highlight params change from URL navigation */
  onHighlightChange?: (hl: HighlightRequest | null) => void
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
  AGENT: 'agent',
  TEAM: 'team',
  SETTINGS: 'settings',
  HL_FILE: 'hlFile',
  HL_LINE: 'hlLine',
  HL_TEXT: 'hlText',
} as const

/**
 * Parse URL search parameters into UrlState
 */
function parseUrlState(search: string): UrlState {
  const params = new URLSearchParams(search)
  const hlLineRaw = params.get(URL_PARAMS.HL_LINE)
  const hlLine = hlLineRaw ? parseInt(hlLineRaw, 10) : null
  return {
    skillId: params.get(URL_PARAMS.SKILL),
    agentId: params.get(URL_PARAMS.AGENT),
    teamId: params.get(URL_PARAMS.TEAM),
    settingsOpen: params.get(URL_PARAMS.SETTINGS) === 'true',
    hlFile: params.get(URL_PARAMS.HL_FILE),
    hlLine: hlLine !== null && !Number.isNaN(hlLine) ? hlLine : null,
    hlText: params.get(URL_PARAMS.HL_TEXT),
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

  if (state.agentId) {
    params.set(URL_PARAMS.AGENT, state.agentId)
  }

  if (state.teamId) {
    params.set(URL_PARAMS.TEAM, state.teamId)
  }

  if (state.settingsOpen) {
    params.set(URL_PARAMS.SETTINGS, 'true')
  }

  if (state.hlFile) {
    params.set(URL_PARAMS.HL_FILE, state.hlFile)
  }

  if (state.hlLine !== null && state.hlLine !== undefined) {
    params.set(URL_PARAMS.HL_LINE, String(state.hlLine))
  }

  if (state.hlText) {
    params.set(URL_PARAMS.HL_TEXT, state.hlText)
  }

  const search = params.toString()
  return search ? `?${search}` : ''
}

/**
 * Extract a HighlightRequest from UrlState, or null if not present.
 */
function extractHighlightRequest(state: UrlState): HighlightRequest | null {
  if (state.hlLine !== null && state.hlText) {
    return {
      file: state.hlFile ?? undefined,
      line: state.hlLine,
      text: state.hlText,
    }
  }
  return null
}

/**
 * Custom hook for syncing UI state with URL search parameters.
 *
 * Uses the browser History API to update the URL without page reloads.
 * Handles browser back/forward navigation via the popstate event.
 */
export function useUrlState(options: UseUrlStateOptions): UseUrlStateReturn {
  const { onSkillIdChange, onAgentIdChange, onTeamIdChange, onSettingsOpenChange } = options

  // Store current URL state to detect changes
  const currentStateRef = useRef<UrlState>({
    skillId: null,
    agentId: null,
    teamId: null,
    settingsOpen: false,
    hlFile: null,
    hlLine: null,
    hlText: null,
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
      return { skillId: null, agentId: null, teamId: null, settingsOpen: false, hlFile: null, hlLine: null, hlText: null }
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
      newState.agentId === currentStateRef.current.agentId &&
      newState.teamId === currentStateRef.current.teamId &&
      newState.settingsOpen === currentStateRef.current.settingsOpen &&
      newState.hlFile === currentStateRef.current.hlFile &&
      newState.hlLine === currentStateRef.current.hlLine &&
      newState.hlText === currentStateRef.current.hlText
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
    opts.onAgentIdChange(urlState.agentId)
    opts.onTeamIdChange(urlState.teamId)
    opts.onSettingsOpenChange(urlState.settingsOpen)
    opts.onHighlightChange?.(extractHighlightRequest(urlState))
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
    const hlRequest = extractHighlightRequest(initialState)
    if (initialState.skillId || initialState.agentId || initialState.teamId || initialState.settingsOpen || hlRequest) {
      // Use setTimeout to ensure this runs after initial render
      setTimeout(() => {
        if (initialState.skillId) {
          onSkillIdChange(initialState.skillId)
        }
        if (initialState.agentId) {
          onAgentIdChange(initialState.agentId)
        }
        if (initialState.teamId) {
          onTeamIdChange(initialState.teamId)
        }
        if (initialState.settingsOpen) {
          onSettingsOpenChange(initialState.settingsOpen)
        }
        if (hlRequest) {
          options.onHighlightChange?.(hlRequest)
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
