/**
 * useSidebarPersistence - Hook for persisting sidebar state to localStorage.
 *
 * Persists:
 * - isCollapsed: Whether the sidebar is collapsed
 * - expandedNodes: Which tree nodes are expanded
 * - filterState: Active filter configuration
 * - sortConfig: Current sort field and direction
 * - viewMode: Current view mode (tree/list/card)
 *
 * Sidebar width is persisted separately by useResizableSidebar.
 */

import { useEffect, useCallback, useRef } from 'react'
import type { FilterState, SortConfig, ViewMode } from '@/types/filterSort'
import { DEFAULT_FILTER_STATE, DEFAULT_SORT_CONFIG, DEFAULT_VIEW_MODE } from '@/types/filterSort'

/** localStorage key for sidebar state */
const STORAGE_KEY = 'pm.sidebarState'

/** Debounce delay for localStorage writes (ms) */
const DEBOUNCE_MS = 300

export interface SidebarPersistedState {
  /** Whether the sidebar is collapsed */
  isCollapsed: boolean
  /** IDs of expanded tree nodes */
  expandedNodes: string[]
  /** Filter configuration */
  filterState: FilterState
  /** Sort configuration */
  sortConfig: SortConfig
  /** View mode */
  viewMode: ViewMode
  /** Active sidebar tab (skills, agents, teams) */
  activeTab: string
  /** Search query for skills */
  searchQuery: string
  /** Search mode for skills */
  searchMode: 'quick' | 'content'
  /** Content search options */
  contentSearchOptions: {
    caseSensitive: boolean
    wholeWord: boolean
    regex: boolean
  }
}

const DEFAULT_STATE: SidebarPersistedState = {
  isCollapsed: false,
  expandedNodes: [],
  filterState: DEFAULT_FILTER_STATE,
  sortConfig: DEFAULT_SORT_CONFIG,
  viewMode: DEFAULT_VIEW_MODE,
  activeTab: 'skills',
  searchQuery: '',
  searchMode: 'quick',
  contentSearchOptions: {
    caseSensitive: false,
    wholeWord: false,
    regex: false,
  },
}

/**
 * Load sidebar state from localStorage.
 * Returns default state if not found or invalid.
 * Gracefully ignores old format keys (selectedTags, selectedFolders).
 */
export function loadSidebarState(): SidebarPersistedState {
  if (typeof window === 'undefined') return DEFAULT_STATE

  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (!stored) return DEFAULT_STATE

    const parsed = JSON.parse(stored) as Record<string, unknown>

    return {
      isCollapsed: typeof parsed.isCollapsed === 'boolean' ? parsed.isCollapsed : DEFAULT_STATE.isCollapsed,
      expandedNodes: Array.isArray(parsed.expandedNodes) ? parsed.expandedNodes : DEFAULT_STATE.expandedNodes,
      filterState: validateFilterState(parsed.filterState),
      sortConfig: validateSortConfig(parsed.sortConfig),
      viewMode: validateViewMode(parsed.viewMode),
      activeTab: typeof parsed.activeTab === 'string' ? parsed.activeTab : DEFAULT_STATE.activeTab,
      searchQuery: typeof parsed.searchQuery === 'string' ? parsed.searchQuery : DEFAULT_STATE.searchQuery,
      searchMode: parsed.searchMode === 'content' ? 'content' : DEFAULT_STATE.searchMode,
      contentSearchOptions: validateContentSearchOptions(parsed.contentSearchOptions),
    }
  } catch {
    return DEFAULT_STATE
  }
}

function validateContentSearchOptions(raw: unknown): SidebarPersistedState['contentSearchOptions'] {
  const defaults = DEFAULT_STATE.contentSearchOptions
  if (!raw || typeof raw !== 'object') return defaults
  const obj = raw as Record<string, unknown>
  return {
    caseSensitive: typeof obj.caseSensitive === 'boolean' ? obj.caseSensitive : defaults.caseSensitive,
    wholeWord: typeof obj.wholeWord === 'boolean' ? obj.wholeWord : defaults.wholeWord,
    regex: typeof obj.regex === 'boolean' ? obj.regex : defaults.regex,
  }
}

function validateFilterState(raw: unknown): FilterState {
  if (!raw || typeof raw !== 'object') return DEFAULT_FILTER_STATE
  const obj = raw as Record<string, unknown>
  return {
    storage: Array.isArray(obj.storage) ? obj.storage.filter((s): s is string => typeof s === 'string') : [],
    tags: Array.isArray(obj.tags) ? obj.tags.filter((t): t is string => typeof t === 'string') : [],
    usagePreset: ['usedThisWeek', 'neverUsed', 'top10'].includes(obj.usagePreset as string)
      ? (obj.usagePreset as FilterState['usagePreset'])
      : null,
    minRating: typeof obj.minRating === 'number' && obj.minRating >= 1 && obj.minRating <= 5
      ? obj.minRating
      : null,
    status: ['all', 'draft', 'published'].includes(obj.status as string)
      ? (obj.status as FilterState['status'])
      : 'all',
  }
}

function validateSortConfig(raw: unknown): SortConfig {
  if (!raw || typeof raw !== 'object') return DEFAULT_SORT_CONFIG
  const obj = raw as Record<string, unknown>
  const validFields = ['alphabetical', 'mostUsed', 'recentlyUsed', 'recentlyUpdated', 'rating']
  return {
    field: validFields.includes(obj.field as string)
      ? (obj.field as SortConfig['field'])
      : DEFAULT_SORT_CONFIG.field,
    direction: obj.direction === 'desc' ? 'desc' : 'asc',
  }
}

function validateViewMode(raw: unknown): ViewMode {
  if (['tree', 'list', 'card'].includes(raw as string)) return raw as ViewMode
  return DEFAULT_VIEW_MODE
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
  /** Current filter state */
  filterState: FilterState
  /** Current sort config */
  sortConfig: SortConfig
  /** Current view mode */
  viewMode: ViewMode
  /** Current active tab */
  activeTab: string
  /** Current search query */
  searchQuery: string
  /** Current search mode */
  searchMode: 'quick' | 'content'
  /** Current content search options */
  contentSearchOptions: {
    caseSensitive: boolean
    wholeWord: boolean
    regex: boolean
  }
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
  const {
    isCollapsed,
    expandedNodes,
    filterState,
    sortConfig,
    viewMode,
    activeTab,
    searchQuery,
    searchMode,
    contentSearchOptions,
  } = options

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
        filterState,
        sortConfig,
        viewMode,
        activeTab,
        searchQuery,
        searchMode,
        contentSearchOptions,
      })
    }, DEBOUNCE_MS)

    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current)
      }
    }
  }, [isCollapsed, expandedNodes, filterState, sortConfig, viewMode, activeTab, searchQuery, searchMode, contentSearchOptions])

  const getInitialState = useCallback((): SidebarPersistedState => {
    return loadSidebarState()
  }, [])

  return {
    getInitialState,
  }
}
