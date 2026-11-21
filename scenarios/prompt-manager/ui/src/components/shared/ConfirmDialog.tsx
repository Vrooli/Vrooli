/**
 * ConfirmDialog - Modal dialog for confirming destructive actions.
 *
 * Used for:
 * - Confirming deletion
 * - Confirming discard of unsaved changes
 */

import { useEffect, useId } from 'react'
import { AlertTriangle } from 'lucide-react'
import { OverlayActionBar, type OverlayActionList } from '@vrooli/react-component-library/OverlayActionBar/1.0.1'
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
  const confirmActionTestId = useId()

  // Auto-focus confirm button when dialog opens
  useEffect(() => {
    if (isOpen) {
      const timer = window.setTimeout(() => {
        document.querySelector<HTMLButtonElement>(`[data-testid="${confirmActionTestId}"]`)?.focus()
      }, 0)
      return () => window.clearTimeout(timer)
    }
  }, [confirmActionTestId, isOpen])

  const actions = [
    {
      id: 'confirm',
      label: isLoading ? 'Processing...' : confirmLabel,
      onSelect: onConfirm,
      disabled: isLoading,
      primary: true,
      testId: confirmActionTestId,
    },
    { id: 'cancel', label: cancelLabel, onSelect: onClose, disabled: isLoading },
  ] satisfies OverlayActionList

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
      <OverlayActionBar
        actions={actions}
        className={cn('prompt-overlay-actions', 'prompt-overlay-actions--secondary-left', `prompt-overlay-actions--${variant}`)}
      />
    </Dialog>
  )
}
