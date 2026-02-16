/**
 * ConfirmDialog - Modal dialog for confirming destructive actions.
 *
 * Used for:
 * - Confirming deletion
 * - Confirming discard of unsaved changes
 */

import { useEffect, useRef } from 'react'
import { AlertTriangle } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Dialog } from './Dialog'

interface ConfirmDialogProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => void
  title: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
  variant?: 'danger' | 'warning'
  isLoading?: boolean
}

/**
 * Confirmation dialog component.
 */
export function ConfirmDialog({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  variant = 'danger',
  isLoading = false,
}: ConfirmDialogProps) {
  const confirmButtonRef = useRef<HTMLButtonElement>(null)

  // Auto-focus confirm button when dialog opens
  useEffect(() => {
    if (isOpen) {
      setTimeout(() => confirmButtonRef.current?.focus(), 0)
    }
  }, [isOpen])

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      isLoading={isLoading}
      maxWidth="max-w-md"
      titleId="confirm-dialog-title"
      descriptionId="confirm-dialog-description"
    >
      {/* Icon */}
      <div
        className={cn(
          'w-12 h-12 mx-auto mb-4 rounded-full flex items-center justify-center',
          variant === 'danger' ? 'bg-red-500/20' : 'bg-amber-500/20'
        )}
      >
        <AlertTriangle
          className={cn(
            'h-6 w-6',
            variant === 'danger' ? 'text-red-400' : 'text-amber-400'
          )}
        />
      </div>

      {/* Title */}
      <h2
        id="confirm-dialog-title"
        className="text-lg font-semibold text-white text-center mb-2"
      >
        {title}
      </h2>

      {/* Message */}
      <p
        id="confirm-dialog-description"
        className="text-sm text-slate-400 text-center mb-6"
      >
        {message}
      </p>

      {/* Actions */}
      <div className="flex gap-3">
        <button
          type="button"
          onClick={onClose}
          disabled={isLoading}
          className={cn(
            'flex-1 px-4 py-2 text-sm font-medium rounded-lg',
            'bg-slate-800 text-slate-300 hover:bg-slate-700 hover:text-white',
            'border border-white/10 transition-colors',
            isLoading && 'opacity-50 cursor-not-allowed'
          )}
        >
          {cancelLabel}
        </button>
        <button
          ref={confirmButtonRef}
          type="button"
          onClick={onConfirm}
          disabled={isLoading}
          className={cn(
            'flex-1 px-4 py-2 text-sm font-medium rounded-lg transition-colors',
            variant === 'danger'
              ? 'bg-red-600 text-white hover:bg-red-500'
              : 'bg-amber-600 text-white hover:bg-amber-500',
            isLoading && 'opacity-50 cursor-not-allowed'
          )}
        >
          {isLoading ? 'Processing...' : confirmLabel}
        </button>
      </div>
    </Dialog>
  )
}
