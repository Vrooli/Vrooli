/**
 * ToolbarDropdown - Dropdown menu for toolbar buttons.
 *
 * Used to group related toolbar buttons in a dropdown on small screens.
 * Provides click-outside-to-close behavior.
 */

import { useState, useRef, useEffect, useLayoutEffect, type ReactNode } from 'react'
import { ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'

interface ToolbarDropdownProps {
  /** Icon to show in the trigger button */
  icon: ReactNode
  /** Label for the dropdown trigger */
  label: string
  /** Children are rendered inside the dropdown panel */
  children: ReactNode
  /** Whether any item in the dropdown is currently active */
  hasActiveItem?: boolean
  /** Whether to show the chevron icon in the trigger */
  showChevron?: boolean
  /** Alignment for the dropdown panel */
  align?: 'left' | 'right'
  /** Additional class names for the trigger button */
  className?: string
  /** Optional data-testid for the trigger button */
  testId?: string
}

/**
 * Dropdown component for grouping toolbar buttons.
 */
export function ToolbarDropdown({
  icon,
  label,
  children,
  hasActiveItem = false,
  showChevron = true,
  align = 'left',
  className,
  testId,
}: ToolbarDropdownProps) {
  const [isOpen, setIsOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)
  const [position, setPosition] = useState<{ top: number; left: number; width: number }>({ top: 0, left: 0, width: 180 })

  // Close on click outside
  useEffect(() => {
    if (!isOpen) return

    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isOpen])

  useLayoutEffect(() => {
    if (!isOpen || !triggerRef.current) return

    const trigger = triggerRef.current.getBoundingClientRect()
    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight
    const measuredWidth = panelRef.current?.scrollWidth ?? 180
    const width = viewportWidth < 640
      ? viewportWidth - 16
      : Math.min(Math.max(trigger.width, measuredWidth), viewportWidth - 16)
    const estimatedHeight = Math.min(panelRef.current?.scrollHeight ?? 260, viewportHeight - 16)

    let left = align === 'right' ? trigger.right - width : trigger.left
    let top = trigger.bottom + 4

    if (left + width > viewportWidth - 8) {
      left = viewportWidth - width - 8
    }
    if (left < 8) {
      left = 8
    }

    if (top + estimatedHeight > viewportHeight - 8) {
      top = Math.max(8, trigger.top - estimatedHeight - 4)
    }

    setPosition({ top, left, width })
  }, [isOpen, align])

  // Close on Escape key
  useEffect(() => {
    if (!isOpen) return

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setIsOpen(false)
      }
    }

    document.addEventListener('keydown', handleEscape)
    return () => {
      document.removeEventListener('keydown', handleEscape)
    }
  }, [isOpen])

  return (
    <div ref={dropdownRef} className="relative">
      {/* Trigger button */}
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className={cn(
          'flex items-center gap-1 px-2 py-1.5 rounded transition-colors',
          hasActiveItem
            ? 'bg-primary/30 text-primary'
            : 'text-muted-foreground hover:text-foreground hover:bg-muted',
          className
        )}
        title={label}
        aria-label={label}
        data-testid={testId}
      >
        {icon}
        {showChevron && (
          <ChevronDown className={cn('h-3 w-3 transition-transform', isOpen && 'rotate-180')} />
        )}
      </button>

      {/* Dropdown panel */}
      {isOpen && (
        <div
          ref={panelRef}
          style={{
            position: 'fixed',
            top: position.top,
            left: position.left,
            width: position.width,
            maxWidth: 'calc(100vw - 16px)',
            maxHeight: 'calc(100vh - 16px)',
          }}
          className={cn(
            'z-50 overflow-y-auto',
            'bg-card border border-border rounded-lg shadow-lg',
            'py-1 min-w-[120px]'
          )}
          onClick={() => setIsOpen(false)}
        >
          {children}
        </div>
      )}
    </div>
  )
}

interface DropdownItemProps {
  onClick: () => void
  isActive?: boolean
  disabled?: boolean
  icon: ReactNode
  label: string
  testId?: string
}

/**
 * Item component for use inside ToolbarDropdown.
 */
export function DropdownItem({
  onClick,
  isActive = false,
  disabled = false,
  icon,
  label,
  testId,
}: DropdownItemProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={cn(
        'w-full flex items-center gap-2 px-3 py-1.5 text-sm transition-colors',
        isActive
          ? 'bg-primary/20 text-primary'
          : 'text-foreground hover:bg-muted',
        disabled && 'opacity-50 cursor-not-allowed'
      )}
      data-testid={testId}
    >
      <span className="w-4 h-4 flex items-center justify-center">{icon}</span>
      <span>{label}</span>
    </button>
  )
}
