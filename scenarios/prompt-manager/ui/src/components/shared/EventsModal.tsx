/**
 * EventsModal - Displays run details and event stream for a heartbeat run.
 *
 * Two tabs:
 * - Info: Run metadata (timestamps, duration, status, team/member links)
 * - Events: Event stream by type (message, tool_call, tool_result, status, metric, log, error)
 *
 * Supports live polling for currently-running agents with auto-scroll.
 */

import { useState, useEffect, useRef, useCallback } from 'react'
import * as Tabs from '@radix-ui/react-tabs'
import { Loader2, ChevronDown, ChevronRight, Copy, Check, Clock, Timer, Activity, AlertCircle, Users, User, Calendar, Key, Hash } from 'lucide-react'
import { Dialog } from './Dialog'
import { MarkdownRenderer } from '@/components/markdown/MarkdownRenderer'
import { getRunEvents, getRunDetails, type RunEvent, type RunDetails } from '@/services/heartbeatService'
import { cn } from '@/lib/utils'

interface EventsModalProps {
  isOpen: boolean
  onClose: () => void
  runId: string
  agentName?: string
  /** When true, poll for new events every 3s */
  live?: boolean
  /** Error context from the heartbeat execution (shown when no events exist) */
  errorMessage?: string
  // Context for Info tab
  teamId?: string
  agentId?: string
  teamName?: string
  schedule?: string
  profileKey?: string
  /** Navigate to team view */
  onNavigateToTeam?: (teamId: string) => void
  /** Navigate to member view */
  onNavigateToMember?: (teamId: string, agentId: string) => void
}

