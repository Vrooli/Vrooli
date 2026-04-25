/**
 * DecisionLogView - Production-ready decision log with human oversight controls.
 *
 * Features:
 * - Accept / Reject / Edit / Delete controls on each decision card
 * - Status visual indicators (pending, accepted, rejected)
 * - Inline editing of decision text, rationale, and context tag
 * - Delete confirmation via ConfirmDialog
 * - Context grouping with accordion, supersede tracking, context filter
 */

import { useEffect, useState, useCallback } from 'react'
import { Plus, ChevronDown, Loader2, Scale, Pencil, X, Check, Trash2, CheckCircle, Star, Clock } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamMember } from '@/types/team'
import type { Agent } from '@/types/agent'
import { AgentColorBadge } from '@/components/shared/AgentColorBadge'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { formatRelativePastTime } from '@/lib/timeUtils'
import * as heartbeatService from '@/services/heartbeatService'
import type { DecisionEntry, DecisionModifications } from '@/services/heartbeatService'

interface DecisionLogViewProps {
  teamId: string
  members: TeamMember[]
  allAgents?: Agent[]
  decisionMode?: string
}

/** Status badge rendering */
function StatusBadge({ status }: { status?: DecisionEntry['status'] }) {
  if (status === 'accepted') {
    return (
      <span className="inline-flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wide px-1.5 py-0.5 rounded-full bg-emerald-500/15 text-emerald-400">
        <Check className="h-2.5 w-2.5" />
        Accepted
      </span>
    )
  }
  if (status === 'rejected') {
    return (
      <span className="inline-flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wide px-1.5 py-0.5 rounded-full bg-red-500/15 text-red-400">
        <X className="h-2.5 w-2.5" />
        Rejected
      </span>
    )
  }
  if (status === 'running') {
    return (
      <span className="inline-flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wide px-1.5 py-0.5 rounded-full bg-blue-500/15 text-blue-400">
        <Loader2 className="h-2.5 w-2.5 animate-spin" />
        Running
      </span>
    )
  }
  if (status === 'deferred') {
    return (
      <span className="inline-flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wide px-1.5 py-0.5 rounded-full bg-yellow-600/15 text-yellow-500">
        <Clock className="h-2.5 w-2.5" />
        Deferred
      </span>
    )
  }
  if (status === 'completed') {
    return (
      <span className="inline-flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wide px-1.5 py-0.5 rounded-full bg-slate-500/15 text-slate-400">
        <CheckCircle className="h-2.5 w-2.5" />
        Completed
      </span>
    )
  }
  return (
    <span className="inline-flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wide px-1.5 py-0.5 rounded-full bg-amber-500/15 text-amber-400">
      <Scale className="h-2.5 w-2.5" />
      Pending
    </span>
  )
}

const OTHER_KEY = '__other__'

/** True if a single-proposal decision was accepted as proposed (new flag or
 *  legacy `__other__ + freeform="accept as proposed"` shape). */
function isAcceptedAsProposed(entry: DecisionEntry): boolean {
  if (entry.accepted_as_proposed) return true
  if ((entry.options?.length ?? 0) === 0 && entry.selected === OTHER_KEY) {
    const f = (entry.freeform ?? '').trim().toLowerCase()
    return f.includes('accept as proposed')
  }
  return false
}

