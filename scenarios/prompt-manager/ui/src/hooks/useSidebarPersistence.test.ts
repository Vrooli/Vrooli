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
} from './useSidebarPersistence'

describe('useSidebarPersistence', () => {
  const STORAGE_KEY = 'pm.sidebarState'

  // Create a working localStorage mock for these tests
  let store: Record<string, string> = {}
  const localStorageMock = {
    getItem: vi.fn((key: string) => store[key] ?? null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key]
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
        selectedTags: [],
      })
    })

    it('should load valid state from localStorage', () => {
      const savedState: SidebarPersistedState = {
        isCollapsed: true,
        expandedNodes: ['folder-1', 'folder-2'],
        selectedTags: ['tag-a', 'tag-b'],
      }
      localStorage.setItem(STORAGE_KEY, JSON.stringify(savedState))

      const state = loadSidebarState()

      expect(state).toEqual(savedState)
    })

    it('should handle invalid JSON gracefully', () => {
      localStorage.setItem(STORAGE_KEY, 'not valid json')

      const state = loadSidebarState()

      expect(state).toEqual({
        isCollapsed: false,
        expandedNodes: [],
        selectedTags: [],
      })
    })

    it('should handle partial data with defaults', () => {
      localStorage.setItem(STORAGE_KEY, JSON.stringify({ isCollapsed: true }))

      const state = loadSidebarState()

      expect(state).toEqual({
        isCollapsed: true,
        expandedNodes: [],
        selectedTags: [],
      })
    })

    it('should handle invalid types with defaults', () => {
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({
          isCollapsed: 'not a boolean',
          expandedNodes: 'not an array',
          selectedTags: 123,
        })
      )

      const state = loadSidebarState()

      expect(state).toEqual({
        isCollapsed: false,
        expandedNodes: [],
        selectedTags: [],
      })
    })
  })

  describe('saveSidebarState', () => {
    it('should save state to localStorage', () => {
      const state: SidebarPersistedState = {
        isCollapsed: true,
        expandedNodes: ['node-1'],
        selectedTags: ['tag-1'],
      }

      saveSidebarState(state)

      const saved = localStorage.getItem(STORAGE_KEY)
      expect(saved).toBe(JSON.stringify(state))
    })
  })

  describe('useSidebarPersistence hook', () => {
    it('should return getInitialState function', () => {
      const { result } = renderHook(() =>
        useSidebarPersistence({
          isCollapsed: false,
          expandedNodes: new Set(),
          selectedTags: [],
        })
      )

      expect(typeof result.current.getInitialState).toBe('function')
    })

    it('should persist state changes after debounce', () => {
      const { rerender } = renderHook(
        ({ isCollapsed, expandedNodes, selectedTags }) =>
          useSidebarPersistence({ isCollapsed, expandedNodes, selectedTags }),
        {
          initialProps: {
            isCollapsed: false,
            expandedNodes: new Set<string>(),
            selectedTags: [] as string[],
          },
        }
      )

      // Update state
      rerender({
        isCollapsed: true,
        expandedNodes: new Set(['folder-1']),
        selectedTags: ['tag-1'],
      })

      // Should not be saved yet (debounced)
      expect(localStorage.getItem(STORAGE_KEY)).toBeNull()

      // Fast-forward past debounce delay
      act(() => {
        vi.advanceTimersByTime(350)
      })

      // Now it should be saved
      const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}')
      expect(saved).toEqual({
        isCollapsed: true,
        expandedNodes: ['folder-1'],
        selectedTags: ['tag-1'],
      })
    })

    it('should cancel pending save on unmount', () => {
      const { unmount, rerender } = renderHook(
        ({ isCollapsed, expandedNodes, selectedTags }) =>
          useSidebarPersistence({ isCollapsed, expandedNodes, selectedTags }),
        {
          initialProps: {
            isCollapsed: false,
            expandedNodes: new Set<string>(),
            selectedTags: [] as string[],
          },
        }
      )

      // Update state
      rerender({
        isCollapsed: true,
        expandedNodes: new Set<string>(),
        selectedTags: [],
      })

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
        ({ isCollapsed, expandedNodes, selectedTags }) =>
          useSidebarPersistence({ isCollapsed, expandedNodes, selectedTags }),
        {
          initialProps: {
            isCollapsed: false,
            expandedNodes: new Set<string>(),
            selectedTags: [] as string[],
          },
        }
      )

      // Rapidly change state multiple times
      rerender({
        isCollapsed: true,
        expandedNodes: new Set<string>(),
        selectedTags: [],
      })

      act(() => {
        vi.advanceTimersByTime(100)
      })

      rerender({
        isCollapsed: true,
        expandedNodes: new Set(['a']),
        selectedTags: [],
      })

      act(() => {
        vi.advanceTimersByTime(100)
      })

      rerender({
        isCollapsed: true,
        expandedNodes: new Set(['a', 'b']),
        selectedTags: ['tag'],
      })

      // Nothing saved yet
      expect(localStorage.getItem(STORAGE_KEY)).toBeNull()

      // Fast-forward past debounce
      act(() => {
        vi.advanceTimersByTime(350)
      })

      // Only the final state should be saved
      const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}')
      expect(saved).toEqual({
        isCollapsed: true,
        expandedNodes: ['a', 'b'],
        selectedTags: ['tag'],
      })
    })

    it('should load correct initial state via getInitialState', () => {
      // Pre-populate localStorage
      const savedState: SidebarPersistedState = {
        isCollapsed: true,
        expandedNodes: ['saved-folder'],
        selectedTags: ['saved-tag'],
      }
      localStorage.setItem(STORAGE_KEY, JSON.stringify(savedState))

      const { result } = renderHook(() =>
        useSidebarPersistence({
          isCollapsed: false,
          expandedNodes: new Set(),
          selectedTags: [],
        })
      )

      const initialState = result.current.getInitialState()

      expect(initialState).toEqual(savedState)
    })
  })
})