export function EventsModal({
  isOpen, onClose, runId, agentName, live, errorMessage,
  teamId, agentId, teamName, schedule, profileKey,
  onNavigateToTeam, onNavigateToMember,
}: EventsModalProps) {
  const [events, setEvents] = useState<RunEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [autoScroll, setAutoScroll] = useState(true)
  const listRef = useRef<HTMLDivElement>(null)
  const lastSequenceRef = useRef<number>(-1)

  // Run details for Info tab
  const [runDetails, setRunDetails] = useState<RunDetails | null>(null)
  const [runDetailsLoading, setRunDetailsLoading] = useState(true)
  const [runDetailsError, setRunDetailsError] = useState<string | null>(null)

  // Reset state when runId changes or modal opens
  useEffect(() => {
    if (!isOpen) return
    setEvents([])
    setLoading(true)
    setError(null)
    setAutoScroll(true)
    lastSequenceRef.current = -1
    setRunDetails(null)
    setRunDetailsLoading(true)
    setRunDetailsError(null)
  }, [isOpen, runId])

  // Fetch run details
  useEffect(() => {
    if (!isOpen) return
    let cancelled = false
    void (async () => {
      try {
        const details = await getRunDetails(runId)
        if (!cancelled) {
          setRunDetails(details)
          setRunDetailsError(null)
        }
      } catch (err) {
        if (!cancelled) {
          setRunDetailsError(err instanceof Error ? err.message : 'Failed to load run details')
        }
      } finally {
        if (!cancelled) setRunDetailsLoading(false)
      }
    })()
    return () => { cancelled = true }
  }, [isOpen, runId])

  // Fetch events
  const fetchEvents = useCallback(async (incremental: boolean) => {
    try {
      const data = await getRunEvents(runId, {
        afterSequence: incremental ? lastSequenceRef.current : undefined,
      })
      if (data.length > 0) {
        lastSequenceRef.current = Math.max(...data.map((e) => e.sequence))
        setEvents((prev) => incremental ? [...prev, ...data] : data)
      }
      setError(null)
    } catch (err) {
      if (!incremental) {
        setError(err instanceof Error ? err.message : 'Failed to load events')
      }
    } finally {
      setLoading(false)
    }
  }, [runId])

  // Initial fetch
  useEffect(() => {
    if (!isOpen) return
    void fetchEvents(false)
  }, [isOpen, fetchEvents])

  // Live polling
  useEffect(() => {
    if (!isOpen || !live) return
    const interval = setInterval(() => void fetchEvents(true), 3000)
    return () => clearInterval(interval)
  }, [isOpen, live, fetchEvents])

  // Auto-scroll to bottom on new events
  useEffect(() => {
    if (autoScroll && listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight
    }
  }, [events, autoScroll])

  // Pause auto-scroll if user scrolls up
  const handleScroll = useCallback(() => {
    if (!listRef.current) return
    const { scrollTop, scrollHeight, clientHeight } = listRef.current
    setAutoScroll(scrollHeight - scrollTop - clientHeight < 40)
  }, [])

  const title = agentName ? `${agentName} — Heartbeat Run` : 'Heartbeat Run'

  return (
    <Dialog isOpen={isOpen} onClose={onClose} title={title} maxWidth="max-w-2xl">
      <Tabs.Root defaultValue="info">
        <Tabs.List className="flex border-b border-border mb-4">
          <Tabs.Trigger
            value="info"
            className={cn(
              'flex items-center gap-1.5 px-3 py-2 text-sm font-medium',
              'border-b-2 transition-colors',
              'data-[state=active]:border-primary data-[state=active]:text-primary',
              'data-[state=inactive]:border-transparent data-[state=inactive]:text-muted-foreground',
              'hover:text-foreground'
            )}
          >
            Info
          </Tabs.Trigger>
          <Tabs.Trigger
            value="events"
            className={cn(
              'flex items-center gap-1.5 px-3 py-2 text-sm font-medium',
              'border-b-2 transition-colors',
              'data-[state=active]:border-primary data-[state=active]:text-primary',
              'data-[state=inactive]:border-transparent data-[state=inactive]:text-muted-foreground',
              'hover:text-foreground'
            )}
          >
            Events
            {live && (
              <span className="relative flex h-2 w-2 ml-1">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
                <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500" />
              </span>
            )}
          </Tabs.Trigger>
        </Tabs.List>

        {/* Info Tab */}
        <Tabs.Content value="info">
          <InfoTabContent
            runDetails={runDetails}
            loading={runDetailsLoading}
            error={runDetailsError}
            live={live}
            agentName={agentName}
            teamId={teamId}
            agentId={agentId}
            teamName={teamName}
            schedule={schedule}
            profileKey={profileKey}
            errorMessage={errorMessage}
            onNavigateToTeam={onNavigateToTeam ? (id) => { onNavigateToTeam(id); onClose() } : undefined}
            onNavigateToMember={onNavigateToMember ? (tId, aId) => { onNavigateToMember(tId, aId); onClose() } : undefined}
          />
        </Tabs.Content>

        {/* Events Tab */}
        <Tabs.Content value="events">
          {/* Header row: live indicator + copy all */}
          <div className="flex items-center justify-between mb-4">
            {live ? (
              <div className="flex items-center gap-2">
                <span className="relative flex h-2.5 w-2.5">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
                  <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-emerald-500" />
                </span>
                <span className="text-xs font-medium text-emerald-400">Live</span>
              </div>
            ) : <div />}
            {events.length > 0 && (
              <CopyButton
                text={JSON.stringify(events, null, 2)}
                label="Copy JSON"
                className="px-2 py-1 rounded bg-slate-800 hover:bg-slate-700"
              />
            )}
          </div>

          {/* Event list */}
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : error ? (
            <p className="text-sm text-destructive py-4">{error}</p>
          ) : events.length === 0 ? (
            <div className="py-4 space-y-3">
              <p className="text-sm text-muted-foreground">No events recorded for this run.</p>
              {errorMessage && (
                <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3">
                  <p className="text-xs font-semibold uppercase tracking-wider text-red-400 mb-1">Execution Error</p>
                  <p className="text-sm text-red-300 whitespace-pre-wrap">{errorMessage}</p>
                </div>
              )}
              {!errorMessage && !live && (
                <p className="text-xs text-muted-foreground/70">
                  The run may have failed before producing events. Check the Recent Heartbeats section for error details.
                </p>
              )}
            </div>
          ) : (
            <div
              ref={listRef}
              onScroll={handleScroll}
              className="space-y-2 max-h-[65vh] overflow-y-auto pr-1"
            >
              {events.map((event) => (
                <EventRow key={event.id || `${event.runId}-${event.sequence}`} event={event} />
              ))}
            </div>
          )}
        </Tabs.Content>
      </Tabs.Root>
    </Dialog>
  )
}

