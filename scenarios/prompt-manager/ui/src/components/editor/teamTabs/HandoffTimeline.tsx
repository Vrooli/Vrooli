/**
 * HandoffTimeline - Vertical timeline showing handoff history with date grouping.
 */

import { useEffect, useMemo, useState, useCallback } from 'react'
import { ChevronDown, Loader2, Clock, AlertCircle, Trash2, Search } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamMember } from '@/types/team'
import type { Agent } from '@/types/agent'
import { AgentColorBadge } from '@/components/shared/AgentColorBadge'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { MarkdownRenderer } from '@/components/markdown'
import { formatRelativePastTime, formatDateGroup } from '@/lib/timeUtils'
import * as heartbeatService from '@/services/heartbeatService'
import type { HandoffHistoryEntry } from '@/services/heartbeatService'

interface HandoffTimelineProps {
  teamId: string
  members: TeamMember[]
  allAgents?: Agent[]
}

const PAGE_SIZE = 10

interface DateGroup {
  label: string
  entries: { entry: HandoffHistoryEntry; index: number }[]
}

function groupEntriesByDate(entries: HandoffHistoryEntry[]): DateGroup[] {
  const groups: DateGroup[] = []
  let currentLabel = ''

  entries.forEach((entry, index) => {
    const date = new Date(entry.timestamp)
    const label = formatDateGroup(date)

    if (label !== currentLabel) {
      currentLabel = label
      groups.push({ label, entries: [] })
    }
    const lastGroup = groups[groups.length - 1]
    if (lastGroup) lastGroup.entries.push({ entry, index })
  })

  return groups
}

function HandoffContent({ content }: { content: string }) {
  return (
    <MarkdownRenderer
      content={content}
      className="text-sm text-muted-foreground break-words [&_*]:break-words [&_pre]:max-w-full [&_pre]:overflow-x-auto"
    />
  )
}

