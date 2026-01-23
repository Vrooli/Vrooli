/**
 * AvatarCustomizeModal - Modal for customizing avatar appearance.
 *
 * Features:
 * - Name editing
 * - Color pickers for body, head, and accent colors
 * - Live preview
 */

import { useState, useEffect, useRef, useCallback } from 'react'
import { X, Palette } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Avatar, UpdateAvatarRequest } from '@/types/avatar'
import { DEFAULT_AVATAR_COLORS } from '@/types/avatar'

interface AvatarCustomizeModalProps {
  isOpen: boolean
  onClose: () => void
  avatar: Avatar | null
  onSave: (updates: UpdateAvatarRequest) => Promise<void>
  isLoading?: boolean
}

/**
 * Color preset palette for quick selection.
 */
const COLOR_PRESETS = [
  '#6366f1', // indigo
  '#8b5cf6', // violet
  '#ec4899', // pink
  '#ef4444', // red
  '#f97316', // orange
  '#eab308', // yellow
  '#22c55e', // green
  '#06b6d4', // cyan
  '#3b82f6', // blue
  '#64748b', // slate
]

/**
 * Avatar customization modal component.
 */
export function AvatarCustomizeModal({
  isOpen,
  onClose,
  avatar,
  onSave,
  isLoading = false,
}: AvatarCustomizeModalProps) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const nameInputRef = useRef<HTMLInputElement>(null)

  // Form state
  const [name, setName] = useState('')
  const [bodyColor, setBodyColor] = useState<string>(DEFAULT_AVATAR_COLORS.bodyColor)
  const [headColor, setHeadColor] = useState<string>(DEFAULT_AVATAR_COLORS.headColor)
  const [accentColor, setAccentColor] = useState<string>(DEFAULT_AVATAR_COLORS.accentColor)

  // Initialize form when avatar changes
  useEffect(() => {
    if (avatar) {
      setName(avatar.name)
      setBodyColor(avatar.bodyColor)
      setHeadColor(avatar.headColor)
      setAccentColor(avatar.accentColor)
    }
  }, [avatar])

  // Focus name input when opened
  useEffect(() => {
    if (isOpen) {
      setTimeout(() => nameInputRef.current?.focus(), 0)
    }
  }, [isOpen])

  // Handle escape key
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !isLoading) {
        onClose()
      }
    },
    [onClose, isLoading]
  )

  // Handle click outside
  const handleClickOutside = useCallback(
    (event: MouseEvent) => {
      if (
        dialogRef.current &&
        !dialogRef.current.contains(event.target as Node) &&
        !isLoading
      ) {
        onClose()
      }
    },
    [onClose, isLoading]
  )

  // Set up event listeners
  useEffect(() => {
    if (isOpen) {
      document.addEventListener('keydown', handleKeyDown)
      document.addEventListener('mousedown', handleClickOutside)
      document.body.style.overflow = 'hidden'
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.removeEventListener('mousedown', handleClickOutside)
      document.body.style.overflow = ''
    }
  }, [isOpen, handleKeyDown, handleClickOutside])

  // Handle save
  const handleSave = async () => {
    await onSave({
      name,
      bodyColor,
      headColor,
      accentColor,
    })
    onClose()
  }

  if (!isOpen || !avatar) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" />

      {/* Dialog */}
      <div
        ref={dialogRef}
        className={cn(
          'relative w-full max-w-lg mx-4 p-6',
          'bg-card border border-border rounded-xl shadow-2xl',
          'animate-in fade-in-0 zoom-in-95 duration-150'
        )}
        role="dialog"
        aria-modal="true"
        aria-labelledby="customize-dialog-title"
      >
        {/* Close button */}
        <button
          type="button"
          onClick={onClose}
          disabled={isLoading}
          className={cn(
            'absolute top-4 right-4 p-1 rounded',
            'text-muted-foreground hover:text-foreground hover:bg-muted transition-colors',
            isLoading && 'opacity-50 cursor-not-allowed'
          )}
          aria-label="Close dialog"
        >
          <X className="h-5 w-5" />
        </button>

        {/* Header */}
        <div className="flex items-center gap-3 mb-6">
          <div className="w-10 h-10 rounded-full bg-primary/20 flex items-center justify-center">
            <Palette className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h2 id="customize-dialog-title" className="text-lg font-semibold text-foreground">
              Customize Avatar
            </h2>
            <p className="text-sm text-muted-foreground">Personalize your avatar's appearance</p>
          </div>
        </div>

        {/* Preview */}
        <div className="flex justify-center mb-6">
          <div className="relative">
            {/* Body */}
            <div
              className="w-20 h-28 rounded-full flex items-start justify-center pt-4"
              style={{ backgroundColor: bodyColor }}
            >
              {/* Head */}
              <div
                className="w-12 h-12 rounded-full flex items-center justify-center"
                style={{ backgroundColor: headColor }}
              >
                {/* Accent (antenna) */}
                <div
                  className="absolute -top-2 w-3 h-3 rounded-full"
                  style={{ backgroundColor: accentColor }}
                />
              </div>
            </div>
            {/* Arms */}
            <div
              className="absolute top-8 -left-3 w-4 h-12 rounded-full"
              style={{ backgroundColor: bodyColor }}
            />
            <div
              className="absolute top-8 -right-3 w-4 h-12 rounded-full"
              style={{ backgroundColor: bodyColor }}
            />
          </div>
        </div>

        {/* Name input */}
        <div className="mb-4">
          <label htmlFor="avatar-name" className="block text-sm font-medium text-foreground mb-1">
            Name
          </label>
          <input
            ref={nameInputRef}
            id="avatar-name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className={cn(
              'w-full px-3 py-2 text-sm',
              'bg-muted border border-border rounded-lg',
              'text-foreground placeholder:text-muted-foreground',
              'focus:outline-none focus:ring-2 focus:ring-primary'
            )}
            placeholder="Enter avatar name"
          />
        </div>

        {/* Color pickers */}
        <div className="space-y-4">
          <ColorPicker
            label="Body Color"
            value={bodyColor}
            onChange={setBodyColor}
          />
          <ColorPicker
            label="Head Color"
            value={headColor}
            onChange={setHeadColor}
          />
          <ColorPicker
            label="Accent Color"
            value={accentColor}
            onChange={setAccentColor}
          />
        </div>

        {/* Actions */}
        <div className="flex gap-3 mt-6">
          <button
            type="button"
            onClick={onClose}
            disabled={isLoading}
            className={cn(
              'flex-1 px-4 py-2 text-sm font-medium rounded-lg',
              'bg-muted text-muted-foreground hover:bg-muted/80 hover:text-foreground',
              'border border-border transition-colors',
              isLoading && 'opacity-50 cursor-not-allowed'
            )}
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={() => void handleSave()}
            disabled={isLoading}
            className={cn(
              'flex-1 px-4 py-2 text-sm font-medium rounded-lg transition-colors',
              'bg-primary text-primary-foreground hover:bg-primary/90',
              isLoading && 'opacity-50 cursor-not-allowed'
            )}
          >
            {isLoading ? 'Saving...' : 'Save Changes'}
          </button>
        </div>
      </div>
    </div>
  )
}

