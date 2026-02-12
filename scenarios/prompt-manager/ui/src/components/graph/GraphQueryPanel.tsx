/**
 * GraphQueryPanel - Quick-access buttons for graph query endpoints.
 *
 * Results highlight matching nodes in the graph by updating the
 * graphStore's highlightedNodeIds.
 */

import { useState } from 'react'
import { Search, AlertTriangle, RefreshCw } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useGraphStore } from '@/stores/graphStore'
import {
  getOrphanedSkills,
  getSkilllessAgents,
  getEmptyTeams,
  getUnaffiliatedAgents,
  getCLIlessSkills,
  getCircularRefs,
} from '@/services/graphService'
import type { GraphNode } from '@/lib/schemas'

interface QueryDef {
  id: string
  label: string
  description: string
  run: () => Promise<string[]>
}

interface GraphQueryPanelProps {
  className?: string
}

export function GraphQueryPanel({ className }: GraphQueryPanelProps) {
  const highlightNodes = useGraphStore((s) => s.highlightNodes)
  const clearHighlights = useGraphStore((s) => s.clearHighlights)
  const highlightedNodeIds = useGraphStore((s) => s.highlightedNodeIds)
  const [activeQuery, setActiveQuery] = useState<string | null>(null)
  const [running, setRunning] = useState(false)
  const [resultCount, setResultCount] = useState<number | null>(null)

  const queries: QueryDef[] = [
    {
      id: 'orphaned-skills',
      label: 'Orphaned Skills',
      description: 'Skills not referenced by any agent',
      run: async () => {
        const nodes: GraphNode[] = await getOrphanedSkills()
        return nodes.map((n) => n.id)
      },
    },
    {
      id: 'skillless-agents',
      label: 'Skillless Agents',
      description: 'Agents with no skill references',
      run: async () => {
        const nodes: GraphNode[] = await getSkilllessAgents()
        return nodes.map((n) => n.id)
      },
    },
    {
      id: 'empty-teams',
      label: 'Empty Teams',
      description: 'Teams with no members',
      run: async () => {
        const nodes: GraphNode[] = await getEmptyTeams()
        return nodes.map((n) => n.id)
      },
    },
    {
      id: 'unaffiliated-agents',
      label: 'Unaffiliated Agents',
      description: 'Agents not in any team',
      run: async () => {
        const nodes: GraphNode[] = await getUnaffiliatedAgents()
        return nodes.map((n) => n.id)
      },
    },
    {
      id: 'cliless-skills',
      label: 'CLI-less Skills',
      description: 'Skills with no CLI tool',
      run: async () => {
        const nodes: GraphNode[] = await getCLIlessSkills()
        return nodes.map((n) => n.id)
      },
    },
    {
      id: 'circular-refs',
      label: 'Circular Refs',
      description: 'Circular dependency chains',
      run: async () => {
        const cycles = await getCircularRefs()
        const ids = new Set<string>()
        for (const cycle of cycles) {
          for (const id of cycle) {
            ids.add(id)
          }
        }
        return Array.from(ids)
      },
    },
  ]

  const runQuery = async (query: QueryDef) => {
    if (running) return

    // Toggle off if re-clicking the active query
    if (activeQuery === query.id) {
      setActiveQuery(null)
      setResultCount(null)
      clearHighlights()
      return
    }

    setRunning(true)
    setActiveQuery(query.id)
    try {
      const ids = await query.run()
      highlightNodes(ids)
      setResultCount(ids.length)
    } catch {
      setResultCount(0)
    } finally {
      setRunning(false)
    }
  }

  return (
    <div className={cn('p-2 bg-card border border-border rounded-lg space-y-1.5', className)}>
      <div className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
        <Search className="h-3 w-3" />
        <span>Queries</span>
        {resultCount !== null && activeQuery && (
          <span className="ml-auto text-foreground">
            {resultCount} found
          </span>
        )}
      </div>

      <div className="grid grid-cols-2 gap-1">
        {queries.map((query) => (
          <button
            key={query.id}
            type="button"
            onClick={() => void runQuery(query)}
            disabled={running && activeQuery !== query.id}
            className={cn(
              'px-2 py-1 text-[10px] font-medium rounded border transition-colors text-left',
              activeQuery === query.id
                ? 'bg-primary/20 border-primary/50 text-primary'
                : 'bg-card border-border text-muted-foreground hover:bg-muted hover:text-foreground',
              running && activeQuery !== query.id && 'opacity-50 cursor-not-allowed',
            )}
            title={query.description}
          >
            {query.id === 'circular-refs' ? (
              <AlertTriangle className="h-2.5 w-2.5 inline mr-1" />
            ) : null}
            {running && activeQuery === query.id ? (
              <RefreshCw className="h-2.5 w-2.5 inline mr-1 animate-spin" />
            ) : null}
            {query.label}
          </button>
        ))}
      </div>

      {highlightedNodeIds.size > 0 && (
        <button
          type="button"
          onClick={() => {
            clearHighlights()
            setActiveQuery(null)
            setResultCount(null)
          }}
          className="w-full px-2 py-1 text-[10px] text-muted-foreground hover:text-foreground transition-colors"
        >
          Clear highlights
        </button>
      )}
    </div>
  )
}
