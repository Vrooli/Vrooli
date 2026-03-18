/**
 * DecisionLogView - Accordion-style decision log grouped by context tags.
 */

import { useEffect, useState, useCallback } from 'react'
import { Plus, ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamMember } from '@/types/team'
import type { Agent } from '@/types/agent'
import { AgentColorBadge } from '@/components/shared/AgentColorBadge'
import { formatRelativePastTime } from '@/lib/timeUtils'
import * as heartbeatService from '@/services/heartbeatService'
import type { DecisionEntry } from '@/services/heartbeatService'

interface DecisionLogViewProps {
  teamId: string
  members: TeamMember[]
  allAgents?: Agent[]
}

export function DecisionLogView({ teamId, members, allAgents }: DecisionLogViewProps) {
  const [entries, setEntries] = useState<DecisionEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [contextFilter, setContextFilter] = useState('')
  const [expandedContexts, setExpandedContexts] = useState<Set<string>>(new Set(['__all__']))
  const [showAddForm, setShowAddForm] = useState(false)
  const [supersededIds, setSupersededIds] = useState<Set<string>>(new Set())

  // Add form state
  const [newDecision, setNewDecision] = useState('')
  const [newRationale, setNewRationale] = useState('')
  const [newContext, setNewContext] = useState('')
  const [newBy, setNewBy] = useState('')
  const [newSupersedes, setNewSupersedes] = useState('')

  const loadEntries = useCallback(async () => {
    try {
      setError(null)
      const resp = await heartbeatService.getDecisions(teamId, {
        context: contextFilter || undefined,
        last: 50,
      })
      const respEntries = resp.entries ?? []
      setEntries(respEntries)
      // Build set of superseded decision IDs
      const superseded = new Set<string>()
      for (const e of respEntries) {
        if (e.supersedes) superseded.add(e.supersedes)
      }
      setSupersededIds(superseded)
    } catch (err) {
      console.error('[DecisionLogView] Failed to load decisions:', err)
      setError(err instanceof Error ? err.message : 'Failed to load decisions')
    } finally {
      setLoading(false)
    }
  }, [teamId, contextFilter])

  useEffect(() => {
    void loadEntries()
    const interval = setInterval(() => void loadEntries(), 30_000)
    return () => clearInterval(interval)
  }, [loadEntries])

  const getAgentName = (agentId: string) => {
    const member = members.find(m => m.agentId === agentId)
    return member?.displayName ?? agentId
  }

  const getAgentAppearance = (agentId: string) => {
    return allAgents?.find(a => a.id === agentId)?.appearance ?? null
  }

  const handleAddDecision = async () => {
    if (!newDecision.trim() || !newRationale.trim()) return
    try {
      await heartbeatService.addDecision(teamId, {
        by: newBy || 'ui-user',
        decision: newDecision,
        rationale: newRationale,
        context: newContext || undefined,
        supersedes: newSupersedes || undefined,
      })
      setNewDecision('')
      setNewRationale('')
      setNewContext('')
      setNewBy('')
      setNewSupersedes('')
      setShowAddForm(false)
      void loadEntries()
    } catch {
      // silently fail
    }
  }

  const toggleContext = (ctx: string) => {
    setExpandedContexts(prev => {
      const next = new Set(prev)
      if (next.has(ctx)) next.delete(ctx)
      else next.add(ctx)
      return next
    })
  }

  // Group by context
  const grouped = new Map<string, DecisionEntry[]>()
  for (const entry of entries) {
    const ctx = entry.context || '(untagged)'
    if (!grouped.has(ctx)) grouped.set(ctx, [])
    grouped.get(ctx)!.push(entry)
  }

  // Get unique context tags for filter
  const contextTags = Array.from(new Set(entries.map(e => e.context).filter(Boolean) as string[]))

  if (loading) {
    return <div className="text-sm text-muted-foreground">Loading decisions...</div>
  }

  if (error) {
    return (
      <div className="text-sm text-destructive">
        Error loading decisions: {error}
        <button onClick={() => void loadEntries()} className="ml-2 text-xs underline hover:no-underline">Retry</button>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {/* Toolbar */}
      <div className="flex items-center justify-between gap-2 flex-wrap">
        <select
          value={contextFilter}
          onChange={e => setContextFilter(e.target.value)}
          className="text-xs border rounded px-2 py-1 bg-background text-foreground"
        >
          <option value="">All contexts</option>
          {contextTags.map(tag => (
            <option key={tag} value={tag}>{tag}</option>
          ))}
        </select>
        <button
          onClick={() => setShowAddForm(!showAddForm)}
          className="flex items-center gap-1 text-xs px-2 py-1 border rounded hover:bg-muted"
        >
          <Plus className="h-3 w-3" /> Log Decision
        </button>
      </div>

      {/* Add form */}
      {showAddForm && (
        <div className="border rounded-lg p-3 bg-card space-y-2">
          <input
            type="text"
            placeholder="Decision..."
            value={newDecision}
            onChange={e => setNewDecision(e.target.value)}
            className="w-full text-sm border rounded px-2 py-1 bg-background text-foreground"
          />
          <textarea
            placeholder="Rationale — why this decision was made..."
            value={newRationale}
            onChange={e => setNewRationale(e.target.value)}
            className="w-full text-sm border rounded px-2 py-1 bg-background text-foreground resize-none"
            rows={2}
          />
          <div className="flex items-center gap-2">
            <select
              value={newBy}
              onChange={e => setNewBy(e.target.value)}
              className="text-xs border rounded px-2 py-1 bg-background text-foreground"
            >
              <option value="">By (agent)</option>
              {members.map(m => (
                <option key={m.agentId} value={m.agentId}>{m.displayName ?? m.agentId}</option>
              ))}
            </select>
            <input
              type="text"
              placeholder="Context tag"
              value={newContext}
              onChange={e => setNewContext(e.target.value)}
              className="text-xs border rounded px-2 py-1 bg-background text-foreground w-28"
            />
            <input
              type="text"
              placeholder="Supersedes ID"
              value={newSupersedes}
              onChange={e => setNewSupersedes(e.target.value)}
              className="text-xs border rounded px-2 py-1 bg-background text-foreground w-28"
            />
            <button
              onClick={() => void handleAddDecision()}
              className="text-xs px-3 py-1 bg-primary text-primary-foreground rounded hover:bg-primary/90"
            >
              Log
            </button>
          </div>
        </div>
      )}

      {/* Grouped decisions */}
      {entries.length === 0 ? (
        <div className="text-sm text-muted-foreground">No decisions logged yet.</div>
      ) : (
        Array.from(grouped.entries()).map(([ctx, decisions]) => {
          const isExpanded = expandedContexts.has(ctx) || expandedContexts.has('__all__')
          return (
            <div key={ctx} className="border rounded-lg overflow-hidden">
              <button
                onClick={() => toggleContext(ctx)}
                className="w-full flex items-center justify-between px-3 py-2 bg-muted/50 hover:bg-muted text-sm font-medium"
              >
                <span>{ctx} ({decisions.length} decision{decisions.length !== 1 ? 's' : ''})</span>
                <ChevronDown className={cn('h-4 w-4 transition-transform', isExpanded && 'rotate-180')} />
              </button>
              {isExpanded && (
                <div className="divide-y">
                  {decisions.map(entry => (
                    <div
                      key={entry.id}
                      className={cn('px-3 py-2', supersededIds.has(entry.id) && 'opacity-50')}
                    >
                      <div className="flex items-center justify-between">
                        <div className={cn('text-sm font-medium', supersededIds.has(entry.id) && 'line-through')}>
                          {entry.decision}
                        </div>
                        <div className="flex items-center gap-2">
                          <AgentColorBadge appearance={getAgentAppearance(entry.by)} size="xs" />
                          <span className="text-xs text-muted-foreground">{getAgentName(entry.by)}</span>
                        </div>
                      </div>
                      <div className="text-xs text-muted-foreground mt-0.5">
                        {entry.at ? formatRelativePastTime(new Date(entry.at)) : ''}
                      </div>
                      <div className="text-xs text-muted-foreground mt-1">
                        <span className="font-medium">Rationale:</span> {entry.rationale}
                      </div>
                      {entry.supersedes && (
                        <div className="text-xs text-amber-600 mt-0.5">
                          supersedes: {entry.supersedes}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          )
        })
      )}
    </div>
  )
}
