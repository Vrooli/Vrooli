/**
 * ScopeSelector - Select a default scope skill for steer skills.
 *
 * Displays a dropdown of available scope skills (skills with mode "scope").
 * Used in SkillEditorPanel to set the defaultScope for steer skills.
 */

import { useState, useMemo, useRef, useEffect, useCallback, useLayoutEffect } from 'react'
import { ChevronDown, Shield, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Skill } from '@/types'

interface ScopeSelectorProps {
  /** Currently selected scope skill ID */
  value: string
  /** Callback when scope changes */
  onChange: (scope: string) => void
  /** All available skills (filtered to scopes internally) */
  allSkills: Skill[]
  /** Whether the selector is disabled */
  disabled?: boolean
  /** Additional class names */
  className?: string
}

/**
 * Scope selector component with dropdown picker.
 */
export function ScopeSelector({
  value,
  onChange,
  allSkills,
  disabled,
  className,
}: ScopeSelectorProps) {
  const [isOpen, setIsOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const [position, setPosition] = useState<{ top: number; left: number; width: number }>({ top: 0, left: 0, width: 256 })

  // Filter to only scope skills
  const scopeSkills = useMemo(() => {
    return allSkills.filter((skill) => skill.modes.includes('scope'))
  }, [allSkills])

  // Get currently selected scope skill
  const selectedScope = useMemo(() => {
    if (!value) return null
    return scopeSkills.find((s) => s.id === value) ?? null
  }, [value, scopeSkills])

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

  useLayoutEffect(() => {
    if (!isOpen || !triggerRef.current) return

    const trigger = triggerRef.current.getBoundingClientRect()
    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight
    const width = viewportWidth < 640 ? viewportWidth - 16 : Math.min(320, viewportWidth - 16)
    const estimatedHeight = Math.min(dropdownRef.current?.scrollHeight ?? 320, viewportHeight - 16)

    let left = trigger.left
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
  }, [isOpen, scopeSkills.length])

  // Don't render if no scope skills available
  if (scopeSkills.length === 0) {
    return null
  }

  return (
    <div ref={containerRef} className={cn('relative', className)}>
      {/* Trigger button */}
      <button
        ref={triggerRef}
        type="button"
        onClick={() => !disabled && setIsOpen(!isOpen)}
        disabled={disabled}
        className={cn(
          'flex items-center gap-1 sm:gap-2 px-1.5 sm:px-3 py-1.5 rounded-md text-sm',
          'border border-white/10 bg-slate-800 hover:bg-slate-700 transition-colors',
          disabled && 'opacity-50 cursor-not-allowed',
          selectedScope && 'border-indigo-500/30'
        )}
        title="Select default scope"
        aria-label={`Select default scope${selectedScope ? `, currently ${selectedScope.name}` : ''}`}
      >
        <Shield className="h-4 w-4 text-slate-400" />
        <span className="hidden sm:inline text-slate-300">
          {selectedScope ? selectedScope.name : 'No scope'}
        </span>
        <ChevronDown className="hidden sm:inline h-3 w-3 text-slate-500" />
      </button>

      {/* Dropdown */}
      {isOpen && (
        <div
          ref={dropdownRef}
          style={{
            position: 'fixed',
            top: position.top,
            left: position.left,
            width: position.width,
            maxWidth: 'calc(100vw - 16px)',
            maxHeight: 'calc(100vh - 16px)',
          }}
          className={cn(
            'z-50 py-1 overflow-y-auto',
            'bg-slate-900 border border-white/10 rounded-lg shadow-xl',
            'animate-in fade-in-0 zoom-in-95 duration-100'
          )}
        >
          {/* None option */}
          <button
            type="button"
            onClick={() => {
              onChange('')
              setIsOpen(false)
            }}
            className={cn(
              'w-full px-3 py-2 text-left text-sm flex items-center gap-2',
              'hover:bg-white/5 transition-colors',
              !value && 'bg-indigo-600/20 text-indigo-300'
            )}
          >
            <X className="h-4 w-4 text-slate-500" />
            <span>No default scope</span>
          </button>

          <div className="h-px bg-white/10 my-1" />

          {/* Scope options */}
          {scopeSkills.map((scope) => (
            <button
              key={scope.id}
              type="button"
              onClick={() => {
                onChange(scope.id)
                setIsOpen(false)
              }}
              className={cn(
                'w-full px-3 py-2 text-left text-sm flex flex-col gap-0.5',
                'hover:bg-white/5 transition-colors',
                value === scope.id && 'bg-indigo-600/20 text-indigo-300'
              )}
            >
              <span className="font-medium">{scope.name}</span>
              {scope.description && (
                <span className="text-xs text-slate-500 line-clamp-1">
                  {scope.description}
                </span>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