// ============================================================================
// Info Tab
// ============================================================================

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
  if (hrs > 0) return `${hrs}h ${mins % 60}m`
  if (mins > 0) return `${mins}m ${secs % 60}s`
  return `${secs}s`
}

const STATUS_COLORS: Record<string, string> = {
  completed: 'bg-emerald-500',
  running: 'bg-amber-500 animate-pulse',
  failed: 'bg-red-500',
  cancelled: 'bg-slate-400',
  pending: 'bg-blue-400',
}

interface InfoTabContentProps {
  runDetails: RunDetails | null
  loading: boolean
  error: string | null
  live?: boolean
  agentName?: string
  teamId?: string
  agentId?: string
  teamName?: string
  schedule?: string
  profileKey?: string
  errorMessage?: string
  onNavigateToTeam?: (teamId: string) => void
  onNavigateToMember?: (teamId: string, agentId: string) => void
}

function InfoTabContent({
  runDetails, loading, error: detailsError, live,
  agentName, teamId, agentId, teamName, schedule, profileKey, errorMessage,
  onNavigateToTeam, onNavigateToMember,
}: InfoTabContentProps) {
  // Live elapsed time ticker
  const [elapsed, setElapsed] = useState('')
  const startedAt = runDetails?.startedAt
  const endedAt = runDetails?.endedAt
  const isRunning = live || runDetails?.status === 'running'

  useEffect(() => {
    if (!isRunning || !startedAt) return
    const tick = () => setElapsed(computeDuration(startedAt) ?? '')
    tick()
    const interval = setInterval(tick, 1000)
    return () => clearInterval(interval)
  }, [isRunning, startedAt])

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (detailsError) {
    return <p className="text-sm text-destructive py-4">{detailsError}</p>
  }

  const status = runDetails?.status ?? (live ? 'running' : 'unknown')
  const duration = isRunning ? elapsed : computeDuration(startedAt, endedAt)
  const runError = runDetails?.error ?? errorMessage

  return (
    <dl className="grid gap-3">
      <InfoRow
        icon={<Clock className="h-4 w-4" />}
        label="Started"
        value={formatTimestamp(startedAt)}
      />
      <InfoRow
        icon={<Timer className="h-4 w-4" />}
        label="Duration"
        value={
          isRunning && duration
            ? <span className="text-amber-400">{duration} (running...)</span>
            : duration ?? 'N/A'
        }
      />
      <InfoRow
        icon={<Activity className="h-4 w-4" />}
        label="Status"
        value={
          <span className="inline-flex items-center gap-2">
            <span className={cn('inline-block h-2 w-2 rounded-full flex-shrink-0', STATUS_COLORS[status] ?? 'bg-slate-400')} />
            <span className="capitalize">{status}</span>
          </span>
        }
      />
      {runError && (
        <InfoRow
          icon={<AlertCircle className="h-4 w-4 text-red-400" />}
          label="Error"
          value={<span className="text-red-400 whitespace-pre-wrap">{runError}</span>}
        />
      )}
      {teamName && teamId && (
        <InfoRow
          icon={<Users className="h-4 w-4" />}
          label="Team"
          value={
            onNavigateToTeam ? (
              <button
                type="button"
                onClick={() => onNavigateToTeam(teamId)}
                className="text-primary hover:underline"
              >
                {teamName}
              </button>
            ) : teamName
          }
        />
      )}
      {agentName && teamId && agentId && (
        <InfoRow
          icon={<User className="h-4 w-4" />}
          label="Member"
          value={
            onNavigateToMember ? (
              <button
                type="button"
                onClick={() => onNavigateToMember(teamId, agentId)}
                className="text-primary hover:underline"
              >
                {agentName}
              </button>
            ) : agentName
          }
        />
      )}
      {schedule && (
        <InfoRow
          icon={<Calendar className="h-4 w-4" />}
          label="Schedule"
          value={<span className="font-mono text-xs">{schedule}</span>}
        />
      )}
      {profileKey && (
        <InfoRow
          icon={<Key className="h-4 w-4" />}
          label="Profile"
          value={<span className="font-mono text-xs">{profileKey}</span>}
        />
      )}
      <InfoRow
        icon={<Hash className="h-4 w-4" />}
        label="Run ID"
        value={
          <span className="inline-flex items-center gap-1.5">
            <span className="font-mono text-xs truncate max-w-[200px]" title={runDetails?.id}>
              {runDetails?.id ?? 'N/A'}
            </span>
            {runDetails?.id && <CopyButton text={runDetails.id} />}
          </span>
        }
      />
    </dl>
  )
}

