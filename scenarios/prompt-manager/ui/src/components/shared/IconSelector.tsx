/**
 * IconSelector - Icon picker for prompts.
 *
 * Displays a grid of available icons with search functionality.
 * Adapted from agent-inbox with prompt-relevant icon set.
 */

import { useState, useMemo, useRef, useEffect, useCallback } from 'react'
import { Search } from 'lucide-react'
import { cn } from '@/lib/utils'
import { ICONS, getIcon } from '@/lib/icons'
import { Skeleton } from '@/components/ui/skeleton'

interface IconSelectorProps {
  value: string
  onChange: (value: string) => void
  disabled?: boolean
  isLoading?: boolean
  className?: string
}

/**
 * Icon selector component with dropdown picker.
 */
export function IconSelector({ value, onChange, disabled, isLoading, className }: IconSelectorProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [search, setSearch] = useState('')
  const containerRef = useRef<HTMLDivElement>(null)

  // Filter icons by search
  const filteredIcons = useMemo(() => {
    if (!search.trim()) return ICONS
    const lowerSearch = search.toLowerCase()
    return ICONS.filter((i) => i.name.toLowerCase().includes(lowerSearch))
  }, [search])

  // Get current icon
  const CurrentIcon = getIcon(value)

  // Handle click outside
  const handleClickOutside = useCallback((event: MouseEvent) => {
    if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
      setIsOpen(false)
    }
  }, [])

  // Handle escape key
  const handleEscape = useCallback((event: KeyboardEvent) => {
    if (event.key === 'Escape') {
      setIsOpen(false)
    }
  }, [])

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

  // Reset search when opening
  useEffect(() => {
    if (isOpen) {
      setSearch('')
    }
  }, [isOpen])

  if (isLoading) {
    return <Skeleton className={cn('w-10 h-10 rounded-lg', className)} />
  }

  return (
    <div ref={containerRef} className={cn('relative', className)}>
      {/* Trigger button */}
      <button
        type="button"
        onClick={() => !disabled && setIsOpen(!isOpen)}
        disabled={disabled}
        className={cn(
          'flex items-center justify-center w-10 h-10 rounded-lg border border-white/10',
          'bg-slate-800 hover:bg-slate-700 transition-colors',
          disabled && 'opacity-50 cursor-not-allowed'
        )}
        title="Select icon"
      >
        <CurrentIcon className="h-5 w-5 text-slate-300" />
      </button>

      {/* Dropdown */}
      {isOpen && (
        <div
          className={cn(
            'absolute z-50 mt-1 p-2 w-64',
            'bg-slate-900 border border-white/10 rounded-lg shadow-xl',
            'animate-in fade-in-0 zoom-in-95 duration-100'
          )}
        >
          {/* Search input */}
          <div className="relative mb-2">
            <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search icons..."
              className={cn(
                'w-full pl-8 pr-3 py-1.5 text-sm',
                'bg-slate-800 border border-white/10 rounded-md',
                'text-white placeholder:text-slate-500',
                'focus:outline-none focus:ring-2 focus:ring-indigo-500'
              )}
              autoFocus
            />
          </div>

          {/* Icon grid */}
          <div className="grid grid-cols-6 gap-1 max-h-48 overflow-y-auto">
            {filteredIcons.map(({ name, icon: Icon }) => (
              <button
                key={name}
                type="button"
                onClick={() => {
                  onChange(name)
                  setIsOpen(false)
                }}
                className={cn(
                  'flex items-center justify-center w-9 h-9 rounded',
                  'hover:bg-white/10 transition-colors',
                  value === name ? 'bg-indigo-600/30 text-indigo-300' : 'text-slate-400'
                )}
                title={name}
              >
                <Icon className="h-4 w-4" />
              </button>
            ))}
          </div>

          {filteredIcons.length === 0 && (
            <p className="text-sm text-slate-500 text-center py-2">No icons found</p>
          )}

          {/* Clear button */}
          {value && (
            <button
              type="button"
              onClick={() => {
                onChange('')
                setIsOpen(false)
              }}
              className="w-full mt-2 px-2 py-1 text-xs text-slate-400 hover:text-white hover:bg-white/5 rounded transition-colors"
            >
              Clear icon
            </button>
          )}
        </div>
      )}
    </div>
  )
}
