/**
 * Dialog - Centered modal with backdrop.
 *
 * Used for confirmations, settings, forms, and any centered overlay content.
 * Renders via createPortal to document.body. Supports Esc/click-outside/X to close,
 * scroll lock, and loading guard.
 */

import { useRef, useEffect, useId, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { X } from 'lucide-react'
import { cn } from '../../lib/utils'
import { useModalBehavior } from '../../hooks/useModalBehavior'
import { useSpatialNavContext } from '../../hooks/SpatialNavContext'

export interface DialogProps {
  /** Whether the dialog is visible */
  isOpen: boolean
  /** Callback to close the dialog */
  onClose: () => void
  /** Dialog title (rendered as h2) */
  title?: string
  /** Dialog content */
  children: ReactNode
  /** Tailwind max-width class (default: 'max-w-lg') */
  maxWidth?: string
  /** Block close when a loading/async operation is in progress */
  isLoading?: boolean
  /** data-testid value */
  testId?: string
  /** Custom id for the title element (for aria-labelledby) */
  titleId?: string
  /** Custom id for the description element (for aria-describedby) */
  descriptionId?: string
  /** Additional CSS classes for the dialog panel */
  className?: string
  /** Additional CSS classes for the fixed outer container (controls alignment) */
  containerClassName?: string
}

/**
 * Generic dialog primitive. Handles escape, click-outside, scroll lock,
 * backdrop rendering, and ARIA attributes.
 */
export function Dialog({
  isOpen,
  onClose,
  title,
  children,
  maxWidth = 'max-w-lg',
  isLoading = false,
  testId,
  titleId: customTitleId,
  descriptionId,
  className,
  containerClassName,
}: DialogProps) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const generatedTitleId = useId()
  const effectiveTitleId = customTitleId ?? generatedTitleId

  // Escape key + scroll lock (click-outside handled via backdrop onClick below)
  useModalBehavior({
    isOpen,
    onClose,
    ref: dialogRef,
    preventBodyScroll: true,
    disableCloseOnOutsideClick: true,
    isLoading,
  })

  // Push a spatial nav modal scope so D-pad navigation is trapped inside the dialog.
  const spatialNavRef = useSpatialNavContext();
  const scopeRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const ctrl = spatialNavRef?.current;
    const el = scopeRef.current;
    if (!isOpen || !ctrl || !el) return;
    ctrl.pushScope(el);
    return () => { ctrl.popScope(); };
  }, [isOpen, spatialNavRef]);

  const handleBackdropClick = (e: React.MouseEvent) => {
    // Only close if the click target is the backdrop/container itself, not the panel
    if (e.target === e.currentTarget && !isLoading) {
      onClose()
    }
  }

  if (!isOpen) return null

  return createPortal(
    <div
      ref={scopeRef}
      className={cn("fixed inset-0 z-50 flex items-center justify-center", containerClassName)}
      onMouseDown={handleBackdropClick}
    >
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm pointer-events-none" />

      {/* Dialog panel */}
      <div
        ref={dialogRef}
        className={cn(
          'relative w-full mx-4 p-6',
          maxWidth,
          'bg-slate-900 border border-white/10 rounded-xl shadow-2xl',
          'animate-in fade-in-0 zoom-in-95 duration-150',
          'max-h-[85vh] overflow-y-auto',
          className,
        )}
        role="dialog"
        aria-modal="true"
        aria-labelledby={(title || customTitleId) ? effectiveTitleId : undefined}
        aria-describedby={descriptionId}
        data-testid={testId}
      >
        {/* Close button */}
        <button
          type="button"
          onClick={onClose}
          disabled={isLoading}
          className={cn(
            'absolute top-4 right-4 p-1 rounded',
            'text-slate-400 hover:text-white hover:bg-white/10 transition-colors',
            isLoading && 'opacity-50 cursor-not-allowed',
          )}
          aria-label="Close dialog"
        >
          <X className="h-5 w-5" />
        </button>

        {/* Title */}
        {title && (
          <h2
            id={effectiveTitleId}
            className="text-xl font-semibold text-white mb-6 pr-8"
          >
            {title}
          </h2>
        )}

        {children}
      </div>
    </div>,
    document.body,
  )
}
