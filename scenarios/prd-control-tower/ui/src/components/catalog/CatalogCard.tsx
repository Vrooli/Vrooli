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
  const prdPath = `/prd/${entry.type}/${encodedName}`
  const draftPath = `/draft/${entry.type}/${encodedName}`
  const primaryPath = entry.has_prd ? prdPath : entry.has_draft ? draftPath : null

  const status: StatusKey = entry.has_prd ? 'published' : entry.has_draft ? 'draft' : 'missing'
  const statusMeta = statusMap[status]
  const StatusIcon = statusMeta.icon

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
      <CardContent className="flex flex-wrap items-center justify-between gap-3 border-t border-dashed pt-4 text-sm">
        <div className="flex flex-wrap gap-2" onClick={event => event.stopPropagation()} onKeyDown={event => event.stopPropagation()}>
          {entry.has_prd && (
            <Button variant="ghost" size="sm" asChild>
              <Link to={prdPath}>View PRD</Link>
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