/** Multi-option decision card with lettered choices */
function MultiOptionCard({
  entry,
  isSuperseded,
  isThisStatusLoading,
  getAgentAppearance,
  getAgentName,
  onSelectOption,
  onDelete,
}: {
  entry: DecisionEntry
  isSuperseded: boolean
  isThisStatusLoading: boolean
  getAgentAppearance: (id: string) => Agent['appearance'] | null
  getAgentName: (id: string) => string
  onSelectOption: (
    entry: DecisionEntry,
    key: string,
    freeform?: string,
    notes?: string,
    modifications?: DecisionModifications,
  ) => Promise<void>
  onDelete: () => void
}) {
  const [localSelected, setLocalSelected] = useState<string | null>(entry.selected ?? null)
  const [localFreeform, setLocalFreeform] = useState(entry.freeform ?? '')
  const [localNotes, setLocalNotes] = useState(entry.notes ?? '')
  const [showModsForm, setShowModsForm] = useState(false)
  const [excludedClauses, setExcludedClauses] = useState<string[]>([])
  const [additions, setAdditions] = useState<string[]>([])
  const [modsRationale, setModsRationale] = useState('')
  const [excludedDraft, setExcludedDraft] = useState('')
  const [additionDraft, setAdditionDraft] = useState('')

  // Modifications are immutable once set. If already present, render read-only
  // (see docs/reference/decision-modifications-contract.md).
  const existingMods = entry.modifications ?? null

  const pendingMods: DecisionModifications | undefined = (() => {
    if (existingMods) return undefined
    if (!showModsForm) return undefined
    if (excludedClauses.length === 0 && additions.length === 0 && modsRationale.trim() === '') {
      return undefined
    }
    const m: DecisionModifications = {}
    if (excludedClauses.length > 0) m.excluded_clauses = excludedClauses
    if (additions.length > 0) m.additions = additions
    if (modsRationale.trim() !== '') m.rationale = modsRationale.trim()
    return m
  })()

  const hasSelection = entry.selected != null && entry.selected !== ''
  const isModified = localSelected !== (entry.selected ?? null)
    || localFreeform !== (entry.freeform ?? '')
    || localNotes !== (entry.notes ?? '')
    || pendingMods !== undefined

  return (
    <>
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className={cn(
              'text-sm font-medium',
              isSuperseded && 'line-through text-muted-foreground'
            )}>
              {entry.topic}
            </span>
            <StatusBadge status={entry.status} />
            {!hasSelection && (
              <span className="text-[10px] font-semibold uppercase tracking-wide px-1.5 py-0.5 rounded-full bg-amber-500/15 text-amber-400">
                {entry.options?.length} options
              </span>
            )}
          </div>
          <div className="text-xs text-muted-foreground mt-0.5">
            {entry.at ? formatRelativePastTime(new Date(entry.at)) : ''}
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <AgentColorBadge appearance={getAgentAppearance(entry.by)} size="xs" />
          <span className="text-xs text-muted-foreground">{getAgentName(entry.by)}</span>
        </div>
      </div>

      {entry.description && (
        <div className="text-xs text-muted-foreground mt-1.5">
          {entry.description}
        </div>
      )}

      {entry.rationale && (
        <div className="text-xs text-muted-foreground mt-1.5">
          <span className="font-medium text-foreground/70">Context:</span> {entry.rationale}
        </div>
      )}

      {entry.supersedes && (
        <div className="text-xs text-amber-600 mt-0.5">
          supersedes: {entry.supersedes}
        </div>
      )}

      {/* Option buttons */}
      <div className="flex flex-wrap gap-1.5 mt-2">
        {entry.options?.map(opt => {
          const isSelected = localSelected === opt.key
          return (
            <button
              key={opt.key}
              onClick={() => setLocalSelected(isSelected ? null : opt.key)}
              disabled={isThisStatusLoading}
              className={cn(
                'text-left px-2.5 py-1.5 rounded-md border text-xs transition-colors',
                isSelected
                  ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-400'
                  : opt.recommended
                    ? 'border-cyan-500/40 bg-cyan-500/10 text-cyan-300 hover:border-cyan-400/50 hover:bg-cyan-500/15'
                    : 'border-border hover:border-primary/30 hover:bg-muted text-foreground',
                isThisStatusLoading && 'opacity-50 cursor-not-allowed'
              )}
            >
              <span className="font-bold mr-1">{opt.key})</span>
              <span>{opt.label}</span>
              {opt.recommended && (
                <span className="inline-flex items-center gap-0.5 ml-1.5 text-[10px] font-semibold uppercase tracking-wide text-cyan-400">
                  <Star className="h-2.5 w-2.5 fill-cyan-400" />
                  Recommended
                </span>
              )}
            </button>
          )
        })}
        <button
          onClick={() => setLocalSelected(localSelected === OTHER_KEY ? null : OTHER_KEY)}
          disabled={isThisStatusLoading}
          className={cn(
            'px-2.5 py-1.5 rounded-md border text-xs transition-colors',
            localSelected === OTHER_KEY
              ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-400'
              : 'border-border hover:border-primary/30 hover:bg-muted text-muted-foreground',
            isThisStatusLoading && 'opacity-50 cursor-not-allowed'
          )}
        >
          Other...
        </button>
      </div>

      {/* Show rationale for selected option */}
      {localSelected && localSelected !== OTHER_KEY && (
        <div className="text-xs text-muted-foreground mt-1.5 pl-2 border-l-2 border-emerald-500/30">
          {entry.options?.find(o => o.key === localSelected)?.rationale}
        </div>
      )}

      {/* Freeform textarea for "Other" */}
      {localSelected === OTHER_KEY && (
        <textarea
          value={localFreeform}
          onChange={e => setLocalFreeform(e.target.value)}
          placeholder="Describe your alternative..."
          rows={2}
          className="w-full mt-2 text-xs border border-border rounded-md px-2.5 py-1.5 bg-background text-foreground placeholder:text-muted-foreground/60 resize-none focus:outline-none focus:ring-1 focus:ring-primary/50"
        />
      )}

      {/* Notes textarea (shown when any option selected) */}
      {localSelected && (
        <textarea
          value={localNotes}
          onChange={e => setLocalNotes(e.target.value)}
          placeholder="Add free-form notes (optional)..."
          rows={1}
          className="w-full mt-1.5 text-xs border border-blue-500/20 bg-blue-500/5 rounded-md px-2.5 py-1.5 text-foreground placeholder:text-muted-foreground/60 resize-none focus:outline-none focus:ring-1 focus:ring-primary/50"
        />
      )}

      {/* Modifications: structured exceptions against the selected option's
          rationale. Shown only when selecting a concrete option; read-only
          once set (accept-once immutability). */}
      {localSelected && localSelected !== OTHER_KEY && !existingMods && (
        <div className="mt-1.5">
          {!showModsForm ? (
            <button
              onClick={() => setShowModsForm(true)}
              className="text-[11px] text-muted-foreground hover:text-foreground underline-offset-2 hover:underline"
            >
              + Add modifications (scoped exceptions / additions)
            </button>
          ) : (
            <div className="border border-purple-500/20 bg-purple-500/5 rounded-md p-2 space-y-1.5">
              <div className="text-[11px] font-medium text-foreground/80">Modifications</div>
              <div>
                <div className="text-[10px] uppercase tracking-wide text-muted-foreground">Excluded clauses</div>
                <div className="flex flex-wrap gap-1 mt-0.5">
                  {excludedClauses.map((c, i) => (
                    <span key={i} className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-red-500/10 text-red-300 text-[11px]">
                      {c}
                      <button onClick={() => setExcludedClauses(excludedClauses.filter((_, j) => j !== i))} className="text-red-400 hover:text-red-200">×</button>
                    </span>
                  ))}
                </div>
                <input
                  value={excludedDraft}
                  onChange={e => setExcludedDraft(e.target.value)}
                  onKeyDown={e => {
                    if (e.key === 'Enter' && excludedDraft.trim()) {
                      e.preventDefault()
                      setExcludedClauses([...excludedClauses, excludedDraft.trim()])
                      setExcludedDraft('')
                    }
                  }}
                  placeholder="Press Enter to add a clause to exclude"
                  className="w-full mt-1 text-[11px] border border-border rounded px-1.5 py-0.5 bg-background"
                />
              </div>
              <div>
                <div className="text-[10px] uppercase tracking-wide text-muted-foreground">Additions</div>
                <div className="flex flex-wrap gap-1 mt-0.5">
                  {additions.map((c, i) => (
                    <span key={i} className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-300 text-[11px]">
                      {c}
                      <button onClick={() => setAdditions(additions.filter((_, j) => j !== i))} className="text-emerald-400 hover:text-emerald-200">×</button>
                    </span>
                  ))}
                </div>
                <input
                  value={additionDraft}
                  onChange={e => setAdditionDraft(e.target.value)}
                  onKeyDown={e => {
                    if (e.key === 'Enter' && additionDraft.trim()) {
                      e.preventDefault()
                      setAdditions([...additions, additionDraft.trim()])
                      setAdditionDraft('')
                    }
                  }}
                  placeholder="Press Enter to add an addition"
                  className="w-full mt-1 text-[11px] border border-border rounded px-1.5 py-0.5 bg-background"
                />
              </div>
              <textarea
                value={modsRationale}
                onChange={e => setModsRationale(e.target.value)}
                placeholder="Rationale for modifications..."
                rows={2}
                maxLength={4096}
                className="w-full text-[11px] border border-border rounded px-1.5 py-0.5 bg-background resize-none"
              />
            </div>
          )}
        </div>
      )}

      {/* Read-only rendering of persisted modifications (immutable post-accept) */}
      {existingMods && (
        <div className="mt-1.5 border-l-2 border-purple-500/40 pl-2 space-y-0.5">
          <div className="text-[10px] uppercase tracking-wide text-muted-foreground">Modifications</div>
          {existingMods.excluded_clauses && existingMods.excluded_clauses.length > 0 && (
            <div className="text-[11px]">
              <span className="text-red-400">Excluded:</span> {existingMods.excluded_clauses.join(', ')}
            </div>
          )}
          {existingMods.additions && existingMods.additions.length > 0 && (
            <div className="text-[11px]">
              <span className="text-emerald-400">Additions:</span> {existingMods.additions.join(', ')}
            </div>
          )}
          {existingMods.rationale && (
            <div className="text-[11px] text-muted-foreground">{existingMods.rationale}</div>
          )}
        </div>
      )}

      {/* Actions: save selection + delete */}
      <div className="flex items-center gap-1 mt-2">
        {isModified && localSelected && (
          <button
            onClick={() => void onSelectOption(
              entry,
              localSelected,
              localSelected === OTHER_KEY ? localFreeform : undefined,
              localNotes || undefined,
              pendingMods,
            )}
            disabled={isThisStatusLoading || (localSelected === OTHER_KEY && !localFreeform.trim())}
            className={cn(
              'text-xs px-3 py-1 rounded-md bg-emerald-600 text-white font-medium transition-colors',
              (isThisStatusLoading || (localSelected === OTHER_KEY && !localFreeform.trim()))
                ? 'opacity-50 cursor-not-allowed'
                : 'hover:bg-emerald-500'
            )}
          >
            {isThisStatusLoading ? (
              <span className="flex items-center gap-1.5"><Loader2 className="h-3 w-3 animate-spin" /> Saving...</span>
            ) : 'Confirm Selection'}
          </button>
        )}
        <button
          onClick={onDelete}
          title="Delete decision"
          className="p-1 rounded text-muted-foreground hover:text-destructive hover:bg-red-500/10 transition-colors"
        >
          <Trash2 className="h-3.5 w-3.5" />
        </button>
      </div>
    </>
  )
}