function InfoRow({ icon, label, value }: { icon: React.ReactNode; label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-center gap-3">
      <div className="text-muted-foreground">{icon}</div>
      <dt className="text-sm text-muted-foreground min-w-[80px]">{label}</dt>
      <dd className="text-sm flex-1">{value}</dd>
    </div>
  )
}

// ============================================================================
// Shared Components
// ============================================================================

/** Copy-to-clipboard button with check feedback. */
function CopyButton({ text, label, className }: { text: string; label?: string; className?: string }) {
  const [copied, setCopied] = useState(false)
  const handleCopy = useCallback(() => {
    void navigator.clipboard.writeText(text).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    })
  }, [text])
  return (
    <button
      type="button"
      onClick={handleCopy}
      className={cn(
        'inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-slate-200 transition-colors',
        className,
      )}
      title={label ?? 'Copy'}
    >
      {copied ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
      {label && <span>{copied ? 'Copied' : label}</span>}
    </button>
  )
}

// ============================================================================
// Event Rendering
// ============================================================================

/** Safely stringify an unknown value for display. */
function str(v: unknown, fallback = ''): string {
  if (v == null) return fallback
  if (typeof v === 'string') return v
  if (typeof v === 'number' || typeof v === 'boolean') return String(v)
  return JSON.stringify(v)
}

function EventRow({ event }: { event: RunEvent }) {
  switch (event.eventType) {
    case 'message':
      return <MessageEvent event={event} />
    case 'tool_call':
      return <ToolCallEvent event={event} />
    case 'tool_result':
      return <ToolResultEvent event={event} />
    case 'status':
      return <StatusEvent event={event} />
    case 'metric':
      return <MetricEvent event={event} />
    case 'log':
      return <LogEvent event={event} />
    case 'error':
      return <ErrorEvent event={event} />
    default:
      return <GenericEvent event={event} />
  }
}

function MessageEvent({ event }: { event: RunEvent }) {
  const role = str(event.data.role, 'unknown')
  const content = str(event.data.content)
  return (
    <div className="rounded-lg bg-slate-800/50 p-3">
      <div className="flex items-center justify-between">
        <span className={cn(
          'text-[10px] font-semibold uppercase tracking-wider',
          role === 'assistant' ? 'text-blue-400' : 'text-slate-400',
        )}>
          {role}
        </span>
        <CopyButton text={content} />
      </div>
      <div className="text-sm text-slate-200 mt-1">
        <MarkdownRenderer content={content} />
      </div>
    </div>
  )
}

