/**
 * Drawer - Responsive side-drawer (desktop) / bottom-sheet (mobile).
 *
 * Desktop: slides in from the right as a 420px-wide panel.
 * Mobile: slides up from the bottom as a rounded sheet (max 85vh).
 *
 * Renders via createPortal to document.body. Supports Esc/click-outside to
 * close, scroll lock, and optional footer.
 */

import { useRef, useEffect, useId, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { X } from 'lucide-react'
import { cn } from '../../lib/utils'
import { useModalBehavior } from '../../hooks/useModalBehavior'
import { useIsMobile } from '../../hooks/useMediaQuery'
import { useSpatialNavContext } from '../../hooks/SpatialNavContext'

export interface DrawerProps {
  /** Whether the drawer is visible */
  isOpen: boolean
  /** Callback to close the drawer */
  onClose: () => void
  /** Drawer title (rendered as h2) */
  title: string
  /** Optional subtitle below the title */
  description?: string
  /** Drawer content */
  children: ReactNode
  /** Optional footer (sticky at the bottom) */
  footer?: ReactNode
  /** Additional CSS classes for the drawer panel */
  className?: string
  /** data-testid value */
  testId?: string
}

export function Drawer({
  isOpen,
  onClose,
  title,
  description,
  children,
  footer,
  className,
  testId,
}: DrawerProps) {
  const drawerRef = useRef<HTMLDivElement>(null)
  const titleId = useId()
  const isMobile = useIsMobile()

  useModalBehavior({
    isOpen,
    onClose,
    ref: drawerRef,
    preventBodyScroll: true,
    disableCloseOnOutsideClick: true,
  })

  // Push a spatial nav modal scope so D-pad navigation is trapped inside the drawer.
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
    if (e.target === e.currentTarget) {
      onClose()
    }
  }

  if (!isOpen) return null

  return createPortal(
    <div
      ref={scopeRef}
      className="fixed inset-0 z-50 flex"
      onMouseDown={handleBackdropClick}
    >
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm pointer-events-none" />

      {/* Drawer panel */}
      <div
        ref={drawerRef}
        className={cn(
          'relative flex flex-col bg-slate-900 border-white/10 shadow-2xl',
          isMobile
            ? 'mt-auto w-full max-h-[85vh] rounded-t-2xl border-t animate-in slide-in-from-bottom duration-200'
            : 'ml-auto h-full w-[420px] max-w-[90vw] border-l animate-in slide-in-from-right duration-200',
          className,
        )}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        data-testid={testId}
      >
        {/* Header */}
        <div className={cn(
          'flex items-start justify-between gap-3 border-b border-white/10 px-4 py-3',
          isMobile && 'pt-4',
        )}>
          {/* Drag indicator on mobile */}
          {isMobile && (
            <div className="absolute left-1/2 top-1.5 -translate-x-1/2">
              <div className="h-1 w-8 rounded-full bg-slate-600" />
            </div>
          )}
          <div className="min-w-0 flex-1">
            <h2 id={titleId} className="text-base font-semibold text-white">
              {title}
            </h2>
            {description && (
              <p className="mt-0.5 text-xs text-slate-400">{description}</p>
            )}
          </div>
          <button
            type="button"
            onClick={onClose}
            className="shrink-0 rounded p-1 text-slate-400 transition-colors hover:bg-white/10 hover:text-white"
            aria-label="Close drawer"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Scrollable content */}
        <div className="flex-1 overflow-y-auto">
          {children}
        </div>

        {/* Optional footer */}
        {footer && (
          <div className="border-t border-white/10 px-4 py-3">
            {footer}
          </div>
        )}
      </div>
    </div>,
    document.body,
  )
}