export function DecisionLogView({ teamId, members, allAgents, decisionMode }: DecisionLogViewProps) {
  const [entries, setEntries] = useState<DecisionEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [mutationError, setMutationError] = useState<string | null>(null)
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
  const [addLoading, setAddLoading] = useState(false)

  // Edit state
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editDecision, setEditDecision] = useState('')
  const [editRationale, setEditRationale] = useState('')
  const [editContext, setEditContext] = useState('')
  const [editLoading, setEditLoading] = useState(false)

  // Delete state
  const [deleteTarget, setDeleteTarget] = useState<DecisionEntry | null>(null)
  const [deleteLoading, setDeleteLoading] = useState(false)

  // Mutation loading for accept/reject
  const [statusLoading, setStatusLoading] = useState<string | null>(null)

  const loadEntries = useCallback(async () => {
    try {
      setError(null)
      const resp = await heartbeatService.getDecisions(teamId, {
        context: contextFilter || undefined,
        last: 50,
      })
      const respEntries = resp.entries
      setEntries(respEntries)
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

  const clearMutationError = () => setMutationError(null)

  const handleAddDecision = async () => {
    if (!newDecision.trim() || !newRationale.trim()) return
    setAddLoading(true)
    clearMutationError()
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
    } catch (err) {
      console.error('[DecisionLogView] Failed to add decision:', err)
      setMutationError(err instanceof Error ? err.message : 'Failed to add decision')
    } finally {
      setAddLoading(false)
    }
  }

  const handleUpdateStatus = async (entry: DecisionEntry, status: 'accepted' | 'rejected') => {
    setStatusLoading(entry.id)
    clearMutationError()
    try {
      await heartbeatService.updateDecision(teamId, entry.id, { status })
      void loadEntries()
    } catch (err) {
      console.error('[DecisionLogView] Failed to update decision status:', err)
      setMutationError(err instanceof Error ? err.message : 'Failed to update status')
    } finally {
      setStatusLoading(null)
    }
  }

  const handleSelectOption = async (
    entry: DecisionEntry,
    key: string,
    freeform?: string,
    notes?: string,
    modifications?: DecisionModifications,
  ) => {
    setStatusLoading(entry.id)
    clearMutationError()
    try {
      await heartbeatService.updateDecision(teamId, entry.id, {
        selected: key,
        freeform: freeform ?? null,
        notes: notes ?? null,
        status: 'accepted',
        ...(modifications ? { modifications } : {}),
      })
      void loadEntries()
    } catch (err) {
      console.error('[DecisionLogView] Failed to select option:', err)
      setMutationError(err instanceof Error ? err.message : 'Failed to select option')
    } finally {
      setStatusLoading(null)
    }
  }

  const startEditing = (entry: DecisionEntry) => {
    setEditingId(entry.id)
    setEditDecision(entry.decision)
    setEditRationale(entry.rationale)
    setEditContext(entry.context ?? '')
    clearMutationError()
  }

  const cancelEditing = () => {
    setEditingId(null)
    setEditDecision('')
    setEditRationale('')
    setEditContext('')
  }

  const handleSaveEdit = async () => {
    if (!editingId || !editDecision.trim() || !editRationale.trim()) return
    setEditLoading(true)
    clearMutationError()
    try {
      await heartbeatService.updateDecision(teamId, editingId, {
        decision: editDecision,
        rationale: editRationale,
        context: editContext || undefined,
      })
      cancelEditing()
      void loadEntries()
    } catch (err) {
      console.error('[DecisionLogView] Failed to update decision:', err)
      setMutationError(err instanceof Error ? err.message : 'Failed to save changes')
    } finally {
      setEditLoading(false)
    }
  }

  const handleConfirmDelete = async () => {
    if (!deleteTarget) return
    setDeleteLoading(true)
    clearMutationError()
    try {
      await heartbeatService.deleteDecision(teamId, deleteTarget.id)
      setDeleteTarget(null)
      void loadEntries()
    } catch (err) {
      console.error('[DecisionLogView] Failed to delete decision:', err)
      setMutationError(err instanceof Error ? err.message : 'Failed to delete decision')
    } finally {
      setDeleteLoading(false)
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
    const existing = grouped.get(ctx)
    if (existing) {
      existing.push(entry)
    } else {
      grouped.set(ctx, [entry])
    }
  }

  // Get unique context tags for filter
  const contextTags = Array.from(new Set(entries.map(e => e.context).filter(Boolean) as string[]))

  // --- Render ---

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
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
      {decisionMode === 'approval' && (
        <div className="flex items-center gap-2 px-3 py-2 mb-3 rounded-lg bg-amber-500/10 border border-amber-500/20 text-amber-600 dark:text-amber-400 text-xs">
          <Scale className="h-3.5 w-3.5 flex-shrink-0" />
          <span>This team requires human approval for decisions.</span>
        </div>
      )}

      {/* Toolbar */}
      <div className="flex items-center justify-between gap-2 flex-wrap">
        <select
          value={contextFilter}
          onChange={e => setContextFilter(e.target.value)}
          className="text-xs border border-border rounded px-2 py-1 bg-background text-foreground"
        >
          <option value="">All contexts</option>
          {contextTags.map(tag => (
            <option key={tag} value={tag}>{tag}</option>
          ))}
        </select>
        <button
          onClick={() => { setShowAddForm(!showAddForm); clearMutationError() }}
          className="flex items-center gap-1 text-xs px-2 py-1 border border-border rounded hover:bg-muted transition-colors"
        >
          <Plus className="h-3 w-3" /> Log Decision
        </button>
      </div>

      {/* Mutation error banner */}
      {mutationError && (
        <div className="flex items-center justify-between gap-2 text-xs text-destructive bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2">
          <span>{mutationError}</span>
          <button onClick={clearMutationError} className="text-muted-foreground hover:text-foreground">
            <X className="h-3 w-3" />
          </button>
        </div>
      )}

      {/* Add form */}
      {showAddForm && (
        <div className="border border-border rounded-lg bg-card overflow-hidden">
          <div className="px-4 py-3 border-b border-border bg-muted/30">
            <h4 className="text-sm font-medium text-foreground">Log New Decision</h4>
          </div>
          <div className="p-4 space-y-3">
            <div>
              <label className="block text-xs font-medium text-muted-foreground mb-1">Decision *</label>
              <input
                type="text"
                placeholder="What was decided..."
                value={newDecision}
                onChange={e => setNewDecision(e.target.value)}
                className="w-full text-sm border border-border rounded-md px-3 py-1.5 bg-background text-foreground placeholder:text-muted-foreground/60 focus:outline-none focus:ring-1 focus:ring-primary/50"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-muted-foreground mb-1">Rationale *</label>
              <textarea
                placeholder="Why this decision was made..."
                value={newRationale}
                onChange={e => setNewRationale(e.target.value)}
                className="w-full text-sm border border-border rounded-md px-3 py-1.5 bg-background text-foreground placeholder:text-muted-foreground/60 resize-none focus:outline-none focus:ring-1 focus:ring-primary/50"
                rows={3}
              />
            </div>
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1">By</label>
                <select
                  value={newBy}
                  onChange={e => setNewBy(e.target.value)}
                  className="w-full text-xs border border-border rounded-md px-2 py-1.5 bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary/50"
                >
                  <option value="">Select agent</option>
                  {members.map(m => (
                    <option key={m.agentId} value={m.agentId}>{m.displayName}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1">Context tag</label>
                <input
                  type="text"
                  placeholder="e.g. architecture"
                  value={newContext}
                  onChange={e => setNewContext(e.target.value)}
                  className="w-full text-xs border border-border rounded-md px-2 py-1.5 bg-background text-foreground placeholder:text-muted-foreground/60 focus:outline-none focus:ring-1 focus:ring-primary/50"
                />
              </div>
              <div className="col-span-2 sm:col-span-2">
                <label className="block text-xs font-medium text-muted-foreground mb-1">Supersedes ID</label>
                <input
                  type="text"
                  placeholder="ID of decision this supersedes"
                  value={newSupersedes}
                  onChange={e => setNewSupersedes(e.target.value)}
                  className="w-full text-xs border border-border rounded-md px-2 py-1.5 bg-background text-foreground placeholder:text-muted-foreground/60 focus:outline-none focus:ring-1 focus:ring-primary/50"
                />
              </div>
            </div>
            <div className="flex items-center justify-end gap-2 pt-1">
              <button
                onClick={() => setShowAddForm(false)}
                className="text-xs px-3 py-1.5 rounded-md border border-border text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => void handleAddDecision()}
                disabled={addLoading || !newDecision.trim() || !newRationale.trim()}
                className={cn(
                  'text-xs px-4 py-1.5 rounded-md bg-primary text-primary-foreground font-medium transition-colors',
                  addLoading || !newDecision.trim() || !newRationale.trim()
                    ? 'opacity-50 cursor-not-allowed'
                    : 'hover:bg-primary/90'
                )}
              >
                {addLoading ? (
                  <span className="flex items-center gap-1.5">
                    <Loader2 className="h-3 w-3 animate-spin" /> Saving...
                  </span>
                ) : 'Log Decision'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Grouped decisions */}
      {entries.length === 0 ? (
        <div className="text-sm text-muted-foreground text-center py-8">No decisions logged yet.</div>
      ) : (
        Array.from(grouped.entries()).map(([ctx, decisions]) => {
          const isExpanded = expandedContexts.has(ctx) || expandedContexts.has('__all__')
          return (
            <div key={ctx} className="border border-border rounded-lg overflow-hidden">
              <button
                onClick={() => toggleContext(ctx)}
                className="w-full flex items-center justify-between px-3 py-2 bg-muted/50 hover:bg-muted text-sm font-medium transition-colors"
              >
                <span>{ctx} ({decisions.length} decision{decisions.length !== 1 ? 's' : ''})</span>
                <ChevronDown className={cn('h-4 w-4 transition-transform', isExpanded && 'rotate-180')} />
              </button>
              {isExpanded && (
                <div className="divide-y divide-border">
                  {decisions.map(entry => {
                    const isEditing = editingId === entry.id
                    const isSuperseded = supersededIds.has(entry.id)
                    const isRejected = entry.status === 'rejected'
                    const isAccepted = entry.status === 'accepted'
                    const isThisStatusLoading = statusLoading === entry.id

                    return (
                      <div
                        key={entry.id}
                        className={cn(
                          'px-3 py-3 transition-colors',
                          isSuperseded && 'opacity-50',
                          isAccepted && 'border-l-4 border-emerald-500',
                          isRejected && 'border-l-4 border-red-500',
                          !isAccepted && !isRejected && 'border-l-4 border-transparent'
                        )}
                      >
                        {isEditing ? (
                          /* ---- Inline edit mode ---- */
                          <div className="space-y-2">
                            <div>
                              <label className="block text-[10px] font-medium text-muted-foreground mb-0.5 uppercase tracking-wide">Decision</label>
                              <input
                                type="text"
                                value={editDecision}
                                onChange={e => setEditDecision(e.target.value)}
                                className="w-full text-sm border border-border rounded-md px-2.5 py-1 bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary/50"
                              />
                            </div>
                            <div>
                              <label className="block text-[10px] font-medium text-muted-foreground mb-0.5 uppercase tracking-wide">Rationale</label>
                              <textarea
                                value={editRationale}
                                onChange={e => setEditRationale(e.target.value)}
                                rows={2}
                                className="w-full text-sm border border-border rounded-md px-2.5 py-1 bg-background text-foreground resize-none focus:outline-none focus:ring-1 focus:ring-primary/50"
                              />
                            </div>
                            <div>
                              <label className="block text-[10px] font-medium text-muted-foreground mb-0.5 uppercase tracking-wide">Context tag</label>
                              <input
                                type="text"
                                value={editContext}
                                onChange={e => setEditContext(e.target.value)}
                                className="w-full text-sm border border-border rounded-md px-2.5 py-1 bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary/50"
                              />
                            </div>
                            <div className="flex items-center justify-end gap-2 pt-1">
                              <button
                                onClick={cancelEditing}
                                disabled={editLoading}
                                className="text-xs px-3 py-1 rounded-md border border-border text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
                              >
                                Cancel
                              </button>
                              <button
                                onClick={() => void handleSaveEdit()}
                                disabled={editLoading || !editDecision.trim() || !editRationale.trim()}
                                className={cn(
                                  'text-xs px-3 py-1 rounded-md bg-primary text-primary-foreground font-medium transition-colors',
                                  editLoading || !editDecision.trim() || !editRationale.trim()
                                    ? 'opacity-50 cursor-not-allowed'
                                    : 'hover:bg-primary/90'
                                )}
                              >
                                {editLoading ? (
                                  <span className="flex items-center gap-1.5">
                                    <Loader2 className="h-3 w-3 animate-spin" /> Saving...
                                  </span>
                                ) : 'Save'}
                              </button>
                            </div>
                          </div>
                        ) : (entry.options && entry.options.length > 0) ? (
                          /* ---- Multi-option display mode ---- */
                          <MultiOptionCard
                            entry={entry}
                            isSuperseded={isSuperseded}
                            isThisStatusLoading={isThisStatusLoading}
                            getAgentAppearance={getAgentAppearance}
                            getAgentName={getAgentName}
                            onSelectOption={handleSelectOption}
                            onDelete={() => setDeleteTarget(entry)}
                          />
                        ) : (
                          /* ---- Simple display mode ---- */
                          <>
                            <div className="flex items-start justify-between gap-2">
                              <div className="flex-1 min-w-0">
                                <div className="flex items-center gap-2 flex-wrap">
                                  <span className={cn(
                                    'text-sm font-medium',
                                    isSuperseded && 'line-through text-muted-foreground',
                                    isRejected && 'line-through text-muted-foreground'
                                  )}>
                                    {entry.decision}
                                  </span>
                                  <StatusBadge status={entry.status} />
                                </div>
                                <div className="text-xs text-muted-foreground mt-0.5">
                                  {entry.at ? formatRelativePastTime(new Date(entry.at)) : ''}
                                </div>
                              </div>
                              <div className="flex items-center gap-2 shrink-0">
                                <AgentColorBadge appearance={getAgentAppearance(entry.by)} size="xs" />
                                <span className="text-xs text-muted-foreground">{getAgentName(entry.by)}</span>
                              </div>
                            </div>

                            <div className={cn(
                              'text-xs text-muted-foreground mt-1.5',
                              isRejected && 'line-through'
                            )}>
                              <span className="font-medium text-foreground/70">Rationale:</span> {entry.rationale}
                            </div>

                            {entry.supersedes && (
                              <div className="text-xs text-amber-600 mt-0.5">
                                supersedes: {entry.supersedes}
                              </div>
                            )}

                            {entry.status === 'deferred' && entry.revisit_after && (
                              <div className="text-xs text-yellow-600 mt-0.5 inline-flex items-center gap-1">
                                <Clock className="h-3 w-3" />
                                Revisit after: {entry.revisit_after}
                              </div>
                            )}

                            {isAcceptedAsProposed(entry) && (
                              <div className="text-xs text-emerald-500 mt-0.5">
                                Accepted as proposed
                              </div>
                            )}

                            {/* Action buttons */}
                            <div className="flex items-center gap-1 mt-2">
                              <button
                                onClick={() => void handleUpdateStatus(entry, 'accepted')}
                                disabled={isThisStatusLoading}
                                title="Accept decision"
                                className={cn(
                                  'p-1 rounded transition-colors',
                                  isAccepted
                                    ? 'text-emerald-400 bg-emerald-500/15'
                                    : 'text-muted-foreground hover:text-emerald-400 hover:bg-emerald-500/10',
                                  isThisStatusLoading && 'opacity-50 cursor-not-allowed'
                                )}
                              >
                                {isThisStatusLoading ? (
                                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                                ) : (
                                  <Check className="h-3.5 w-3.5" />
                                )}
                              </button>
                              <button
                                onClick={() => void handleUpdateStatus(entry, 'rejected')}
                                disabled={isThisStatusLoading}
                                title="Reject decision"
                                className={cn(
                                  'p-1 rounded transition-colors',
                                  isRejected
                                    ? 'text-red-400 bg-red-500/15'
                                    : 'text-muted-foreground hover:text-red-400 hover:bg-red-500/10',
                                  isThisStatusLoading && 'opacity-50 cursor-not-allowed'
                                )}
                              >
                                <X className="h-3.5 w-3.5" />
                              </button>
                              <button
                                onClick={() => startEditing(entry)}
                                title="Edit decision"
                                className="p-1 rounded text-muted-foreground hover:text-primary hover:bg-primary/10 transition-colors"
                              >
                                <Pencil className="h-3.5 w-3.5" />
                              </button>
                              <button
                                onClick={() => setDeleteTarget(entry)}
                                title="Delete decision"
                                className="p-1 rounded text-muted-foreground hover:text-destructive hover:bg-red-500/10 transition-colors"
                              >
                                <Trash2 className="h-3.5 w-3.5" />
                              </button>
                            </div>
                          </>
                        )}
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          )
        })
      )}

      {/* Delete confirmation dialog */}
      <ConfirmDialog
        isOpen={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => void handleConfirmDelete()}
        title="Delete Decision"
        message={deleteTarget ? `Are you sure you want to delete "${deleteTarget.topic || deleteTarget.decision}"? This action cannot be undone.` : ''}
        confirmLabel="Delete"
        cancelLabel="Cancel"
        variant="danger"
        isLoading={deleteLoading}
      />
    </div>
  )
}
