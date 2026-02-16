/**
 * TagManagementModal - Full-screen mobile modal for tag management.
 *
 * Features:
 * - Add tags with autocomplete
 * - Remove tags
 * - Inline rename (tap tag text to edit)
 * - Drag reorder via HTML5 DnD
 * - Auto-save: changes apply immediately via onChange
 */

import { useState, useRef, useEffect, useCallback } from 'react'
import { GripVertical, Plus, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useModalBehavior } from '@/hooks/useModalBehavior'

interface TagManagementModalProps {
  /** Whether the modal is visible (default: true, for backward compat with conditional rendering) */
  isOpen?: boolean
  tags: string[]
  onChange: (tags: string[]) => void
  availableTags?: string[]
  onClose: () => void
}

export function TagManagementModal({
  isOpen = true,
  tags,
  onChange,
  availableTags = [],
  onClose,
}: TagManagementModalProps) {
  const [inputValue, setInputValue] = useState('')
  const [editingIndex, setEditingIndex] = useState<number | null>(null)
  const [editValue, setEditValue] = useState('')
  const [dragIndex, setDragIndex] = useState<number | null>(null)
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const editInputRef = useRef<HTMLInputElement>(null)
  const modalRef = useRef<HTMLDivElement>(null)

  // Shared dismiss behavior: Escape closes (unless editing), scroll lock
  useModalBehavior({
    isOpen,
    onClose,
    ref: modalRef,
    preventBodyScroll: true,
    disableCloseOnEsc: editingIndex !== null,
    disableCloseOnOutsideClick: true,
  })

  // When editing and Escape is pressed, cancel editing (not handled by useModalBehavior
  // since disableCloseOnEsc is true during editing)
  useEffect(() => {
    if (!isOpen || editingIndex === null) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setEditingIndex(null)
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, editingIndex])

  // Focus add input on mount
  useEffect(() => {
    if (isOpen) {
      setTimeout(() => inputRef.current?.focus(), 100)
    }
  }, [isOpen])

  // Focus edit input when editing starts
  useEffect(() => {
    if (editingIndex !== null) {
      setTimeout(() => editInputRef.current?.focus(), 0)
    }
  }, [editingIndex])

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
    (index: number) => {
      onChange(tags.filter((_, i) => i !== index))
    },
    [tags, onChange]
  )

  const startRename = useCallback(
    (index: number) => {
      setEditingIndex(index)
      setEditValue(tags[index] ?? '')
    },
    [tags]
  )

  const commitRename = useCallback(() => {
    if (editingIndex === null) return
    const trimmed = editValue.trim()
    if (trimmed && trimmed !== tags[editingIndex]) {
      // Ensure no duplicate
      if (!tags.includes(trimmed)) {
        const updated = [...tags]
        updated[editingIndex] = trimmed
        onChange(updated)
      }
    }
    setEditingIndex(null)
  }, [editingIndex, editValue, tags, onChange])

  const handleEditKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        e.preventDefault()
        commitRename()
      } else if (e.key === 'Escape') {
        setEditingIndex(null)
      }
    },
    [commitRename]
  )

  const handleAddKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' && inputValue.trim()) {
        e.preventDefault()
        addTag(inputValue)
      }
    },
    [inputValue, addTag]
  )

  // Drag and drop
  const handleDragStart = useCallback((index: number) => {
    setDragIndex(index)
  }, [])

  const handleDragOver = useCallback(
    (e: React.DragEvent, index: number) => {
      e.preventDefault()
      if (dragIndex !== null && dragIndex !== index) {
        setDragOverIndex(index)
      }
    },
    [dragIndex]
  )

  const handleDrop = useCallback(
    (index: number) => {
      if (dragIndex === null || dragIndex === index) {
        setDragIndex(null)
        setDragOverIndex(null)
        return
      }
      const updated = [...tags]
      const [moved] = updated.splice(dragIndex, 1)
      if (moved !== undefined) {
        updated.splice(index, 0, moved)
        onChange(updated)
      }
      setDragIndex(null)
      setDragOverIndex(null)
    },
    [dragIndex, tags, onChange]
  )

  const handleDragEnd = useCallback(() => {
    setDragIndex(null)
    setDragOverIndex(null)
  }, [])

  // Autocomplete suggestions
  const lower = inputValue.toLowerCase()
  const filteredSuggestions = availableTags
    .filter((t) => !tags.includes(t))
    .filter((t) => !inputValue || t.toLowerCase().includes(lower))
    .slice(0, 8)

  const trimmedInput = inputValue.trim()
  const isNewTag = trimmedInput && !availableTags.includes(trimmedInput) && !tags.includes(trimmedInput)

  if (!isOpen) return null

  return (
    <div
      ref={modalRef}
      className="fixed inset-0 z-50 bg-background flex flex-col"
      role="dialog"
      aria-modal="true"
      aria-label="Manage Tags"
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-border">
        <h2 className="text-sm font-semibold">Manage Tags</h2>
        <button
          type="button"
          onClick={onClose}
          className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
          aria-label="Close"
        >
          <X className="h-5 w-5" />
        </button>
      </div>

      {/* Add tag input */}
      <div className="px-4 py-3 border-b border-border">
        <div className="flex items-center gap-2">
          <input
            ref={inputRef}
            type="text"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleAddKeyDown}
            placeholder="Add a tag..."
            className={cn(
              'flex-1 px-3 py-2 text-sm',
              'bg-muted border border-border rounded-md',
              'text-foreground placeholder:text-muted-foreground',
              'focus:outline-none focus:ring-2 focus:ring-primary'
            )}
          />
          <button
            type="button"
            onClick={() => addTag(inputValue)}
            disabled={!inputValue.trim()}
            className={cn(
              'p-2 rounded-md transition-colors',
              inputValue.trim()
                ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                : 'bg-muted text-muted-foreground cursor-not-allowed'
            )}
            aria-label="Add tag"
          >
            <Plus className="h-4 w-4" />
          </button>
        </div>

        {/* Suggestions */}
        {inputValue && (filteredSuggestions.length > 0 || isNewTag) && (
          <div className="mt-2 max-h-40 overflow-y-auto rounded-md border border-border bg-popover">
            {isNewTag && (
              <button
                type="button"
                onClick={() => addTag(inputValue)}
                className="w-full px-3 py-2 text-left text-sm hover:bg-muted transition-colors flex items-center justify-between"
              >
                <span className="text-foreground">Create &ldquo;{trimmedInput}&rdquo;</span>
                <span className="px-1.5 py-0.5 text-[10px] bg-emerald-600/30 text-emerald-300 rounded">
                  NEW
                </span>
              </button>
            )}
            {filteredSuggestions.map((suggestion) => (
              <button
                key={suggestion}
                type="button"
                onClick={() => addTag(suggestion)}
                className="w-full px-3 py-2 text-left text-sm text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
              >
                {suggestion}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Tag list */}
      <div className="flex-1 overflow-y-auto">
        {tags.length === 0 ? (
          <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
            No tags yet. Add one above.
          </div>
        ) : (
          tags.map((tag, index) => (
            <div
              key={`${tag}-${index}`}
              draggable
              onDragStart={() => handleDragStart(index)}
              onDragOver={(e) => handleDragOver(e, index)}
              onDrop={() => handleDrop(index)}
              onDragEnd={handleDragEnd}
              className={cn(
                'flex items-center gap-2 px-4 py-2.5 border-b border-border',
                'transition-colors',
                dragIndex === index && 'opacity-50',
                dragOverIndex === index && 'bg-primary/10'
              )}
            >
              {/* Drag handle */}
              <div className="cursor-grab active:cursor-grabbing text-muted-foreground touch-none">
                <GripVertical className="h-4 w-4" />
              </div>

              {/* Tag text / edit input */}
              {editingIndex === index ? (
                <input
                  ref={editInputRef}
                  type="text"
                  value={editValue}
                  onChange={(e) => setEditValue(e.target.value)}
                  onKeyDown={handleEditKeyDown}
                  onBlur={commitRename}
                  className={cn(
                    'flex-1 px-2 py-1 text-sm',
                    'bg-muted border border-primary rounded-md',
                    'text-foreground',
                    'focus:outline-none focus:ring-2 focus:ring-primary'
                  )}
                />
              ) : (
                <button
                  type="button"
                  onClick={() => startRename(index)}
                  className="flex-1 text-left text-sm text-foreground hover:text-primary transition-colors truncate"
                >
                  {tag}
                </button>
              )}

              {/* Delete button */}
              <button
                type="button"
                onClick={() => removeTag(index)}
                className="p-1 rounded hover:bg-destructive/20 text-muted-foreground hover:text-destructive transition-colors"
                aria-label={`Remove ${tag}`}
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          ))
        )}
      </div>

      {/* Footer hint */}
      <div className="px-4 py-2 border-t border-border">
        <p className="text-[10px] text-muted-foreground text-center">
          Tap tag name to rename. Drag to reorder.
        </p>
      </div>
    </div>
  )
}
