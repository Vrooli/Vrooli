/**
 * Tests for the overlay store.
 *
 * Tests cover:
 * - Speech bubbles management
 * - Thinking states
 * - Name tag configuration
 * - Selectors with edge cases
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import {
  useOverlayStore,
  selectAgentSpeechBubbles,
  selectLatestAgentSpeechBubble,
  selectAgentThinking,
} from './overlayStore'

describe('overlayStore', () => {
  beforeEach(() => {
    // Reset store state before each test
    useOverlayStore.getState().reset()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('initial state', () => {
    it('should start with empty speechBubbles', () => {
      const state = useOverlayStore.getState()
      expect(state.speechBubbles).toEqual([])
    })

    it('should start with empty thinkingStates', () => {
      const state = useOverlayStore.getState()
      expect(state.thinkingStates).toEqual({})
    })

    it('should start with overlays visible', () => {
      const state = useOverlayStore.getState()
      expect(state.overlaysVisible).toBe(true)
    })

    it('should start with showAll name tags enabled', () => {
      const state = useOverlayStore.getState()
      expect(state.nameTagConfig.showAll).toBe(true)
    })
  })

  describe('showSpeechBubble', () => {
    it('should add a speech bubble', () => {
      const id = useOverlayStore.getState().showSpeechBubble('agent-1', 'Hello!')

      const state = useOverlayStore.getState()
      expect(state.speechBubbles).toHaveLength(1)
      expect(state.speechBubbles[0]).toMatchObject({
        id,
        agentId: 'agent-1',
        text: 'Hello!',
      })
    })

    it('should return the bubble id', () => {
      const id = useOverlayStore.getState().showSpeechBubble('agent-1', 'Test')
      expect(id).toMatch(/^bubble-\d+$/)
    })

    it('should auto-remove bubble after duration', () => {
      useOverlayStore.getState().showSpeechBubble('agent-1', 'Test', 1000)

      expect(useOverlayStore.getState().speechBubbles).toHaveLength(1)

      vi.advanceTimersByTime(1000)

      expect(useOverlayStore.getState().speechBubbles).toHaveLength(0)
    })

    it('should not auto-remove bubble with duration 0', () => {
      useOverlayStore.getState().showSpeechBubble('agent-1', 'Permanent', 0)

      vi.advanceTimersByTime(10000)

      expect(useOverlayStore.getState().speechBubbles).toHaveLength(1)
    })

    it('should mark bubble as temporary when duration > 0', () => {
      useOverlayStore.getState().showSpeechBubble('agent-1', 'Temp', 5000)

      const state = useOverlayStore.getState()
      expect(state.speechBubbles[0]?.temporary).toBe(true)
    })

    it('should mark bubble as not temporary when duration is 0', () => {
      useOverlayStore.getState().showSpeechBubble('agent-1', 'Permanent', 0)

      const state = useOverlayStore.getState()
      expect(state.speechBubbles[0]?.temporary).toBe(false)
    })
  })

  describe('hideSpeechBubble', () => {
    it('should remove a specific bubble', () => {
      const id1 = useOverlayStore.getState().showSpeechBubble('agent-1', 'First', 0)
      useOverlayStore.getState().showSpeechBubble('agent-1', 'Second', 0)

      useOverlayStore.getState().hideSpeechBubble(id1)

      const state = useOverlayStore.getState()
      expect(state.speechBubbles).toHaveLength(1)
      expect(state.speechBubbles[0]?.text).toBe('Second')
    })

    it('should do nothing for non-existent id', () => {
      useOverlayStore.getState().showSpeechBubble('agent-1', 'Test', 0)

      useOverlayStore.getState().hideSpeechBubble('non-existent')

      expect(useOverlayStore.getState().speechBubbles).toHaveLength(1)
    })
  })

  describe('hideAllSpeechBubbles', () => {
    it('should remove all bubbles for a specific agent', () => {
      useOverlayStore.getState().showSpeechBubble('agent-1', 'First', 0)
      useOverlayStore.getState().showSpeechBubble('agent-1', 'Second', 0)
      useOverlayStore.getState().showSpeechBubble('agent-2', 'Other', 0)

      useOverlayStore.getState().hideAllSpeechBubbles('agent-1')

      const state = useOverlayStore.getState()
      expect(state.speechBubbles).toHaveLength(1)
      expect(state.speechBubbles[0]?.agentId).toBe('agent-2')
    })
  })

  describe('setThinking', () => {
    it('should set thinking state for an agent', () => {
      useOverlayStore.getState().setThinking('agent-1', true, 'Processing...')

      const state = useOverlayStore.getState()
      expect(state.thinkingStates['agent-1']).toEqual({
        agentId: 'agent-1',
        isThinking: true,
        label: 'Processing...',
      })
    })

    it('should update thinking state', () => {
      useOverlayStore.getState().setThinking('agent-1', true)
      useOverlayStore.getState().setThinking('agent-1', false)

      const state = useOverlayStore.getState()
      expect(state.thinkingStates['agent-1']?.isThinking).toBe(false)
    })
  })

  describe('clearThinking', () => {
    it('should remove thinking state for an agent', () => {
      useOverlayStore.getState().setThinking('agent-1', true)
      useOverlayStore.getState().setThinking('agent-2', true)

      useOverlayStore.getState().clearThinking('agent-1')

      const state = useOverlayStore.getState()
      expect(state.thinkingStates['agent-1']).toBeUndefined()
      expect(state.thinkingStates['agent-2']).toBeDefined()
    })
  })

  describe('shouldShowNameTag', () => {
    it('should return false when overlays are not visible', () => {
      useOverlayStore.getState().setOverlaysVisible(false)

      const result = useOverlayStore.getState().shouldShowNameTag('agent-1', false)
      expect(result).toBe(false)
    })

    it('should return false when agent is in neverShowFor list', () => {
      useOverlayStore.getState().updateNameTagConfig({ neverShowFor: ['agent-1'] })

      const result = useOverlayStore.getState().shouldShowNameTag('agent-1', true)
      expect(result).toBe(false)
    })

    it('should return true when agent is in alwaysShowFor list', () => {
      useOverlayStore.getState().updateNameTagConfig({
        showAll: false,
        alwaysShowFor: ['agent-1'],
      })

      const result = useOverlayStore.getState().shouldShowNameTag('agent-1', false)
      expect(result).toBe(true)
    })

    it('should respect showOnHover setting', () => {
      useOverlayStore.getState().updateNameTagConfig({
        showAll: false,
        showOnHover: true,
      })

      expect(useOverlayStore.getState().shouldShowNameTag('agent-1', false)).toBe(false)
      expect(useOverlayStore.getState().shouldShowNameTag('agent-1', true)).toBe(true)
    })

    it('should return showAll value when no other conditions match', () => {
      useOverlayStore.getState().updateNameTagConfig({ showAll: true })
      expect(useOverlayStore.getState().shouldShowNameTag('agent-1', false)).toBe(true)

      useOverlayStore.getState().updateNameTagConfig({ showAll: false })
      expect(useOverlayStore.getState().shouldShowNameTag('agent-1', false)).toBe(false)
    })
  })

  describe('cleanupExpiredBubbles', () => {
    it('should remove expired temporary bubbles', () => {
      useOverlayStore.getState().showSpeechBubble('agent-1', 'Short', 100)
      useOverlayStore.getState().showSpeechBubble('agent-1', 'Long', 10000)
      useOverlayStore.getState().showSpeechBubble('agent-1', 'Permanent', 0)

      vi.advanceTimersByTime(500)
      useOverlayStore.getState().cleanupExpiredBubbles()

      const state = useOverlayStore.getState()
      // The long and permanent bubbles should still be present
      expect(state.speechBubbles.length).toBeGreaterThanOrEqual(2)
    })
  })

  describe('selectAgentSpeechBubbles', () => {
    it('should return bubbles for a specific agent', () => {
      useOverlayStore.getState().showSpeechBubble('agent-1', 'First', 0)
      useOverlayStore.getState().showSpeechBubble('agent-2', 'Other', 0)
      useOverlayStore.getState().showSpeechBubble('agent-1', 'Second', 0)

      const state = useOverlayStore.getState()
      const bubbles = selectAgentSpeechBubbles(state, 'agent-1')

      expect(bubbles).toHaveLength(2)
      expect(bubbles[0]?.text).toBe('First')
      expect(bubbles[1]?.text).toBe('Second')
    })

    it('should return empty array for null agentId', () => {
      useOverlayStore.getState().showSpeechBubble('agent-1', 'Test', 0)

      const state = useOverlayStore.getState()
      const bubbles = selectAgentSpeechBubbles(state, null as unknown as string)

      expect(bubbles).toEqual([])
    })

    it('should return empty array for undefined agentId', () => {
      useOverlayStore.getState().showSpeechBubble('agent-1', 'Test', 0)

      const state = useOverlayStore.getState()
      const bubbles = selectAgentSpeechBubbles(state, undefined as unknown as string)

      expect(bubbles).toEqual([])
    })

    it('should return empty array for empty string agentId', () => {
      useOverlayStore.getState().showSpeechBubble('agent-1', 'Test', 0)

      const state = useOverlayStore.getState()
      const bubbles = selectAgentSpeechBubbles(state, '')

      expect(bubbles).toEqual([])
    })
  })

  describe('selectLatestAgentSpeechBubble', () => {
    it('returns the latest bubble for the target agent only', () => {
      useOverlayStore.getState().showSpeechBubble('agent-1', 'First', 0)
      useOverlayStore.getState().showSpeechBubble('agent-2', 'Other', 0)
      useOverlayStore.getState().showSpeechBubble('agent-1', 'Second', 0)

      const latest = selectLatestAgentSpeechBubble(useOverlayStore.getState(), 'agent-1')

      expect(latest?.text).toBe('Second')
      expect(latest?.agentId).toBe('agent-1')
    })

    it('returns null for missing agent id', () => {
      useOverlayStore.getState().showSpeechBubble('agent-1', 'Only', 0)
      expect(selectLatestAgentSpeechBubble(useOverlayStore.getState(), '')).toBeNull()
      expect(selectLatestAgentSpeechBubble(useOverlayStore.getState(), null as unknown as string)).toBeNull()
    })
  })

  describe('selectAgentThinking', () => {
    it('should return thinking state for an agent', () => {
      useOverlayStore.getState().setThinking('agent-1', true, 'Working...')

      const state = useOverlayStore.getState()
      const thinking = selectAgentThinking(state, 'agent-1')

      expect(thinking).toEqual({
        agentId: 'agent-1',
        isThinking: true,
        label: 'Working...',
      })
    })

    it('should return null for non-existent agent', () => {
      const state = useOverlayStore.getState()
      const thinking = selectAgentThinking(state, 'non-existent')

      expect(thinking).toBeNull()
    })
  })

  describe('reset', () => {
    it('should reset all state to initial values', () => {
      // Set up some state
      useOverlayStore.getState().showSpeechBubble('agent-1', 'Test', 0)
      useOverlayStore.getState().setThinking('agent-1', true)
      useOverlayStore.getState().setOverlaysVisible(false)

      // Reset
      useOverlayStore.getState().reset()

      const state = useOverlayStore.getState()
      expect(state.speechBubbles).toEqual([])
      expect(state.thinkingStates).toEqual({})
      expect(state.overlaysVisible).toBe(true)
    })
  })
})
