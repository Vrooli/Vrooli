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

  // Calculate coverage metrics
  const targetsWithRequirements = targets.filter(t => t.linked_requirement_ids && t.linked_requirement_ids.length > 0)
  const orphanedTargets = targets.filter(t => !t.linked_requirement_ids || t.linked_requirement_ids.length === 0)
  const coveragePercent = targets.length > 0 ? Math.round((targetsWithRequirements.length / targets.length) * 100) : 0
  const hasGaps = orphanedTargets.length > 0 || (unmatchedRequirements && unmatchedRequirements.length > 0)

  return (
    <div className="space-y-4">
      {/* Coverage Summary */}
      <div className="rounded-2xl border bg-gradient-to-br from-slate-50 to-white p-4">
        <div className="flex items-center justify-between mb-3">
          <span className="text-sm font-medium text-slate-700">Requirement Coverage</span>
          <div className="flex items-center gap-2">
            <span className={`text-lg font-bold ${coveragePercent >= 90 ? 'text-emerald-600' : coveragePercent >= 70 ? 'text-amber-600' : 'text-red-600'}`}>
              {coveragePercent}%
            </span>
          </div>
        </div>
        <div className="grid grid-cols-3 gap-3 text-center text-xs">
          <div className="rounded-lg bg-white p-2 shadow-sm">
            <p className="font-semibold text-slate-900">{targets.length}</p>
            <p className="text-muted-foreground">Total Targets</p>
          </div>
          <div className="rounded-lg bg-emerald-50 p-2 shadow-sm">
            <p className="font-semibold text-emerald-700">{targetsWithRequirements.length}</p>
            <p className="text-emerald-600">Linked</p>
          </div>
          <div className="rounded-lg bg-amber-50 p-2 shadow-sm">
            <p className="font-semibold text-amber-700">{orphanedTargets.length}</p>
            <p className="text-amber-600">Orphaned</p>
          </div>
        </div>
        {hasGaps && (
          <div className="mt-3 flex items-start gap-2 rounded-lg bg-amber-50 p-2 text-xs text-amber-800">
            <AlertTriangle size={14} className="mt-0.5 shrink-0" />
            <p>
              <strong>Coverage gaps detected:</strong> {orphanedTargets.length > 0 && `${orphanedTargets.length} target(s) without requirements`}
              {orphanedTargets.length > 0 && unmatchedRequirements && unmatchedRequirements.length > 0 && ', '}
              {unmatchedRequirements && unmatchedRequirements.length > 0 && `${unmatchedRequirements.length} requirement(s) without targets`}
            </p>
          </div>
        )}
      </div>

      {/* Orphaned Targets Warning */}
      {orphanedTargets.length > 0 && (
        <div className="rounded-2xl border-2 border-amber-300 bg-amber-50 p-4">
          <p className="flex items-center gap-2 text-sm font-medium text-amber-900 mb-3">
            <AlertTriangle size={16} />
            {orphanedTargets.length} Operational Target{orphanedTargets.length === 1 ? '' : 's'} Without Requirements
          </p>
          <p className="text-xs text-amber-800 mb-3">
            These operational targets are defined in your PRD but have no corresponding requirements in the <code className="bg-amber-100 px-1 rounded">requirements/</code> folder.
            Consider creating requirements to track implementation progress.
          </p>
          <div className="space-y-2">
            {orphanedTargets.map(target => (
              <div
                key={target.id}
                className="rounded-lg border border-amber-200 bg-white p-2 text-sm"
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="flex-1">
                    <p className="font-medium text-amber-900">{target.title}</p>
                    <p className="text-xs text-amber-700">{target.path}</p>
                  </div>
                  <Badge variant="outline" className="border-amber-400 text-amber-700">
                    {target.criticality ?? 'Unknown'}
                  </Badge>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* All Targets List */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium text-slate-700">All operational targets</span>
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
