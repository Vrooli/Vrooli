/**
 * RunInvestigationTab - Investigation agent tab.
 *
 * States:
 * 1. Not started: "Investigate" button + depth selector + optional custom context
 * 2. Running: EventsDisplay for investigation run + Stop
 * 3. Completed: Events + message input for follow-up + "Apply Recommendations"
 * 4. Follow-up running: Updated events stream
 *
 * Flow:
 * - "Investigate" -> createInvestigationRun([runId])
 * - Poll events via EventsDisplay
 * - Follow-up -> continueRun(investigationRunId, message)
 * - "Apply" -> createInvestigationApplyRun(investigationRunId)
 * - On mount: check listRuns({tagPrefix:"investigate-"}) for existing investigations
 */

import { useState, useEffect, useCallback } from 'react'
import { Search as SearchIcon, Loader2, Play, MessageSquare, Wrench, ChevronDown, Info } from 'lucide-react'
import { cn } from '@/lib/utils'
import { EventsDisplay } from '@/components/shared/EventsDisplay'
import {
  createInvestigationRun,
  createInvestigationApplyRun,
  continueRun,
  listRuns,
  getRunDetails,
  type RunDetails,
} from '@/services/heartbeatService'

interface RunInvestigationTabProps {
  runId: string
  className?: string
}

type DepthOption = 'quick' | 'standard' | 'deep'

const DEPTH_OPTIONS: { value: DepthOption; label: string; description: string }[] = [
  { value: 'quick', label: 'Quick', description: 'Fast surface-level scan' },
  { value: 'standard', label: 'Standard', description: 'Balanced analysis' },
  { value: 'deep', label: 'Deep', description: 'Thorough deep dive' },
]

type InvestigationState =
  | { phase: 'idle' }
  | { phase: 'running'; investigationRun: RunDetails; depth?: string; customContext?: string }
  | { phase: 'completed'; investigationRun: RunDetails; depth?: string; customContext?: string }
  | { phase: 'followup'; investigationRun: RunDetails; depth?: string; customContext?: string }
  | { phase: 'applying'; applyRun: RunDetails; investigationRun: RunDetails; depth?: string; customContext?: string }

