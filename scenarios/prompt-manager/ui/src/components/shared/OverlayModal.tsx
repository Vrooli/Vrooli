/**
 * OverlayModal - Shared modal chrome for overlay settings/help dialogs.
 *
 * Thin wrapper around Dialog that adds the "Press Esc to close" footer hint.
 */

import { type ReactNode } from 'react'
import { Dialog } from './Dialog'

interface OverlayModalProps {
  isOpen: boolean
  onClose: () => void
  title: string
  children: ReactNode
  maxWidth?: string
  testId?: string
}

export function OverlayModal({ isOpen, onClose, title, children, maxWidth = 'max-w-lg', testId }: OverlayModalProps) {
  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      title={title}
      maxWidth={maxWidth}
      testId={testId}
    >
      {children}

      <div className="mt-6 pt-4 border-t border-white/10 text-center">
        <span className="text-xs text-slate-500">Press Esc to close</span>
      </div>
    </Dialog>
  )
}
