/**
 * RunListPanel - Panel for listing runs in the sidebar.
 *
 * Features:
 * - Status filter chips: All | Running | Complete | Failed
 * - Scrollable list of runs from useRunData
 * - Status dot + tag label + relative timestamp per row
 * - Search filters by tag text
 * - Auto-refresh via useRunData polling
 */

import { useState, useMemo } from 'react'
import { Activity } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useRunData } from '@/hooks/useRunData'
import { selectors } from '@/constants/selectors'

interface RunListPanelProps {
  selectedRunId: string | null
  onSelectRun: (id: string) => void
  searchQuery?: string
  className?: string
}

const STATUS_FILTERS = [
  { label: 'All', value: '' },
  { label: 'Running', value: 'running' },
  { label: 'Complete', value: 'completed' },
  { label: 'Failed', value: 'failed' },
] as const

const STATUS_DOT_COLORS: Record<string, string> = {
  completed: 'bg-emerald-500',
  running: 'bg-amber-500 animate-pulse',
  failed: 'bg-red-500',
  cancelled: 'bg-slate-400',
  pending: 'bg-blue-400',
}

function formatRelativeTime(iso?: string): string {
  if (!iso) return ''
  const diffMs = Date.now() - new Date(iso).getTime()
  if (Number.isNaN(diffMs) || diffMs < 0) return 'Just now'
  if (diffMs < 60000) return 'Just now'
  if (diffMs < 3600000) {
    const mins = Math.round(diffMs / 60000)
    return `${mins}m ago`
  }
  if (diffMs < 86400000) {
    const hrs = Math.round(diffMs / 3600000)
    return `${hrs}h ago`
  }
  const days = Math.round(diffMs / 86400000)
  return `${days}d ago`
}

export function RunListPanel({
  selectedRunId,
  onSelectRun,
  searchQuery,
  className,
}: RunListPanelProps) {
  const [statusFilter, setStatusFilter] = useState('')
  const { runs, loading, error } = useRunData({
    status: statusFilter || undefined,
  })

  const filteredRuns = useMemo(() => {
    if (!searchQuery) return runs
    const lower = searchQuery.toLowerCase()
    return runs.filter((r) => (r.tag ?? '').toLowerCase().includes(lower))
  }, [runs, searchQuery])

  if (loading) {
    return (
      <div className={cn('flex items-center justify-center py-8', className)}>
        <div className="w-6 h-6 border-2 border-primary border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  if (error) {
    return (
      <div className={cn('px-3 py-8 text-center', className)}>
        <p className="text-sm text-destructive">Failed to load runs</p>
      </div>
    )
  }

  return (
    <div
      className={cn('flex flex-col h-full', className)}
      data-testid={selectors.runs.list}
    >
      {/* Status filter chips */}
      <div className="flex-shrink-0 px-3 py-2 flex gap-1 flex-wrap">
        {STATUS_FILTERS.map((filter) => (
          <button
            key={filter.value}
            type="button"
            onClick={() => setStatusFilter(filter.value)}
            className={cn(
              'px-2 py-1 text-[10px] rounded border transition-colors',
              statusFilter === filter.value
                ? 'bg-primary/10 text-primary border-primary/40'
                : 'text-muted-foreground border-border hover:text-foreground hover:bg-muted/50'
            )}
          >
            {filter.label}
          </button>
        ))}
      </div>

      {/* Run list */}
      <div className="flex-1 overflow-y-auto py-1">
        {runs.length === 0 ? (
          <div className="px-3 py-8 text-center">
            <Activity className="h-8 w-8 mx-auto mb-2 text-muted-foreground" />
            <p className="text-xs text-muted-foreground">No runs yet</p>
          </div>
        ) : filteredRuns.length === 0 ? (
          <div className="px-3 py-8 text-center">
            <Activity className="h-8 w-8 mx-auto mb-2 text-muted-foreground opacity-60" />
            <p className="text-xs text-muted-foreground">No matching runs</p>
          </div>
        ) : (
          filteredRuns.map((run) => (
            <button
              key={run.id}
              type="button"
              onClick={() => onSelectRun(run.id)}
              className={cn(
                'w-full flex items-center gap-3 px-3 py-2 text-left group',
                'hover:bg-muted/50 transition-colors',
                selectedRunId === run.id && 'bg-primary/10'
              )}
              data-testid={selectors.runs.row}
              data-run-id={run.id}
            >
              {/* Status dot */}
              <span
                className={cn(
                  'inline-block h-2.5 w-2.5 rounded-full flex-shrink-0',
                  STATUS_DOT_COLORS[run.status] ?? 'bg-slate-400'
                )}
              />

              {/* Run info */}
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-foreground truncate">
                  {run.tag || run.id.slice(0, 8)}
                </p>
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <span className="capitalize">{run.status}</span>
                </div>
              </div>

              {/* Timestamp */}
              <span className="text-[10px] text-muted-foreground flex-shrink-0">
                {formatRelativeTime(run.startedAt)}
              </span>
            </button>
          ))
        )}
      </div>
    </div>
  )
}
