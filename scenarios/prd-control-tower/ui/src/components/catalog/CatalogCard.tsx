import { Link, type NavigateFunction } from 'react-router-dom'
import { CheckCircle, FileEdit, XCircle, HelpCircle, type LucideIcon } from 'lucide-react'
import { Button } from '../ui/button'
import { Badge } from '../ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card'
import { Tooltip } from '../ui/tooltip'
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
        'flex h-full flex-col justify-between border-slate-200 transition-all hover:-translate-y-0.5 hover:border-slate-400 active:translate-y-0',
        primaryPath ? 'cursor-pointer' : 'opacity-80',
      )}
    >
      <CardHeader className="space-y-4 p-5 sm:p-6">
        <div className="flex items-start justify-between gap-3 sm:gap-4">
          <div className="space-y-2 min-w-0 flex-1">
            <CardTitle className="text-lg sm:text-xl font-bold text-slate-900 break-words leading-tight">{entry.name}</CardTitle>
            <CardDescription className="line-clamp-2 sm:line-clamp-3 text-xs sm:text-sm text-slate-600 leading-relaxed">
              {entry.description || 'No description available'}
            </CardDescription>
          </div>
          <Badge variant="secondary" className="capitalize shrink-0 text-xs font-medium px-2.5 py-1">
            {entry.type}
          </Badge>
        </div>
        <Badge variant={statusMeta.badge} className="w-fit gap-1.5 capitalize text-xs sm:text-sm px-3 py-1.5 font-medium">
          <StatusIcon size={15} className="shrink-0" />
          <span>{statusMeta.label}</span>
        </Badge>
      </CardHeader>
      <CardContent className="space-y-4 border-t border-dashed pt-4 p-5 sm:p-6 sm:pt-5 text-sm">
        {summary ? (
          <div className="space-y-3">
            <div className="flex items-center justify-between text-xs uppercase tracking-wide text-slate-500">
              <span className="text-[10px] sm:text-xs font-semibold">Requirements coverage</span>
              <span className="font-bold text-slate-900 text-sm">{completionRate}%</span>
            </div>
            <div className="h-2.5 w-full rounded-full bg-slate-100 shadow-inner touch-none">
              <div
                className="h-full rounded-full bg-gradient-to-r from-emerald-400 via-emerald-500 to-emerald-600 transition-all duration-500 shadow-sm"
                style={{ width: `${completionRate}%` }}
                role="progressbar"
                aria-valuenow={completionRate ?? 0}
                aria-valuemin={0}
                aria-valuemax={100}
              />
            </div>
            <div className="grid grid-cols-3 gap-1 sm:gap-2 text-[10px] sm:text-xs text-slate-600">
              <span className="flex items-center gap-1">
                <CheckCircle className="text-green-600 shrink-0" size={12} /> <span className="truncate">{summary.completed}</span>
              </span>
              <span className="flex items-center gap-1">
                <FileEdit className="text-blue-600 shrink-0" size={12} /> <span className="truncate">{summary.in_progress}</span>
              </span>
              <span className="flex items-center gap-1">
                <XCircle className="text-slate-400 shrink-0" size={12} /> <span className="truncate">{summary.pending}</span>
              </span>
            </div>
            <div className="flex flex-wrap items-center gap-1.5">
              {summary.p0 > 0 && (
                <Tooltip content="P0: Must-have for launch viability" side="top">
                  <Badge variant="destructive" className="text-[10px] sm:text-[11px] cursor-help">P0: {summary.p0}</Badge>
                </Tooltip>
              )}
              {summary.p1 > 0 && (
                <Tooltip content="P1: Should have post-launch" side="top">
                  <Badge variant="default" className="text-[10px] sm:text-[11px] cursor-help">P1: {summary.p1}</Badge>
                </Tooltip>
              )}
              {summary.p2 > 0 && (
                <Tooltip content="P2: Future enhancement" side="top">
                  <Badge variant="secondary" className="text-[10px] sm:text-[11px] cursor-help">P2: {summary.p2}</Badge>
                </Tooltip>
              )}
              {(summary.p0 > 0 || summary.p1 > 0 || summary.p2 > 0) && (
                <Tooltip content="Criticality levels indicate priority: P0 (critical), P1 (important), P2 (nice-to-have)" side="top">
                  <HelpCircle size={13} className="text-slate-400 hover:text-slate-600 cursor-help shrink-0" />
                </Tooltip>
              )}
            </div>
          </div>
        ) : (
          <p className="text-xs text-muted-foreground">No requirements registered yet.</p>
        )}

        <div className="flex flex-wrap gap-2" onClick={event => event.stopPropagation()} onKeyDown={event => event.stopPropagation()}>
          {entry.has_prd && (
            <Button variant="ghost" size="sm" asChild className="h-10 text-xs sm:text-sm px-3 sm:px-4 hover:bg-slate-100 transition-colors">
              <Link to={`${scenarioPath}?tab=prd`}>View PRD</Link>
            </Button>
          )}
          {entry.has_requirements && (
            <Button variant="ghost" size="sm" asChild className="h-10 text-xs sm:text-sm px-3 sm:px-4 hover:bg-slate-100 transition-colors">
              <Link to={`${scenarioPath}?tab=requirements`}>
                <span className="hidden sm:inline">Requirements</span>
                <span className="sm:hidden">Reqs</span>
              </Link>
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
              className="h-10 text-xs sm:text-sm px-3 sm:px-4 hover:bg-violet-50 hover:text-violet-700 transition-colors disabled:opacity-50"
              title="Create a draft from the published PRD to make changes"
            >
              {preparingId === draftKey ? (
                <span className="flex items-center gap-2">
                  <span className="h-3 w-3 animate-spin rounded-full border-2 border-violet-200 border-t-violet-600" />
                  Preparing…
                </span>
              ) : (
                'Edit PRD'
              )}
            </Button>
          )}
          {entry.has_draft && (
            <Button variant="ghost" size="sm" asChild className="h-10 text-xs sm:text-sm px-3 sm:px-4 hover:bg-blue-50 hover:text-blue-700 transition-colors">
              <Link to={draftPath}>View draft</Link>
            </Button>
          )}
          {!entry.has_prd && !entry.has_draft && (
            <span className="text-xs text-slate-500 italic">No PRD yet</span>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