/**
 * Color picker component with presets and custom input.
 */
interface ColorPickerProps {
  label: string
  value: string
  onChange: (color: string) => void
}

function ColorPicker({ label, value, onChange }: ColorPickerProps) {
  return (
    <div>
      <label className="block text-sm font-medium text-foreground mb-2">
        {label}
      </label>
      <div className="flex items-center gap-3">
        {/* Color presets */}
        <div className="flex gap-1 flex-wrap">
          {COLOR_PRESETS.map((color) => (
            <button
              key={color}
              type="button"
              onClick={() => onChange(color)}
              className={cn(
                'w-6 h-6 rounded-full border-2 transition-all',
                value === color ? 'border-foreground scale-110' : 'border-transparent hover:scale-105'
              )}
              style={{ backgroundColor: color }}
              title={color}
            />
          ))}
        </div>

        {/* Custom color input */}
        <div className="flex items-center gap-2 ml-auto">
          <input
            type="color"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            className="w-8 h-8 rounded cursor-pointer border border-border"
          />
          <input
            type="text"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            className={cn(
              'w-20 px-2 py-1 text-xs font-mono',
              'bg-muted border border-border rounded',
              'text-foreground',
              'focus:outline-none focus:ring-1 focus:ring-primary'
            )}
          />
        </div>
      </div>
    </div>
  )
}
