/**
 * PromptTab - Annotated prompt assembly view.
 *
 * Shows the fully constructed prompt as labeled, collapsible section cards.
 * Each card identifies its source (agent file, team doc, generated) and
 * provides navigation back to the editable source.
 */

import { useCallback, useEffect, useMemo, useState } from 'react'
import { copyToClipboard } from '@/lib/clipboard'
import {
  Copy,
  Code,
  FileText,
  RefreshCw,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { toast } from '@/hooks/use-toast'
import type { Agent } from '@/types/agent'
import type { PromptSection, AgentTeamMembership } from '@/lib/schemas'
import * as agentService from '@/services/agentService'
import { SectionCard } from './SectionCard'

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface PromptTabProps {
  agent: Agent
  hasUnsavedChanges?: boolean
  /** Switch to Files tab and highlight a specific file */
  onNavigateToFile?: (filePath: string) => void
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function PromptTab({
  agent,
  hasUnsavedChanges = false,
  onNavigateToFile,
}: PromptTabProps) {
  const [sections, setSections] = useState<PromptSection[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [viewMode, setViewMode] = useState<'markdown' | 'raw'>('markdown')
  const [collapsedSections, setCollapsedSections] = useState<Set<number>>(
    () => new Set(),
  )

  // Team selector
  const [memberships, setMemberships] = useState<AgentTeamMembership[]>([])
  const [selectedTeamId, setSelectedTeamId] = useState<string>('')

  // Load team memberships
  useEffect(() => {
    let cancelled = false
    void agentService.getAgentTeams(agent.id).then((result) => {
      if (!cancelled) setMemberships(result)
    })
    return () => {
      cancelled = true
    }
  }, [agent.id])

  // Load structured prompt
  const loadSections = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const response = await agentService.previewAgentPromptStructured(
        agent.id,
        selectedTeamId || undefined,
      )
      setSections(response.sections)
    } catch {
      setSections([])
      setError('Unable to build prompt preview. Check the API and try again.')
    } finally {
      setIsLoading(false)
    }
  }, [agent.id, selectedTeamId])

  useEffect(() => {
    void loadSections()
  }, [loadSections])

  // Derived
  const totalChars = useMemo(
    () => sections.reduce((sum, s) => sum + s.content.length, 0),
    [sections],
  )

  const handleCopyAll = useCallback(async () => {
    if (sections.length === 0) return
    const full = sections.map((s) => s.content).join('\n\n---\n\n')
    try {
      await copyToClipboard(full)
      toast({ title: 'Prompt copied', description: 'Full prompt copied to clipboard.' })
    } catch {
      toast({ title: 'Copy failed', description: 'Unable to copy. Try again.' })
    }
  }, [sections])

  const toggleCollapsed = useCallback((index: number) => {
    setCollapsedSections((prev) => {
      const next = new Set(prev)
      if (next.has(index)) next.delete(index)
      else next.add(index)
      return next
    })
  }, [])

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <div className="flex h-full flex-col space-y-3">
      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-2">
        {/* Team selector */}
        {memberships.length > 0 && (
          <select
            value={selectedTeamId}
            onChange={(e) => setSelectedTeamId(e.target.value)}
            className="rounded-md border border-border bg-muted px-2 py-1.5 text-xs font-medium text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
          >
            <option value="">Agent only</option>
            {memberships.map((m) => (
              <option key={m.teamId} value={m.teamId}>
                {m.teamDisplayName}
              </option>
            ))}
          </select>
        )}

        <div className="flex-1" />

        {/* Char count */}
        {sections.length > 0 && (
          <span className="text-[11px] text-muted-foreground tabular-nums">
            {totalChars.toLocaleString()} chars
          </span>
        )}

        {/* View mode toggle */}
        <div className="flex items-center rounded-md border border-border">
          <button
            type="button"
            onClick={() => setViewMode('markdown')}
            className={cn(
              'inline-flex items-center gap-1 rounded-l-md px-2 py-1.5 text-xs font-medium transition-colors',
              viewMode === 'markdown'
                ? 'bg-primary/15 text-primary'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground',
            )}
            title="Rendered markdown"
          >
            <FileText className="h-3.5 w-3.5" />
          </button>
          <button
            type="button"
            onClick={() => setViewMode('raw')}
            className={cn(
              'inline-flex items-center gap-1 rounded-r-md px-2 py-1.5 text-xs font-medium transition-colors',
              viewMode === 'raw'
                ? 'bg-primary/15 text-primary'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground',
            )}
            title="Raw text"
          >
            <Code className="h-3.5 w-3.5" />
          </button>
        </div>

        {/* Refresh */}
        <button
          type="button"
          onClick={() => void loadSections()}
          disabled={isLoading}
          className={cn(
            'inline-flex items-center gap-1.5 rounded-md border border-border px-2.5 py-1.5 text-xs font-medium',
            'text-foreground transition-colors hover:bg-muted',
            isLoading && 'cursor-not-allowed opacity-50',
          )}
        >
          <RefreshCw className={cn('h-3.5 w-3.5', isLoading && 'animate-spin')} />
          Refresh
        </button>

        {/* Copy all */}
        <button
          type="button"
          onClick={() => void handleCopyAll()}
          disabled={sections.length === 0}
          className={cn(
            'inline-flex items-center gap-1.5 rounded-md border border-border px-2.5 py-1.5 text-xs font-medium',
            'text-foreground transition-colors hover:bg-muted',
            sections.length === 0 && 'cursor-not-allowed opacity-50',
          )}
        >
          <Copy className="h-3.5 w-3.5" />
          Copy All
        </button>
      </div>

      {/* Unsaved warning */}
      {hasUnsavedChanges && (
        <div className="rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-700 dark:text-amber-200/90">
          Preview uses last saved files. Save changes to update.
        </div>
      )}

      {/* Section cards */}
      <div className="min-h-0 flex-1 overflow-y-auto space-y-2 pb-2">
        {isLoading ? (
          <div className="flex h-32 items-center justify-center text-sm text-muted-foreground">
            Building prompt preview…
          </div>
        ) : error ? (
          <div className="flex h-32 items-center justify-center text-sm text-amber-700 dark:text-amber-200">
            {error}
          </div>
        ) : sections.length === 0 ? (
          <div className="flex h-32 items-center justify-center text-sm text-muted-foreground">
            No prompt content available.
          </div>
        ) : (
          sections.map((section, idx) => (
            <SectionCard
              key={`${section.kind}-${section.label}-${idx}`}
              index={idx}
              section={section}
              isCollapsed={collapsedSections.has(idx)}
              onToggle={() => toggleCollapsed(idx)}
              viewMode={viewMode}
              onNavigateToFile={onNavigateToFile}
            />
          ))
        )}
      </div>
    </div>
  )
}

