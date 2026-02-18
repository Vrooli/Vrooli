/**
 * RunInvestigationTab - Investigation agent tab.
 *
 * Flow:
 * - Discover investigations linked to this run via investigates_run_id filter
 * - "Investigate" starts a new investigation run
 * - Follow-up supports both continue current run and start-new-investigation
 * - "Apply" starts investigation-apply run from selected investigation
 */

import { useState, useEffect, useCallback } from 'react'
import { Search as SearchIcon, Loader2, Play, MessageSquare, Wrench, ChevronDown, Info, RefreshCw, Copy, Check, SendHorizontal, ChevronLeft } from 'lucide-react'
import { cn } from '@/lib/utils'
import { EventsDisplay } from '@/components/shared/EventsDisplay'
import {
  createInvestigationRun,
  createInvestigationApplyRun,
  continueRun,
  listRuns,
  getRunDetails,
  type RunDetails,
  type RunEvent,
} from '@/services/heartbeatService'

interface RunInvestigationTabProps {
  runId: string
  className?: string
}

type DepthOption = 'quick' | 'standard' | 'deep'
type InvestigationAction = 'apply' | 'followUp' | 'newInvestigation'

const DEPTH_OPTIONS: { value: DepthOption; label: string; description: string }[] = [
  { value: 'quick', label: 'Quick', description: 'Fast surface-level scan' },
  { value: 'standard', label: 'Standard', description: 'Balanced analysis' },
  { value: 'deep', label: 'Deep', description: 'Thorough deep dive' },
]

function isActiveStatus(status: string): boolean {
  return status === 'running' || status === 'pending' || status === 'starting'
}

function formatRunLabel(run: RunDetails, index: number): string {
  const ts = run.startedAt || run.endedAt || ''
  if (!ts) return `Investigation ${index + 1}`
  const parsed = new Date(ts)
  if (Number.isNaN(parsed.getTime())) return `Investigation ${index + 1}`
  return `Investigation ${index + 1} · ${parsed.toLocaleString()}`
}

