import { Bot, CheckCircle2, ExternalLink, RefreshCw, TriangleAlert } from 'lucide-react'

import { Button } from '../ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card'
import { Badge } from '../ui/badge'
import { useEcosystemTask } from '../../hooks/useEcosystemTask'

interface EcosystemTaskPanelProps {
  entityType?: string
  entityName?: string
  criticalIssues: number
  coverageIssues: number
}

export function EcosystemTaskPanel({ entityType, entityName, criticalIssues, coverageIssues }: EcosystemTaskPanelProps) {
  const { status, loading, error, createTask, creating, refresh } = useEcosystemTask(entityType, entityName)

  const disableCreate = criticalIssues > 0 || coverageIssues > 0
  const disableReason = criticalIssues > 0
    ? 'Resolve critical target linkages before dispatching an agent.'
    : coverageIssues > 0
      ? 'Map outstanding requirements to targets before dispatching an agent.'
      : ''

  const renderBody = () => {
    if (!entityType || !entityName) {
      return <p className="text-sm text-muted-foreground">Select a specific scenario to view automation status.</p>
    }

    if (!status?.configured) {
      return (
        <div className="rounded-2xl border border-dashed border-slate-200 bg-slate-50 p-4 text-sm text-muted-foreground">
          Set <strong>ECOSYSTEM_MANAGER_URL</strong> in the scenario environment to enable dispatching tasks.
        </div>
      )
    }

    if (error) {
      return (
        <div className="flex items-center justify-between rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
          <div className="flex items-center gap-2">
            <TriangleAlert size={16} /> {error}
          </div>
          <Button variant="outline" size="sm" onClick={refresh} disabled={loading}>
            <RefreshCw size={14} className={loading ? 'animate-spin' : ''} />
          </Button>
        </div>
      )
    }

    if (status?.task) {
      return (
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-slate-600">Active agent</p>
              <p className="text-lg font-semibold text-slate-900">{status.task.title}</p>
            </div>
            <Badge>{status.task.status}</Badge>
          </div>
          <div className="rounded-2xl border bg-white p-4 text-sm text-slate-600">
            <p>
              Priority <strong className="text-slate-900">{status.task.priority}</strong> · Category{' '}
              <strong className="text-slate-900">{status.task.category || 'n/a'}</strong>
            </p>
            <p className="text-xs text-muted-foreground">Target: {status.task.target}</p>
          </div>
          <div className="flex gap-2">
            {status.task.view_url && (
              <Button asChild variant="outline" size="sm">
                <a href={status.task.view_url} target="_blank" rel="noreferrer" className="flex items-center gap-2">
                  Open in ecosystem-manager <ExternalLink size={14} />
                </a>
              </Button>
            )}
            <Button variant="ghost" size="sm" onClick={refresh} disabled={loading}>
              Refresh
              <RefreshCw size={14} className={loading ? 'ml-2 animate-spin' : 'ml-2'} />
            </Button>
          </div>
        </div>
      )
    }

    return (
      <div className="space-y-3">
        <p className="text-sm text-muted-foreground">No agent is currently working this scenario.</p>
        {disableCreate && (
          <div className="rounded-xl border border-amber-200 bg-amber-50 p-3 text-xs text-amber-900">
            {disableReason}
          </div>
        )}
        <div className="flex gap-3">
          <Button onClick={createTask} disabled={disableCreate || creating} className="gap-2">
            {creating ? <RefreshCw size={16} className="animate-spin" /> : <CheckCircle2 size={16} />}
            Create ecosystem task
          </Button>
          <Button variant="ghost" size="sm" onClick={refresh} disabled={loading}>
            Refresh
          </Button>
        </div>
      </div>
    )
  }

  return (
    <Card className="border bg-white/90">
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-lg">
          <span className="rounded-xl bg-slate-100 p-2 text-slate-600">
            <Bot size={18} />
          </span>
          Ecosystem-manager task
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4 text-sm">
        <p className="text-muted-foreground">
          Kick off the ecosystem-manager agent once requirements are stable. It will iterate until every linked requirement is green.
        </p>
        {renderBody()}
      </CardContent>
    </Card>
  )
}
