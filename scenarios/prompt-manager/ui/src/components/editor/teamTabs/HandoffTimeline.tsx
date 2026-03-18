/**
 * HandoffTimeline - Vertical timeline/feed showing handoff history.
 */

import { useEffect, useState, useCallback } from 'react'
import { ChevronDown } from 'lucide-react'
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

  if (loading) {
    return <div className="text-sm text-muted-foreground">Loading handoffs...</div>
  }

  if (error) {
    return (
      <div className="text-sm text-destructive">
        Error loading handoffs: {error}
        <button onClick={() => void loadEntries()} className="ml-2 text-xs underline hover:no-underline">Retry</button>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {/* Filter */}
      <div className="flex items-center gap-2">
        <select
          value={agentFilter}
          onChange={e => setAgentFilter(e.target.value)}
          className="text-xs border rounded px-2 py-1 bg-background text-foreground"
        >
          <option value="">All members</option>
          {members.map(m => (
            <option key={m.agentId} value={m.agentId}>{m.displayName ?? m.agentId}</option>
          ))}
        </select>
      </div>

      {entries.length === 0 ? (
        <div className="text-sm text-muted-foreground">No handoffs recorded yet.</div>
      ) : (
        <>
          {entries.map((entry, i) => {
            const id = `${entry.runId}-${i}`
            const isExpanded = expandedIds.has(id)
            const { text, truncated } = truncateContent(entry.content)

            return (
              <div
                key={id}
                className="border rounded-lg p-3 bg-card"
              >
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <AgentColorBadge appearance={getAgentAppearance(entry.agentId)} size="xs" />
                    <span className="text-sm font-medium">{getAgentName(entry.agentId)}</span>
                  </div>
                  <span className="text-xs text-muted-foreground">
                    {formatRelativePastTime(new Date(entry.timestamp))}
                  </span>
                </div>
                <div className="text-sm text-muted-foreground whitespace-pre-wrap">
                  {isExpanded ? entry.content : text}
                </div>
                {truncated && (
                  <button
                    onClick={() => toggleExpand(id)}
                    className="text-xs text-primary mt-1 flex items-center gap-0.5 hover:underline"
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
