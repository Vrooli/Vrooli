/**
 * Tests for useHoverHighlight hook.
 *
 * Tests cover:
 * - Pointer events
 * - Cursor changes
 * - Hover state management
 * - Disabled state
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useHoverHighlight, useIsAnythingHovered, useHoveredObjectId } from './useHoverHighlight'
import { useInteractionStore } from '@/stores/interactionStore'

describe('useHoverHighlight', () => {
  beforeEach(() => {
    // Reset store state before each test
    useInteractionStore.getState().reset()
    // Reset cursor
    document.body.style.cursor = 'auto'
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('basic functionality', () => {
    it('should return isHovered as false initially', () => {
      const { result } = renderHook(() => useHoverHighlight('object-1'))

      expect(result.current.isHovered).toBe(false)
    })

    it('should return hover event handlers', () => {
      const { result } = renderHook(() => useHoverHighlight('object-1'))

      expect(typeof result.current.onPointerOver).toBe('function')
      expect(typeof result.current.onPointerOut).toBe('function')
    })

    it('should return hoverProps object', () => {
      const { result } = renderHook(() => useHoverHighlight('object-1'))

      expect(result.current.hoverProps).toHaveProperty('onPointerOver')
      expect(result.current.hoverProps).toHaveProperty('onPointerOut')
    })
  })

  describe('onPointerOver', () => {
    it('should set hovered state when pointer enters', () => {
      const { result } = renderHook(() => useHoverHighlight('object-1'))

      act(() => {
        result.current.onPointerOver({ stopPropagation: vi.fn() })
      })

      expect(useInteractionStore.getState().hoveredObjectId).toBe('object-1')
    })

    it('should update isHovered return value', () => {
      const { result } = renderHook(() => useHoverHighlight('object-1'))

      act(() => {
        result.current.onPointerOver({ stopPropagation: vi.fn() })
      })

      expect(result.current.isHovered).toBe(true)
    })

    it('should call stopPropagation', () => {
      const { result } = renderHook(() => useHoverHighlight('object-1'))
      const stopPropagation = vi.fn()

      act(() => {
        result.current.onPointerOver({ stopPropagation })
      })

      expect(stopPropagation).toHaveBeenCalled()
    })

    it('should change cursor to pointer by default', () => {
      const { result } = renderHook(() => useHoverHighlight('object-1'))

      act(() => {
        result.current.onPointerOver({ stopPropagation: vi.fn() })
      })

      expect(document.body.style.cursor).toBe('pointer')
    })

    it('should use custom cursor when specified', () => {
      const { result } = renderHook(() =>
        useHoverHighlight('object-1', { cursor: 'grab' })
      )

      act(() => {
        result.current.onPointerOver({ stopPropagation: vi.fn() })
      })

      expect(document.body.style.cursor).toBe('grab')
    })

    it('should not set hover when disabled', () => {
      const { result } = renderHook(() =>
        useHoverHighlight('object-1', { enabled: false })
      )

      act(() => {
        result.current.onPointerOver({ stopPropagation: vi.fn() })
      })

      expect(useInteractionStore.getState().hoveredObjectId).toBeNull()
    })

    it('should not set hover during drag', () => {
      // Start dragging another object
      useInteractionStore.getState().startDrag('other-object', [0, 0, 0])

      const { result } = renderHook(() => useHoverHighlight('object-1'))

      act(() => {
        result.current.onPointerOver({ stopPropagation: vi.fn() })
      })

      expect(useInteractionStore.getState().hoveredObjectId).toBeNull()
    })
  })

  describe('onPointerOut', () => {
    it('should clear hovered state when pointer leaves', () => {
      const { result } = renderHook(() => useHoverHighlight('object-1'))

      // First hover
      act(() => {
        result.current.onPointerOver({ stopPropagation: vi.fn() })
      })

      // Then leave
      act(() => {
        result.current.onPointerOut()
      })

      expect(useInteractionStore.getState().hoveredObjectId).toBeNull()
    })

    it('should reset cursor to auto', () => {
      const { result } = renderHook(() => useHoverHighlight('object-1'))

      act(() => {
        result.current.onPointerOver({ stopPropagation: vi.fn() })
      })

      act(() => {
        result.current.onPointerOut()
      })

      expect(document.body.style.cursor).toBe('auto')
    })

    it('should do nothing when disabled', () => {
      // Set hover state manually
      useInteractionStore.getState().setHovered('object-1')

      const { result } = renderHook(() =>
        useHoverHighlight('object-1', { enabled: false })
      )

      act(() => {
        result.current.onPointerOut()
      })

      // Should still be hovered since we're disabled
      expect(useInteractionStore.getState().hoveredObjectId).toBe('object-1')
    })
  })

  describe('isHovered return value', () => {
    it('should return false when disabled even if technically hovered', () => {
      // Manually set hover
      useInteractionStore.getState().setHovered('object-1')

      const { result } = renderHook(() =>
        useHoverHighlight('object-1', { enabled: false })
      )

      expect(result.current.isHovered).toBe(false)
    })

    it('should be true only for the hovered object', () => {
      const { result: result1 } = renderHook(() => useHoverHighlight('object-1'))
      const { result: result2 } = renderHook(() => useHoverHighlight('object-2'))

      act(() => {
        result1.current.onPointerOver({ stopPropagation: vi.fn() })
      })

      expect(result1.current.isHovered).toBe(true)
      expect(result2.current.isHovered).toBe(false)
    })
  })

  describe('multiple objects', () => {
    it('should correctly track hover across multiple objects', () => {
      const { result: result1 } = renderHook(() => useHoverHighlight('object-1'))
      const { result: result2 } = renderHook(() => useHoverHighlight('object-2'))

      // Hover first object
      act(() => {
        result1.current.onPointerOver({ stopPropagation: vi.fn() })
      })
      expect(result1.current.isHovered).toBe(true)
      expect(result2.current.isHovered).toBe(false)

      // Hover second object
      act(() => {
        result2.current.onPointerOver({ stopPropagation: vi.fn() })
      })
      expect(result1.current.isHovered).toBe(false)
      expect(result2.current.isHovered).toBe(true)
    })
  })
})

describe('useIsAnythingHovered', () => {
  beforeEach(() => {
    useInteractionStore.getState().reset()
  })

  it('should return false when nothing is hovered', () => {
    const { result } = renderHook(() => useIsAnythingHovered())
    expect(result.current).toBe(false)
  })

  it('should return true when something is hovered', () => {
    useInteractionStore.getState().setHovered('object-1')

    const { result } = renderHook(() => useIsAnythingHovered())
    expect(result.current).toBe(true)
  })
})

describe('useHoveredObjectId', () => {
  beforeEach(() => {
    useInteractionStore.getState().reset()
  })

  it('should return null when nothing is hovered', () => {
    const { result } = renderHook(() => useHoveredObjectId())
    expect(result.current).toBeNull()
  })

  it('should return hovered object id', () => {
    useInteractionStore.getState().setHovered('object-1')

    const { result } = renderHook(() => useHoveredObjectId())
    expect(result.current).toBe('object-1')
  })
})
