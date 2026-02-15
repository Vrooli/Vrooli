/**
 * useResizableSplitPanel - Hook for managing a resizable split panel.
 *
 * Provides:
 * - Panel width state with localStorage persistence
 * - Mouse drag handlers for resize
 * - Min/max constraints with ResizeObserver clamping
 * - Cursor feedback during drag
 * - Snap-close: drag below a threshold to collapse the panel to zero width
 */

import { useState, useEffect, useCallback, useRef } from 'react'

const DEFAULT_WIDTH = 560
const MIN_WIDTH = 320
const MAX_WIDTH_RATIO = 0.75 // Maximum 75% of container width
const SNAP_CLOSE_THRESHOLD = 200 // Snap to collapsed when released below this width

interface UseResizableSplitPanelOptions {
  /** Default width if not stored in localStorage */
  defaultWidth?: number
  /** Minimum panel width */
  minWidth?: number
  /** Maximum ratio of container width (0-1) */
  maxWidthRatio?: number
  /** Which side the resizable panel is anchored to */
  anchor?: 'left' | 'right'
  /** localStorage key for persistence */
  storageKey?: string
  /** Width threshold below which panel snaps to collapsed (0). Set to 0 to disable snap-close. */
  snapCloseThreshold?: number
}

interface UseResizableSplitPanelResult {
  /** Current panel width in pixels */
  width: number
  /** Whether currently resizing */
  isResizing: boolean
  /** Whether the panel is snapped closed (width === 0) */
  isCollapsed: boolean
  /** Ref to attach to the container element for ResizeObserver */
  containerRef: React.RefObject<HTMLDivElement>
  /** Mouse down handler for the resize handle */
  handleResizeStart: (e: React.MouseEvent) => void
  /** Expand panel to its previous width (or defaultWidth) */
  expand: () => void
  /** Collapse panel to zero width */
  collapse: () => void
}

export function useResizableSplitPanel(
  options: UseResizableSplitPanelOptions = {}
): UseResizableSplitPanelResult {
  const {
    defaultWidth = DEFAULT_WIDTH,
    minWidth = MIN_WIDTH,
    maxWidthRatio = MAX_WIDTH_RATIO,
    anchor = 'left',
    storageKey = 'pm.editorSplitWidth',
    snapCloseThreshold = SNAP_CLOSE_THRESHOLD,
  } = options

  const containerRef = useRef<HTMLDivElement>(null)
  const resizeRef = useRef<{ startX: number; startWidth: number; maxWidth: number } | null>(null)
  const prevWidthRef = useRef<number>(defaultWidth)

  // Initialize width from localStorage
  const [width, setWidth] = useState(() => {
    if (typeof window === 'undefined') return defaultWidth
    const stored = localStorage.getItem(storageKey)
    if (stored) {
      const parsed = Number(stored)
      // Accept 0 (collapsed) or any value >= minWidth
      if (Number.isFinite(parsed) && (parsed === 0 || parsed >= minWidth)) {
        if (parsed > 0) prevWidthRef.current = parsed
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
        // Don't clamp when collapsed
        if (prev === 0) return 0
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
      const direction = anchor === 'right' ? -1 : 1
      const rawWidth = resizeRef.current.startWidth + delta * direction

      // When snap-close is enabled, allow dragging below minWidth (down to 0) for visual feedback
      const clampedWidth = snapCloseThreshold > 0
        ? Math.max(0, Math.min(resizeRef.current.maxWidth, rawWidth))
        : Math.max(minWidth, Math.min(resizeRef.current.maxWidth, rawWidth))
      setWidth(clampedWidth)
    }

    const handleMouseUp = () => {
      setIsResizing(false)

      // Snap-close: if released below threshold, collapse; otherwise clamp to minWidth
      if (snapCloseThreshold > 0) {
        setWidth((prev) => {
          if (prev < snapCloseThreshold && prev < minWidth) {
            // Store the width we started dragging from for expand restoration
            prevWidthRef.current = resizeRef.current?.startWidth ?? defaultWidth
            resizeRef.current = null
            return 0
          }
          resizeRef.current = null
          return prev < minWidth ? minWidth : prev
        })
      } else {
        resizeRef.current = null
      }

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
  }, [anchor, isResizing, minWidth, snapCloseThreshold, defaultWidth])

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

  const expand = useCallback(() => {
    const restored = prevWidthRef.current > 0 ? prevWidthRef.current : defaultWidth
    setWidth(restored)
  }, [defaultWidth])

  const collapse = useCallback(() => {
    if (width > 0) {
      prevWidthRef.current = width
    }
    setWidth(0)
  }, [width])

  return {
    width,
    isResizing,
    isCollapsed: width === 0,
    containerRef,
    handleResizeStart,
    expand,
    collapse,
  }
}
