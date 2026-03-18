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
import { Menu, X, Info, List, Search, Loader2, MoreHorizontal, Clock3, Activity } from 'lucide-react'
import { TabList, TabTrigger } from '../shared/TabTrigger'
import { cn } from '@/lib/utils'
import { getRunDetails, type RunDetails } from '@/services/heartbeatService'
import { selectors } from '@/constants/selectors'
import { useIsCompactHeader } from '@/hooks/useMediaQuery'
import { DropdownItem, ToolbarDropdown } from './ToolbarDropdown'
import { RunInfoTab } from './runTabs/RunInfoTab'
import { RunEventsTab } from './runTabs/RunEventsTab'
import { RunInvestigationTab } from './runTabs/RunInvestigationTab'

interface RunEditorPanelProps {
  runId: string
  onClose: () => void
  onOpenSidebar?: () => void
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
  onOpenSidebar,
  className,
}: RunEditorPanelProps) {
  const [activeTab, setActiveTab] = useState('info')
  const [runDetails, setRunDetails] = useState<RunDetails | null>(null)
  const [loading, setLoading] = useState(true)
  const [elapsed, setElapsed] = useState('')
  const isCompactHeader = useIsCompactHeader()
  const isMobileSidebarToggle = Boolean(onOpenSidebar)

  // Fetch run details
  useEffect(() => {
    const controller = new AbortController()
    setLoading(true)

    void (async () => {
      try {
        const details = await getRunDetails(runId)
        if (controller.signal.aborted) return
        setRunDetails(details)
      } catch {
        // Non-critical for header display
      } finally {
        if (!controller.signal.aborted) setLoading(false)
      }
    })()

    return () => controller.abort()
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
        <div className="flex items-center gap-2 min-w-0">
          {/* Close button */}
          <button
            type="button"
            onClick={onOpenSidebar ?? onClose}
            className="h-9 w-9 flex items-center justify-center rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
            aria-label={isMobileSidebarToggle ? 'Open sidebar' : 'Close editor'}
            title={isMobileSidebarToggle ? 'Open sidebar' : 'Close (Esc)'}
          >
            {isMobileSidebarToggle ? <Menu className="h-5 w-5" /> : <X className="h-5 w-5" />}
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
            <span className="text-xs text-muted-foreground flex-shrink-0 max-[389px]:hidden">
              {duration}
            </span>
          )}

          {/* Status text badge */}
          <span
            className={cn(
              'px-2 py-1 text-xs font-medium rounded-full capitalize max-[389px]:hidden',
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

          {isCompactHeader && (
            <ToolbarDropdown
              icon={<MoreHorizontal className="h-4 w-4" />}
              label="Run details"
              showChevron={false}
              align="right"
              className="h-9 w-9 p-0 rounded-lg"
            >
              <DropdownItem
                onClick={() => void 0}
                disabled
                icon={<Activity className="h-4 w-4" />}
                label={`Status: ${status}`}
              />
              {duration && (
                <DropdownItem
                  onClick={() => void 0}
                  disabled
                  icon={<Clock3 className="h-4 w-4" />}
                  label={`Duration: ${duration}`}
                />
              )}
            </ToolbarDropdown>
          )}
        </div>
      </div>

      {/* Tabs */}
      <Tabs.Root
        value={activeTab}
        onValueChange={setActiveTab}
        className="flex-1 flex flex-col min-h-0 overflow-hidden"
      >
        <TabList>
          <TabTrigger value="info" icon={<Info className="h-4 w-4" />} label="Info" testId={selectors.runEditor.tabInfo} />
          <TabTrigger value="events" icon={<List className="h-4 w-4" />} label="Events" testId={selectors.runEditor.tabEvents} live={isRunning} />
          <TabTrigger value="investigation" icon={<Search className="h-4 w-4" />} label="Investigation" testId={selectors.runEditor.tabInvestigation} />
        </TabList>

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

