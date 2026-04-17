/**
 * FloatingPanel - Draggable non-modal panel with no backdrop.
 *
 * Used for settings/help overlays so users can keep panels open while
 * interacting with content behind them.
 *
 * Features:
 * - Escape to close (via useModalBehavior)
 * - Viewport clamping on window resize
 * - Full header bar as drag handle (desktop)
 * - Mobile bottom tray mode (undraggable, slides up from bottom)
 */

import { useCallback, useEffect, useId, useRef, useState, type ReactNode } from 'react'
import { GripVertical, X, Minus } from 'lucide-react'
import { cn } from '../../lib/utils'
import { useModalBehavior } from '../../hooks/useModalBehavior'
import { useIsMobile } from '../../hooks/useMediaQuery'
import { useSpatialNavContext } from '../../hooks/SpatialNavContext'

interface Position {
  x: number
  y: number
}

interface FloatingPanelProps {
  isOpen: boolean
  onClose: () => void
  title: string
  children: ReactNode
  initialPosition?: Position
  className?: string
  testId?: string
}

const DEFAULT_POSITION: Position = { x: 24, y: 76 }

export function FloatingPanel({
  isOpen,
  onClose,
  title,
  children,
  initialPosition = DEFAULT_POSITION,
  className,
  testId,
}: FloatingPanelProps) {
  const [position, setPosition] = useState(initialPosition)
  const [isDragging, setIsDragging] = useState(false)
  const dragOffset = useRef({ x: 0, y: 0 })
  const panelRef = useRef<HTMLDivElement>(null)
  const titleId = useId()
  const isMobile = useIsMobile()

  // Escape to close (no scroll lock, no click-outside for non-modal panels)
  useModalBehavior({
    isOpen,
    onClose,
    ref: panelRef,
    disableCloseOnOutsideClick: true,
  })

  // Push a spatial nav modal scope so D-pad navigation is trapped inside the panel.
  const spatialNavRef = useSpatialNavContext();
  useEffect(() => {
    const ctrl = spatialNavRef?.current;
    const el = panelRef.current;
    if (!isOpen || !ctrl || !el) return;
    ctrl.pushScope(el);
    return () => { ctrl.popScope(); };
  }, [isOpen, spatialNavRef]);

  const clampPosition = useCallback((next: Position): Position => {
    const panelWidth = panelRef.current?.offsetWidth ?? 560
    const panelHeight = panelRef.current?.offsetHeight ?? 500
    const maxX = Math.max(8, window.innerWidth - panelWidth - 8)
    const maxY = Math.max(8, window.innerHeight - panelHeight - 8)
    return {
      x: Math.min(Math.max(next.x, 8), maxX),
      y: Math.min(Math.max(next.y, 8), maxY),
    }
  }, [])

  // Clamp position on window resize
  useEffect(() => {
    if (!isOpen || isMobile) return

    const handleResize = () => {
      setPosition((prev) => clampPosition(prev))
    }

    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [isOpen, isMobile, clampPosition])

  const handleDragStart = useCallback((e: React.MouseEvent) => {
    if (!panelRef.current || isMobile) return
    setIsDragging(true)
    const rect = panelRef.current.getBoundingClientRect()
    dragOffset.current = {
      x: e.clientX - rect.left,
      y: e.clientY - rect.top,
    }

    const handleMove = (moveEvent: MouseEvent) => {
      const next = {
        x: moveEvent.clientX - dragOffset.current.x,
        y: moveEvent.clientY - dragOffset.current.y,
      }
      setPosition(clampPosition(next))
    }

    const handleUp = () => {
      setIsDragging(false)
      window.removeEventListener('mousemove', handleMove)
      window.removeEventListener('mouseup', handleUp)
    }

    window.addEventListener('mousemove', handleMove)
    window.addEventListener('mouseup', handleUp)
  }, [clampPosition, isMobile])

  if (!isOpen) return null

  // Mobile: bottom tray mode
  if (isMobile) {
    return (
      <div
        ref={panelRef}
        className={cn(
          'fixed inset-x-0 bottom-0 z-40',
          'bg-slate-900/95 border-t border-white/10 rounded-t-2xl shadow-2xl backdrop-blur-sm',
          'animate-in slide-in-from-bottom duration-200',
          className,
        )}
        role="dialog"
        aria-modal="false"
        aria-labelledby={titleId}
        data-testid={testId}
      >
        {/* Drag handle indicator (visual only) */}
        <div className="flex justify-center pt-2 pb-1">
          <Minus className="h-5 w-8 text-slate-500" />
        </div>
        <div className="flex items-center gap-2 px-3 py-1 border-b border-white/10">
          <h2 id={titleId} className="flex-1 text-sm font-semibold text-slate-100">
            {title}
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="p-1 rounded text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition-colors"
            aria-label="Close panel"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="p-4 max-h-[85vh] overflow-y-auto">
          {children}
        </div>
      </div>
    )
  }

  // Desktop: draggable floating panel
  return (
    <div
      ref={panelRef}
      className={cn(
        'fixed z-40 w-[92vw] max-w-2xl',
        'bg-slate-900/95 border border-white/10 rounded-xl shadow-2xl backdrop-blur-sm',
        isDragging && 'select-none cursor-grabbing',
        className,
      )}
      style={{ left: position.x, top: position.y }}
      role="dialog"
      aria-modal="false"
      aria-labelledby={titleId}
      data-testid={testId}
    >
      {/* Full header as drag handle */}
      <div
        className={cn(
          'flex items-center gap-2 px-3 py-2 border-b border-white/10 cursor-grab',
          isDragging && 'cursor-grabbing',
        )}
        onMouseDown={handleDragStart}
      >
        <GripVertical className="h-4 w-4 text-slate-400" />
        <h2 id={titleId} className="text-sm font-semibold text-slate-100">
          {title}
        </h2>
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation()
            onClose()
          }}
          onMouseDown={(e) => e.stopPropagation()}
          className="ml-auto p-1 rounded text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition-colors"
          aria-label="Close panel"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
      <div className="p-4 max-h-[85vh] overflow-y-auto">
        {children}
      </div>
    </div>
  )
}