export function HandoffTimeline({ teamId, members, allAgents }: HandoffTimelineProps) {
  const [entries, setEntries] = useState<HandoffHistoryEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [pageSize, setPageSize] = useState(PAGE_SIZE)
  const [agentFilter, setAgentFilter] = useState('')
  const [searchQuery, setSearchQuery] = useState('')
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set())
  const [showClearConfirm, setShowClearConfirm] = useState(false)
  const [clearing, setClearing] = useState(false)

  const loadEntries = useCallback(async () => {
    try {
      setError(null)
      const resp = await heartbeatService.getHandoffHistory(teamId, {
        agent: agentFilter || undefined,
        last: pageSize,
      })
      setEntries(resp.entries)
    } catch (err) {
      console.error('[HandoffTimeline] Failed to load handoffs:', err)
      setError(err instanceof Error ? err.message : 'Failed to load handoffs')
    } finally {
      setLoading(false)
    }
  }, [teamId, agentFilter, pageSize])

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

  const toggleExpand = (id: string) => {
    setExpandedIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const truncateContent = (content: string, maxLines = 3) => {
    const lines = content.split('\n')
    if (lines.length <= maxLines) return { text: content, truncated: false }
    return { text: lines.slice(0, maxLines).join('\n'), truncated: true }
  }

  const handleClear = async () => {
    setClearing(true)
    try {
      await heartbeatService.clearHandoffHistory(teamId, agentFilter || undefined)
      setShowClearConfirm(false)
      setExpandedIds(new Set())
      await loadEntries()
    } catch (err) {
      console.error('[HandoffTimeline] Failed to clear handoffs:', err)
    } finally {
      setClearing(false)
    }
  }

  const clearMessage = agentFilter
    ? `Clear all handoff history for ${getAgentName(agentFilter)}? This cannot be undone.`
    : 'Clear all handoff history for this team? This cannot be undone.'

  const filteredEntries = useMemo(() => {
    const query = searchQuery.trim().toLowerCase()
    if (!query) return entries
    return entries.filter((entry) => {
      const agentName = getAgentName(entry.agentId)
      return (
        entry.content.toLowerCase().includes(query) ||
        entry.runId.toLowerCase().includes(query) ||
        entry.agentId.toLowerCase().includes(query) ||
        agentName.toLowerCase().includes(query)
      )
    })
  }, [entries, searchQuery, getAgentName])

  const dateGroups = groupEntriesByDate(filteredEntries)

  // Loading state
  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
        <Loader2 className="h-6 w-6 animate-spin mb-2" />
        <span className="text-sm">Loading handoffs...</span>
      </div>
    )
  }

  // Error state
  if (error) {
    return (
      <div className="border border-destructive/30 rounded-lg p-4 bg-destructive/5">
        <div className="flex items-center gap-2 mb-1">
          <AlertCircle className="h-4 w-4 text-destructive flex-shrink-0" />
          <span className="text-sm font-medium text-destructive">Error loading handoffs</span>
        </div>
        <p className="text-sm text-muted-foreground ml-6">{error}</p>
        <button
          onClick={() => void loadEntries()}
          className="ml-6 mt-2 text-xs text-primary hover:underline"
        >
          Retry
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Filter + Clear */}
      <div className="flex items-center gap-2 flex-wrap">
        <select
          value={agentFilter}
          onChange={e => setAgentFilter(e.target.value)}
          className="text-xs border border-border rounded px-2 py-1 bg-background text-foreground"
        >
          <option value="">All members</option>
          {members.map(m => (
            <option key={m.agentId} value={m.agentId}>{m.displayName}</option>
          ))}
        </select>
        <div className="relative min-w-[180px] flex-1">
          <Search className="absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <input
            type="search"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search handoffs..."
            className="w-full rounded border border-border bg-background py-1 pl-7 pr-2 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary/50"
          />
        </div>
        {entries.length > 0 && (
          <button
            onClick={() => setShowClearConfirm(true)}
            disabled={clearing}
            className="ml-auto text-xs text-muted-foreground hover:text-destructive transition-colors p-1 rounded hover:bg-destructive/10"
            title="Clear handoff history"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        )}
      </div>

      {entries.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <Clock className="h-8 w-8 mb-3 opacity-40" />
          <p className="text-sm font-medium mb-1">No handoffs recorded yet</p>
          <p className="text-xs text-center max-w-xs opacity-70">
            Handoffs are automatically captured at the end of each heartbeat.
          </p>
        </div>
      ) : filteredEntries.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <Search className="h-8 w-8 mb-3 opacity-40" />
          <p className="text-sm font-medium mb-1">No matching handoffs</p>
        </div>
      ) : (
        <div className="relative pl-5">
          {/* Vertical timeline line */}
          <div className="absolute left-[7px] top-2 bottom-2 w-0.5 bg-border" />

          {dateGroups.map((group, gi) => (
            <div key={group.label} className={cn(gi > 0 && 'mt-4')}>
              {/* Date group header */}
              <div className="relative flex items-center gap-2 mb-3">
                <div className="absolute -left-5 w-4 h-4 rounded-full bg-muted border-2 border-border flex items-center justify-center">
                  <div className="w-1.5 h-1.5 rounded-full bg-muted-foreground/50" />
                </div>
                <span className="text-xs font-medium text-muted-foreground">{group.label}</span>
              </div>

              {/* Entries in this date group */}
              <div className="space-y-3">
                {group.entries.map(({ entry, index }) => {
                  const id = `${entry.runId}-${index}`
                  const isExpanded = expandedIds.has(id)
                  const { text, truncated } = truncateContent(entry.content)

                  return (
                    <div key={id} className="relative">
                      {/* Timeline dot */}
                      <div className="absolute -left-5 top-3 w-2.5 h-2.5 rounded-full bg-primary/30 border-2 border-primary/50" />

                      {/* Card */}
                      <div className="min-w-0 overflow-hidden border border-border rounded-lg p-3 bg-card">
                        <div className="flex items-center justify-between gap-2 mb-2">
                          <div className="flex min-w-0 items-center gap-2">
                            <AgentColorBadge appearance={getAgentAppearance(entry.agentId)} size="sm" />
                            <span className="truncate text-sm font-medium">{getAgentName(entry.agentId)}</span>
                          </div>
                          <span className="shrink-0 text-xs text-muted-foreground">
                            {formatRelativePastTime(new Date(entry.timestamp))}
                          </span>
                        </div>
                        <HandoffContent content={isExpanded ? entry.content : text} />
                        {truncated && (
                          <button
                            onClick={() => toggleExpand(id)}
                            className="text-xs text-primary mt-1.5 flex items-center gap-0.5 hover:underline"
                          >
                            <ChevronDown className={cn('h-3 w-3 transition-transform', isExpanded && 'rotate-180')} />
                            {isExpanded ? 'Show less' : 'Show more'}
                          </button>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          ))}

          {entries.length >= pageSize && (
            <button
              onClick={() => setPageSize(s => s + PAGE_SIZE)}
              className="w-full text-xs text-primary py-2 hover:underline mt-3"
            >
              Load more
            </button>
          )}
        </div>
      )}

      <ConfirmDialog
        isOpen={showClearConfirm}
        onClose={() => setShowClearConfirm(false)}
        onConfirm={() => void handleClear()}
        title="Clear Handoff History"
        message={clearMessage}
        confirmLabel="Clear"
        variant="danger"
        isLoading={clearing}
      />
    </div>
  )
}