export function RunInvestigationTab({ runId, className }: RunInvestigationTabProps) {
  const [investigations, setInvestigations] = useState<RunDetails[]>([])
  const [selectedInvestigationId, setSelectedInvestigationId] = useState<string | null>(null)
  const [applyRun, setApplyRun] = useState<RunDetails | null>(null)

  const [depth, setDepth] = useState<DepthOption>('standard')
  const [customContext, setCustomContext] = useState('')
  const [followUpMessage, setFollowUpMessage] = useState('')
  const [selectedAction, setSelectedAction] = useState<InvestigationAction | null>('followUp')
  const [isSubmittingAction, setIsSubmittingAction] = useState(false)
  const [copiedJson, setCopiedJson] = useState(false)
  const [filteredEvents, setFilteredEvents] = useState<RunEvent[]>([])

  const [checkingExisting, setCheckingExisting] = useState(true)
  const [isStarting, setIsStarting] = useState(false)
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [systemContextOpen, setSystemContextOpen] = useState(false)

  const selectedInvestigation = investigations.find((r) => r.id === selectedInvestigationId) ?? null

  const loadInvestigations = useCallback(async (showSpinner = false) => {
    if (showSpinner) setIsRefreshing(true)
    try {
      const response = await listRuns({
        investigatesRunId: runId,
        limit: 50,
      })
      const runs = response.runs
      setInvestigations(runs)
      if (runs.length === 0) {
        setSelectedInvestigationId(null)
      } else if (!selectedInvestigationId || !runs.some((r) => r.id === selectedInvestigationId)) {
        const firstRun = runs[0]
        if (firstRun) {
          setSelectedInvestigationId(firstRun.id)
        }
      }
    } catch (err) {
      if (showSpinner) {
        setError(err instanceof Error ? err.message : 'Failed to refresh investigations')
      }
    } finally {
      if (showSpinner) setIsRefreshing(false)
    }
  }, [runId, selectedInvestigationId])

  useEffect(() => {
    const controller = new AbortController()
    void (async () => {
      try {
        const response = await listRuns({
          investigatesRunId: runId,
          limit: 50,
        })
        if (controller.signal.aborted) return
        const runs = response.runs
        setInvestigations(runs)
        setSelectedInvestigationId(runs[0]?.id ?? null)
      } catch {
        // Start with idle state if lookup fails
      } finally {
        if (!controller.signal.aborted) setCheckingExisting(false)
      }
    })()
    return () => controller.abort()
  }, [runId])

  useEffect(() => {
    if (!selectedInvestigation || !isActiveStatus(selectedInvestigation.status)) return
    const id = selectedInvestigation.id
    const interval = setInterval(() => {
      void (async () => {
        try {
          const updated = await getRunDetails(id)
          setInvestigations((prev) => prev.map((r) => (r.id === id ? updated : r)))
        } catch {
          // ignore transient poll errors
        }
      })()
    }, 3000)
    return () => clearInterval(interval)
  }, [selectedInvestigation])

  useEffect(() => {
    if (!applyRun || !isActiveStatus(applyRun.status)) return
    const id = applyRun.id
    const interval = setInterval(() => {
      void (async () => {
        try {
          const updated = await getRunDetails(id)
          setApplyRun(updated)
        } catch {
          // ignore transient poll errors
        }
      })()
    }, 3000)
    return () => clearInterval(interval)
  }, [applyRun])

  const handleInvestigate = useCallback(async (message?: string) => {
    setIsStarting(true)
    setError(null)
    try {
      const invRun = await createInvestigationRun([runId], {
        depth,
        customContext: message ?? (customContext || undefined),
      })
      setInvestigations((prev) => [invRun, ...prev])
      setSelectedInvestigationId(invRun.id)
      if (!message) {
        setCustomContext('')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start investigation')
    } finally {
      setIsStarting(false)
    }
  }, [runId, depth, customContext])

  const handleContinue = useCallback(async (message: string) => {
    if (!selectedInvestigation || !message.trim()) return
    setError(null)
    try {
      await continueRun(selectedInvestigation.id, message)
      const updated = await getRunDetails(selectedInvestigation.id)
      setInvestigations((prev) => prev.map((r) => (r.id === selectedInvestigation.id ? updated : r)))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send follow-up')
    }
  }, [selectedInvestigation])

  const handleNewInvestigationFromFollowup = useCallback(async (message: string) => {
    if (!message.trim()) return
    await handleInvestigate(message.trim())
  }, [handleInvestigate])

  const handleApply = useCallback(async (message?: string) => {
    if (!selectedInvestigation) return
    setError(null)
    try {
      const nextApplyRun = await createInvestigationApplyRun(selectedInvestigation.id, message?.trim() || undefined)
      setApplyRun(nextApplyRun)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to apply recommendations')
    }
  }, [selectedInvestigation])

  const handleCopyJson = useCallback(() => {
    const text = JSON.stringify(filteredEvents, null, 2)
    void navigator.clipboard.writeText(text).then(() => {
      setCopiedJson(true)
      setTimeout(() => setCopiedJson(false), 1500)
    })
  }, [filteredEvents])

  const getActionLabel = useCallback((action: InvestigationAction | null): string => {
    if (action === 'apply') return 'Apply'
    if (action === 'followUp') return 'Follow-Up'
    if (action === 'newInvestigation') return 'New'
    return ''
  }, [])

  const actionCanSubmit = selectedAction === 'apply' ? true : Boolean(followUpMessage.trim())

  const handleActionSubmit = useCallback(async () => {
    if (!selectedAction || isSubmittingAction) return
    const message = followUpMessage.trim()
    if (selectedAction !== 'apply' && !message) return

    setIsSubmittingAction(true)
    try {
      if (selectedAction === 'apply') {
        await handleApply(message)
      } else if (selectedAction === 'followUp') {
        await handleContinue(message)
      } else {
        await handleNewInvestigationFromFollowup(message)
      }
      setFollowUpMessage('')
      setSelectedAction('followUp')
    } finally {
      setIsSubmittingAction(false)
    }
  }, [selectedAction, isSubmittingAction, followUpMessage, handleApply, handleContinue, handleNewInvestigationFromFollowup])

  if (checkingExisting) {
    return (
      <div className={cn('flex items-center justify-center py-12', className)}>
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className={cn('flex flex-col gap-4', className)}>
      {error && (
        <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3">
          <p className="text-sm text-red-400">{error}</p>
        </div>
      )}

      {!selectedInvestigation && (
        <div className="space-y-4">
          <div className="flex items-start gap-3 p-4 bg-muted rounded-lg">
            <SearchIcon className="h-5 w-5 text-muted-foreground mt-0.5 flex-shrink-0" />
            <div>
              <p className="text-sm font-medium text-foreground">Investigate this run</p>
              <p className="text-xs text-muted-foreground mt-1">
                Start an AI investigation to analyze what happened during this run and suggest improvements.
              </p>
            </div>
          </div>

          <div>
            <label className="text-xs text-muted-foreground block mb-2">Investigation Depth</label>
            <div className="inline-flex rounded-lg border border-border overflow-hidden">
              {DEPTH_OPTIONS.map((opt) => (
                <button
                  key={opt.value}
                  type="button"
                  onClick={() => setDepth(opt.value)}
                  title={opt.description}
                  className={cn(
                    'px-3 py-1.5 text-xs font-medium transition-colors',
                    'border-r border-border last:border-r-0',
                    depth === opt.value
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-muted text-muted-foreground hover:bg-muted/80 hover:text-foreground'
                  )}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          </div>

          <div className="space-y-2">
            <label className="text-xs font-medium text-foreground block">Investigation Context</label>
            <button
              type="button"
              onClick={() => setSystemContextOpen(!systemContextOpen)}
              className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
            >
              <ChevronDown className={cn('h-3.5 w-3.5 transition-transform', systemContextOpen && 'rotate-180')} />
              <Info className="h-3.5 w-3.5" />
              <span>System Context (auto-generated)</span>
            </button>
            {systemContextOpen && (
              <div className="rounded-md border border-border bg-muted/50 p-3 text-xs text-muted-foreground ml-5">
                The investigation agent automatically receives context about the run including:
                team configuration, agent profile, heartbeat schedule, and run events.
              </div>
            )}

            <textarea
              value={customContext}
              onChange={(e) => setCustomContext(e.target.value)}
              placeholder="Describe what you expected or any specific areas to investigate..."
              className={cn(
                'w-full px-3 py-2 text-sm rounded-md border border-border bg-muted',
                'text-foreground placeholder:text-muted-foreground',
                'focus:outline-none focus:ring-2 focus:ring-primary',
                'resize-y min-h-[60px]'
              )}
              rows={3}
            />
          </div>

          <button
            type="button"
            onClick={() => void handleInvestigate()}
            disabled={isStarting}
            className={cn(
              'flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium rounded-lg transition-colors',
              'bg-primary hover:bg-primary/90 text-primary-foreground',
              isStarting && 'opacity-50 cursor-not-allowed'
            )}
          >
            {isStarting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
            {isStarting ? 'Starting...' : 'Investigate'}
          </button>
        </div>
      )}

      {selectedInvestigation && (
        <div className="space-y-4">
          <div className="flex items-center justify-between gap-2 flex-wrap">
            <div className="min-w-[200px] flex-1">
              {investigations.length > 1 ? (
                <select
                  value={selectedInvestigationId ?? ''}
                  onChange={(e) => setSelectedInvestigationId(e.target.value)}
                  aria-label="Select investigation"
                  className={cn(
                    'w-full rounded-md border border-border bg-muted px-3 py-2 text-xs text-foreground',
                    'focus:outline-none focus:ring-2 focus:ring-primary'
                  )}
                >
                  {investigations.map((run, idx) => (
                    <option key={run.id} value={run.id}>
                      {formatRunLabel(run, idx)}
                    </option>
                  ))}
                </select>
              ) : (
                <p className="rounded-md border border-border bg-muted px-3 py-2 text-xs text-foreground" title={selectedInvestigation.id}>
                  {formatRunLabel(selectedInvestigation, 0)}
                </p>
              )}
            </div>

            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => void loadInvestigations(true)}
                aria-label="Refresh investigations"
                className="inline-flex items-center gap-1 rounded border border-border px-2 py-1 text-xs text-muted-foreground hover:text-foreground"
              >
                <RefreshCw className={cn('h-3.5 w-3.5', isRefreshing && 'animate-spin')} />
                <span className="hidden sm:inline">Refresh</span>
              </button>
              <button
                type="button"
                onClick={handleCopyJson}
                aria-label="Copy investigation JSON"
                className="inline-flex items-center gap-1 rounded border border-border px-2 py-1 text-xs text-muted-foreground hover:text-foreground"
              >
                {copiedJson ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
                <span className="hidden sm:inline">{copiedJson ? 'Copied' : 'Copy JSON'}</span>
              </button>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <span
              className={cn(
                'h-2.5 w-2.5 rounded-full',
                isActiveStatus(selectedInvestigation.status) && 'bg-emerald-500',
                selectedInvestigation.status === 'completed' && 'bg-blue-500',
                selectedInvestigation.status === 'failed' && 'bg-red-500',
                !isActiveStatus(selectedInvestigation.status) && selectedInvestigation.status !== 'completed' && selectedInvestigation.status !== 'failed' && 'bg-slate-500'
              )}
            />
            <span className="text-xs text-muted-foreground">
              Selected investigation status: <span className="font-medium text-foreground">{selectedInvestigation.status}</span>
            </span>
          </div>

          <EventsDisplay
            runId={selectedInvestigation.id}
            live={isActiveStatus(selectedInvestigation.status)}
            showHeader={false}
            onFilteredEventsChange={setFilteredEvents}
          />

          {!isActiveStatus(selectedInvestigation.status) && (
            <>
              {!selectedAction ? (
                <div className="flex flex-wrap gap-2">
                  <button
                    type="button"
                    onClick={() => setSelectedAction('apply')}
                    className="inline-flex items-center gap-1 rounded-lg px-3 py-2 text-xs font-medium border border-primary text-primary hover:bg-primary/10"
                  >
                    <Wrench className="h-3.5 w-3.5" /> Apply
                  </button>
                  <button
                    type="button"
                    onClick={() => setSelectedAction('followUp')}
                    className="inline-flex items-center gap-1 rounded-lg px-3 py-2 text-xs font-medium bg-primary text-primary-foreground hover:bg-primary/90"
                  >
                    <MessageSquare className="h-3.5 w-3.5" /> Follow-Up
                  </button>
                  <button
                    type="button"
                    onClick={() => setSelectedAction('newInvestigation')}
                    className="inline-flex items-center gap-1 rounded-lg px-3 py-2 text-xs font-medium border border-border text-muted-foreground hover:text-foreground"
                  >
                    <Play className="h-3.5 w-3.5" /> New
                  </button>
                </div>
              ) : (
                <div className="space-y-2">
                  <button
                    type="button"
                    onClick={() => {
                      setSelectedAction(null)
                      setFollowUpMessage('')
                    }}
                    className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
                  >
                    <ChevronLeft className="h-3.5 w-3.5" />
                    {getActionLabel(selectedAction)}
                  </button>

                  <div className="relative">
                    <textarea
                      value={followUpMessage}
                      onChange={(e) => setFollowUpMessage(e.target.value)}
                      placeholder={`Add a message for ${getActionLabel(selectedAction).toLowerCase()}...`}
                      className={cn(
                        'w-full pr-11 px-3 py-2 text-sm rounded-md border border-border bg-muted',
                        'text-foreground placeholder:text-muted-foreground',
                        'focus:outline-none focus:ring-2 focus:ring-primary',
                        'resize-y min-h-[88px]'
                      )}
                      rows={3}
                    />
                    <button
                      type="button"
                      onClick={() => void handleActionSubmit()}
                      disabled={!actionCanSubmit || isSubmittingAction}
                      aria-label={`Send ${getActionLabel(selectedAction).toLowerCase()}`}
                      className={cn(
                        'absolute bottom-2 right-2 inline-flex items-center justify-center rounded-md h-8 w-8 transition-colors',
                        actionCanSubmit && !isSubmittingAction
                          ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                          : 'bg-muted border border-border text-muted-foreground cursor-not-allowed'
                      )}
                    >
                      {isSubmittingAction ? <Loader2 className="h-4 w-4 animate-spin" /> : <SendHorizontal className="h-4 w-4" />}
                    </button>
                  </div>
                </div>
              )}
            </>
          )}

          {applyRun && (
            <div className="space-y-2">
              <p className="text-xs text-amber-400 font-medium">Apply run ({applyRun.status})</p>
              <EventsDisplay runId={applyRun.id} live={isActiveStatus(applyRun.status)} />
            </div>
          )}
        </div>
      )}
    </div>
  )
}
