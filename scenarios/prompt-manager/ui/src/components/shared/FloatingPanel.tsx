/**
 * FloatingPanel - Draggable non-modal panel with no backdrop.
 *
 * Used for settings/help overlays so users can keep panels open while
 * interacting with content behind them.
 */

import { useCallback, useId, useRef, useState, type ReactNode } from 'react'
import { GripVertical, X } from 'lucide-react'
import { cn } from '@/lib/utils'

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
  const [position, setPosition] = useState<Position>(initialPosition)
  const [isDragging, setIsDragging] = useState(false)
  const dragOffset = useRef({ x: 0, y: 0 })
  const panelRef = useRef<HTMLDivElement>(null)
  const titleId = useId()

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

  const handleDragStart = useCallback((e: React.MouseEvent) => {
    if (!panelRef.current) return
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
  }, [clampPosition])

  if (!isOpen) return null

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
      <div className="flex items-center gap-2 px-3 py-2 border-b border-white/10">
        <button
          type="button"
          onMouseDown={handleDragStart}
          className={cn(
            'p-1 rounded cursor-grab text-slate-400 hover:text-white hover:bg-white/10 transition-colors',
            isDragging && 'cursor-grabbing',
          )}
          title="Drag to move"
          aria-label="Drag panel"
        >
          <GripVertical className="h-4 w-4" />
        </button>
        <h2 id={titleId} className="text-sm font-semibold text-white">
          {title}
        </h2>
        <button
          type="button"
          onClick={onClose}
          className="ml-auto p-1 rounded text-slate-400 hover:text-white hover:bg-white/10 transition-colors"
          aria-label="Close panel"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
      <div className="p-4 max-h-[70vh] overflow-y-auto">
        {children}
      </div>
    </div>
  )
}
