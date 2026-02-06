/**
 * InlineEditableText - Click-to-edit text component.
 *
 * Display mode: text with subtle hover indication
 * Edit mode: input field (triggered by click or Enter key)
 * Save on blur or Enter, cancel on Escape
 */

import { useState, useRef, useEffect, useCallback } from 'react'
import { cn } from '@/lib/utils'
import { Skeleton } from '@/components/ui/skeleton'

interface InlineEditableTextProps {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  error?: string
  as?: 'h1' | 'h2' | 'h3' | 'span' | 'p'
  className?: string
  inputClassName?: string
  disabled?: boolean
  isLoading?: boolean
  displayTestId?: string
  inputTestId?: string
}

/**
 * Inline editable text component.
 */
export function InlineEditableText({
  value,
  onChange,
  placeholder = 'Click to edit...',
  error,
  as = 'span',
  className,
  inputClassName,
  disabled,
  isLoading,
  displayTestId,
  inputTestId,
}: InlineEditableTextProps) {
  if (isLoading) {
    const skeletonSize = as === 'h1' ? 'h-6 w-48' : as === 'h2' ? 'h-5 w-48' : as === 'h3' ? 'h-5 w-40' : 'h-4 w-32'
    return <Skeleton className={cn(skeletonSize, className)} />
  }

  const [isEditing, setIsEditing] = useState(false)
  const [editValue, setEditValue] = useState(value)
  const inputRef = useRef<HTMLInputElement>(null)

  // Sync edit value when value prop changes
  useEffect(() => {
    if (!isEditing) {
      setEditValue(value)
    }
  }, [value, isEditing])

  // Focus input when entering edit mode
  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus()
      inputRef.current.select()
    }
  }, [isEditing])

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
      if (e.key === 'Enter') {
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

  // Base styles for the text element
  const baseTextStyles = cn(
    'cursor-pointer transition-colors rounded px-1 -mx-1',
    'hover:bg-muted/50 focus:outline-none focus:ring-2 focus:ring-primary/50',
    disabled && 'cursor-not-allowed opacity-50',
    error && 'text-red-400'
  )

  // Typography styles based on element type
  const typographyStyles = {
    h1: 'text-2xl font-bold',
    h2: 'text-lg font-semibold',
    h3: 'text-base font-medium',
    span: 'text-sm',
    p: 'text-sm',
  }

  if (isEditing) {
    return (
      <div className={cn('relative', className)}>
        <input
          ref={inputRef}
          type="text"
          value={editValue}
          onChange={(e) => setEditValue(e.target.value)}
          onBlur={handleSave}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          className={cn(
            'w-full px-2 py-1 bg-muted border rounded text-foreground',
            'placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary',
            error ? 'border-red-500' : 'border-border',
            typographyStyles[as],
            inputClassName
          )}
          data-testid={inputTestId}
        />
        {error && <p className="absolute -bottom-5 left-0 text-xs text-red-400">{error}</p>}
      </div>
    )
  }

  const displayValue = value || placeholder
  const isEmpty = !value

  // Render the appropriate HTML element
  const Element = as
  return (
    <Element
      onClick={handleClick}
      onKeyDown={handleDisplayKeyDown}
      tabIndex={disabled ? -1 : 0}
      role="button"
      aria-label={`Edit ${displayValue}`}
      className={cn(
        baseTextStyles,
        typographyStyles[as],
        isEmpty && 'text-muted-foreground italic',
        className
      )}
      data-testid={displayTestId}
    >
      {displayValue}
    </Element>
  )
}
