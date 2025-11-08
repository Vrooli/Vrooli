import { useState, useCallback, useRef, useEffect, createContext, useContext, ReactNode } from 'react'

interface ConfirmDialogOptions {
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  variant?: 'danger' | 'warning' | 'info'
}

interface ConfirmDialogContextValue {
  confirm: (options: ConfirmDialogOptions) => Promise<boolean>
}

const ConfirmDialogContext = createContext<ConfirmDialogContextValue | null>(null)

export function useConfirm() {
  const context = useContext(ConfirmDialogContext)
  if (!context) {
    throw new Error('useConfirm must be used within ConfirmDialogProvider')
  }
  return context.confirm
}

interface DialogState extends ConfirmDialogOptions {
  isOpen: boolean
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialogProvider({ children }: { children: ReactNode }) {
  const [dialogState, setDialogState] = useState<DialogState>({
    isOpen: false,
    message: '',
    onConfirm: () => {},
    onCancel: () => {},
  })

  // Use ref to store current onCancel callback for escape handler
  const onCancelRef = useRef<(() => void) | null>(null)

  const confirm = useCallback((options: ConfirmDialogOptions): Promise<boolean> => {
    return new Promise((resolve) => {
      const onCancel = () => {
        setDialogState((prev) => ({ ...prev, isOpen: false }))
        resolve(false)
      }

      // Store cancel callback in ref for escape handler
      onCancelRef.current = onCancel

      setDialogState({
        isOpen: true,
        title: options.title || 'Confirm',
        message: options.message,
        confirmText: options.confirmText || 'Confirm',
        cancelText: options.cancelText || 'Cancel',
        variant: options.variant || 'info',
        onConfirm: () => {
          setDialogState((prev) => ({ ...prev, isOpen: false }))
          resolve(true)
        },
        onCancel,
      })
    })
  }, [])

  // Handle escape key globally when dialog is open
  useEffect(() => {
    if (!dialogState.isOpen) {
      return
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        if (onCancelRef.current) {
          onCancelRef.current()
        }
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => {
      window.removeEventListener('keydown', handleKeyDown)
    }
  }, [dialogState.isOpen])

  if (!dialogState.isOpen) {
    return (
      <ConfirmDialogContext.Provider value={{ confirm }}>
        {children}
      </ConfirmDialogContext.Provider>
    )
  }

  return (
    <ConfirmDialogContext.Provider value={{ confirm }}>
      {children}
      <div className="confirm-dialog-overlay" onClick={dialogState.onCancel}>
        <div
          className="confirm-dialog"
          role="dialog"
          aria-modal="true"
          aria-labelledby="confirm-dialog-title"
          aria-describedby="confirm-dialog-message"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="confirm-dialog__header">
            <h3 id="confirm-dialog-title">{dialogState.title}</h3>
          </div>
          <div className="confirm-dialog__body">
            <p id="confirm-dialog-message">{dialogState.message}</p>
          </div>
          <div className="confirm-dialog__footer">
            <button
              type="button"
              className="confirm-dialog__button confirm-dialog__button--cancel"
              onClick={dialogState.onCancel}
            >
              {dialogState.cancelText}
            </button>
            <button
              type="button"
              className={`confirm-dialog__button confirm-dialog__button--confirm confirm-dialog__button--${dialogState.variant}`}
              onClick={dialogState.onConfirm}
              autoFocus
            >
              {dialogState.confirmText}
            </button>
          </div>
        </div>
      </div>
    </ConfirmDialogContext.Provider>
  )
}
