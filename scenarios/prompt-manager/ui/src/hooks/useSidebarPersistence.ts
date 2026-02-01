/**
 * useSidebarPersistence - Hook for persisting sidebar state to localStorage.
 *
 * Persists:
 * - isCollapsed: Whether the sidebar is collapsed
 * - expandedNodes: Which tree nodes are expanded
 * - selectedTags: Active tag filters
 *
 * Note: Search query is intentionally NOT persisted as it's typically transient.
 * Sidebar width is persisted separately by useResizableSidebar.
 */

import { useEffect, useCallback, useRef } from 'react'

/** localStorage key for sidebar state */
const STORAGE_KEY = 'pm.sidebarState'

/** Debounce delay for localStorage writes (ms) */
const DEBOUNCE_MS = 300

export interface SidebarPersistedState {
  /** Whether the sidebar is collapsed */
  isCollapsed: boolean
  /** IDs of expanded tree nodes */
  expandedNodes: string[]
  /** Selected tag filters */
  selectedTags: string[]
  /** Selected folder filters */
  selectedFolders: string[]
  /** Active sidebar tab (skills, agents, teams) */
  activeTab: string
  /** Search query for skills */
  searchQuery: string
}

const DEFAULT_STATE: SidebarPersistedState = {
  isCollapsed: false,
  expandedNodes: [],
  selectedTags: [],
  selectedFolders: [],
  activeTab: 'skills',
  searchQuery: '',
}

/**
 * Load sidebar state from localStorage.
 * Returns default state if not found or invalid.
 */
export function loadSidebarState(): SidebarPersistedState {
  if (typeof window === 'undefined') return DEFAULT_STATE

  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (!stored) return DEFAULT_STATE

    const parsed = JSON.parse(stored) as Partial<SidebarPersistedState>

    // Validate and merge with defaults
    return {
      isCollapsed: typeof parsed.isCollapsed === 'boolean' ? parsed.isCollapsed : DEFAULT_STATE.isCollapsed,
      expandedNodes: Array.isArray(parsed.expandedNodes) ? parsed.expandedNodes : DEFAULT_STATE.expandedNodes,
      selectedTags: Array.isArray(parsed.selectedTags) ? parsed.selectedTags : DEFAULT_STATE.selectedTags,
      selectedFolders: Array.isArray(parsed.selectedFolders) ? parsed.selectedFolders : DEFAULT_STATE.selectedFolders,
      activeTab: typeof parsed.activeTab === 'string' ? parsed.activeTab : DEFAULT_STATE.activeTab,
      searchQuery: typeof parsed.searchQuery === 'string' ? parsed.searchQuery : DEFAULT_STATE.searchQuery,
    }
  } catch {
    return DEFAULT_STATE
  }
}

/**
 * Save sidebar state to localStorage.
 */
export function saveSidebarState(state: SidebarPersistedState): void {
  if (typeof window === 'undefined') return

  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
  } catch {
    // Ignore localStorage errors (quota exceeded, etc.)
  }
}

export interface UseSidebarPersistenceOptions {
  /** Current collapsed state */
  isCollapsed: boolean
  /** Current expanded nodes */
  expandedNodes: Set<string>
  /** Current selected tags */
  selectedTags: string[]
  /** Current selected folders */
  selectedFolders: string[]
  /** Current active tab */
  activeTab: string
  /** Current search query */
  searchQuery: string
}

export interface UseSidebarPersistenceReturn {
  /** Get initial state from localStorage */
  getInitialState: () => SidebarPersistedState
}

/**
 * Hook for persisting sidebar state to localStorage.
 *
 * Usage:
 * 1. Call getInitialState() on mount to get saved state
 * 2. Pass current state to the hook - it auto-persists on changes
 */
export function useSidebarPersistence(options: UseSidebarPersistenceOptions): UseSidebarPersistenceReturn {
  const { isCollapsed, expandedNodes, selectedTags, selectedFolders, activeTab, searchQuery } = options

  // Debounce timer ref
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Persist state changes (debounced)
  useEffect(() => {
    // Clear any pending save
    if (timerRef.current) {
      clearTimeout(timerRef.current)
    }

    // Schedule debounced save
    timerRef.current = setTimeout(() => {
      saveSidebarState({
        isCollapsed,
        expandedNodes: Array.from(expandedNodes),
        selectedTags,
        selectedFolders,
        activeTab,
        searchQuery,
      })
    }, DEBOUNCE_MS)

    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current)
      }
    }
  }, [isCollapsed, expandedNodes, selectedTags, selectedFolders, activeTab, searchQuery])

  const getInitialState = useCallback((): SidebarPersistedState => {
    return loadSidebarState()
  }, [])

  return {
    getInitialState,
  }
}
