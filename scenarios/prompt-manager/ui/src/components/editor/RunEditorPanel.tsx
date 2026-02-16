/**
 * RunEditorPanel - Full-panel editor for viewing run details.
 *
 * Features:
 * - Header with close (X), status badge, run title (from tag), duration
 * - Three Radix tabs: Info, Events, Investigation
 * - Fetches run details on mount/change
 */

import { useState, useEffect } from 'react'
import * as Tabs from '@radix-ui/react-tabs'
import { X, Info, List, Search, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { getRunDetails, type RunDetails } from '@/services/heartbeatService'
import { selectors } from '@/constants/selectors'
import { RunInfoTab } from './runTabs/RunInfoTab'
import { RunEventsTab } from './runTabs/RunEventsTab'
import { RunInvestigationTab } from './runTabs/RunInvestigationTab'

interface RunEditorPanelProps {
  runId: string
  onClose: () => void
  className?: string
}

const STATUS_COLORS: Record<string, string> = {
  completed: 'bg-emerald-500',
  running: 'bg-amber-500 animate-pulse',
  failed: 'bg-red-500',
  cancelled: 'bg-slate-400',
  pending: 'bg-blue-400',
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

export function RunEditorPanel({
  runId,
  onClose,
  className,
}: RunEditorPanelProps) {
  const [activeTab, setActiveTab] = useState('info')
  const [runDetails, setRunDetails] = useState<RunDetails | null>(null)
  const [loading, setLoading] = useState(true)
  const [elapsed, setElapsed] = useState('')

  // Fetch run details
  useEffect(() => {
    let cancelled = false
    setLoading(true)

    void (async () => {
      try {
        const details = await getRunDetails(runId)
        if (!cancelled) setRunDetails(details)
      } catch {
        // Non-critical for header display
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()

    return () => { cancelled = true }
  }, [runId])

  // Live elapsed ticker
  const isRunning = runDetails?.status === 'running'
  const startedAt = runDetails?.startedAt

  useEffect(() => {
    if (!isRunning || !startedAt) return
    const tick = () => setElapsed(computeDuration(startedAt) ?? '')
    tick()
    const interval = setInterval(tick, 1000)
    return () => clearInterval(interval)
  }, [isRunning, startedAt])

  // Keyboard shortcut: Escape to close
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  const status = runDetails?.status ?? 'unknown'
  const title = runDetails?.tag || `Run ${runId.slice(0, 8)}`
  const duration = isRunning ? elapsed : computeDuration(startedAt, runDetails?.endedAt)

  return (
    <div className={cn('h-full flex flex-col bg-card/50', className)}>
      {/* Header */}
      <div
        className="flex-shrink-0 px-4 py-3 border-b border-border"
        data-testid={selectors.runEditor.header}
      >
        <div className="flex items-center gap-3">
          {/* Close button */}
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
            aria-label="Close editor"
            title="Close (Esc)"
          >
            <X className="h-5 w-5" />
          </button>

          {/* Status badge */}
          {loading ? (
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          ) : (
            <span
              className={cn(
                'inline-block h-3 w-3 rounded-full flex-shrink-0',
                STATUS_COLORS[status] ?? 'bg-slate-400'
              )}
              title={status}
            />
          )}

          {/* Run title */}
          <div className="flex-1 min-w-0">
            <h2 className="text-lg font-semibold text-foreground truncate">
              {title}
            </h2>
          </div>

          {/* Duration */}
          {duration && (
            <span className="text-xs text-muted-foreground flex-shrink-0">
              {duration}
            </span>
          )}

          {/* Status text badge */}
          <span
            className={cn(
              'px-2 py-1 text-xs font-medium rounded-full capitalize',
              status === 'completed' && 'bg-emerald-500/15 text-emerald-500',
              status === 'running' && 'bg-amber-500/15 text-amber-500',
              status === 'failed' && 'bg-red-500/15 text-red-500',
              status === 'cancelled' && 'bg-slate-500/20 text-slate-400',
              status === 'pending' && 'bg-blue-500/15 text-blue-400',
              !['completed', 'running', 'failed', 'cancelled', 'pending'].includes(status) && 'bg-slate-500/20 text-slate-400'
            )}
          >
            {status}
          </span>
        </div>
      </div>

      {/* Tabs */}
      <Tabs.Root
        value={activeTab}
        onValueChange={setActiveTab}
        className="flex-1 flex flex-col min-h-0 overflow-hidden"
      >
        <Tabs.List className="flex-shrink-0 flex border-b border-border px-4">
          <TabTrigger value="info" icon={<Info className="h-4 w-4" />} label="Info" testId={selectors.runEditor.tabInfo} />
          <TabTrigger value="events" icon={<List className="h-4 w-4" />} label="Events" testId={selectors.runEditor.tabEvents} live={isRunning} />
          <TabTrigger value="investigation" icon={<Search className="h-4 w-4" />} label="Investigation" testId={selectors.runEditor.tabInvestigation} />
        </Tabs.List>

        <div className="flex-1 min-h-0 flex flex-col">
          <Tabs.Content
            value="info"
            className="flex-1 min-h-0 overflow-y-auto p-4 data-[state=inactive]:hidden"
          >
            <RunInfoTab runId={runId} />
          </Tabs.Content>

          <Tabs.Content
            value="events"
            className="flex-1 min-h-0 overflow-y-auto p-4 data-[state=inactive]:hidden"
          >
            <RunEventsTab runId={runId} live={isRunning} />
          </Tabs.Content>

          <Tabs.Content
            value="investigation"
            className="flex-1 min-h-0 overflow-y-auto p-4 data-[state=inactive]:hidden"
          >
            <RunInvestigationTab runId={runId} />
          </Tabs.Content>
        </div>
      </Tabs.Root>
    </div>
  )
}

interface TabTriggerProps {
  value: string
  icon: React.ReactNode
  label: string
  testId?: string
  live?: boolean
}

function TabTrigger({ value, icon, label, testId, live }: TabTriggerProps) {
  return (
    <Tabs.Trigger
      value={value}
      className={cn(
        'flex items-center gap-1.5 px-3 py-2 text-sm font-medium',
        'border-b-2 transition-colors',
        'data-[state=active]:border-primary data-[state=active]:text-primary',
        'data-[state=inactive]:border-transparent data-[state=inactive]:text-muted-foreground',
        'hover:text-foreground'
      )}
      data-testid={testId}
    >
      {icon}
      {label}
      {live && (
        <span className="relative flex h-2 w-2 ml-1">
          <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
          <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500" />
        </span>
      )}
    </Tabs.Trigger>
  )
}
