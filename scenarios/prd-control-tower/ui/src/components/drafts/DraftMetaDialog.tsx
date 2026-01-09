import { X } from 'lucide-react'
import type { Draft } from '../../types'
import type { DraftMetrics } from '../../utils/formatters'
import { formatDate } from '../../utils/formatters'

interface DraftMetaDialogProps {
  draft: Draft
  metrics: DraftMetrics
  open: boolean
  onClose: () => void
}

export function DraftMetaDialog({ draft, metrics, open, onClose }: DraftMetaDialogProps) {
  if (!open) {
    return null
  }

  const stats: Array<{ label: string; value: string }> = [
    { label: 'Last updated', value: formatDate(draft.updated_at) },
    { label: 'Created', value: formatDate(draft.created_at) },
    { label: 'Status', value: draft.status },
    { label: 'Type', value: draft.entity_type },
    { label: 'File size', value: `${metrics.sizeKb.toFixed(1)} KB` },
    { label: 'Word count', value: metrics.wordCount.toLocaleString() },
    { label: 'Character count', value: metrics.characterCount.toLocaleString() },
  ]

  if (draft.owner) {
    stats.splice(2, 0, { label: 'Owner', value: draft.owner })
  }

  if (metrics.estimatedReadMinutes > 0) {
    stats.push({
      label: 'Estimated read time',
      value: `${metrics.estimatedReadMinutes} ${metrics.estimatedReadMinutes === 1 ? 'minute' : 'minutes'}`,
    })
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/70 px-4"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-labelledby="draft-info-dialog-title"
    >
      <div
        className="w-full max-w-xl rounded-2xl border bg-white shadow-2xl"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-start justify-between border-b px-6 py-4">
          <div>
            <h3 id="draft-info-dialog-title" className="text-lg font-semibold">
              Draft details
            </h3>
            <p className="text-sm text-muted-foreground">
              Supplemental metadata and helpful stats for <strong>{draft.entity_name}</strong>.
            </p>
          </div>
          <button
            type="button"
            className="rounded-full border p-1 text-slate-500 hover:bg-slate-50"
            onClick={onClose}
            aria-label="Close draft details"
          >
            <X size={16} />
          </button>
        </div>
        <dl className="grid grid-cols-1 gap-4 px-6 py-5 text-sm sm:grid-cols-2">
          {stats.map((stat) => (
            <div key={stat.label} className="rounded-xl border bg-slate-50/70 px-4 py-3">
              <dt className="text-xs uppercase tracking-wide text-slate-500">{stat.label}</dt>
              <dd className="text-base font-medium text-slate-900">{stat.value}</dd>
            </div>
          ))}
        </dl>
      </div>
    </div>
  )
}
