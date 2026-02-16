/**
 * Popover - Anchored overlay without backdrop.
 *
 * Used for context menus, filter popovers, and any anchored floating content.
 * Supports fixed positioning (x/y) for context menus and click-outside/Esc to close.
 */

import { useRef, useEffect, type ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { useModalBehavior } from '@/hooks/useModalBehavior'

export interface PopoverProps {
  /** Whether the popover is visible */
  isOpen: boolean
  /** Callback to close the popover */
  onClose: () => void
  /** Fixed X position (for context menus) */
  x?: number
  /** Fixed Y position (for context menus) */
  y?: number
  /** Popover content */
  children: ReactNode
  /** Additional CSS classes for the container */
  className?: string
  /** Delay click-outside listener (prevents instant close from triggering right-click) */
  delayClickOutside?: boolean
  /** data-testid value */
  testId?: string
}

/**
 * Generic popover primitive. Handles escape, click-outside, and viewport clamping.
 */
export function Popover({
  isOpen,
  onClose,
  x,
  y,
  children,
  className,
  delayClickOutside = false,
  testId,
}: PopoverProps) {
  const menuRef = useRef<HTMLDivElement>(null)

  useModalBehavior({
    isOpen,
    onClose,
    ref: menuRef,
    delayClickOutside,
  })

  // Viewport clamping
  useEffect(() => {
    if (!isOpen || !menuRef.current || x === undefined || y === undefined) return

    const rect = menuRef.current.getBoundingClientRect()
    let adjustedX = x
    let adjustedY = y

    if (x + rect.width > window.innerWidth) {
      adjustedX = window.innerWidth - rect.width - 8
    }
    if (y + rect.height > window.innerHeight) {
      adjustedY = window.innerHeight - rect.height - 8
    }

    menuRef.current.style.left = `${adjustedX}px`
    menuRef.current.style.top = `${adjustedY}px`
  }, [isOpen, x, y])

  if (!isOpen) return null

  return (
    <div
      ref={menuRef}
      className={cn(
        'fixed z-50 min-w-[180px] overflow-visible rounded-md',
        'bg-popover border border-border shadow-lg',
        'animate-in fade-in-0 zoom-in-95 duration-100',
        className,
      )}
      style={x !== undefined && y !== undefined ? { left: x, top: y } : undefined}
      data-testid={testId}
    >
      {children}
    </div>
  )
}
