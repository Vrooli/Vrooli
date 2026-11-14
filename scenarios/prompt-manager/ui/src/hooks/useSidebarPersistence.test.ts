/**
 * Tests for useSidebarPersistence hook.
 *
 * Tests cover:
 * - Loading state from localStorage
 * - Saving state to localStorage
 * - Debounced persistence
 * - Invalid/missing data handling
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import {
  useSidebarPersistence,
  loadSidebarState,
  saveSidebarState,
  type SidebarPersistedState,
  type UseSidebarPersistenceOptions,
} from './useSidebarPersistence'
import { DEFAULT_FILTER_STATE, DEFAULT_SORT_CONFIG, DEFAULT_VIEW_MODE, DEFAULT_DETAIL_MODE } from '@/types/filterSort'

const DEFAULT_CONTENT_SEARCH_OPTIONS = {
  caseSensitive: false,
  wholeWord: false,
  regex: false,
}

const makeOptions = (
  overrides: Partial<UseSidebarPersistenceOptions> = {}
): UseSidebarPersistenceOptions => {
  const contentSearchOptions = {
    ...DEFAULT_CONTENT_SEARCH_OPTIONS,
    ...overrides.contentSearchOptions,
  }

  return {
    isCollapsed: false,
    expandedNodes: new Set<string>(),
    filterState: DEFAULT_FILTER_STATE,
    sortConfig: DEFAULT_SORT_CONFIG,
    viewMode: DEFAULT_VIEW_MODE,
    detailMode: DEFAULT_DETAIL_MODE,
    activeTab: 'skills',
    searchQuery: '',
    searchMode: 'quick',
    ...overrides,
    contentSearchOptions,
  }
}

describe('useSidebarPersistence', () => {
  const STORAGE_KEY = 'pm.sidebarState'
  const STORAGE_SCHEMA_VERSION = 2

  // Create a working localStorage mock for these tests
  let store: Record<string, string> = {}
  const localStorageMock = {
    getItem: vi.fn((key: string) => store[key] ?? null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value
    }),
    removeItem: vi.fn((key: string) => {
      // Use destructuring to avoid dynamic delete lint error
      const { [key]: _, ...rest } = store
      store = rest
    }),
    clear: vi.fn(() => {
      store = {}
    }),
  }

  beforeEach(() => {
    store = {}
    vi.clearAllMocks()
    Object.defineProperty(window, 'localStorage', {
      value: localStorageMock,
      writable: true,
    })
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('loadSidebarState', () => {
    it('should return default state when localStorage is empty', () => {
      const state = loadSidebarState()

      expect(state).toEqual({
        isCollapsed: false,
        expandedNodes: [],
        filterState: DEFAULT_FILTER_STATE,
        sortConfig: DEFAULT_SORT_CONFIG,
        viewMode: DEFAULT_VIEW_MODE,
        detailMode: DEFAULT_DETAIL_MODE,
        activeTab: 'skills',
        searchQuery: '',
        searchMode: 'quick',
        contentSearchOptions: {
          caseSensitive: false,
          wholeWord: false,
          regex: false,
        },
      })
    })

    it('should load valid state from localStorage', () => {
      const savedState: SidebarPersistedState = {
        isCollapsed: true,
        expandedNodes: ['folder-1', 'folder-2'],
        filterState: { storage: ['local', 'core'], tags: ['tag-a', 'tag-b'], usagePreset: null, minRating: null, status: 'all' },
        sortConfig: { field: 'mostUsed', direction: 'desc' },
        viewMode: 'list',
        detailMode: 'full',
        activeTab: 'agents',
        searchQuery: 'test query',
        searchMode: 'content',
        contentSearchOptions: {
          caseSensitive: true,
          wholeWord: true,
          regex: false,
        },
      }
      localStorage.setItem(STORAGE_KEY, JSON.stringify({
        schemaVersion: STORAGE_SCHEMA_VERSION,
        ...savedState,
      }))

      const state = loadSidebarState()

      expect(state).toEqual(savedState)
    })

    it('should load ai searchMode from localStorage', () => {
      const savedState: SidebarPersistedState = {
        isCollapsed: false,
        expandedNodes: [],
        filterState: DEFAULT_FILTER_STATE,
        sortConfig: DEFAULT_SORT_CONFIG,
        viewMode: DEFAULT_VIEW_MODE,
        detailMode: DEFAULT_DETAIL_MODE,
        activeTab: 'skills',
        searchQuery: '',
        searchMode: 'ai',
        contentSearchOptions: {
          caseSensitive: false,
          wholeWord: false,
          regex: false,
        },
      }
      localStorage.setItem(STORAGE_KEY, JSON.stringify({
        schemaVersion: STORAGE_SCHEMA_VERSION,
        ...savedState,
      }))

      const state = loadSidebarState()

      expect(state.searchMode).toBe('ai')
    })

    it('should handle invalid JSON gracefully', () => {
      localStorage.setItem(STORAGE_KEY, 'not valid json')

      const state = loadSidebarState()

      expect(state).toEqual({
        isCollapsed: false,
        expandedNodes: [],
        filterState: DEFAULT_FILTER_STATE,
        sortConfig: DEFAULT_SORT_CONFIG,
        viewMode: DEFAULT_VIEW_MODE,
        detailMode: DEFAULT_DETAIL_MODE,
        activeTab: 'skills',
        searchQuery: '',
        searchMode: 'quick',
        contentSearchOptions: {
          caseSensitive: false,
          wholeWord: false,
          regex: false,
        },
      })
    })

    it('should handle partial data with defaults', () => {
      localStorage.setItem(STORAGE_KEY, JSON.stringify({
        schemaVersion: STORAGE_SCHEMA_VERSION,
        isCollapsed: true,
      }))

      const state = loadSidebarState()

      expect(state).toEqual({
        isCollapsed: true,
        expandedNodes: [],
        filterState: DEFAULT_FILTER_STATE,
        sortConfig: DEFAULT_SORT_CONFIG,
        viewMode: DEFAULT_VIEW_MODE,
        detailMode: DEFAULT_DETAIL_MODE,
        activeTab: 'skills',
        searchQuery: '',
        searchMode: 'quick',
        contentSearchOptions: {
          caseSensitive: false,
          wholeWord: false,
          regex: false,
        },
      })
    })

    it('should reset unversioned persisted state to defaults', () => {
      localStorage.setItem(STORAGE_KEY, JSON.stringify({
        isCollapsed: true,
        filterState: { storage: [], tags: [], usagePreset: null, minRating: null, status: 'draft' },
        searchQuery: 'stale query',
      }))

      const state = loadSidebarState()

      expect(state).toEqual({
        isCollapsed: false,
        expandedNodes: [],
        filterState: DEFAULT_FILTER_STATE,
        sortConfig: DEFAULT_SORT_CONFIG,
        viewMode: DEFAULT_VIEW_MODE,
        detailMode: DEFAULT_DETAIL_MODE,
        activeTab: 'skills',
        searchQuery: '',
        searchMode: 'quick',
        contentSearchOptions: {
          caseSensitive: false,
          wholeWord: false,
          regex: false,
        },
      })
    })

    it('should handle invalid types with defaults', () => {
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({
          schemaVersion: STORAGE_SCHEMA_VERSION,
          isCollapsed: 'not a boolean',
          expandedNodes: 'not an array',
          filterState: 123,
        })
      )

      const state = loadSidebarState()

      expect(state).toEqual({
        isCollapsed: false,
        expandedNodes: [],
        filterState: DEFAULT_FILTER_STATE,
        sortConfig: DEFAULT_SORT_CONFIG,
        viewMode: DEFAULT_VIEW_MODE,
        detailMode: DEFAULT_DETAIL_MODE,
        activeTab: 'skills',
        searchQuery: '',
        searchMode: 'quick',
        contentSearchOptions: {
          caseSensitive: false,
          wholeWord: false,
          regex: false,
        },
      })
    })
  })

  describe('saveSidebarState', () => {
    it('should save state to localStorage', () => {
      const state: SidebarPersistedState = {
        isCollapsed: true,
        expandedNodes: ['node-1'],
        filterState: { storage: ['local'], tags: ['tag-1'], usagePreset: null, minRating: null, status: 'all' },
        sortConfig: DEFAULT_SORT_CONFIG,
        viewMode: DEFAULT_VIEW_MODE,
        detailMode: 'full',
        activeTab: 'teams',
        searchQuery: 'find me',
        searchMode: 'quick',
        contentSearchOptions: {
          caseSensitive: false,
          wholeWord: false,
          regex: false,
        },
      }

      saveSidebarState(state)

      const saved = localStorage.getItem(STORAGE_KEY)
      expect(JSON.parse(saved || '{}')).toEqual({
        schemaVersion: STORAGE_SCHEMA_VERSION,
        ...state,
      })
    })
  })

  describe('useSidebarPersistence hook', () => {
    it('should return getInitialState function', () => {
      const { result } = renderHook(() =>
        useSidebarPersistence(makeOptions())
      )

      expect(typeof result.current.getInitialState).toBe('function')
    })

    it('should persist state changes after debounce', () => {
      const { rerender } = renderHook(
        (props: UseSidebarPersistenceOptions) => useSidebarPersistence(props),
        {
          initialProps: makeOptions(),
        }
      )

      // Update state
      rerender(makeOptions({
        isCollapsed: true,
        expandedNodes: new Set(['folder-1']),
        filterState: { storage: ['local'], tags: ['tag-1'], usagePreset: null, minRating: null, status: 'all' },
        sortConfig: { field: 'mostUsed', direction: 'desc' },
        viewMode: 'list',
        activeTab: 'agents',
        searchQuery: 'test',
        searchMode: 'content',
        contentSearchOptions: {
          caseSensitive: true,
          wholeWord: false,
          regex: true,
        },
      }))

      // Should not be saved yet (debounced)
      expect(localStorage.getItem(STORAGE_KEY)).toBeNull()

      // Fast-forward past debounce delay
      act(() => {
        vi.advanceTimersByTime(350)
      })

      // Now it should be saved
      const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}') as SidebarPersistedState & { schemaVersion: number }
      expect(saved).toEqual({
        schemaVersion: STORAGE_SCHEMA_VERSION,
        isCollapsed: true,
        expandedNodes: ['folder-1'],
        filterState: { storage: ['local'], tags: ['tag-1'], usagePreset: null, minRating: null, status: 'all' },
        sortConfig: { field: 'mostUsed', direction: 'desc' },
        viewMode: 'list',
        detailMode: 'full',
        activeTab: 'agents',
        searchQuery: 'test',
        searchMode: 'content',
        contentSearchOptions: {
          caseSensitive: true,
          wholeWord: false,
          regex: true,
        },
      })
    })

    it('should cancel pending save on unmount', () => {
      const { unmount, rerender } = renderHook(
        (props: UseSidebarPersistenceOptions) => useSidebarPersistence(props),
        {
          initialProps: makeOptions(),
        }
      )

      // Update state
      rerender(makeOptions({
        isCollapsed: true,
      }))

      // Unmount before debounce completes
      unmount()

      // Fast-forward
      act(() => {
        vi.advanceTimersByTime(350)
      })

      // Should NOT have saved (unmounted)
      expect(localStorage.getItem(STORAGE_KEY)).toBeNull()
    })

    it('should debounce rapid state changes', () => {
      const { rerender } = renderHook(
        (props: UseSidebarPersistenceOptions) => useSidebarPersistence(props),
        {
          initialProps: makeOptions(),
        }
      )

      // Rapidly change state multiple times
      rerender(makeOptions({
        isCollapsed: true,
      }))

      act(() => {
        vi.advanceTimersByTime(100)
      })

      rerender(makeOptions({
        isCollapsed: true,
        expandedNodes: new Set(['a']),
        filterState: { storage: ['local'], tags: [], usagePreset: null, minRating: null, status: 'all' },
        activeTab: 'agents',
        searchQuery: 'search',
        searchMode: 'content',
        contentSearchOptions: {
          caseSensitive: true,
          wholeWord: false,
          regex: false,
        },
      }))

      act(() => {
        vi.advanceTimersByTime(100)
      })

      rerender(makeOptions({
        isCollapsed: true,
        expandedNodes: new Set(['a', 'b']),
        filterState: { storage: ['local', 'core'], tags: ['tag'], usagePreset: null, minRating: null, status: 'all' },
        sortConfig: { field: 'recentlyUpdated', direction: 'desc' },
        viewMode: 'card',
        activeTab: 'teams',
        searchQuery: 'final',
        searchMode: 'content',
        contentSearchOptions: {
          caseSensitive: true,
          wholeWord: true,
          regex: true,
        },
      }))

      // Nothing saved yet
      expect(localStorage.getItem(STORAGE_KEY)).toBeNull()

      // Fast-forward past debounce
      act(() => {
        vi.advanceTimersByTime(350)
      })

      // Only the final state should be saved
      const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}') as SidebarPersistedState & { schemaVersion: number }
      expect(saved).toEqual({
        schemaVersion: STORAGE_SCHEMA_VERSION,
        isCollapsed: true,
        expandedNodes: ['a', 'b'],
        filterState: { storage: ['local', 'core'], tags: ['tag'], usagePreset: null, minRating: null, status: 'all' },
        sortConfig: { field: 'recentlyUpdated', direction: 'desc' },
        viewMode: 'card',
        detailMode: 'full',
        activeTab: 'teams',
        searchQuery: 'final',
        searchMode: 'content',
        contentSearchOptions: {
          caseSensitive: true,
          wholeWord: true,
          regex: true,
        },
      })
    })

    it('should load correct initial state via getInitialState', () => {
      // Pre-populate localStorage
      const savedState: SidebarPersistedState = {
        isCollapsed: true,
        expandedNodes: ['saved-folder'],
        filterState: { storage: ['local'], tags: ['saved-tag'], usagePreset: null, minRating: null, status: 'all' },
        sortConfig: { field: 'mostUsed', direction: 'desc' },
        viewMode: 'list',
        detailMode: 'full',
        activeTab: 'agents',
        searchQuery: 'saved query',
        searchMode: 'content',
        contentSearchOptions: {
          caseSensitive: true,
          wholeWord: false,
          regex: true,
        },
      }
      localStorage.setItem(STORAGE_KEY, JSON.stringify({
        schemaVersion: STORAGE_SCHEMA_VERSION,
        ...savedState,
      }))

      const { result } = renderHook(() =>
        useSidebarPersistence(makeOptions())
      )

      const initialState = result.current.getInitialState()

      expect(initialState).toEqual(savedState)
    })
  })
})
