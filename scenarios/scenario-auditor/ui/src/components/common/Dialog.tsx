import { ReactNode, useEffect } from 'react'
import { X } from 'lucide-react'
import clsx from 'clsx'

interface DialogProps {
  open: boolean
  onClose: () => void
  title?: string
  children: ReactNode
  className?: string
  size?: 'sm' | 'md' | 'lg'
}

export function Dialog({ 
  open, 
  onClose, 
  title, 
  children, 
  className,
  size = 'md' 
}: DialogProps) {
  useEffect(() => {
    if (open) {
      document.body.style.overflow = 'hidden'
    } else {
      document.body.style.overflow = 'unset'
    }
    return () => {
      document.body.style.overflow = 'unset'
    }
  }, [open])

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && open) {
        onClose()
      }
    }
    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [open, onClose])

  if (!open) return null

  const sizeClasses = {
    sm: 'max-w-md',
    md: 'max-w-lg',
    lg: 'max-w-2xl'
  }

  return (
    <>
      {/* Backdrop */}
      <div 
        className="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm animate-in fade-in"
        onClick={onClose}
      />
      
      {/* Dialog */}
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div 
          className={clsx(
            'relative bg-white rounded-xl shadow-xl w-full animate-in zoom-in-95',
            sizeClasses[size],
            className
          )}
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          {title && (
            <div className="flex items-center justify-between border-b border-dark-200 px-6 py-4">
              <h2 className="text-lg font-semibold text-dark-900">{title}</h2>
              <button
                onClick={onClose}
                className="rounded-lg p-1 text-dark-500 hover:bg-dark-100 hover:text-dark-700 transition-colors"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
          )}

          {/* Content */}
          <div className={clsx(
            'overflow-y-auto',
            title ? 'p-6' : 'p-6 pt-12'
          )}>
            {!title && (
              <button
                onClick={onClose}
                className="absolute right-4 top-4 rounded-lg p-1 text-dark-500 hover:bg-dark-100 hover:text-dark-700 transition-colors"
              >
                <X className="h-5 w-5" />
              </button>
            )}
            {children}
          </div>
        </div>
      </div>
    </>
  )
}