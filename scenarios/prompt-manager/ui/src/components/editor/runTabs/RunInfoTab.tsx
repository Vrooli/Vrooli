/**
 * RunInfoTab - Run metadata display tab.
 *
 * Displays run info in grouped card sections:
 * - Status Overview (badge, duration, timestamps)
 * - Execution Details (tag, profile, session ID, run ID)
 * - Error Card (conditional, red-bordered)
 */

import { useState, useEffect } from 'react'
import { Clock, Timer, Tag, Key, Calendar, Hash, AlertCircle, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { getRunDetails, retryRun, type RunDetails } from '@/services/heartbeatService'
import { CopyButton } from '@/components/shared/EventsDisplay'
import { toast } from '@/hooks/use-toast'

interface RunInfoTabProps {
  runId: string
  className?: string
}

const STATUS_STYLES: Record<string, { bg: string; text: string; border: string }> = {
  completed: { bg: 'bg-emerald-500/15', text: 'text-emerald-400', border: 'border-emerald-500/30' },
  running: { bg: 'bg-amber-500/15', text: 'text-amber-400', border: 'border-amber-500/30' },
  failed: { bg: 'bg-red-500/15', text: 'text-red-400', border: 'border-red-500/30' },
  cancelled: { bg: 'bg-slate-500/15', text: 'text-slate-400', border: 'border-slate-500/30' },
  pending: { bg: 'bg-blue-500/15', text: 'text-blue-400', border: 'border-blue-500/30' },
}

const STATUS_DOT: Record<string, string> = {
  completed: 'bg-emerald-500',
  running: 'bg-amber-500 animate-pulse',
  failed: 'bg-red-500',
  cancelled: 'bg-slate-400',
  pending: 'bg-blue-400',
}

function formatTimestamp(iso?: string): string {
  if (!iso) return 'Unknown'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}

function computeDuration(startedAt?: string, endedAt?: string): string | null {
  if (!startedAt) return null
  const start = new Date(startedAt).getTime()
  if (Number.isNaN(start)) return null
  const end = endedAt ? new Date(endedAt).getTime() : Date.now()
  if (Number.isNaN(end)) return null
  const diffMs = end - start
  if (diffMs < 0) return null
  const secs = Math.floor(diffMs / 1000)
  const mins = Math.floor(secs / 60)
  const hrs = Math.floor(mins / 60)
  if (hrs > 0) return `${hrs}h ${mins % 60}m ${secs % 60}s`
  if (mins > 0) return `${mins}m ${secs % 60}s`
  return `${secs}s`
}

function truncateId(id: string, chars = 8): string {
  if (id.length <= chars + 3) return id
  return `${id.slice(0, chars)}...`
}

export function RunInfoTab({ runId, className }: RunInfoTabProps) {
  const [runDetails, setRunDetails] = useState<RunDetails | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [elapsed, setElapsed] = useState('')
  const [retrying, setRetrying] = useState(false)

  // Fetch run details
  useEffect(() => {
    const controller = new AbortController()
    setLoading(true)
    setError(null)

    void (async () => {
      try {
        const details = await getRunDetails(runId)
        if (controller.signal.aborted) return
        setRunDetails(details)
      } catch (err) {
        if (controller.signal.aborted) return
        setError(err instanceof Error ? err.message : 'Failed to load run details')
      } finally {
        if (!controller.signal.aborted) setLoading(false)
      }
    })()

    return () => controller.abort()
  }, [runId])

  // Live elapsed time ticker
  const isRunning = runDetails?.status === 'running'
  const startedAt = runDetails?.startedAt

  useEffect(() => {
    if (!isRunning || !startedAt) return
    const tick = () => setElapsed(computeDuration(startedAt) ?? '')
    tick()
    const interval = setInterval(tick, 1000)
    return () => clearInterval(interval)
  }, [isRunning, startedAt])

  if (loading) {
    return (
      <div className={cn('flex items-center justify-center py-12', className)}>
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error) {
    return <p className={cn('text-sm text-destructive py-4', className)}>{error}</p>
  }

  const status = runDetails?.status ?? 'unknown'
  const styles = STATUS_STYLES[status] ?? { bg: 'bg-slate-500/15', text: 'text-slate-400', border: 'border-slate-500/30' }
  const duration = isRunning ? elapsed : computeDuration(startedAt, runDetails?.endedAt)
  const canRetry = runDetails?.actions?.canRetry ?? !!runDetails?.error

  const handleRetry = async () => {
    if (!runDetails?.id) return
    setRetrying(true)
    try {
      const result = await retryRun(runDetails.id)
      toast({
        title: 'Retry triggered',
        description: result.runId ? `Started run ${truncateId(result.runId, 12)}` : undefined,
        variant: 'success',
      })
    } catch (err) {
      toast({
        title: 'Failed to retry run',
        description: err instanceof Error ? err.message : 'Unknown error',
        variant: 'destructive',
      })
    } finally {
      setRetrying(false)
    }
  }

  return (
    <div className={cn('space-y-4', className)}>
      {/* Status Overview */}
      <section className="bg-muted/30 rounded-lg p-4">
        <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Status Overview</h4>
        <div className="flex items-center gap-3 mb-3">
          <span className={cn('inline-flex items-center gap-2 px-3 py-1 rounded-full text-sm font-medium border', styles.bg, styles.text, styles.border)}>
            <span className={cn('inline-block h-2 w-2 rounded-full flex-shrink-0', STATUS_DOT[status] ?? 'bg-slate-400')} />
            <span className="capitalize">{status}</span>
          </span>
        </div>
        {duration && (
          <div className="mb-3">
            <div className="flex items-center gap-2 text-muted-foreground mb-1">
              <Timer className="h-3.5 w-3.5" />
              <span className="text-xs">Duration</span>
            </div>
            <p className={cn('text-2xl font-semibold tabular-nums', isRunning && 'text-amber-400')}>
              {duration}
              {isRunning && <span className="text-sm font-normal ml-1.5 text-amber-400/70">(running)</span>}
            </p>
          </div>
        )}
        <div className="grid gap-1.5 text-sm">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Clock className="h-3.5 w-3.5" />
            <span className="text-xs min-w-[52px]">Started</span>
            <span className="text-foreground text-xs">{formatTimestamp(startedAt)}</span>
          </div>
          {runDetails?.endedAt && (
            <div className="flex items-center gap-2 text-muted-foreground">
              <Clock className="h-3.5 w-3.5" />
              <span className="text-xs min-w-[52px]">Ended</span>
              <span className="text-foreground text-xs">{formatTimestamp(runDetails.endedAt)}</span>
            </div>
          )}
        </div>
      </section>

      {/* Execution Details */}
      <section className="bg-muted/30 rounded-lg p-4">
        <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">Execution Details</h4>
        <dl className="grid gap-2.5">
          {runDetails?.tag && (
            <div className="flex items-center gap-3">
              <Tag className="h-4 w-4 text-muted-foreground" />
              <dt className="text-xs text-muted-foreground min-w-[72px]">Tag</dt>
              <dd className="text-sm font-mono">{runDetails.tag}</dd>
            </div>
          )}
          {runDetails?.profileId && (
            <div className="flex items-center gap-3">
              <Key className="h-4 w-4 text-muted-foreground" />
              <dt className="text-xs text-muted-foreground min-w-[72px]">Profile</dt>
              <dd className="text-sm font-mono">{runDetails.profileId}</dd>
            </div>
          )}
          {runDetails?.sessionId && (
            <div className="flex items-center gap-3">
              <Calendar className="h-4 w-4 text-muted-foreground" />
              <dt className="text-xs text-muted-foreground min-w-[72px]">Session</dt>
              <dd className="text-sm flex items-center gap-1.5">
                <span className="font-mono text-xs" title={runDetails.sessionId}>
                  {truncateId(runDetails.sessionId)}
                </span>
                <CopyButton text={runDetails.sessionId} />
              </dd>
            </div>
          )}
          <div className="flex items-center gap-3">
            <Hash className="h-4 w-4 text-muted-foreground" />
            <dt className="text-xs text-muted-foreground min-w-[72px]">Run ID</dt>
            <dd className="text-sm flex items-center gap-1.5">
              <span className="font-mono text-xs" title={runDetails?.id}>
                {runDetails?.id ? truncateId(runDetails.id) : 'N/A'}
              </span>
              {runDetails?.id && <CopyButton text={runDetails.id} />}
            </dd>
          </div>
        </dl>
      </section>

      {/* Error Card */}
      {runDetails?.error && (
        <section className="bg-red-500/10 border border-red-500/30 rounded-lg p-4">
          <div className="flex items-center gap-2 mb-2">
            <AlertCircle className="h-4 w-4 text-red-400" />
            <h4 className="text-xs font-semibold uppercase tracking-wider text-red-400">Error</h4>
            <div className="ml-auto flex items-center gap-2">
              <button
                type="button"
                onClick={() => void handleRetry()}
                disabled={!canRetry || retrying}
                className={cn(
                  'px-2.5 py-1 text-xs rounded-md border transition-colors',
                  canRetry && !retrying
                    ? 'border-red-400/40 text-red-300 hover:bg-red-500/20'
                    : 'border-red-900/40 text-red-500/60 cursor-not-allowed'
                )}
              >
                {retrying ? 'Retrying...' : 'Retry'}
              </button>
              <CopyButton text={runDetails.error} />
            </div>
          </div>
          <p className="text-sm text-red-400 whitespace-pre-wrap font-mono">{runDetails.error}</p>
        </section>
      )}
    </div>
  )
}
