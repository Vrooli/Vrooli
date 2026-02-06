/**
 * TagChipsEditor - Inline tag editing with chips.
 *
 * Display tags as removable chips
 * [+] button opens popover for adding tags
 * Autocomplete from existing tags in codebase
 *
 * Note: This component now accepts tags as string[] (not comma-separated).
 */

import { useState, useRef, useEffect, useCallback } from 'react'
import { X, Plus, Tag } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Skeleton } from '@/components/ui/skeleton'

interface TagChipsEditorProps {
  /** Tags array */
  value: string[]
  onChange: (value: string[]) => void
  /** Available tags for autocomplete */
  availableTags?: string[]
  placeholder?: string
  disabled?: boolean
  isLoading?: boolean
  className?: string
}

/**
 * Tag chips editor component.
 */
export function TagChipsEditor({
  value,
  onChange,
  availableTags = [],
  placeholder = 'Add tags...',
  disabled,
  isLoading,
  className,
}: TagChipsEditorProps) {
  if (isLoading) {
    return (
      <div className={cn('flex items-center gap-1', className)}>
        <Tag className="h-3 w-3 text-muted-foreground mr-0.5" />
        <Skeleton className="h-5 w-14 rounded-full" />
        <Skeleton className="h-5 w-16 rounded-full" />
        <Skeleton className="h-5 w-12 rounded-full" />
      </div>
    )
  }

  const [isPopoverOpen, setIsPopoverOpen] = useState(false)
  const [inputValue, setInputValue] = useState('')
  const containerRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  // Tags are now passed as an array directly
  const tags = value

  // Filter suggestions based on input and exclude already-selected tags
  const lower = inputValue.toLowerCase()
  const filteredSuggestions = availableTags
    .filter((t) => !tags.includes(t))
    .filter((t) => !inputValue || t.toLowerCase().includes(lower))
    .slice(0, 10)

  // Check if current input is a new tag
  const trimmedInput = inputValue.trim()
  const isNewTag = trimmedInput && !availableTags.includes(trimmedInput) && !tags.includes(trimmedInput)

  // Handle click outside to close popover
  const handleClickOutside = useCallback((event: MouseEvent) => {
    if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
      setIsPopoverOpen(false)
      setInputValue('')
    }
  }, [])

  const handleEscape = useCallback((event: KeyboardEvent) => {
    if (event.key === 'Escape') {
      setIsPopoverOpen(false)
      setInputValue('')
    }
  }, [])

  useEffect(() => {
    if (isPopoverOpen) {
      document.addEventListener('mousedown', handleClickOutside)
      document.addEventListener('keydown', handleEscape)
      // Focus input when popover opens
      setTimeout(() => inputRef.current?.focus(), 0)
    }
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [isPopoverOpen, handleClickOutside, handleEscape])

  const addTag = useCallback(
    (tag: string) => {
      const trimmed = tag.trim()
      if (trimmed && !tags.includes(trimmed)) {
        onChange([...tags, trimmed])
      }
      setInputValue('')
      inputRef.current?.focus()
    },
    [tags, onChange]
  )

  const removeTag = useCallback(
    (tagToRemove: string) => {
      onChange(tags.filter((t) => t !== tagToRemove))
    },
    [tags, onChange]
  )

  const handleInputKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' && inputValue.trim()) {
        e.preventDefault()
        addTag(inputValue)
      } else if (e.key === 'Backspace' && !inputValue && tags.length > 0) {
        // Remove last tag on backspace when input is empty
        const lastTag = tags[tags.length - 1]
        if (lastTag) {
          removeTag(lastTag)
        }
      }
    },
    [inputValue, addTag, tags, removeTag]
  )

  return (
    <div ref={containerRef} className={cn('relative', className)}>
      <div className="flex items-center gap-1 flex-wrap">
        {/* Tag icon */}
        <Tag className="h-3 w-3 text-muted-foreground mr-0.5" />

        {/* Existing tags as chips */}
        {tags.map((tag) => (
          <span
            key={tag}
            className={cn(
              'inline-flex items-center gap-1 px-2 py-0.5',
              'text-xs bg-primary/20 text-primary rounded-full',
              'whitespace-nowrap'
            )}
          >
            {tag}
            {!disabled && (
              <button
                type="button"
                onClick={() => removeTag(tag)}
                className="p-0.5 hover:bg-primary/30 rounded-full transition-colors"
                title={`Remove ${tag}`}
              >
                <X className="h-2.5 w-2.5" />
              </button>
            )}
          </span>
        ))}

        {/* Add button */}
        {!disabled && (
          <button
            type="button"
            onClick={() => setIsPopoverOpen(true)}
            className={cn(
              'inline-flex items-center gap-0.5 px-1.5 py-0.5 text-xs',
              'text-muted-foreground hover:text-foreground',
              'hover:bg-muted/50 rounded transition-colors'
            )}
            title="Add tag"
          >
            <Plus className="h-3 w-3" />
          </button>
        )}

        {/* Empty state */}
        {tags.length === 0 && !isPopoverOpen && (
          <span className="text-xs text-muted-foreground italic">{placeholder}</span>
        )}
      </div>

      {/* Add tag popover */}
      {isPopoverOpen && (
        <div
          className={cn(
            'absolute z-50 mt-1 p-2 w-56',
            'bg-popover border border-border rounded-lg shadow-xl',
            'animate-in fade-in-0 zoom-in-95 duration-100'
          )}
        >
          <input
            ref={inputRef}
            type="text"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleInputKeyDown}
            placeholder="Type tag name..."
            className={cn(
              'w-full px-2 py-1.5 text-sm',
              'bg-muted border border-border rounded-md',
              'text-foreground placeholder:text-muted-foreground',
              'focus:outline-none focus:ring-2 focus:ring-primary'
            )}
          />

          {/* Suggestions list */}
          {(filteredSuggestions.length > 0 || isNewTag) && (
            <div className="mt-2 max-h-32 overflow-y-auto">
              {/* New tag option */}
              {isNewTag && (
                <button
                  type="button"
                  onClick={() => addTag(inputValue)}
                  className={cn(
                    'w-full px-2 py-1.5 text-left text-sm rounded',
                    'hover:bg-muted transition-colors',
                    'flex items-center justify-between'
                  )}
                >
                  <span className="text-foreground">Create "{inputValue.trim()}"</span>
                  <span className="px-1.5 py-0.5 text-[10px] bg-emerald-600/30 text-emerald-300 rounded">
                    NEW
                  </span>
                </button>
              )}

              {/* Existing tag suggestions */}
              {filteredSuggestions.map((suggestion) => (
                <button
                  key={suggestion}
                  type="button"
                  onClick={() => addTag(suggestion)}
                  className={cn(
                    'w-full px-2 py-1.5 text-left text-sm rounded',
                    'text-muted-foreground hover:bg-muted hover:text-foreground transition-colors'
                  )}
                >
                  {suggestion}
                </button>
              ))}
            </div>
          )}

          {/* Help text */}
          <p className="mt-2 text-[10px] text-muted-foreground">
            Press Enter to add, Escape to close
          </p>
        </div>
      )}
    </div>
  )
}
