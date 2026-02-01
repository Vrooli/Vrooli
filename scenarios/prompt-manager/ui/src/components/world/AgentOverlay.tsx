/**
 * AgentOverlay - Overlay UI that appears when camera is zoomed to an agent.
 *
 * Features:
 * - Close button to exit zoom
 * - Action buttons: Customize, Set Skills, Delete, Duplicate
 * - Draggable positioning
 */

import { useState, useRef, useCallback } from 'react'
import { X, Palette, Zap, Copy, Trash2, GripVertical } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Agent } from '@/types/agent'

interface AgentOverlayProps {
  agent: Agent | null
  isVisible: boolean
  onClose: () => void
  onCustomize: () => void
  onSetSkills: () => void
  onDuplicate: () => void
  onDelete: () => void
}

/**
 * Agent overlay component for quick actions.
 */
export function AgentOverlay({
  agent,
  isVisible,
  onClose,
  onCustomize,
  onSetSkills,
  onDuplicate,
  onDelete,
}: AgentOverlayProps) {
  // Drag state
  const [position, setPosition] = useState({ x: 20, y: 20 })
  const [isDragging, setIsDragging] = useState(false)
  const dragOffset = useRef({ x: 0, y: 0 })
  const overlayRef = useRef<HTMLDivElement>(null)

  // Handle drag start
  const handleDragStart = useCallback((e: React.MouseEvent) => {
    if (!overlayRef.current) return

    setIsDragging(true)
    const rect = overlayRef.current.getBoundingClientRect()
    dragOffset.current = {
      x: e.clientX - rect.left,
      y: e.clientY - rect.top,
    }

    // Add move and up listeners to window
    const handleMove = (moveEvent: MouseEvent) => {
      const newX = moveEvent.clientX - dragOffset.current.x
      const newY = moveEvent.clientY - dragOffset.current.y

      // Clamp to window bounds
      const maxX = window.innerWidth - (overlayRef.current?.offsetWidth ?? 200)
      const maxY = window.innerHeight - (overlayRef.current?.offsetHeight ?? 200)

      setPosition({
        x: Math.max(0, Math.min(newX, maxX)),
        y: Math.max(0, Math.min(newY, maxY)),
      })
    }

    const handleUp = () => {
      setIsDragging(false)
      window.removeEventListener('mousemove', handleMove)
      window.removeEventListener('mouseup', handleUp)
    }

    window.addEventListener('mousemove', handleMove)
    window.addEventListener('mouseup', handleUp)
  }, [])

  if (!isVisible || !agent) return null

  const bodyColor = agent.appearance?.body ?? '#6366f1'
  const headColor = agent.appearance?.head ?? '#a5b4fc'

  return (
    <div
      ref={overlayRef}
      className={cn(
        'fixed z-40 w-64',
        'bg-card/95 backdrop-blur-sm border border-border rounded-xl shadow-2xl',
        'animate-in fade-in-0 slide-in-from-left-2 duration-200',
        isDragging && 'select-none cursor-grabbing'
      )}
      style={{
        left: position.x,
        top: position.y,
      }}
    >
      {/* Header with drag handle */}
      <div className="flex items-center gap-2 px-3 py-2 border-b border-border">
        {/* Drag handle */}
        <button
          type="button"
          onMouseDown={handleDragStart}
          className={cn(
            'p-1 rounded cursor-grab',
            'text-muted-foreground hover:text-foreground hover:bg-muted transition-colors',
            isDragging && 'cursor-grabbing'
          )}
          title="Drag to move"
        >
          <GripVertical className="h-4 w-4" />
        </button>

        {/* Agent info */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            {/* Mini agent preview */}
            <div
              className="w-6 h-6 rounded-full flex items-center justify-center flex-shrink-0"
              style={{ backgroundColor: bodyColor }}
            >
              <div
                className="w-3 h-3 rounded-full"
                style={{ backgroundColor: headColor }}
              />
            </div>
            <span className="text-sm font-medium text-foreground truncate">
              {agent.displayName}
            </span>
          </div>
        </div>

        {/* Close button */}
        <button
          type="button"
          onClick={onClose}
          className={cn(
            'p-1 rounded',
            'text-muted-foreground hover:text-foreground hover:bg-muted transition-colors'
          )}
          title="Close (Esc)"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Action buttons */}
      <div className="p-2 space-y-1">
        <OverlayButton
          icon={<Palette className="h-4 w-4" />}
          label="Customize"
          description="Change colors and name"
          onClick={onCustomize}
        />
        <OverlayButton
          icon={<Zap className="h-4 w-4" />}
          label="Set Skills"
          description={`${agent.skills.length} skills assigned`}
          onClick={onSetSkills}
        />
        <OverlayButton
          icon={<Copy className="h-4 w-4" />}
          label="Duplicate"
          description="Create a copy"
          onClick={onDuplicate}
        />
        <OverlayButton
          icon={<Trash2 className="h-4 w-4" />}
          label="Delete"
          description="Remove this agent"
          onClick={onDelete}
          variant="danger"
        />
      </div>

      {/* Skills preview */}
      {agent.skills.length > 0 && (
        <div className="px-3 pb-3">
          <p className="text-xs text-muted-foreground mb-1">Assigned Skills:</p>
          <div className="flex flex-wrap gap-1">
            {agent.skills.slice(0, 4).map((skillId) => (
              <span
                key={skillId}
                className="px-1.5 py-0.5 text-[10px] bg-primary/20 text-primary rounded"
              >
                {skillId.substring(0, 8)}...
              </span>
            ))}
            {agent.skills.length > 4 && (
              <span className="px-1.5 py-0.5 text-[10px] text-muted-foreground">
                +{agent.skills.length - 4} more
              </span>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

/**
 * Individual action button in the overlay.
 */
interface OverlayButtonProps {
  icon: React.ReactNode
  label: string
  description: string
  onClick: () => void
  variant?: 'default' | 'danger'
}

function OverlayButton({
  icon,
  label,
  description,
  onClick,
  variant = 'default',
}: OverlayButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'w-full flex items-center gap-3 px-3 py-2 rounded-lg text-left',
        'transition-colors',
        variant === 'danger'
          ? 'hover:bg-destructive/10 hover:text-destructive'
          : 'hover:bg-muted'
      )}
    >
      <div
        className={cn(
          'p-1.5 rounded-md',
          variant === 'danger' ? 'bg-destructive/10 text-destructive' : 'bg-muted text-muted-foreground'
        )}
      >
        {icon}
      </div>
      <div className="flex-1 min-w-0">
        <p
          className={cn(
            'text-sm font-medium',
            variant === 'danger' ? 'text-destructive' : 'text-foreground'
          )}
        >
          {label}
        </p>
        <p className="text-xs text-muted-foreground truncate">{description}</p>
      </div>
    </button>
  )
}
