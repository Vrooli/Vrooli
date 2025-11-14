import { CheckCircle2, AlertTriangle } from 'lucide-react'
import type { DraftSaveStatus } from '../../types'
import { cn } from '../../lib/utils'

interface SaveStatusNotificationProps {
  saveStatus: DraftSaveStatus | null
}

/**
 * Save status notification component for draft editor.
 * Displays success or error messages after save operations.
 */
export function SaveStatusNotification({ saveStatus }: SaveStatusNotificationProps) {
  if (!saveStatus) {
    return null
  }

  return (
    <div
      className={cn(
        'flex items-center gap-2 rounded-xl border px-4 py-2 text-sm',
        saveStatus.type === 'success' ? 'border-emerald-200 bg-emerald-50 text-emerald-800' : 'border-amber-200 bg-amber-50 text-amber-900',
      )}
    >
      {saveStatus.type === 'success' ? <CheckCircle2 size={18} /> : <AlertTriangle size={18} />}
      <span>{saveStatus.message}</span>
    </div>
  )
}
