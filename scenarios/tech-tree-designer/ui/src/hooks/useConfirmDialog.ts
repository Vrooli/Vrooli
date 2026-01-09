import { useCallback, useState } from 'react'

interface ConfirmDialogState {
  isOpen: boolean
  title: string
  message: string
  confirmLabel: string
  isDangerous: boolean
  onConfirm: (() => void | Promise<void>) | null
}

const defaultState: ConfirmDialogState = {
  isOpen: false,
  title: '',
  message: '',
  confirmLabel: 'Confirm',
  isDangerous: false,
  onConfirm: null
}

/**
 * Hook for managing confirmation dialog state.
 * Provides a clean API for showing confirmation dialogs for destructive actions.
 */
export const useConfirmDialog = () => {
  const [state, setState] = useState<ConfirmDialogState>(defaultState)
  const [isProcessing, setIsProcessing] = useState(false)

  const showConfirm = useCallback(
    (options: {
      title: string
      message: string
      confirmLabel?: string
      isDangerous?: boolean
      onConfirm: () => void | Promise<void>
    }) => {
      setState({
        isOpen: true,
        title: options.title,
        message: options.message,
        confirmLabel: options.confirmLabel || 'Confirm',
        isDangerous: options.isDangerous ?? true,
        onConfirm: options.onConfirm
      })
    },
    []
  )

  const handleConfirm = useCallback(async () => {
    if (!state.onConfirm) {
      return
    }

    setIsProcessing(true)
    try {
      const result = state.onConfirm()
      // Await if it's a Promise
      await Promise.resolve(result)
      setState(defaultState)
    } catch (error) {
      console.error('Confirmation action failed:', error)
      // Keep dialog open on error so user can see what happened
    } finally {
      setIsProcessing(false)
    }
  }, [state.onConfirm])

  const handleCancel = useCallback(() => {
    if (isProcessing) {
      return
    }
    setState(defaultState)
  }, [isProcessing])

  return {
    confirmState: state,
    isProcessing,
    showConfirm,
    handleConfirm,
    handleCancel
  }
}
