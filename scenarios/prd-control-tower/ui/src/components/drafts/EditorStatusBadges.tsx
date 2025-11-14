import { Info, FileEdit, AlertTriangle, CheckCircle2, StickyNote } from 'lucide-react'
import { Link } from 'react-router-dom'
import type { Draft, OperationalTarget } from '../../types'
import { Badge } from '../ui/badge'
import { Button } from '../ui/button'

interface EditorStatusBadgesProps {
  draft: Draft
  metaDialogOpen: boolean
  onOpenMeta: () => void
  targetsData?: {
    targets: OperationalTarget[]
    unmatched_requirements: unknown[]
  } | null
  completenessPercent?: number
}

/**
 * Status badges component for draft editor header.
 * Displays entity type, draft status, template compliance, and target coverage warnings.
 */
export function EditorStatusBadges({ draft, metaDialogOpen, onOpenMeta, targetsData, completenessPercent }: EditorStatusBadgesProps) {
  const statusVariant = draft.status === 'draft' ? 'warning' : 'success'

  // Calculate unlinked targets
  const unlinkedTargets = targetsData?.targets?.filter(t => !t.linked_requirement_ids || t.linked_requirement_ids.length === 0) || []
  const hasUnlinkedTargets = unlinkedTargets.length > 0

  return (
    <div className="flex flex-wrap items-center gap-3 pt-2">
      <Badge variant="secondary" className="capitalize">
        {draft.entity_type}
      </Badge>
      <Badge variant={statusVariant} className="gap-1 capitalize">
        <FileEdit size={14} /> {draft.status === 'draft' ? 'Draft in progress' : draft.status}
      </Badge>

      {/* Backlog origin badge */}
      {draft.source_backlog_id && (
        <Badge variant="outline" className="gap-1 border-amber-300 bg-amber-50 text-amber-700 hover:bg-amber-100">
          <StickyNote size={14} />
          <Link to="/backlog" className="hover:underline">
            From Backlog
          </Link>
        </Badge>
      )}

      {/* Template compliance badge */}
      {completenessPercent !== undefined && (
        <Badge
          variant={completenessPercent >= 90 ? 'success' : completenessPercent >= 70 ? 'warning' : 'destructive'}
          className="gap-1"
        >
          {completenessPercent >= 90 ? <CheckCircle2 size={14} /> : <AlertTriangle size={14} />}
          Template: {completenessPercent}%
        </Badge>
      )}

      {/* Unlinked targets warning */}
      {hasUnlinkedTargets && (
        <Badge variant="destructive" className="gap-1">
          <AlertTriangle size={14} />
          {unlinkedTargets.length} target{unlinkedTargets.length === 1 ? '' : 's'} without requirements
        </Badge>
      )}

      <Button type="button" variant="ghost" size="sm" onClick={onOpenMeta} aria-haspopup="dialog" aria-expanded={metaDialogOpen}>
        <Info size={16} className="mr-2" /> Details
      </Button>
    </div>
  )
}