function ToolCallEvent({ event }: { event: RunEvent }) {
  const [open, setOpen] = useState(false)
  const toolName = str(event.data.tool_name ?? event.data.name, 'tool')
  const input = event.data.input ?? event.data.arguments
  return (
    <div className="rounded-lg border border-amber-500/20 bg-amber-500/5 p-3">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 w-full text-left"
      >
        {open ? <ChevronDown className="h-3 w-3 text-amber-400" /> : <ChevronRight className="h-3 w-3 text-amber-400" />}
        <span className="text-xs font-medium bg-amber-500/20 text-amber-300 px-1.5 py-0.5 rounded">{toolName}</span>
        <span className="text-[10px] text-muted-foreground">tool call</span>
      </button>
      {open && input != null && (
        <pre className="text-xs text-slate-300 mt-2 overflow-x-auto whitespace-pre-wrap font-mono bg-slate-900/50 p-2 rounded">
          {typeof input === 'string' ? input : JSON.stringify(input, null, 2)}
        </pre>
      )}
    </div>
  )
}

function ToolResultEvent({ event }: { event: RunEvent }) {
  const [open, setOpen] = useState(false)
  const success = event.data.success !== false && !event.data.error
  const output = event.data.output ?? event.data.result ?? event.data.content
  return (
    <div className={cn(
      'rounded-lg border p-3',
      success ? 'border-emerald-500/20 bg-emerald-500/5' : 'border-red-500/20 bg-red-500/5',
    )}>
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 w-full text-left"
      >
        {open ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
        <span className={cn('text-xs font-medium', success ? 'text-emerald-400' : 'text-red-400')}>
          {success ? 'Success' : 'Failed'}
        </span>
        <span className="text-[10px] text-muted-foreground">tool result</span>
      </button>
      {open && output != null && (
        <pre className="text-xs text-slate-300 mt-2 overflow-x-auto whitespace-pre-wrap font-mono bg-slate-900/50 p-2 rounded">
          {typeof output === 'string' ? output : JSON.stringify(output, null, 2)}
        </pre>
      )}
    </div>
  )
}

function StatusEvent({ event }: { event: RunEvent }) {
  const status = str(event.data.status ?? event.data.new_status)
  const from = str(event.data.from_status ?? event.data.old_status)
  return (
    <div className="flex justify-center py-1">
      <span className="text-xs bg-slate-700 text-slate-300 px-3 py-1 rounded-full">
        {from ? `${from} → ${status}` : status}
      </span>
    </div>
  )
}

function MetricEvent({ event }: { event: RunEvent }) {
  const entries = Object.entries(event.data).filter(([k]) => k !== 'type')
  return (
    <div className="text-xs text-muted-foreground px-3 py-1 flex gap-4 flex-wrap">
      {entries.map(([k, v]) => (
        <span key={k}><span className="text-slate-500">{k}:</span> {str(v)}</span>
      ))}
    </div>
  )
}

function LogEvent({ event }: { event: RunEvent }) {
  const level = str(event.data.level, 'info')
  const message = str(event.data.message ?? event.data.content)
  return (
    <div className={cn(
      'font-mono text-xs px-3 py-1.5 rounded',
      level === 'error' && 'text-red-400 bg-red-500/5',
      level === 'warn' && 'text-amber-400 bg-amber-500/5',
      (level === 'info' || level !== 'error' && level !== 'warn') && 'text-slate-400',
    )}>
      {message}
    </div>
  )
}

function ErrorEvent({ event }: { event: RunEvent }) {
  const message = str(event.data.message ?? event.data.error, 'Unknown error')
  return (
    <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3">
      <p className="text-sm text-red-400 font-medium">{message}</p>
    </div>
  )
}

function GenericEvent({ event }: { event: RunEvent }) {
  return (
    <div className="text-xs text-muted-foreground px-3 py-1">
      <span className="text-slate-500">[{event.eventType}]</span>{' '}
      {JSON.stringify(event.data)}
    </div>
  )
}
