/**
 * useResizableSidebar - Hook for managing a resizable sidebar.
 *
 * Provides:
 * - Sidebar width state with localStorage persistence
 * - Mouse drag handlers for resize
 * - Min/max constraints with ResizeObserver clamping
 * - Cursor feedback during drag
 *
 * Based on the resize pattern from git-control-tower.
 */

import { useState, useEffect, useCallback, useRef } from 'react'

const STORAGE_KEY = 'pm.sidebarWidth'
const DEFAULT_WIDTH = 280
const MIN_WIDTH = 200
const MAX_WIDTH_RATIO = 0.5 // Maximum 50% of container width

interface UseResizableSidebarOptions {
  /** Default width if not stored in localStorage */
  defaultWidth?: number
  /** Minimum sidebar width */
  minWidth?: number
  /** Maximum ratio of container width (0-1) */
  maxWidthRatio?: number
  /** localStorage key for persistence */
  storageKey?: string
}

interface UseResizableSidebarResult {
  /** Current sidebar width in pixels */
  width: number
  /** Whether currently resizing */
  isResizing: boolean
  /** Ref to attach to the container element for ResizeObserver */
  containerRef: React.RefObject<HTMLDivElement>
  /** Mouse down handler for the resize handle */
  handleResizeStart: (e: React.MouseEvent) => void
}

export function useResizableSidebar(
  options: UseResizableSidebarOptions = {}
): UseResizableSidebarResult {
  const {
    defaultWidth = DEFAULT_WIDTH,
    minWidth = MIN_WIDTH,
    maxWidthRatio = MAX_WIDTH_RATIO,
    storageKey = STORAGE_KEY,
  } = options

  const containerRef = useRef<HTMLDivElement>(null)
  const resizeRef = useRef<{ startX: number; startWidth: number; maxWidth: number } | null>(null)

  // Initialize width from localStorage
  const [width, setWidth] = useState(() => {
    if (typeof window === 'undefined') return defaultWidth
    const stored = localStorage.getItem(storageKey)
    if (stored) {
      const parsed = Number(stored)
      if (Number.isFinite(parsed) && parsed >= minWidth) {
        return parsed
      }
    }
    return defaultWidth
  })

  const [isResizing, setIsResizing] = useState(false)

  // Persist width to localStorage
  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem(storageKey, String(width))
    }
  }, [width, storageKey])

  // ResizeObserver to clamp width when container resizes
  useEffect(() => {
    if (!containerRef.current || typeof ResizeObserver === 'undefined') return

    const clamp = () => {
      if (!containerRef.current) return
      const containerWidth = containerRef.current.clientWidth
      const maxWidth = Math.floor(containerWidth * maxWidthRatio)

      setWidth((prev) => {
        if (prev > maxWidth) return Math.max(minWidth, maxWidth)
        if (prev < minWidth) return minWidth
        return prev
      })
    }

    clamp()
    const observer = new ResizeObserver(clamp)
    observer.observe(containerRef.current)

    return () => observer.disconnect()
  }, [minWidth, maxWidthRatio])

  // Mouse event handlers for resizing
  useEffect(() => {
    if (!isResizing) return

    const handleMouseMove = (e: MouseEvent) => {
      if (!resizeRef.current) return

      const delta = e.clientX - resizeRef.current.startX
      const newWidth = resizeRef.current.startWidth + delta
      const clampedWidth = Math.max(minWidth, Math.min(resizeRef.current.maxWidth, newWidth))
      setWidth(clampedWidth)
    }

    const handleMouseUp = () => {
      setIsResizing(false)
      resizeRef.current = null
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }

    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
    window.addEventListener('mousemove', handleMouseMove)
    window.addEventListener('mouseup', handleMouseUp)

    return () => {
      window.removeEventListener('mousemove', handleMouseMove)
      window.removeEventListener('mouseup', handleMouseUp)
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }
  }, [isResizing, minWidth])

  // Handler to start resizing
  const handleResizeStart = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault()
      if (!containerRef.current) return

      const containerWidth = containerRef.current.clientWidth
      const maxWidth = Math.floor(containerWidth * maxWidthRatio)

      resizeRef.current = {
        startX: e.clientX,
        startWidth: width,
        maxWidth,
      }
      setIsResizing(true)
    },
    [width, maxWidthRatio]
  )

  return {
    width,
    isResizing,
    containerRef,
    handleResizeStart,
  }
}
