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
  /** Ref for the sidebar panel whose width is live-mutated during drag */
  panelRef: React.RefObject<HTMLDivElement | null>
  /** Ref to attach to the container element for ResizeObserver */
  containerRef: React.RefObject<HTMLDivElement | null>
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
  const panelRef = useRef<HTMLDivElement>(null)
  const resizeRef = useRef<{ startX: number; startWidth: number; maxWidth: number; frame: number | null; latestWidth: number } | null>(null)

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

  // Persist width to localStorage
  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem(storageKey, String(width))
    }
  }, [width, storageKey])

  useEffect(() => {
    panelRef.current?.style.setProperty('--pm-sidebar-width', `${width}px`)
  }, [width])

  // ResizeObserver to clamp width when container resizes
  useEffect(() => {
    if (!containerRef.current || typeof ResizeObserver === 'undefined') return

    const clamp = () => {
      if (!containerRef.current) return
      const containerWidth = containerRef.current.clientWidth
      const maxWidth = Math.floor(containerWidth * maxWidthRatio)

      setWidth((prev) => {
        const next = Math.max(minWidth, Math.min(maxWidth, prev))
        if (panelRef.current) {
          panelRef.current.style.setProperty('--pm-sidebar-width', `${next}px`)
        }
        if (next !== prev) return next
        return prev
      })
    }

    clamp()
    const observer = new ResizeObserver(clamp)
    observer.observe(containerRef.current)

    return () => observer.disconnect()
  }, [minWidth, maxWidthRatio])

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
        frame: null,
        latestWidth: width,
      }

      const handleMouseMove = (event: MouseEvent) => {
        const resize = resizeRef.current
        if (!resize) return

        const delta = event.clientX - resize.startX
        resize.latestWidth = Math.max(minWidth, Math.min(resize.maxWidth, resize.startWidth + delta))
        if (resize.frame !== null) return

        resize.frame = window.requestAnimationFrame(() => {
          const current = resizeRef.current
          if (!current) return
          panelRef.current?.style.setProperty('--pm-sidebar-width', `${current.latestWidth}px`)
          current.frame = null
        })
      }

      const finishResize = () => {
        const resize = resizeRef.current
        if (resize && resize.frame !== null) {
          window.cancelAnimationFrame(resize.frame)
        }
        if (resize) {
          panelRef.current?.style.setProperty('--pm-sidebar-width', `${resize.latestWidth}px`)
          setWidth(resize.latestWidth)
        }
        resizeRef.current = null
        document.body.style.cursor = ''
        document.body.style.userSelect = ''
        panelRef.current?.removeAttribute('data-resizing')
        window.removeEventListener('mousemove', handleMouseMove)
        window.removeEventListener('mouseup', finishResize)
      }

      document.body.style.cursor = 'col-resize'
      document.body.style.userSelect = 'none'
      panelRef.current?.setAttribute('data-resizing', 'true')
      window.addEventListener('mousemove', handleMouseMove)
      window.addEventListener('mouseup', finishResize)
    },
    [width, maxWidthRatio, minWidth]
  )

  return {
    width,
    panelRef,
    containerRef,
    handleResizeStart,
  }
}
