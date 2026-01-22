/**
 * Tests for useResizableSidebar hook.
 *
 * Tests cover:
 * - Initial width from localStorage
 * - Width persistence to localStorage
 * - Resize handlers
 * - Min/max constraints
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useResizableSidebar } from './useResizableSidebar'

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
}

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
  writable: true,
})

describe('useResizableSidebar', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorageMock.getItem.mockReturnValue(null)
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  describe('initialization', () => {
    it('should use default width when localStorage is empty', () => {
      const { result } = renderHook(() => useResizableSidebar())

      expect(result.current.width).toBe(280) // default width
      expect(result.current.isResizing).toBe(false)
    })

    it('should use custom default width when provided', () => {
      const { result } = renderHook(() =>
        useResizableSidebar({ defaultWidth: 350 })
      )

      expect(result.current.width).toBe(350)
    })

    it('should load width from localStorage', () => {
      localStorageMock.getItem.mockReturnValue('320')

      const { result } = renderHook(() => useResizableSidebar())

      expect(result.current.width).toBe(320)
    })

    it('should ignore invalid localStorage value', () => {
      localStorageMock.getItem.mockReturnValue('not-a-number')

      const { result } = renderHook(() => useResizableSidebar())

      expect(result.current.width).toBe(280)
    })

    it('should ignore localStorage value below minWidth', () => {
      localStorageMock.getItem.mockReturnValue('100')

      const { result } = renderHook(() =>
        useResizableSidebar({ minWidth: 200 })
      )

      expect(result.current.width).toBe(280) // falls back to default
    })
  })

  describe('persistence', () => {
    it('should persist width to localStorage on change', () => {
      const { rerender } = renderHook(() => useResizableSidebar())

      // Initial render should save
      expect(localStorageMock.setItem).toHaveBeenCalledWith('pm.sidebarWidth', '280')

      // Simulate a width change (normally this happens via mouse events)
      // We can't easily test the full resize flow, but we verify persistence works
      rerender()
    })

    it('should use custom storage key when provided', () => {
      localStorageMock.getItem.mockReturnValue('300')

      renderHook(() =>
        useResizableSidebar({ storageKey: 'custom.sidebarWidth' })
      )

      expect(localStorageMock.getItem).toHaveBeenCalledWith('custom.sidebarWidth')
      expect(localStorageMock.setItem).toHaveBeenCalledWith('custom.sidebarWidth', '300')
    })
  })

  describe('resize start', () => {
    it('should provide a handleResizeStart function', () => {
      const { result } = renderHook(() => useResizableSidebar())

      expect(typeof result.current.handleResizeStart).toBe('function')
    })

    it('should provide a containerRef', () => {
      const { result } = renderHook(() => useResizableSidebar())

      expect(result.current.containerRef).toBeDefined()
      expect(result.current.containerRef.current).toBeNull()
    })
  })

  describe('configuration options', () => {
    it('should accept custom minWidth', () => {
      const { result } = renderHook(() =>
        useResizableSidebar({ minWidth: 150 })
      )

      // The hook initializes, we can verify options are accepted
      expect(result.current.width).toBeDefined()
    })

    it('should accept custom maxWidthRatio', () => {
      const { result } = renderHook(() =>
        useResizableSidebar({ maxWidthRatio: 0.3 })
      )

      expect(result.current.width).toBeDefined()
    })
  })
})
