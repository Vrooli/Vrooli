/**
 * OverlayModal - Shared modal chrome for overlay settings/help dialogs.
 *
 * Provides consistent modal behavior: backdrop, escape key, click-outside,
 * body scroll prevention, close button.
 */

import { useEffect, useRef, useCallback, type ReactNode } from 'react'
import { X } from 'lucide-react'
import { cn } from '@/lib/utils'

interface OverlayModalProps {
  isOpen: boolean
  onClose: () => void
  title: string
  children: ReactNode
  maxWidth?: string
  testId?: string
}

export function OverlayModal({ isOpen, onClose, title, children, maxWidth = 'max-w-lg', testId }: OverlayModalProps) {
  const dialogRef = useRef<HTMLDivElement>(null)

  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose()
      }
    },
    [onClose]
  )

  const handleClickOutside = useCallback(
    (event: MouseEvent) => {
      if (dialogRef.current && !dialogRef.current.contains(event.target as Node)) {
        onClose()
      }
    },
    [onClose]
  )

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

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" />

      <div
        ref={dialogRef}
        className={cn(
          'relative w-full mx-4 p-6',
          maxWidth,
          'bg-slate-900 border border-white/10 rounded-xl shadow-2xl',
          'animate-in fade-in-0 zoom-in-95 duration-150',
          'max-h-[85vh] overflow-y-auto'
        )}
        role="dialog"
        aria-modal="true"
        aria-labelledby="overlay-modal-title"
        data-testid={testId}
      >
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

        <h2
          id="overlay-modal-title"
          className="text-xl font-semibold text-white mb-6"
        >
          {title}
        </h2>

        {children}

        <div className="mt-6 pt-4 border-t border-white/10 text-center">
          <span className="text-xs text-slate-500">Press Esc to close</span>
        </div>
      </div>
    </div>
  )
}
