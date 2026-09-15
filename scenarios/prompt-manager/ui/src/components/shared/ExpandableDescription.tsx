/**
 * ExpandableDescription - Truncated description with inline expansion.
 *
 * Collapsed: 1-2 lines with ellipsis (line-clamp-2)
 * Expanded: textarea for editing (inline, not modal)
 * Save on blur or Ctrl+Enter, cancel on Escape
 */

import { useState, useRef, useEffect, useCallback } from 'react'
import { cn } from '@/lib/utils'
import { Skeleton } from '@/components/ui/skeleton'

interface ExpandableDescriptionProps {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  error?: string
  className?: string
  disabled?: boolean
  isLoading?: boolean
  maxLines?: number
}

/**
 * Expandable description component with inline editing.
 */
export function ExpandableDescription({
  value,
  onChange,
  placeholder = 'Click to add description...',
  error,
  className,
  disabled,
  isLoading,
  maxLines = 2,
}: ExpandableDescriptionProps) {
  const [isEditing, setIsEditing] = useState(false)
  const [editValue, setEditValue] = useState(value)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  // Sync edit value when value prop changes
  useEffect(() => {
    if (!isEditing) {
      setEditValue(value)
    }
  }, [value, isEditing])

  // Focus and auto-resize textarea when entering edit mode
  useEffect(() => {
    if (isEditing && textareaRef.current) {
      textareaRef.current.focus()
      // Move cursor to end
      textareaRef.current.selectionStart = textareaRef.current.value.length
      // Auto-resize
      autoResize(textareaRef.current)
    }
  }, [isEditing])

  function autoResize(textarea: HTMLTextAreaElement) {
    textarea.style.height = 'auto'
    textarea.style.height = `${Math.max(textarea.scrollHeight, 60)}px`
  }

  const handleSave = useCallback(() => {
    const trimmed = editValue.trim()
    if (trimmed !== value) {
      onChange(trimmed)
    }
    setIsEditing(false)
  }, [editValue, value, onChange])

  const handleCancel = useCallback(() => {
    setEditValue(value)
    setIsEditing(false)
  }, [value])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
        e.preventDefault()
        handleSave()
      } else if (e.key === 'Escape') {
        e.preventDefault()
        handleCancel()
      }
    },
    [handleSave, handleCancel]
  )

  const handleClick = useCallback(() => {
    if (!disabled) {
      setIsEditing(true)
    }
  }, [disabled])

  const handleDisplayKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if ((e.key === 'Enter' || e.key === ' ') && !disabled) {
        e.preventDefault()
        setIsEditing(true)
      }
    },
    [disabled]
  )

  const handleInput = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setEditValue(e.target.value)
    autoResize(e.target)
  }, [])

  if (isLoading) {
    return (
      <div className={cn('space-y-1.5', className)}>
        <Skeleton className="h-3 w-full" />
        <Skeleton className="h-3 w-2/3" />
      </div>
    )
  }

  // Line clamp class based on maxLines
  const lineClampClass = {
    1: 'line-clamp-1',
    2: 'line-clamp-2',
    3: 'line-clamp-3',
    4: 'line-clamp-4',
  }[maxLines] || 'line-clamp-2'

  if (isEditing) {
    return (
      <div className={cn('relative', className)}>
        <textarea
          ref={textareaRef}
          value={editValue}
          onChange={handleInput}
          onBlur={handleSave}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          rows={2}
          className={cn(
            'w-full px-2 py-1.5 bg-muted border rounded text-sm text-foreground resize-none',
            'placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary',
            error ? 'border-red-500' : 'border-border'
          )}
        />
        <p className="mt-0.5 text-[10px] text-muted-foreground">
          Ctrl+Enter to save, Escape to cancel
        </p>
        {error && <p className="mt-1 text-xs text-red-400">{error}</p>}
      </div>
    )
  }

  const displayValue = value || placeholder
  const isEmpty = !value

  return (
    <p
      onClick={handleClick}
      onKeyDown={handleDisplayKeyDown}
      tabIndex={disabled ? -1 : 0}
      role="button"
      aria-label={isEmpty ? 'Add description' : 'Edit description'}
      className={cn(
        'text-sm cursor-pointer transition-colors rounded px-1 -mx-1 py-0.5',
        'hover:bg-muted/50 focus:outline-none focus:ring-2 focus:ring-primary/50',
        lineClampClass,
        disabled && 'cursor-not-allowed opacity-50',
        isEmpty && 'text-muted-foreground italic',
        error && 'text-red-400',
        className
      )}
    >
      {displayValue}
    </p>
  )
}
