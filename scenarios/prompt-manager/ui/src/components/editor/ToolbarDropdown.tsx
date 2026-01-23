/**
 * ToolbarDropdown - Dropdown menu for toolbar buttons.
 *
 * Used to group related toolbar buttons in a dropdown on small screens.
 * Provides click-outside-to-close behavior.
 */

import { useState, useRef, useEffect, type ReactNode } from 'react'
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
  /** Additional class names for the trigger button */
  className?: string
}

/**
 * Dropdown component for grouping toolbar buttons.
 */
export function ToolbarDropdown({
  icon,
  label,
  children,
  hasActiveItem = false,
  className,
}: ToolbarDropdownProps) {
  const [isOpen, setIsOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)

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
      >
        {icon}
        <ChevronDown className={cn('h-3 w-3 transition-transform', isOpen && 'rotate-180')} />
      </button>

      {/* Dropdown panel */}
      {isOpen && (
        <div
          className={cn(
            'absolute top-full left-0 mt-1 z-50',
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
    >
      <span className="w-4 h-4 flex items-center justify-center">{icon}</span>
      <span>{label}</span>
    </button>
  )
}
