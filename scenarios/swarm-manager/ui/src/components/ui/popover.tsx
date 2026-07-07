/**
 * Popover - Anchored overlay without backdrop.
 *
 * Used for context menus, filter popovers, and any anchored floating content.
 * Supports fixed positioning (x/y) for context menus and click-outside/Esc to close.
 */

import { useRef, useState, useLayoutEffect, type ReactNode, type RefObject } from 'react'
import { createPortal } from 'react-dom'
import { cn } from '../../lib/utils'
import { useModalBehavior } from '../../hooks/useModalBehavior'

type PopoverPlacement = 'bottom-start' | 'bottom-end' | 'top-start' | 'top-end'

export interface PopoverProps {
  /** Whether the popover is visible */
  isOpen: boolean
  /** Callback to close the popover */
  onClose: () => void
  /** Fixed X position (for context menus) */
  x?: number
  /** Fixed Y position (for context menus) */
  y?: number
  /** Trigger element to anchor the popover to */
  triggerRef?: RefObject<HTMLElement | null>
  /** Trigger-anchored placement */
  placement?: PopoverPlacement
  /** Gap between the trigger and the popover */
  offset?: number
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
  triggerRef,
  placement = 'bottom-start',
  offset = 8,
  children,
  className,
  delayClickOutside = false,
  testId,
}: PopoverProps) {
  const menuRef = useRef<HTMLDivElement>(null)
  // Measure-then-reveal: the anchored node is measured while hidden, then its
  // final position is written and it is revealed — all inside a pre-paint
  // useLayoutEffect, so a trigger-anchored menu never paints a top-left frame
  // before flying into place.
  const [positioned, setPositioned] = useState(false)

  useModalBehavior({
    isOpen,
    onClose,
    ref: menuRef,
    delayClickOutside,
  })

  useLayoutEffect(() => {
    if (!isOpen) {
      // Re-hide so a reopened menu is measured again before it reveals.
      setPositioned(false)
      return
    }
    if (!menuRef.current) return

    const rect = menuRef.current.getBoundingClientRect()
    const triggerRect = triggerRef?.current?.getBoundingClientRect()
    let nextX = x
    let nextY = y

    if (triggerRect) {
      const isTop = placement.startsWith('top')
      const isEnd = placement.endsWith('end')
      nextX = isEnd ? triggerRect.right - rect.width : triggerRect.left
      nextY = isTop ? triggerRect.top - rect.height - offset : triggerRect.bottom + offset
    }

    if (nextX !== undefined && nextY !== undefined) {
      let adjustedX = nextX
      let adjustedY = nextY

      if (nextX + rect.width > window.innerWidth) {
        adjustedX = window.innerWidth - rect.width - 8
      }
      if (nextY + rect.height > window.innerHeight) {
        adjustedY = window.innerHeight - rect.height - 8
      }

      menuRef.current.style.left = `${Math.max(8, adjustedX)}px`
      menuRef.current.style.top = `${Math.max(8, adjustedY)}px`
    }

    // Reveal in the same pre-paint commit that wrote the position.
    setPositioned(true)
  }, [isOpen, offset, placement, triggerRef, x, y])

  if (!isOpen) return null

  const hasPosition = x !== undefined && y !== undefined
  const hasTrigger = Boolean(triggerRef?.current)
  const positionStyle = hasPosition ? { left: x, top: y } : hasTrigger ? { left: 0, top: 0 } : {}

  return createPortal(
    <div
      ref={menuRef}
      className={cn(
        'fixed z-50 min-w-[180px] overflow-visible rounded-md',
        'bg-slate-900 border border-white/10 shadow-lg',
        'animate-in fade-in-0 zoom-in-95 duration-100',
        className,
      )}
      style={{ ...positionStyle, visibility: positioned ? undefined : 'hidden' }}
      data-testid={testId}
    >
      {children}
    </div>,
    document.body,
  )
}