export function RunInvestigationTab({ runId, className }: RunInvestigationTabProps) {
  const [state, setState] = useState<InvestigationState>({ phase: 'idle' })
  const [depth, setDepth] = useState<DepthOption>('standard')
  const [customContext, setCustomContext] = useState('')
  const [followUpMessage, setFollowUpMessage] = useState('')
  const [isStarting, setIsStarting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [checkingExisting, setCheckingExisting] = useState(true)
  const [systemContextOpen, setSystemContextOpen] = useState(false)

  // Check for existing investigation runs on mount
  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const response = await listRuns({ tagPrefix: `investigate-${runId}` })
        if (cancelled) return
        const existing = response.runs[0]
        if (existing) {
          if (existing.status === 'running' || existing.status === 'pending') {
            setState({ phase: 'running', investigationRun: existing })
          } else if (existing.status === 'completed') {
            setState({ phase: 'completed', investigationRun: existing })
          }
        }
      } catch {
        // Ignore - just start fresh
      } finally {
        if (!cancelled) setCheckingExisting(false)
      }
    })()
    return () => { cancelled = true }
  }, [runId])

  // Poll investigation run status when running
  useEffect(() => {
    if (state.phase !== 'running' && state.phase !== 'followup') return
    const invRun = state.investigationRun

    const poll = async () => {
      try {
        const updated = await getRunDetails(invRun.id)
        if (updated.status === 'completed' || updated.status === 'failed') {
          setState((prev) => {
            if (prev.phase === 'running' || prev.phase === 'followup') {
              return { phase: 'completed', investigationRun: updated, depth: prev.depth, customContext: prev.customContext }
            }
            return prev
          })
        }
      } catch {
        // Ignore poll errors
      }
    }

    const interval = setInterval(() => void poll(), 3000)
    return () => clearInterval(interval)
  }, [state])

  const handleInvestigate = useCallback(async () => {
    setIsStarting(true)
    setError(null)
    try {
      const invRun = await createInvestigationRun([runId], {
        depth,
        customContext: customContext || undefined,
      })
      setState({
        phase: 'running',
        investigationRun: invRun,
        depth,
        customContext: customContext || undefined,
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start investigation')
    } finally {
      setIsStarting(false)
    }
  }, [runId, depth, customContext])

  const handleFollowUp = useCallback(async () => {
    if (state.phase !== 'completed' || !followUpMessage.trim()) return
    setError(null)
    try {
      await continueRun(state.investigationRun.id, followUpMessage)
      setState({ phase: 'followup', investigationRun: state.investigationRun, depth: state.depth, customContext: state.customContext })
      setFollowUpMessage('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send follow-up')
    }
  }, [state, followUpMessage])

  const handleApply = useCallback(async () => {
    if (state.phase !== 'completed') return
    setError(null)
    try {
      const applyRun = await createInvestigationApplyRun(state.investigationRun.id)
      setState({ phase: 'applying', applyRun, investigationRun: state.investigationRun, depth: state.depth, customContext: state.customContext })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to apply recommendations')
    }
  }, [state])

  if (checkingExisting) {
    return (
      <div className={cn('flex items-center justify-center py-12', className)}>
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  // Config summary shown for running/completed/followup/applying states
  const configSummary = state.phase !== 'idle' && (state.depth || state.customContext) ? (
    <div className="rounded-lg border border-border bg-muted/50 p-3 text-xs text-muted-foreground space-y-1">
      {state.depth && (
        <p><span className="font-medium text-foreground">Depth:</span> {DEPTH_OPTIONS.find((d) => d.value === state.depth)?.label ?? state.depth}</p>
      )}
      {state.customContext && (
        <p><span className="font-medium text-foreground">Custom context:</span> {state.customContext}</p>
      )}
    </div>
  ) : null

  return (
    <div className={cn('flex flex-col gap-4', className)}>
      {error && (
        <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3">
          <p className="text-sm text-red-400">{error}</p>
        </div>
      )}

      {/* Idle state: start investigation */}
      {state.phase === 'idle' && (
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

          {/* Depth selector */}
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

          {/* Investigation Context */}
          <div className="space-y-2">
            <label className="text-xs font-medium text-foreground block">Investigation Context</label>

            {/* Collapsible system context info */}
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

            <div>
              <label className="text-xs text-muted-foreground block mb-1">
                Custom Context (your additions)
              </label>
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
            {isStarting ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Play className="h-4 w-4" />
            )}
            {isStarting ? 'Starting...' : 'Investigate'}
          </button>
        </div>
      )}

      {/* Running state: show events */}
      {(state.phase === 'running' || state.phase === 'followup') && (
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <span className="relative flex h-2.5 w-2.5">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
              <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-emerald-500" />
            </span>
            <span className="text-xs font-medium text-emerald-400">Investigation in progress</span>
          </div>
          {configSummary}
          <EventsDisplay
            runId={state.investigationRun.id}
            live
          />
        </div>
      )}

      {/* Completed state: events + follow-up + apply */}
      {state.phase === 'completed' && (
        <div className="space-y-4">
          {configSummary}
          <EventsDisplay
            runId={state.investigationRun.id}
          />

          {/* Follow-up input */}
          <div className="flex gap-2">
            <input
              type="text"
              value={followUpMessage}
              onChange={(e) => setFollowUpMessage(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && followUpMessage.trim()) {
                  void handleFollowUp()
                }
              }}
              placeholder="Ask a follow-up question..."
              className={cn(
                'flex-1 px-3 py-2 text-sm rounded-md border border-border bg-muted',
                'text-foreground placeholder:text-muted-foreground',
                'focus:outline-none focus:ring-2 focus:ring-primary'
              )}
            />
            <button
              type="button"
              onClick={() => void handleFollowUp()}
              disabled={!followUpMessage.trim()}
              className={cn(
                'p-2 rounded-lg transition-colors',
                followUpMessage.trim()
                  ? 'bg-primary hover:bg-primary/90 text-primary-foreground'
                  : 'bg-muted text-muted-foreground cursor-not-allowed'
              )}
              title="Send follow-up"
            >
              <MessageSquare className="h-4 w-4" />
            </button>
          </div>

          {/* Apply button */}
          <button
            type="button"
            onClick={() => void handleApply()}
            className={cn(
              'flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium rounded-lg transition-colors',
              'border border-primary text-primary hover:bg-primary/10'
            )}
          >
            <Wrench className="h-4 w-4" />
            Apply Recommendations
          </button>
        </div>
      )}

      {/* Applying state */}
      {state.phase === 'applying' && (
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <span className="relative flex h-2.5 w-2.5">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-amber-400 opacity-75" />
              <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-amber-500" />
            </span>
            <span className="text-xs font-medium text-amber-400">Applying recommendations</span>
          </div>
          {configSummary}
          <EventsDisplay
            runId={state.applyRun.id}
            live
          />
        </div>
      )}
    </div>
  )
}
