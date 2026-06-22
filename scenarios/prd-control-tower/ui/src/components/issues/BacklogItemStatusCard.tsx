import { useEffect, useRef, useState } from 'react'
import { CheckCircle2, ExternalLink, Loader2, ListOrdered, AlertCircle, Copy } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card'
import { Button } from '../ui/button'
import { Badge } from '../ui/badge'
import { cn } from '../../lib/utils'
import { fetchBacklogItemStatus, type BacklogFeedback } from '../../services/issues'

// BacklogItemStatusCard renders the per-item feedback contract for a backlog
// item filed into swarm-manager: live status, queue position ("N items ahead"),
// a deep link into swarm-manager's own UI, and a dedup notice. It polls the
// consumer's own /issues/status endpoint and stops once the item is terminal.
// Thin presentation — recreated per scenario; deliberately NO time-based ETA
// (queue position is the honest signal in a deep variable-runtime queue).

const TERMINAL_STATUSES = new Set(['completed', 'failed', 'needs_followup'])
const POLL_INTERVAL_MS = 5000

type StatusTone = 'pending' | 'active' | 'done' | 'failed'

function toneForStatus(status: string): StatusTone {
  switch (status) {
    case 'completed':
      return 'done'
    case 'failed':
      return 'failed'
    case 'in_progress':
    case 'in_review':
    case 'review_pending':
      return 'active'
    default:
      return 'pending'
  }
}

const TONE_BADGE: Record<StatusTone, string> = {
  pending: 'bg-slate-100 text-slate-700 border-slate-200',
  active: 'bg-sky-100 text-sky-800 border-sky-200',
  done: 'bg-emerald-100 text-emerald-800 border-emerald-200',
  failed: 'bg-rose-100 text-rose-800 border-rose-200',
}

const STATUS_LABEL: Record<string, string> = {
  backlog: 'In backlog',
  researching: 'Researching',
  ready: 'Ready',
  queued: 'Queued',
  in_progress: 'In progress',
  in_review: 'In review',
  review_pending: 'Awaiting review',
  completed: 'Completed',
  failed: 'Failed',
  needs_followup: 'Needs follow-up',
}

export interface BacklogItemStatusCardProps {
  feedback: BacklogFeedback
  /** Poll the status endpoint for live updates. Default true. */
  poll?: boolean
  className?: string
}

export function BacklogItemStatusCard({ feedback, poll = true, className }: BacklogItemStatusCardProps) {
  const [current, setCurrent] = useState<BacklogFeedback>(feedback)
  const [error, setError] = useState<string | null>(null)
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Reset when a new item is reported.
  useEffect(() => {
    setCurrent(feedback)
    setError(null)
  }, [feedback])

  useEffect(() => {
    if (!poll) return
    if (TERMINAL_STATUSES.has(current.status)) return

    let cancelled = false
    const tick = async () => {
      try {
        const next = await fetchBacklogItemStatus(current.kind, current.name)
        if (cancelled) return
        setCurrent(next)
        setError(null)
        if (!TERMINAL_STATUSES.has(next.status)) {
          timer.current = setTimeout(tick, POLL_INTERVAL_MS)
        }
      } catch (e) {
        if (cancelled) return
        setError(e instanceof Error ? e.message : 'Failed to refresh status')
        timer.current = setTimeout(tick, POLL_INTERVAL_MS)
      }
    }
    timer.current = setTimeout(tick, POLL_INTERVAL_MS)
    return () => {
      cancelled = true
      if (timer.current) clearTimeout(timer.current)
    }
  }, [poll, current.kind, current.name, current.status])

  const tone = toneForStatus(current.status)
  const isTerminal = TERMINAL_STATUSES.has(current.status)
  const position = current.queue_position

  return (
    <Card className={cn('border-slate-200', className)} data-testid="backlog-item-status-card">
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-base">
          {isTerminal ? (
            <CheckCircle2 className="h-4 w-4 text-emerald-600" />
          ) : (
            <Loader2 className="h-4 w-4 animate-spin text-sky-600" />
          )}
          <span>Backlog item filed</span>
          <Badge variant="outline" className={cn('ml-auto', TONE_BADGE[tone])}>
            {STATUS_LABEL[current.status] ?? current.status}
          </Badge>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3 text-sm">
        {current.deduped && (
          <div className="flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50 p-2 text-amber-800">
            <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
            <span>This matched an existing open item — your report was merged into it.</span>
          </div>
        )}

        <div className="flex items-center gap-2 text-slate-600">
          <span className="font-mono text-xs">{current.item_id}</span>
          <button
            type="button"
            className="text-slate-400 hover:text-slate-600"
            onClick={() => void navigator.clipboard?.writeText(current.item_id)}
            aria-label="Copy item id"
          >
            <Copy className="h-3.5 w-3.5" />
          </button>
        </div>

        {typeof position === 'number' && position !== null && (
          <div className="flex items-center gap-2 text-slate-700">
            <ListOrdered className="h-4 w-4 text-slate-500" />
            <span>
              {position === 0 ? 'Next up in the queue' : `${position} item${position === 1 ? '' : 's'} ahead in the queue`}
            </span>
          </div>
        )}

        {error && <div className="text-xs text-rose-600">{error}</div>}

        <Button asChild variant="outline" size="sm" className="w-full">
          <a href={current.deep_link} target="_blank" rel="noopener noreferrer">
            Open in Swarm Manager
            <ExternalLink className="ml-1.5 h-3.5 w-3.5" />
          </a>
        </Button>
      </CardContent>
    </Card>
  )
}
