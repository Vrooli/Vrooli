/**
 * EventsDisplay - Reusable event stream renderer for run events.
 *
 * Supports:
 * - Self-fetching mode (fetches via getRunEvents when no events prop provided)
 * - Pass-through mode (renders provided events directly)
 * - Live polling (3s interval when live=true)
 * - Auto-scroll with pause-on-scroll
 * - Filter by event types
 * - Text search filtering
 */

import { useState, useEffect, useRef, useCallback } from 'react'
import { Loader2, ChevronDown, ChevronRight, Copy, Check } from 'lucide-react'
import { MarkdownRenderer } from '@/components/markdown/MarkdownRenderer'
import { getRunEvents, type RunEvent } from '@/services/heartbeatService'
import { cn } from '@/lib/utils'

// ============================================================================
// Helpers
// ============================================================================

/** Safely stringify an unknown value for display. */
function str(v: unknown, fallback = ''): string {
  if (v == null) return fallback
  if (typeof v === 'string') return v
  if (typeof v === 'number' || typeof v === 'boolean') return String(v)
  return JSON.stringify(v)
}

// ============================================================================
// CopyButton
// ============================================================================

/** Copy-to-clipboard button with check feedback. */
export function CopyButton({ text, label, className }: { text: string; label?: string; className?: string }) {
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
// Event Type Components
// ============================================================================

export function EventRow({ event }: { event: RunEvent }) {
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

export function MessageEvent({ event }: { event: RunEvent }) {
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

export function ToolCallEvent({ event }: { event: RunEvent }) {
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

export function ToolResultEvent({ event }: { event: RunEvent }) {
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

export function StatusEvent({ event }: { event: RunEvent }) {
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

export function MetricEvent({ event }: { event: RunEvent }) {
  const entries = Object.entries(event.data).filter(([k]) => k !== 'type')
  return (
    <div className="text-xs text-muted-foreground px-3 py-1 flex gap-4 flex-wrap">
      {entries.map(([k, v]) => (
        <span key={k}><span className="text-slate-500">{k}:</span> {str(v)}</span>
      ))}
    </div>
  )
}

export function LogEvent({ event }: { event: RunEvent }) {
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

export function ErrorEvent({ event }: { event: RunEvent }) {
  const message = str(event.data.message ?? event.data.error, 'Unknown error')
  return (
    <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3">
      <p className="text-sm text-red-400 font-medium">{message}</p>
    </div>
  )
}

export function GenericEvent({ event }: { event: RunEvent }) {
  return (
    <div className="text-xs text-muted-foreground px-3 py-1">
      <span className="text-slate-500">[{event.eventType}]</span>{' '}
      {JSON.stringify(event.data)}
    </div>
  )
}

// ============================================================================
// EventsDisplay Component
// ============================================================================

interface EventsDisplayProps {
  /** Run ID to fetch events for (required for self-fetching mode) */
  runId?: string
  /** When true, poll for new events every 3s */
  live?: boolean
  /** Pass events directly instead of self-fetching */
  events?: RunEvent[]
  /** Filter to specific event types */
  filterTypes?: RunEvent['eventType'][]
  /** Text search query to filter events */
  searchQuery?: string
  /** Additional CSS class */
  className?: string
  /** Error context shown when no events exist */
  errorMessage?: string
}

export function EventsDisplay({
  runId,
  live,
  events: externalEvents,
  filterTypes,
  searchQuery,
  className,
  errorMessage,
}: EventsDisplayProps) {
  const [fetchedEvents, setFetchedEvents] = useState<RunEvent[]>([])
  const [loading, setLoading] = useState(!externalEvents)
  const [error, setError] = useState<string | null>(null)
  const [autoScroll, setAutoScroll] = useState(true)
  const listRef = useRef<HTMLDivElement>(null)
  const lastSequenceRef = useRef<number>(-1)

  const selfFetching = !externalEvents && !!runId

  // Reset state when runId changes
  useEffect(() => {
    if (!selfFetching) return
    setFetchedEvents([])
    setLoading(true)
    setError(null)
    setAutoScroll(true)
    lastSequenceRef.current = -1
  }, [selfFetching, runId])

  // Fetch events
  const fetchEvents = useCallback(async (incremental: boolean) => {
    if (!runId) return
    try {
      const data = await getRunEvents(runId, {
        afterSequence: incremental ? lastSequenceRef.current : undefined,
      })
      if (data.length > 0) {
        lastSequenceRef.current = Math.max(...data.map((e) => e.sequence))
        setFetchedEvents((prev) => incremental ? [...prev, ...data] : data)
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
    if (!selfFetching) return
    void fetchEvents(false)
  }, [selfFetching, fetchEvents])

  // Live polling
  useEffect(() => {
    if (!selfFetching || !live) return
    const interval = setInterval(() => void fetchEvents(true), 3000)
    return () => clearInterval(interval)
  }, [selfFetching, live, fetchEvents])

  // Auto-scroll to bottom on new events
  const allEvents = externalEvents ?? fetchedEvents
  useEffect(() => {
    if (autoScroll && listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight
    }
  }, [allEvents, autoScroll])

  // Pause auto-scroll if user scrolls up
  const handleScroll = useCallback(() => {
    if (!listRef.current) return
    const { scrollTop, scrollHeight, clientHeight } = listRef.current
    setAutoScroll(scrollHeight - scrollTop - clientHeight < 40)
  }, [])

  // Apply filters
  let filteredEvents = allEvents
  if (filterTypes && filterTypes.length > 0) {
    filteredEvents = filteredEvents.filter((e) => filterTypes.includes(e.eventType))
  }
  if (searchQuery) {
    const q = searchQuery.toLowerCase()
    filteredEvents = filteredEvents.filter((e) => {
      const dataStr = JSON.stringify(e.data).toLowerCase()
      return dataStr.includes(q) || e.eventType.toLowerCase().includes(q)
    })
  }

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

  if (filteredEvents.length === 0) {
    return (
      <div className={cn('py-4 space-y-3', className)}>
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
    )
  }

  return (
    <div className={className}>
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
        <CopyButton
          text={JSON.stringify(filteredEvents, null, 2)}
          label="Copy JSON"
          className="px-2 py-1 rounded bg-slate-800 hover:bg-slate-700"
        />
      </div>

      {/* Event list */}
      <div
        ref={listRef}
        onScroll={handleScroll}
        className="space-y-2 max-h-[65vh] overflow-y-auto pr-1"
      >
        {filteredEvents.map((event) => (
          <EventRow key={event.id || `${event.runId}-${event.sequence}`} event={event} />
        ))}
      </div>
    </div>
  )
}
