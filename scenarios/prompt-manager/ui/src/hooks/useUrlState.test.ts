/**
 * Tests for useUrlState hook.
 *
 * Tests cover:
 * - URL parsing on mount
 * - URL updates without page reload
 * - popstate event handling (browser back/forward)
 * - Dirty state integration
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useUrlState, type UseUrlStateOptions } from './useUrlState'

describe('useUrlState', () => {
  // Store original window properties
  const originalLocation = window.location
  const originalHistory = window.history

  // Mock functions
  let mockReplaceState: ReturnType<typeof vi.fn>
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  let addEventListenerSpy: any
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  let removeEventListenerSpy: any
  let popstateHandler: ((event: PopStateEvent) => void) | null = null

  beforeEach(() => {
    // Mock window.location
    Object.defineProperty(window, 'location', {
      value: {
        pathname: '/',
        search: '',
      },
      writable: true,
      configurable: true,
    })

    // Mock window.history.replaceState
    mockReplaceState = vi.fn()
    Object.defineProperty(window, 'history', {
      value: {
        replaceState: mockReplaceState,
        pushState: vi.fn(),
        state: null,
      },
      writable: true,
      configurable: true,
    })

    // Spy on addEventListener/removeEventListener
    addEventListenerSpy = vi.spyOn(window, 'addEventListener').mockImplementation(
      (type: string, listener: EventListenerOrEventListenerObject) => {
        if (type === 'popstate' && typeof listener === 'function') {
          popstateHandler = listener as (event: PopStateEvent) => void
        }
      }
    )
    removeEventListenerSpy = vi.spyOn(window, 'removeEventListener')
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
    popstateHandler = null
    Object.defineProperty(window, 'location', { value: originalLocation, configurable: true })
    Object.defineProperty(window, 'history', { value: originalHistory, configurable: true })
  })

  function createOptions(overrides: Partial<UseUrlStateOptions> = {}): UseUrlStateOptions {
    return {
      onSkillIdChange: vi.fn(),
      onAgentIdChange: vi.fn(),
      onTeamIdChange: vi.fn(),
      onSettingsOpenChange: vi.fn(),
      isDirty: false,
      storeCurrentChanges: vi.fn(),
      ...overrides,
    }
  }

  describe('URL parsing on mount', () => {
    it('should parse skill ID from URL', () => {
      vi.useFakeTimers()
      window.location.search = '?skill=test-skill-123'

      const options = createOptions()
      renderHook(() => useUrlState(options))

      // Run timers to trigger the deferred state application
      act(() => {
        vi.runAllTimers()
      })

      expect(options.onSkillIdChange).toHaveBeenCalledWith('test-skill-123')
    })

    it('should parse settings open state from URL', () => {
      vi.useFakeTimers()
      window.location.search = '?settings=true'

      const options = createOptions()
      renderHook(() => useUrlState(options))

      act(() => {
        vi.runAllTimers()
      })

      expect(options.onSettingsOpenChange).toHaveBeenCalledWith(true)
    })

    it('should parse both skill ID and settings from URL', () => {
      vi.useFakeTimers()
      window.location.search = '?skill=skill-456&settings=true'

      const options = createOptions()
      renderHook(() => useUrlState(options))

      act(() => {
        vi.runAllTimers()
      })

      expect(options.onSkillIdChange).toHaveBeenCalledWith('skill-456')
      expect(options.onSettingsOpenChange).toHaveBeenCalledWith(true)
    })

    it('should handle empty URL', () => {
      vi.useFakeTimers()
      window.location.search = ''

      const options = createOptions()
      renderHook(() => useUrlState(options))

      act(() => {
        vi.runAllTimers()
      })

      // Should not call handlers when no state in URL
      expect(options.onSkillIdChange).not.toHaveBeenCalled()
      expect(options.onSettingsOpenChange).not.toHaveBeenCalled()
    })

    it('should return correct initial state from getInitialState', () => {
      window.location.search = '?skill=my-skill&settings=true'

      const options = createOptions()
      const { result } = renderHook(() => useUrlState(options))

      const initialState = result.current.getInitialState()

      expect(initialState).toEqual({
        skillId: 'my-skill',
        agentId: null,
        teamId: null,
        settingsOpen: true,
      })
    })
  })

  describe('URL updates', () => {
    it('should update URL with skill ID', () => {
      const options = createOptions()
      const { result } = renderHook(() => useUrlState(options))

      act(() => {
        result.current.updateUrl({ skillId: 'new-skill' })
      })

      expect(mockReplaceState).toHaveBeenCalledWith(
        expect.objectContaining({ skillId: 'new-skill', agentId: null, teamId: null, settingsOpen: false }),
        '',
        '/?skill=new-skill'
      )
    })

    it('should update URL with settings open', () => {
      const options = createOptions()
      const { result } = renderHook(() => useUrlState(options))

      act(() => {
        result.current.updateUrl({ settingsOpen: true })
      })

      expect(mockReplaceState).toHaveBeenCalledWith(
        expect.objectContaining({ skillId: null, agentId: null, teamId: null, settingsOpen: true }),
        '',
        '/?settings=true'
      )
    })

    it('should update URL with both skill ID and settings', () => {
      const options = createOptions()
      const { result } = renderHook(() => useUrlState(options))

      act(() => {
        result.current.updateUrl({ skillId: 'skill-abc', settingsOpen: true })
      })

      expect(mockReplaceState).toHaveBeenCalledWith(
        expect.objectContaining({ skillId: 'skill-abc', agentId: null, teamId: null, settingsOpen: true }),
        '',
        '/?skill=skill-abc&settings=true'
      )
    })

    it('should clear URL when skill ID is null', () => {
      const options = createOptions()
      const { result } = renderHook(() => useUrlState(options))

      // First set a skill
      act(() => {
        result.current.updateUrl({ skillId: 'skill-123' })
      })

      // Then clear it
      act(() => {
        result.current.updateUrl({ skillId: null })
      })

      expect(mockReplaceState).toHaveBeenLastCalledWith(
        expect.objectContaining({ skillId: null, agentId: null, teamId: null, settingsOpen: false }),
        '',
        '/'
      )
    })

    it('should not update URL when state has not changed', () => {
      const options = createOptions()
      const { result } = renderHook(() => useUrlState(options))

      // Initial call
      act(() => {
        result.current.updateUrl({ skillId: 'skill-1' })
      })

      const callCount = mockReplaceState.mock.calls.length

      // Same state should not trigger update
      act(() => {
        result.current.updateUrl({ skillId: 'skill-1' })
      })

      expect(mockReplaceState.mock.calls.length).toBe(callCount)
    })

    it('should merge partial updates with current state', () => {
      const options = createOptions()
      const { result } = renderHook(() => useUrlState(options))

      // Set skill ID
      act(() => {
        result.current.updateUrl({ skillId: 'skill-merge' })
      })

      // Update only settings, skill should be preserved
      act(() => {
        result.current.updateUrl({ settingsOpen: true })
      })

      expect(mockReplaceState).toHaveBeenLastCalledWith(
        expect.objectContaining({ skillId: 'skill-merge', agentId: null, teamId: null, settingsOpen: true }),
        '',
        '/?skill=skill-merge&settings=true'
      )
    })
  })

  describe('popstate event handling', () => {
    it('should add popstate event listener on mount', () => {
      const options = createOptions()
      renderHook(() => useUrlState(options))

      expect(addEventListenerSpy).toHaveBeenCalledWith('popstate', expect.any(Function))
    })

    it('should remove popstate event listener on unmount', () => {
      const options = createOptions()
      const { unmount } = renderHook(() => useUrlState(options))

      unmount()

      expect(removeEventListenerSpy).toHaveBeenCalledWith('popstate', expect.any(Function))
    })

    it('should call onSkillIdChange on popstate with state', () => {
      const options = createOptions()
      renderHook(() => useUrlState(options))

      // Simulate browser back/forward
      const event = new PopStateEvent('popstate', {
        state: { skillId: 'back-skill', agentId: null, teamId: null, settingsOpen: false },
      })

      act(() => {
        popstateHandler?.(event)
      })

      expect(options.onSkillIdChange).toHaveBeenCalledWith('back-skill')
    })

    it('should call onSettingsOpenChange on popstate with state', () => {
      const options = createOptions()
      renderHook(() => useUrlState(options))

      const event = new PopStateEvent('popstate', {
        state: { skillId: null, agentId: null, teamId: null, settingsOpen: true },
      })

      act(() => {
        popstateHandler?.(event)
      })

      expect(options.onSettingsOpenChange).toHaveBeenCalledWith(true)
    })

    it('should parse URL when popstate has no state', () => {
      window.location.search = '?skill=url-parsed-skill'

      const options = createOptions()
      renderHook(() => useUrlState(options))

      // Simulate popstate without state (e.g., manual URL change)
      const event = new PopStateEvent('popstate', { state: null })

      act(() => {
        popstateHandler?.(event)
      })

      expect(options.onSkillIdChange).toHaveBeenCalledWith('url-parsed-skill')
    })
  })

  describe('dirty state integration', () => {
    it('should call storeCurrentChanges before popstate when dirty', () => {
      const storeCurrentChanges = vi.fn()
      const options = createOptions({ isDirty: true, storeCurrentChanges })
      renderHook(() => useUrlState(options))

      const event = new PopStateEvent('popstate', {
        state: { skillId: 'new-skill', agentId: null, teamId: null, settingsOpen: false },
      })

      act(() => {
        popstateHandler?.(event)
      })

      expect(storeCurrentChanges).toHaveBeenCalled()
    })

    it('should not call storeCurrentChanges when not dirty', () => {
      const storeCurrentChanges = vi.fn()
      const options = createOptions({ isDirty: false, storeCurrentChanges })
      renderHook(() => useUrlState(options))

      const event = new PopStateEvent('popstate', {
        state: { skillId: 'new-skill', agentId: null, teamId: null, settingsOpen: false },
      })

      act(() => {
        popstateHandler?.(event)
      })

      expect(storeCurrentChanges).not.toHaveBeenCalled()
    })

    it('should use updated isDirty value in popstate handler', () => {
      const storeCurrentChanges = vi.fn()
      const options = createOptions({ isDirty: false, storeCurrentChanges })
      const { rerender } = renderHook(
        ({ opts }) => useUrlState(opts),
        { initialProps: { opts: options } }
      )

      // Update to dirty state
      const dirtyOptions = createOptions({ isDirty: true, storeCurrentChanges })
      rerender({ opts: dirtyOptions })

      const event = new PopStateEvent('popstate', {
        state: { skillId: 'other-skill', agentId: null, teamId: null, settingsOpen: false },
      })

      act(() => {
        popstateHandler?.(event)
      })

      expect(storeCurrentChanges).toHaveBeenCalled()
    })
  })
})
