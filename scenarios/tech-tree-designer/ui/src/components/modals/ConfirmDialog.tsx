import React from 'react'

interface ConfirmDialogProps {
  isOpen: boolean
  title: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
  isDangerous?: boolean
  isProcessing?: boolean
  onConfirm: () => void
  onCancel: () => void
}

/**
 * ConfirmDialog - Reusable confirmation dialog for destructive actions.
 *
 * Used for operations like deleting stages, sectors, or unlinking scenarios.
 * Supports danger mode (red confirm button) for destructive actions.
 */
const ConfirmDialog: React.FC<ConfirmDialogProps> = ({
  isOpen,
  title,
  message,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  isDangerous = false,
  isProcessing = false,
  onConfirm,
  onCancel
}) => {
  if (!isOpen) {
    return null
  }

  const handleKeyDown = (event: React.KeyboardEvent) => {
    if (event.key === 'Escape' && !isProcessing) {
      onCancel()
    } else if (event.key === 'Enter' && !isProcessing) {
      onConfirm()
    }
  }

  return (
    <div
      className="modal-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="confirm-dialog-title"
      onKeyDown={handleKeyDown}
    >
      <div className="modal modal--compact">
        <h3 id="confirm-dialog-title">{title}</h3>
        <p className="confirm-dialog__message">{message}</p>
        <div className="modal-actions">
          <button
            type="button"
            className="button button--ghost"
            onClick={onCancel}
            disabled={isProcessing}
          >
            {cancelLabel}
          </button>
          <button
            type="button"
            className={isDangerous ? 'button button--danger' : 'button'}
            onClick={onConfirm}
            disabled={isProcessing}
            autoFocus
          >
            {isProcessing ? 'Processing…' : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}

export default ConfirmDialog
