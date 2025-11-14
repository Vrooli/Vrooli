import { AlertTriangle, Link as LinkIcon } from 'lucide-react'
import type { OperationalTarget, RequirementRecord } from '../../types'
import { Badge } from '../ui/badge'

interface OperationalTargetsPanelProps {
  targets: OperationalTarget[] | null
  unmatchedRequirements: RequirementRecord[] | null
  loading: boolean
  error: string | null
}

const statusVariantMap: Record<string, { label: string; variant: 'success' | 'warning' }> = {
  complete: { label: 'Complete', variant: 'success' },
  pending: { label: 'Planned', variant: 'warning' },
}

export function OperationalTargetsPanel({ targets, unmatchedRequirements, loading, error }: OperationalTargetsPanelProps) {
  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading coverage…</p>
  }
  if (error) {
    return (
      <p className="flex items-center gap-2 text-sm text-amber-700">
        <AlertTriangle size={16} /> {error}
      </p>
    )
  }
  if (!targets || targets.length === 0) {
    return <p className="text-sm text-muted-foreground">No operational targets defined yet.</p>
  }

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium text-slate-700">Operational targets</span>
          <span className="text-xs text-muted-foreground">{targets.length} total</span>
        </div>
        <div className="space-y-3">
          {targets.map(target => (
            <div
              key={target.id}
              className="rounded-2xl border border-slate-200 bg-white/70 p-3 text-sm shadow-sm"
            >
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="font-medium text-slate-900">{target.title}</p>
                  <p className="text-xs text-muted-foreground">{target.path}</p>
                </div>
                <div className="flex flex-wrap gap-2">
                  <Badge variant="secondary">{target.category}</Badge>
                  {target.criticality && (
                    <Badge variant="outline">{target.criticality}</Badge>
                  )}
                  <Badge variant={statusVariantMap[target.status]?.variant ?? 'outline'}>
                    {statusVariantMap[target.status]?.label ?? target.status}
                  </Badge>
                </div>
              </div>
              {target.notes && <p className="mt-2 text-xs text-muted-foreground">{target.notes}</p>}
              {target.linked_requirement_ids?.length ? (
                <p className="mt-2 flex items-center gap-2 text-xs text-emerald-700">
                  <LinkIcon size={14} /> Linked to {target.linked_requirement_ids.length} requirement
                  {target.linked_requirement_ids.length === 1 ? '' : 's'}
                </p>
              ) : (
                <p className="mt-2 flex items-center gap-2 text-xs text-amber-700">
                  <AlertTriangle size={14} /> No linked requirements yet
                </p>
              )}
            </div>
          ))}
        </div>
      </div>

      {unmatchedRequirements && unmatchedRequirements.length > 0 && (
        <div>
          <p className="flex items-center gap-2 text-sm font-medium text-slate-700">
            <AlertTriangle size={16} /> Requirements without operational targets
          </p>
          <ul className="mt-2 list-disc pl-5 text-xs text-muted-foreground">
            {unmatchedRequirements.slice(0, 5).map(req => (
              <li key={req.id}>
                <span className="font-medium text-slate-800">{req.title}</span>
                {req.prd_ref ? ` · ${req.prd_ref}` : ''}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}
