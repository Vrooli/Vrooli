/**
 * KnowledgeLogView - Team knowledge log with CRUD controls.
 *
 * Features:
 * - Add / Edit / Delete knowledge entries
 * - Topic filtering and grouping with accordion
 * - Supersede tracking (strikethrough for superseded entries)
 * - 30-second auto-refresh
 */

import { useEffect, useMemo, useState, useCallback } from 'react'
import { Plus, ChevronDown, Loader2, Pencil, X, Trash2, Search } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamMember } from '@/types/team'
import type { Agent } from '@/types/agent'
import { AgentColorBadge } from '@/components/shared/AgentColorBadge'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { MarkdownRenderer } from '@/components/markdown'
import { formatRelativePastTime } from '@/lib/timeUtils'
import * as heartbeatService from '@/services/heartbeatService'
import type { KnowledgeEntry } from '@/services/heartbeatService'

interface KnowledgeLogViewProps {
  teamId: string
  members: TeamMember[]
  allAgents?: Agent[]
}

export function KnowledgeLogView({ teamId, members, allAgents }: KnowledgeLogViewProps) {
  const [entries, setEntries] = useState<KnowledgeEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [mutationError, setMutationError] = useState<string | null>(null)
  const [topicFilter, setTopicFilter] = useState('')
  const [searchQuery, setSearchQuery] = useState('')
  const [expandedTopics, setExpandedTopics] = useState<Set<string>>(new Set(['__all__']))
  const [showAddForm, setShowAddForm] = useState(false)
  const [supersededIds, setSupersededIds] = useState<Set<string>>(new Set())

  // Add form state. Identity is no longer a form field — every UI
  // write is attributed to `kind=operator-direct` and the API derives
  // the `caller` display string. Operators can attach an optional
  // freeform note (e.g., "acting-as researcher") via newCallerNote;
  // it is never used by validators. Canon:
  // docs/agent-system/RUNTIME_ATTRIBUTION.md.
  const [newTopic, setNewTopic] = useState('')
  const [newContent, setNewContent] = useState('')
  const [newSource, setNewSource] = useState('')
  const [newCallerNote, setNewCallerNote] = useState('')
  const [newSupersedes, setNewSupersedes] = useState('')
  const [addLoading, setAddLoading] = useState(false)

  // Edit state
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editTopic, setEditTopic] = useState('')
  const [editContent, setEditContent] = useState('')
  const [editSource, setEditSource] = useState('')
  const [editLoading, setEditLoading] = useState(false)

  // Delete state
  const [deleteTarget, setDeleteTarget] = useState<KnowledgeEntry | null>(null)
  const [deleteLoading, setDeleteLoading] = useState(false)

  const loadEntries = useCallback(async () => {
    try {
      setError(null)
      const resp = await heartbeatService.getTeamCorpus(teamId, {
        topic: topicFilter || undefined,
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
      console.error('[KnowledgeLogView] Failed to load knowledge:', err)
      setError(err instanceof Error ? err.message : 'Failed to load knowledge')
    } finally {
      setLoading(false)
    }
  }, [teamId, topicFilter])

  useEffect(() => {
    void loadEntries()
    const interval = setInterval(() => void loadEntries(), 30_000)
    return () => clearInterval(interval)
  }, [loadEntries])

  const getAgentName = useCallback((agentId: string) => {
    const member = members.find(m => m.agentId === agentId)
    return member?.displayName ?? agentId
  }, [members])

  const getAgentAppearance = useCallback((agentId: string) => {
    return allAgents?.find(a => a.id === agentId)?.appearance ?? null
  }, [allAgents])

  const clearMutationError = () => setMutationError(null)

  const handleAdd = async () => {
    if (!newTopic.trim() || !newContent.trim()) return
    setAddLoading(true)
    clearMutationError()
    try {
      await heartbeatService.addKnowledge(teamId, {
        topic: newTopic,
        content: newContent,
        caller_note: newCallerNote || undefined,
        source: newSource || undefined,
        supersedes: newSupersedes || undefined,
      })
      setNewTopic('')
      setNewContent('')
      setNewSource('')
      setNewCallerNote('')
      setNewSupersedes('')
      setShowAddForm(false)
      void loadEntries()
    } catch (err) {
      console.error('[KnowledgeLogView] Failed to add knowledge:', err)
      setMutationError(err instanceof Error ? err.message : 'Failed to add knowledge')
    } finally {
      setAddLoading(false)
    }
  }

  const startEditing = (entry: KnowledgeEntry) => {
    setEditingId(entry.id)
    setEditTopic(entry.topic)
    setEditContent(entry.content)
    setEditSource(entry.source ?? '')
    clearMutationError()
  }

  const cancelEditing = () => {
    setEditingId(null)
    setEditTopic('')
    setEditContent('')
    setEditSource('')
  }

  const handleSaveEdit = async () => {
    if (!editingId || !editTopic.trim() || !editContent.trim()) return
    setEditLoading(true)
    clearMutationError()
    try {
      await heartbeatService.updateKnowledge(teamId, editingId, {
        topic: editTopic,
        content: editContent,
        source: editSource || undefined,
      })
      cancelEditing()
      void loadEntries()
    } catch (err) {
      console.error('[KnowledgeLogView] Failed to update knowledge:', err)
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
      await heartbeatService.deleteKnowledge(teamId, deleteTarget.id)
      setDeleteTarget(null)
      void loadEntries()
    } catch (err) {
      console.error('[KnowledgeLogView] Failed to delete knowledge:', err)
      setMutationError(err instanceof Error ? err.message : 'Failed to delete knowledge')
    } finally {
      setDeleteLoading(false)
    }
  }

  const toggleTopic = (topic: string) => {
    setExpandedTopics(prev => {
      const next = new Set(prev)
      if (next.has(topic)) next.delete(topic)
      else next.add(topic)
      return next
    })
  }

  const filteredEntries = useMemo(() => {
    const query = searchQuery.trim().toLowerCase()
    if (!query) return entries
    return entries.filter((entry) => {
      const agentName = getAgentName(entry.caller)
      return [
        entry.id,
        entry.caller,
        agentName,
        entry.topic,
        entry.content,
        entry.source,
        entry.supersedes,
      ].some((value) => (value ?? '').toLowerCase().includes(query))
    })
  }, [entries, searchQuery, getAgentName])

  // Group by topic
  const grouped = new Map<string, KnowledgeEntry[]>()
  for (const entry of filteredEntries) {
    const t = entry.topic || '(untagged)'
    const existing = grouped.get(t)
    if (existing) {
      existing.push(entry)
    } else {
      grouped.set(t, [entry])
    }
  }

  const topicTags = Array.from(new Set(entries.map(e => e.topic).filter(Boolean)))

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
        Error loading knowledge: {error}
        <button onClick={() => void loadEntries()} className="ml-2 text-xs underline hover:no-underline">Retry</button>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {/* Toolbar */}
      <div className="flex items-center justify-between gap-2 flex-wrap">
        <select
          value={topicFilter}
          onChange={e => setTopicFilter(e.target.value)}
          className="text-xs border border-border rounded px-2 py-1 bg-background text-foreground"
        >
          <option value="">All topics</option>
          {topicTags.map(tag => (
            <option key={tag} value={tag}>{tag}</option>
          ))}
        </select>
        <div className="relative min-w-[180px] flex-1">
          <Search className="absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <input
            type="search"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search knowledge..."
            className="w-full rounded border border-border bg-background py-1 pl-7 pr-2 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary/50"
          />
        </div>
        <button
          onClick={() => { setShowAddForm(!showAddForm); clearMutationError() }}
          className="flex items-center gap-1 text-xs px-2 py-1 border border-border rounded hover:bg-muted transition-colors"
        >
          <Plus className="h-3 w-3" /> Add Knowledge
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
            <h4 className="text-sm font-medium text-foreground">Add Knowledge Entry</h4>
          </div>
          <div className="p-4 space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1">Topic *</label>
                <input
                  type="text"
                  placeholder="e.g. api-patterns, conventions"
                  value={newTopic}
                  onChange={e => setNewTopic(e.target.value)}
                  className="w-full text-sm border border-border rounded-md px-3 py-1.5 bg-background text-foreground placeholder:text-muted-foreground/60 focus:outline-none focus:ring-1 focus:ring-primary/50"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1" title="Optional freeform note. Identity is auto-attributed as operator-direct.">Note</label>
                <input
                  type="text"
                  placeholder="optional, e.g. acting-as researcher"
                  value={newCallerNote}
                  onChange={e => setNewCallerNote(e.target.value)}
                  className="w-full text-sm border border-border rounded-md px-3 py-1.5 bg-background text-foreground placeholder:text-muted-foreground/60 focus:outline-none focus:ring-1 focus:ring-primary/50"
                  maxLength={256}
                />
              </div>
            </div>
            <div>
              <label className="block text-xs font-medium text-muted-foreground mb-1">Content *</label>
              <textarea
                placeholder="What was discovered or learned..."
                value={newContent}
                onChange={e => setNewContent(e.target.value)}
                className="w-full text-sm border border-border rounded-md px-3 py-1.5 bg-background text-foreground placeholder:text-muted-foreground/60 resize-none focus:outline-none focus:ring-1 focus:ring-primary/50"
                rows={3}
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1">Source</label>
                <input
                  type="text"
                  placeholder="e.g. codebase exploration, API docs"
                  value={newSource}
                  onChange={e => setNewSource(e.target.value)}
                  className="w-full text-xs border border-border rounded-md px-2 py-1.5 bg-background text-foreground placeholder:text-muted-foreground/60 focus:outline-none focus:ring-1 focus:ring-primary/50"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1">Supersedes ID</label>
                <input
                  type="text"
                  placeholder="ID of entry this replaces"
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
                onClick={() => void handleAdd()}
                disabled={addLoading || !newTopic.trim() || !newContent.trim()}
                className={cn(
                  'text-xs px-4 py-1.5 rounded-md bg-primary text-primary-foreground font-medium transition-colors',
                  addLoading || !newTopic.trim() || !newContent.trim()
                    ? 'opacity-50 cursor-not-allowed'
                    : 'hover:bg-primary/90'
                )}
              >
                {addLoading ? (
                  <span className="flex items-center gap-1.5">
                    <Loader2 className="h-3 w-3 animate-spin" /> Saving...
                  </span>
                ) : 'Add Entry'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Grouped entries */}
      {entries.length === 0 ? (
        <div className="text-sm text-muted-foreground text-center py-8">No knowledge entries yet.</div>
      ) : filteredEntries.length === 0 ? (
        <div className="text-sm text-muted-foreground text-center py-8">No matching knowledge entries.</div>
      ) : (
        Array.from(grouped.entries()).map(([topic, topicEntries]) => {
          const isExpanded = expandedTopics.has(topic) || expandedTopics.has('__all__')
          return (
            <div key={topic} className="border border-border rounded-lg overflow-hidden">
              <button
                onClick={() => toggleTopic(topic)}
                className="w-full flex items-center justify-between px-3 py-2 bg-muted/50 hover:bg-muted text-sm font-medium transition-colors"
              >
                <span>{topic} ({topicEntries.length} entr{topicEntries.length !== 1 ? 'ies' : 'y'})</span>
                <ChevronDown className={cn('h-4 w-4 transition-transform', isExpanded && 'rotate-180')} />
              </button>
              {isExpanded && (
                <div className="divide-y divide-border">
                  {topicEntries.map(entry => {
                    const isEditing = editingId === entry.id
                    const isSuperseded = supersededIds.has(entry.id)

                    return (
                      <div
                        key={entry.id}
                        className={cn(
                          'min-w-0 overflow-hidden px-3 py-3 transition-colors',
                          isSuperseded && 'opacity-50'
                        )}
                      >
                        {isEditing ? (
                          <div className="space-y-2">
                            <div>
                              <label className="block text-[10px] font-medium text-muted-foreground mb-0.5 uppercase tracking-wide">Topic</label>
                              <input
                                type="text"
                                value={editTopic}
                                onChange={e => setEditTopic(e.target.value)}
                                className="w-full text-sm border border-border rounded-md px-2.5 py-1 bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary/50"
                              />
                            </div>
                            <div>
                              <label className="block text-[10px] font-medium text-muted-foreground mb-0.5 uppercase tracking-wide">Content</label>
                              <textarea
                                value={editContent}
                                onChange={e => setEditContent(e.target.value)}
                                rows={3}
                                className="w-full text-sm border border-border rounded-md px-2.5 py-1 bg-background text-foreground resize-none focus:outline-none focus:ring-1 focus:ring-primary/50"
                              />
                            </div>
                            <div>
                              <label className="block text-[10px] font-medium text-muted-foreground mb-0.5 uppercase tracking-wide">Source</label>
                              <input
                                type="text"
                                value={editSource}
                                onChange={e => setEditSource(e.target.value)}
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
                                disabled={editLoading || !editTopic.trim() || !editContent.trim()}
                                className={cn(
                                  'text-xs px-3 py-1 rounded-md bg-primary text-primary-foreground font-medium transition-colors',
                                  editLoading || !editTopic.trim() || !editContent.trim()
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
                        ) : (
                          <>
                            <div className="flex items-start justify-between gap-2">
                              <div className="flex-1 min-w-0">
                                <MarkdownRenderer
                                  content={entry.content}
                                  className="break-words text-sm text-foreground [&_*]:break-words [&_pre]:max-w-full [&_pre]:overflow-x-auto"
                                />
                                <div className="flex items-center gap-2 mt-1 text-xs text-muted-foreground">
                                  {entry.at ? formatRelativePastTime(new Date(entry.at)) : ''}
                                  {entry.source && (
                                    <span className="text-muted-foreground/70">via {entry.source}</span>
                                  )}
                                </div>
                              </div>
                              <div className="flex items-center gap-2 shrink-0">
                                <AgentColorBadge appearance={getAgentAppearance(entry.caller)} size="xs" />
                                <span className="max-w-[12rem] truncate text-xs text-muted-foreground">{getAgentName(entry.caller)}</span>
                              </div>
                            </div>

                            {entry.supersedes && (
                              <div className="text-xs text-amber-600 mt-0.5">
                                supersedes: {entry.supersedes}
                              </div>
                            )}

                            <div className="flex items-center gap-1 mt-2">
                              <button
                                onClick={() => startEditing(entry)}
                                title="Edit"
                                className="p-1 rounded text-muted-foreground hover:text-primary hover:bg-primary/10 transition-colors"
                              >
                                <Pencil className="h-3.5 w-3.5" />
                              </button>
                              <button
                                onClick={() => setDeleteTarget(entry)}
                                title="Delete"
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

      <ConfirmDialog
        isOpen={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => void handleConfirmDelete()}
        title="Delete Knowledge"
        message={deleteTarget ? `Are you sure you want to delete this knowledge entry? This action cannot be undone.` : ''}
        confirmLabel="Delete"
        cancelLabel="Cancel"
        variant="danger"
        isLoading={deleteLoading}
      />
    </div>
  )
}
