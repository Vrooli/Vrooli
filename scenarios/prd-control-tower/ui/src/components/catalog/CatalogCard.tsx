import { Link, type NavigateFunction } from 'react-router-dom'
import { CheckCircle, FileEdit, XCircle, type LucideIcon } from 'lucide-react'
import { Button } from '../ui/button'
import { Badge } from '../ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card'
import { cn } from '../../lib/utils'
import type { CatalogEntry } from '../../types'

type StatusKey = 'draft' | 'published' | 'missing'

const statusMap: Record<StatusKey, { label: string; icon: LucideIcon; badge: 'success' | 'warning' | 'outline' }> = {
  draft: { label: 'Draft pending', icon: FileEdit, badge: 'warning' },
  published: { label: 'Has PRD', icon: CheckCircle, badge: 'success' },
  missing: { label: 'Missing', icon: XCircle, badge: 'outline' },
}

export interface CatalogCardProps {
  entry: CatalogEntry
  navigate: NavigateFunction
  prepareDraft: (entityType: string, entityName: string) => Promise<void>
  preparingId: string | null
}

export function CatalogCard({ entry, navigate, prepareDraft, preparingId }: CatalogCardProps) {
  const encodedName = encodeURIComponent(entry.name)
  const draftKey = `${entry.type}:${entry.name}`
  const scenarioPath = `/scenario/${entry.type}/${encodedName}`
  const draftPath = `/draft/${entry.type}/${encodedName}`
  const primaryPath = entry.has_prd ? scenarioPath : entry.has_draft ? draftPath : null

  const status: StatusKey = entry.has_prd ? 'published' : entry.has_draft ? 'draft' : 'missing'
  const statusMeta = statusMap[status]
  const StatusIcon = statusMeta.icon
  const summary = entry.requirements_summary
  const completionRate = summary && summary.total > 0 ? Math.round((summary.completed / summary.total) * 100) : null

  const onNavigate = () => {
    if (primaryPath) {
      navigate(primaryPath)
    }
  }

  return (
    <Card
      role={primaryPath ? 'link' : undefined}
      tabIndex={primaryPath ? 0 : -1}
      onClick={onNavigate}
      onKeyDown={event => {
        if (!primaryPath) return
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault()
          onNavigate()
        }
      }}
      className={cn(
        'flex h-full flex-col justify-between border-slate-200 transition hover:-translate-y-0.5 hover:border-slate-400',
        primaryPath ? 'cursor-pointer' : 'opacity-80',
      )}
    >
      <CardHeader className="space-y-3">
        <div className="flex items-start justify-between gap-4">
          <div className="space-y-1">
            <CardTitle className="text-xl font-semibold text-slate-900">{entry.name}</CardTitle>
            <CardDescription className="line-clamp-3 text-sm text-slate-600">
              {entry.description || 'No description available'}
            </CardDescription>
          </div>
          <Badge variant="secondary" className="capitalize">
            {entry.type}
          </Badge>
        </div>
        <Badge variant={statusMeta.badge} className="w-fit gap-1 capitalize">
          <StatusIcon size={14} />
          {statusMeta.label}
        </Badge>
      </CardHeader>
      <CardContent className="space-y-4 border-t border-dashed pt-4 text-sm">
        {summary ? (
          <div className="space-y-2">
            <div className="flex items-center justify-between text-xs uppercase tracking-wide text-slate-500">
              <span>Requirements coverage</span>
              <span className="font-semibold text-slate-900">{completionRate}%</span>
            </div>
            <div className="h-2 w-full rounded-full bg-slate-100">
              <div
                className="h-full rounded-full bg-gradient-to-r from-emerald-400 to-emerald-600 transition-all"
                style={{ width: `${completionRate}%` }}
              />
            </div>
            <div className="grid grid-cols-3 gap-2 text-xs text-slate-600">
              <span className="flex items-center gap-1">
                <CheckCircle className="text-green-600" size={12} /> {summary.completed}
              </span>
              <span className="flex items-center gap-1">
                <FileEdit className="text-blue-600" size={12} /> {summary.in_progress}
              </span>
              <span className="flex items-center gap-1">
                <XCircle className="text-slate-400" size={12} /> {summary.pending}
              </span>
            </div>
            <div className="flex flex-wrap gap-1">
              {summary.p0 > 0 && (
                <Badge variant="destructive" className="text-[11px]">P0: {summary.p0}</Badge>
              )}
              {summary.p1 > 0 && (
                <Badge variant="default" className="text-[11px]">P1: {summary.p1}</Badge>
              )}
              {summary.p2 > 0 && (
                <Badge variant="secondary" className="text-[11px]">P2: {summary.p2}</Badge>
              )}
            </div>
          </div>
        ) : (
          <p className="text-xs text-muted-foreground">No requirements registered yet.</p>
        )}

        <div className="flex flex-wrap gap-2" onClick={event => event.stopPropagation()} onKeyDown={event => event.stopPropagation()}>
          {entry.has_prd && (
            <Button variant="ghost" size="sm" asChild>
              <Link to={`${scenarioPath}?tab=prd`}>View PRD</Link>
            </Button>
          )}
          {entry.has_requirements && (
            <Button variant="ghost" size="sm" asChild>
              <Link to={`${scenarioPath}?tab=requirements`}>Requirements</Link>
            </Button>
          )}
          {entry.has_prd && (
            <Button
              variant="ghost"
              size="sm"
              onClick={event => {
                event.stopPropagation()
                prepareDraft(entry.type, entry.name)
              }}
              disabled={preparingId === draftKey}
            >
              {preparingId === draftKey ? 'Preparing…' : 'Edit PRD'}
            </Button>
          )}
          {entry.has_draft && (
            <Button variant="ghost" size="sm" asChild>
              <Link to={draftPath}>View draft</Link>
            </Button>
          )}
          {!entry.has_prd && !entry.has_draft && (
            <span className="text-xs text-muted-foreground">No PRD yet</span>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
