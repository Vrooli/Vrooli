/**
 * CategoryPathEditor - Dynamic hierarchical path editor with suggestions.
 *
 * Used for editing mode paths on prompts.
 * Each level shows suggestions from existing prompts and allows custom values.
 * Custom values show a "NEW" badge to indicate a new category will be created.
 *
 * Adapted from agent-inbox/ui/src/components/shared/CategoryPathEditor.tsx
 */

import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import { Plus, X, ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'

interface CategoryPathEditorProps {
  value: string[]
  onChange: (value: string[]) => void
  getSuggestionsAtLevel: (level: number, parentPath: string[]) => string[]
  label?: string
  placeholder?: string
  disabled?: boolean
  required?: boolean
  error?: string
}

interface ComboboxProps {
  value: string
  onChange: (value: string) => void
  suggestions: string[]
  placeholder?: string
  disabled?: boolean
  onDelete?: () => void
  showDelete?: boolean
  level: number
}

function Combobox({
  value,
  onChange,
  suggestions,
  placeholder,
  disabled,
  onDelete,
  showDelete,
  level,
}: ComboboxProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [inputValue, setInputValue] = useState(value)
  const containerRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  // Sync inputValue with value prop
  useEffect(() => {
    setInputValue(value)
  }, [value])

  const handleClickOutside = useCallback(
    (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false)
        // Commit value on blur
        if (inputValue !== value) {
          onChange(inputValue)
        }
      }
    },
    [inputValue, value, onChange]
  )

  const handleEscape = useCallback(
    (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setIsOpen(false)
        setInputValue(value) // Revert to original
      }
    },
    [value]
  )

  useEffect(() => {
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
      document.addEventListener('keydown', handleEscape)
    }
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [isOpen, handleClickOutside, handleEscape])

  // Filter suggestions based on input
  const filteredSuggestions = useMemo(() => {
    if (!inputValue.trim()) return suggestions
    const lower = inputValue.toLowerCase()
    return suggestions.filter((s) => s.toLowerCase().includes(lower))
  }, [suggestions, inputValue])

  // Check if current value is custom (not in suggestions)
  const isCustomValue = useMemo(() => {
    return inputValue.trim() !== '' && !suggestions.includes(inputValue)
  }, [inputValue, suggestions])

  const handleInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setInputValue(e.target.value)
    setIsOpen(true)
  }, [])

  const handleSelect = useCallback(
    (suggestion: string) => {
      setInputValue(suggestion)
      onChange(suggestion)
      setIsOpen(false)
    },
    [onChange]
  )

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        e.preventDefault()
        if (inputValue.trim()) {
          onChange(inputValue.trim())
          setIsOpen(false)
        }
      } else if (e.key === 'ArrowDown') {
        e.preventDefault()
        setIsOpen(true)
      }
    },
    [inputValue, onChange]
  )

  return (
    <div ref={containerRef} className="relative flex items-center gap-1">
      <div className="relative flex-1">
        <div className="relative">
          <input
            ref={inputRef}
            type="text"
            value={inputValue}
            onChange={handleInputChange}
            onFocus={() => setIsOpen(true)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder || `Level ${level + 1}`}
            disabled={disabled}
            className={cn(
              'w-full px-3 py-2 pr-8 bg-muted border border-border rounded-lg text-foreground text-sm',
              'placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary',
              disabled && 'opacity-50 cursor-not-allowed'
            )}
          />
          <button
            type="button"
            onClick={() => !disabled && setIsOpen(!isOpen)}
            disabled={disabled}
            className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
          >
            <ChevronDown className={cn('h-4 w-4 transition-transform', isOpen && 'rotate-180')} />
          </button>
        </div>

        {/* Custom value badge */}
        {isCustomValue && inputValue.trim() && (
          <span className="absolute -right-14 top-1/2 -translate-y-1/2 px-1.5 py-0.5 text-[10px] font-medium bg-emerald-600/30 text-emerald-300 rounded">
            NEW
          </span>
        )}

        {/* Dropdown */}
        {isOpen && (filteredSuggestions.length > 0 || inputValue.trim()) && (
          <div
            className={cn(
              'absolute z-50 mt-1 w-full max-h-48 overflow-y-auto',
              'bg-popover border border-border rounded-lg shadow-xl',
              'animate-in fade-in-0 zoom-in-95 duration-100'
            )}
          >
            {filteredSuggestions.length > 0 ? (
              filteredSuggestions.map((suggestion) => (
                <button
                  key={suggestion}
                  type="button"
                  onClick={() => handleSelect(suggestion)}
                  className={cn(
                    'w-full px-3 py-2 text-left text-sm transition-colors',
                    'hover:bg-muted focus:bg-muted focus:outline-none',
                    suggestion === value ? 'text-primary bg-primary/20' : 'text-muted-foreground'
                  )}
                >
                  {suggestion}
                </button>
              ))
            ) : (
              <div className="px-3 py-2 text-sm text-muted-foreground">
                Press Enter to create "{inputValue}"
              </div>
            )}
          </div>
        )}
      </div>

      {/* Delete button */}
      {showDelete && (
        <button
          type="button"
          onClick={onDelete}
          disabled={disabled}
          className={cn(
            'p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-destructive transition-colors',
            disabled && 'opacity-50 cursor-not-allowed'
          )}
          title="Remove level"
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  )
}

export function CategoryPathEditor({
  value,
  onChange,
  getSuggestionsAtLevel,
  label = 'Modes',
  placeholder,
  disabled,
  required,
  error,
}: CategoryPathEditorProps) {
  // Handle value change at a specific level
  const handleLevelChange = useCallback(
    (level: number, newValue: string) => {
      const newPath = [...value]
      if (newValue.trim()) {
        newPath[level] = newValue.trim()
        // Truncate any levels after this one when changing
        onChange(newPath.slice(0, level + 1))
      } else {
        // Clear this level and all after
        onChange(newPath.slice(0, level))
      }
    },
    [value, onChange]
  )

  // Handle delete at a specific level
  const handleDelete = useCallback(
    (level: number) => {
      onChange(value.slice(0, level))
    },
    [value, onChange]
  )

  // Add a new level
  const handleAddLevel = useCallback(() => {
    onChange([...value, ''])
  }, [value, onChange])

  // Get suggestions for each level
  const getSuggestions = useCallback(
    (level: number): string[] => {
      const parentPath = value.slice(0, level)
      return getSuggestionsAtLevel(level, parentPath)
    },
    [value, getSuggestionsAtLevel]
  )

  // Determine how many levels to show
  // Show existing levels + 1 empty level if the last level has a value
  const levels = useMemo(() => {
    if (value.length === 0) return [0] // At least one level
    return value.map((_, i) => i)
  }, [value])

  // Path preview text
  const pathPreview = value.filter(Boolean).join(' / ') || '(none)'

  return (
    <div className="space-y-2">
      {/* Label */}
      <label className="block text-sm font-medium text-muted-foreground">
        {label}
        {required && <span className="text-red-400 ml-1">*</span>}
      </label>

      {/* Combobox levels */}
      <div className="space-y-2">
        {levels.map((level) => (
          <Combobox
            key={level}
            level={level}
            value={value[level] || ''}
            onChange={(v) => handleLevelChange(level, v)}
            suggestions={getSuggestions(level)}
            placeholder={
              level === 0 ? placeholder || 'Select or type mode...' : 'Subcategory (optional)'
            }
            disabled={disabled}
            onDelete={() => handleDelete(level)}
            showDelete={level > 0 || (level === 0 && !required)}
          />
        ))}
      </div>

      {/* Add level button */}
      {value.length > 0 && value[value.length - 1]?.trim() && (
        <button
          type="button"
          onClick={handleAddLevel}
          disabled={disabled}
          className={cn(
            'flex items-center gap-1.5 px-2 py-1.5 text-xs text-muted-foreground hover:text-foreground',
            'hover:bg-muted rounded-lg transition-colors',
            disabled && 'opacity-50 cursor-not-allowed'
          )}
        >
          <Plus className="h-3 w-3" />
          Add level
        </button>
      )}

      {/* Path preview */}
      <p className="text-xs text-muted-foreground">Path: {pathPreview}</p>

      {/* Error message */}
      {error && <p className="text-xs text-red-400">{error}</p>}
    </div>
  )
}
