/**
 * HandoffTimeline - Vertical timeline/feed showing handoff history.
 */

import { useEffect, useState, useCallback } from 'react'
import { ChevronDown, Loader2, Clock, AlertCircle } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamMember } from '@/types/team'
import type { Agent } from '@/types/agent'
import { AgentColorBadge } from '@/components/shared/AgentColorBadge'
import { formatRelativePastTime } from '@/lib/timeUtils'
import * as heartbeatService from '@/services/heartbeatService'
import type { HandoffHistoryEntry } from '@/services/heartbeatService'

interface HandoffTimelineProps {
  teamId: string
  members: TeamMember[]
  allAgents?: Agent[]
}

const PAGE_SIZE = 10

/**
 * Renders handoff content with basic markdown-like styling.
 * - Lines starting with `**` get bold/accent treatment
 * - Lines starting with `- ` render as styled list items
 * - Everything else renders as plain pre-wrapped text
 */
function HandoffContent({ content }: { content: string }) {
  const lines = content.split('\n')

  return (
    <div className="text-sm text-muted-foreground whitespace-pre-wrap">
      {lines.map((line, i) => {
        // Bold label lines like "**Status**: ..." or "**Completed**:"
        if (line.startsWith('**')) {
          const match = line.match(/^\*\*(.+?)\*\*(.*)$/)
          if (match) {
            return (
              <div key={i} className="text-foreground">
                <span className="font-semibold">{match[1]}</span>
                <span className="text-muted-foreground">{match[2]}</span>
              </div>
            )
          }
        }

        // List items
        if (line.startsWith('- ')) {
          return (
            <div key={i} className="flex gap-2 pl-1">
              <span className="text-muted-foreground/60 select-none" aria-hidden="true">&bull;</span>
              <span>{line.slice(2)}</span>
            </div>
          )
        }

        // Plain text (preserve empty lines)
        return <div key={i}>{line || '\u00A0'}</div>
      })}
    </div>
  )
}

export function HandoffTimeline({ teamId, members, allAgents }: HandoffTimelineProps) {
  const [entries, setEntries] = useState<HandoffHistoryEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [pageSize, setPageSize] = useState(PAGE_SIZE)
  const [agentFilter, setAgentFilter] = useState('')
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set())

  const loadEntries = useCallback(async () => {
    try {
      setError(null)
      const resp = await heartbeatService.getHandoffHistory(teamId, {
        agent: agentFilter || undefined,
        last: pageSize,
      })
      setEntries(resp.entries ?? [])
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

  const getAgentName = (agentId: string) => {
    const member = members.find(m => m.agentId === agentId)
    return member?.displayName ?? agentId
  }

  const getAgentAppearance = (agentId: string) => {
    return allAgents?.find(a => a.id === agentId)?.appearance ?? null
  }

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
      {/* Filter */}
      <div className="flex items-center gap-2">
        <select
          value={agentFilter}
          onChange={e => setAgentFilter(e.target.value)}
          className="text-xs border border-border rounded px-2 py-1 bg-background text-foreground"
        >
          <option value="">All members</option>
          {members.map(m => (
            <option key={m.agentId} value={m.agentId}>{m.displayName ?? m.agentId}</option>
          ))}
        </select>
      </div>

      {entries.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <Clock className="h-8 w-8 mb-3 opacity-40" />
          <p className="text-sm font-medium mb-1">No handoffs recorded yet</p>
          <p className="text-xs text-center max-w-xs opacity-70">
            Handoffs are automatically captured at the end of each heartbeat.
          </p>
        </div>
      ) : (
        <>
          {entries.map((entry, i) => {
            const id = `${entry.runId}-${i}`
            const isExpanded = expandedIds.has(id)
            const { text, truncated } = truncateContent(entry.content)

            return (
              <div
                key={id}
                className="border border-border rounded-lg p-3 bg-card border-l-4 border-l-primary/30"
              >
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <AgentColorBadge appearance={getAgentAppearance(entry.agentId)} size="sm" />
                    <span className="text-sm font-medium">{getAgentName(entry.agentId)}</span>
                  </div>
                  <span className="text-xs text-muted-foreground">
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
            )
          })}

          {entries.length >= pageSize && (
            <button
              onClick={() => setPageSize(s => s + PAGE_SIZE)}
              className="w-full text-xs text-primary py-2 hover:underline"
            >
              Load more
            </button>
          )}
        </>
      )}
    </div>
  )
}
