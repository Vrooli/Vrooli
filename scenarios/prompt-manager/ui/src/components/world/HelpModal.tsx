/**
 * HelpModal - Modal dialog explaining the skill tree environment.
 *
 * Provides information about:
 * - What avatars represent
 * - How to interact with the 3D environment
 * - Camera controls explanation
 */

import { useEffect, useRef, useCallback } from 'react'
import { X, User, Eye, Map, MousePointer, Move3D } from 'lucide-react'
import { cn } from '@/lib/utils'

interface HelpModalProps {
  isOpen: boolean
  onClose: () => void
}

/**
 * Help modal component for the skill tree environment.
 */
export function HelpModal({ isOpen, onClose }: HelpModalProps) {
  const dialogRef = useRef<HTMLDivElement>(null)

  // Handle escape key
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose()
      }
    },
    [onClose]
  )

  // Handle click outside
  const handleClickOutside = useCallback(
    (event: MouseEvent) => {
      if (dialogRef.current && !dialogRef.current.contains(event.target as Node)) {
        onClose()
      }
    },
    [onClose]
  )

  // Set up event listeners
  useEffect(() => {
    if (isOpen) {
      document.addEventListener('keydown', handleKeyDown)
      document.addEventListener('mousedown', handleClickOutside)

      // Prevent body scroll
      document.body.style.overflow = 'hidden'
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.removeEventListener('mousedown', handleClickOutside)
      document.body.style.overflow = ''
    }
  }, [isOpen, handleKeyDown, handleClickOutside])

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" />

      {/* Dialog */}
      <div
        ref={dialogRef}
        className={cn(
          'relative w-full max-w-lg mx-4 p-6',
          'bg-slate-900 border border-white/10 rounded-xl shadow-2xl',
          'animate-in fade-in-0 zoom-in-95 duration-150',
          'max-h-[85vh] overflow-y-auto'
        )}
        role="dialog"
        aria-modal="true"
        aria-labelledby="help-dialog-title"
      >
        {/* Close button */}
        <button
          type="button"
          onClick={onClose}
          className={cn(
            'absolute top-4 right-4 p-1 rounded',
            'text-slate-400 hover:text-white hover:bg-white/10 transition-colors'
          )}
          aria-label="Close dialog"
        >
          <X className="h-5 w-5" />
        </button>

        {/* Title */}
        <h2
          id="help-dialog-title"
          className="text-xl font-semibold text-white mb-6"
        >
          Avatar Environment
        </h2>

        {/* Content sections */}
        <div className="space-y-6">
          {/* What are avatars */}
          <section>
            <h3 className="text-sm font-medium text-indigo-400 mb-2">
              What are Avatars?
            </h3>
            <p className="text-sm text-slate-300 leading-relaxed">
              Avatars are visual characters that can have prompts (skills) assigned to them.
              Each avatar can hold a collection of related prompts, making it easy to organize
              and manage your prompt library.
            </p>
          </section>

          {/* How to interact */}
          <section>
            <h3 className="text-sm font-medium text-indigo-400 mb-3">
              Interactions
            </h3>
            <div className="space-y-3">
              <HelpItem
                icon={<MousePointer className="h-4 w-4" />}
                title="Click an Avatar"
                description="Opens the avatar's menu where you can customize it, assign skills, duplicate, or delete it."
              />
              <HelpItem
                icon={<Move3D className="h-4 w-4" />}
                title="Drag to Orbit"
                description="Click and drag anywhere in the environment to rotate the camera around the scene."
              />
            </div>
          </section>

          {/* Camera modes */}
          <section>
            <h3 className="text-sm font-medium text-indigo-400 mb-3">
              Camera Views
            </h3>
            <p className="text-sm text-slate-400 mb-3">
              Use the camera button in the top-right to cycle through views:
            </p>
            <div className="space-y-3">
              <HelpItem
                icon={<User className="h-4 w-4" />}
                title="Focus on Avatar"
                description="Zooms in on the selected avatar (or the first one if none selected)."
              />
              <HelpItem
                icon={<Eye className="h-4 w-4" />}
                title="Default View"
                description="Standard perspective view showing all avatars in the environment."
              />
              <HelpItem
                icon={<Map className="h-4 w-4" />}
                title="Aerial View"
                description="Top-down view of the entire environment, useful for seeing all avatars at once."
              />
            </div>
          </section>
        </div>

        {/* Close action */}
        <div className="mt-6 pt-4 border-t border-white/10">
          <button
            type="button"
            onClick={onClose}
            className={cn(
              'w-full px-4 py-2 text-sm font-medium rounded-lg',
              'bg-indigo-600 text-white hover:bg-indigo-500',
              'transition-colors'
            )}
          >
            Got it
          </button>
        </div>
      </div>
    </div>
  )
}

/**
 * Individual help item with icon.
 */
interface HelpItemProps {
  icon: React.ReactNode
  title: string
  description: string
}

function HelpItem({ icon, title, description }: HelpItemProps) {
  return (
    <div className="flex gap-3">
      <div className="p-2 rounded-lg bg-slate-800 text-slate-400 shrink-0">
        {icon}
      </div>
      <div>
        <p className="text-sm font-medium text-white">{title}</p>
        <p className="text-xs text-slate-400 mt-0.5">{description}</p>
      </div>
    </div>
  )
}
