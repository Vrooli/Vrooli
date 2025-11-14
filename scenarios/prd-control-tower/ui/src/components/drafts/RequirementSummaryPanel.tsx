import { AlertTriangle, FolderTree } from 'lucide-react'
import type { RequirementGroup } from '../../types'

interface RequirementSummaryPanelProps {
  groups: RequirementGroup[] | null
  loading: boolean
  error: string | null
}

function countRequirements(groups: RequirementGroup[] | null): number {
  if (!groups) {
    return 0
  }
  return groups.reduce((sum, group) => sum + group.requirements.length + countRequirements(group.children ?? []), 0)
}

export function RequirementSummaryPanel({ groups, loading, error }: RequirementSummaryPanelProps) {
  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading requirements…</p>
  }
  if (error) {
    return (
      <p className="flex items-center gap-2 text-sm text-amber-700">
        <AlertTriangle size={16} /> {error}
      </p>
    )
  }
  if (!groups || groups.length === 0) {
    return <p className="text-sm text-muted-foreground">No modular requirements defined yet.</p>
  }

  const total = countRequirements(groups)

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2 text-sm font-medium text-slate-700">
        <FolderTree size={16} /> Requirements registry
      </div>
      <p className="text-xs text-muted-foreground">{total} tracked requirement{total === 1 ? '' : 's'}</p>
      <ul className="space-y-1 text-sm">
        {groups.map(group => (
          <li key={group.id} className="rounded-xl border border-slate-200 bg-white/70 px-3 py-2 text-slate-800">
            <div className="flex items-center justify-between">
              <span className="font-medium">{group.name}</span>
              <span className="text-xs text-muted-foreground">{group.requirements.length} items</span>
            </div>
            {group.description && <p className="text-xs text-muted-foreground">{group.description}</p>}
          </li>
        ))}
      </ul>
    </div>
  )
}
