import { CheckCircle2, AlertTriangle } from 'lucide-react'
import type { DraftSaveStatus } from '../../types'
import { cn } from '../../lib/utils'

interface SaveStatusNotificationProps {
  saveStatus: DraftSaveStatus | null
}

/**
 * Save status notification component for draft editor.
 * Displays success or error messages after save operations with animated feedback.
 */
export function SaveStatusNotification({ saveStatus }: SaveStatusNotificationProps) {
  if (!saveStatus) {
    return null
  }

  const isSuccess = saveStatus.type === 'success'

  return (
    <div
      className={cn(
        'flex items-center gap-2 rounded-xl border px-4 py-2 text-sm animate-in fade-in slide-in-from-top-2 duration-300',
        isSuccess
          ? 'border-emerald-200 bg-emerald-50 text-emerald-800 shadow-sm'
          : 'border-amber-200 bg-amber-50 text-amber-900 shadow-sm',
      )}
      role="status"
      aria-live="polite"
    >
      {isSuccess ? (
        <CheckCircle2 size={18} className="shrink-0 animate-in zoom-in duration-300" />
      ) : (
        <AlertTriangle size={18} className="shrink-0 animate-pulse" />
      )}
      <span className="font-medium">{saveStatus.message}</span>
      {isSuccess && (
        <span className="ml-auto text-xs text-emerald-600">✓</span>
      )}
    </div>
  )
}
