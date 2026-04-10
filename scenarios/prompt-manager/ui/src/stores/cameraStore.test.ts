/**
 * Tests for the camera store.
 *
 * Tests cover:
 * - Camera position and target
 * - Camera modes
 * - Zoom to agent functionality
 * - History management
 * - Mode cycling
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { useCameraStore } from './cameraStore'

describe('cameraStore', () => {
  beforeEach(() => {
    // Reset store state before each test
    useCameraStore.setState({
      position: [0, 15, 15],
      target: [0, 0, 0],
      zoom: 1,
      mode: 'freeform',
      focusedAgentId: null,
      history: [],
      isAnimating: false,
    })
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('initial state', () => {
    it('should start in freeform mode', () => {
      const state = useCameraStore.getState()
      expect(state.mode).toBe('freeform')
    })

    it('should start with default position', () => {
      const state = useCameraStore.getState()
      expect(state.position).toEqual([0, 15, 15])
    })

    it('should start with default target', () => {
      const state = useCameraStore.getState()
      expect(state.target).toEqual([0, 0, 0])
    })

    it('should start with no focused agent', () => {
      const state = useCameraStore.getState()
      expect(state.focusedAgentId).toBeNull()
    })

    it('should start with empty history', () => {
      const state = useCameraStore.getState()
      expect(state.history).toEqual([])
    })

    it('should start not animating', () => {
      const state = useCameraStore.getState()
      expect(state.isAnimating).toBe(false)
    })
  })

  describe('setPosition', () => {
    it('should update camera position', () => {
      useCameraStore.getState().setPosition([5, 10, 15])
      expect(useCameraStore.getState().position).toEqual([5, 10, 15])
    })
  })

  describe('setTarget', () => {
    it('should update camera target', () => {
      useCameraStore.getState().setTarget([1, 2, 3])
      expect(useCameraStore.getState().target).toEqual([1, 2, 3])
    })
  })

  describe('setZoom', () => {
    it('should update zoom level', () => {
      useCameraStore.getState().setZoom(2.5)
      expect(useCameraStore.getState().zoom).toBe(2.5)
    })
  })

  describe('setMode', () => {
    it('should update camera mode', () => {
      useCameraStore.getState().setMode('top-down')
      expect(useCameraStore.getState().mode).toBe('top-down')
    })
  })

  describe('setIsAnimating', () => {
    it('should set animation state', () => {
      useCameraStore.getState().setIsAnimating(true)
      expect(useCameraStore.getState().isAnimating).toBe(true)
    })
  })

  describe('zoomToAgent', () => {
    it('should set mode to zoomed-agent', () => {
      useCameraStore.getState().zoomToAgent('agent-1', [0, 0, 0])
      expect(useCameraStore.getState().mode).toBe('zoomed-agent')
    })

    it('should set focused agent id', () => {
      useCameraStore.getState().zoomToAgent('agent-1', [0, 0, 0])
      expect(useCameraStore.getState().focusedAgentId).toBe('agent-1')
    })

    it('should calculate camera position based on agent position', () => {
      useCameraStore.getState().zoomToAgent('agent-1', [5, 0, 3])

      const state = useCameraStore.getState()
      // Camera should be above and in front of agent
      expect(state.position).toEqual([5, 2, 8]) // agent pos + [0, 2, 5]
    })

    it('should set target to agent position', () => {
      useCameraStore.getState().zoomToAgent('agent-1', [5, 0, 3])
      expect(useCameraStore.getState().target).toEqual([5, 0, 3])
    })

    it('should set zoom to 2', () => {
      useCameraStore.getState().zoomToAgent('agent-1', [0, 0, 0])
      expect(useCameraStore.getState().zoom).toBe(2)
    })

    it('should push current state to history', () => {
      useCameraStore.getState().zoomToAgent('agent-1', [0, 0, 0])
      expect(useCameraStore.getState().history).toHaveLength(1)
    })

    it('should set animating to true', () => {
      useCameraStore.getState().zoomToAgent('agent-1', [0, 0, 0])
      expect(useCameraStore.getState().isAnimating).toBe(true)
    })

    it('should reset animating after timeout', () => {
      useCameraStore.getState().zoomToAgent('agent-1', [0, 0, 0])

      vi.advanceTimersByTime(1000)

      expect(useCameraStore.getState().isAnimating).toBe(false)
    })
  })

  describe('exitZoom', () => {
    it('should restore from history', () => {
      // Set initial state
      useCameraStore.setState({
        position: [1, 2, 3],
        target: [4, 5, 6],
        zoom: 1.5,
        mode: 'freeform',
      })

      // Zoom to agent
      useCameraStore.getState().zoomToAgent('agent-1', [0, 0, 0])

      // Exit zoom
      useCameraStore.getState().exitZoom()

      const state = useCameraStore.getState()
      expect(state.position).toEqual([1, 2, 3])
      expect(state.target).toEqual([4, 5, 6])
      expect(state.zoom).toBe(1.5)
      expect(state.mode).toBe('freeform')
    })

    it('should clear focused agent', () => {
      useCameraStore.getState().zoomToAgent('agent-1', [0, 0, 0])
      useCameraStore.getState().exitZoom()

      expect(useCameraStore.getState().focusedAgentId).toBeNull()
    })

    it('should return to default when no history', () => {
      useCameraStore.getState().exitZoom()

      const state = useCameraStore.getState()
      expect(state.position).toEqual([0, 15, 15])
      expect(state.target).toEqual([0, 0, 0])
      expect(state.mode).toBe('freeform')
    })

    it('should set animating to true', () => {
      useCameraStore.getState().exitZoom()
      expect(useCameraStore.getState().isAnimating).toBe(true)
    })
  })

  describe('setTopDown', () => {
    it('should set mode to top-down', () => {
      useCameraStore.getState().setTopDown()
      expect(useCameraStore.getState().mode).toBe('top-down')
    })

    it('should set top-down position', () => {
      useCameraStore.getState().setTopDown()
      expect(useCameraStore.getState().position).toEqual([0, 20, 0.1])
    })

    it('should push current state to history', () => {
      useCameraStore.getState().setTopDown()
      expect(useCameraStore.getState().history).toHaveLength(1)
    })

    it('should clear focused agent', () => {
      useCameraStore.setState({ focusedAgentId: 'agent-1' })
      useCameraStore.getState().setTopDown()
      expect(useCameraStore.getState().focusedAgentId).toBeNull()
    })
  })

  describe('setFreeform', () => {
    it('should set mode to freeform', () => {
      useCameraStore.setState({ mode: 'top-down' })
      useCameraStore.getState().setFreeform()
      expect(useCameraStore.getState().mode).toBe('freeform')
    })

    it('should set default position', () => {
      useCameraStore.setState({ position: [1, 1, 1] })
      useCameraStore.getState().setFreeform()
      expect(useCameraStore.getState().position).toEqual([0, 15, 15])
    })

    it('should push current state to history', () => {
      useCameraStore.getState().setFreeform()
      expect(useCameraStore.getState().history).toHaveLength(1)
    })
  })

  describe('cycleCameraMode', () => {
    it('should cycle from zoomed-agent to freeform', () => {
      useCameraStore.setState({ mode: 'zoomed-agent' })
      useCameraStore.getState().cycleCameraMode()

      expect(useCameraStore.getState().mode).toBe('freeform')
    })

    it('should cycle from freeform to top-down', () => {
      useCameraStore.setState({ mode: 'freeform' })
      useCameraStore.getState().cycleCameraMode()

      expect(useCameraStore.getState().mode).toBe('top-down')
    })

    it('should cycle from top-down to zoomed-agent when agent provided', () => {
      useCameraStore.setState({ mode: 'top-down' })
      useCameraStore.getState().cycleCameraMode('agent-1', [0, 0, 0])

      expect(useCameraStore.getState().mode).toBe('zoomed-agent')
      expect(useCameraStore.getState().focusedAgentId).toBe('agent-1')
    })

    it('should cycle from top-down to freeform when no agent provided', () => {
      useCameraStore.setState({ mode: 'top-down' })
      useCameraStore.getState().cycleCameraMode()

      expect(useCameraStore.getState().mode).toBe('freeform')
    })
  })

  describe('pushHistory', () => {
    it('should add current state to history', () => {
      useCameraStore.setState({
        position: [1, 2, 3],
        target: [4, 5, 6],
        zoom: 2,
        mode: 'freeform',
      })

      useCameraStore.getState().pushHistory()

      expect(useCameraStore.getState().history).toEqual([
        {
          position: [1, 2, 3],
          target: [4, 5, 6],
          zoom: 2,
          mode: 'freeform',
        },
      ])
    })

    it('should limit history to 10 entries', () => {
      for (let i = 0; i < 15; i++) {
        useCameraStore.setState({ zoom: i })
        useCameraStore.getState().pushHistory()
      }

      expect(useCameraStore.getState().history).toHaveLength(10)
    })
  })

  describe('popHistory', () => {
    it('should return and remove last entry', () => {
      useCameraStore.setState({
        history: [
          { position: [1, 1, 1], target: [0, 0, 0], zoom: 1, mode: 'freeform' as const },
          { position: [2, 2, 2], target: [0, 0, 0], zoom: 2, mode: 'top-down' as const },
        ],
      })

      const entry = useCameraStore.getState().popHistory()

      expect(entry).toEqual({
        position: [2, 2, 2],
        target: [0, 0, 0],
        zoom: 2,
        mode: 'top-down',
      })
      expect(useCameraStore.getState().history).toHaveLength(1)
    })

    it('should return null when history is empty', () => {
      const entry = useCameraStore.getState().popHistory()
      expect(entry).toBeNull()
    })
  })

  describe('clearHistory', () => {
    it('should clear all history', () => {
      useCameraStore.setState({
        history: [
          { position: [1, 1, 1], target: [0, 0, 0], zoom: 1, mode: 'freeform' as const },
          { position: [2, 2, 2], target: [0, 0, 0], zoom: 2, mode: 'top-down' as const },
        ],
      })

      useCameraStore.getState().clearHistory()

      expect(useCameraStore.getState().history).toEqual([])
    })
  })
})
