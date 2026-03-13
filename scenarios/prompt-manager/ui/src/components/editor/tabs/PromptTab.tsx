/**
 * PromptTab - Annotated prompt assembly view.
 *
 * Shows the fully constructed prompt as labeled, collapsible section cards.
 * Each card identifies its source (agent file, team doc, generated) and
 * provides navigation back to the editable source.
 */

import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  ChevronDown,
  ChevronRight,
  Copy,
  Code,
  ExternalLink,
  FileText,
  GitBranch,
  Network,
  Heart,
  Inbox,
  RefreshCw,
  Users,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { toast } from '@/hooks/use-toast'
import type { Agent } from '@/types/agent'
import type { PromptSection, AgentTeamMembership } from '@/lib/schemas'
import { MarkdownRenderer } from '@/components/markdown/MarkdownRenderer'
import * as agentService from '@/services/agentService'

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const KIND_META: Record<
  string,
  { icon: typeof FileText; color: string; badgeLabel: string }
> = {
  'agent-file': {
    icon: FileText,
    color: 'bg-blue-500/15 text-blue-400 border-blue-500/25',
    badgeLabel: 'Agent file',
  },
  'team-responsibilities': {
    icon: Users,
    color: 'bg-green-500/15 text-green-400 border-green-500/25',
    badgeLabel: 'Team',
  },
  'team-relationships': {
    icon: GitBranch,
    color: 'bg-purple-500/15 text-purple-400 border-purple-500/25',
    badgeLabel: 'Team',
  },
  'team-coordination': {
    icon: Network,
    color: 'bg-orange-500/15 text-orange-400 border-orange-500/25',
    badgeLabel: 'Team',
  },
  'team-inbox': {
    icon: Inbox,
    color: 'bg-yellow-500/15 text-yellow-400 border-yellow-500/25',
    badgeLabel: 'Team',
  },
  'heartbeat-task': {
    icon: Heart,
    color: 'bg-red-500/15 text-red-400 border-red-500/25',
    badgeLabel: 'Team',
  },
}

const FALLBACK_META: { icon: typeof FileText; color: string; badgeLabel: string } = {
  icon: FileText,
  color: 'bg-blue-500/15 text-blue-400 border-blue-500/25',
  badgeLabel: 'Agent file',
}

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
      await navigator.clipboard.writeText(full)
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

// ---------------------------------------------------------------------------
// SectionCard
// ---------------------------------------------------------------------------

interface SectionCardProps {
  index: number
  section: PromptSection
  isCollapsed: boolean
  onToggle: () => void
  viewMode: 'markdown' | 'raw'
  onNavigateToFile?: (filePath: string) => void
}

function SectionCard({
  index,
  section,
  isCollapsed,
  onToggle,
  viewMode,
  onNavigateToFile,
}: SectionCardProps) {
  const meta = KIND_META[section.kind] ?? FALLBACK_META
  const Icon = meta.icon

  return (
    <div className="rounded-lg border border-border bg-card/50 overflow-hidden">
      {/* Header */}
      <button
        type="button"
        onClick={onToggle}
        className="flex w-full items-center gap-2 px-3 py-2 text-left hover:bg-muted/50 transition-colors"
      >
        {/* Order number */}
        <span className="flex h-5 w-5 flex-shrink-0 items-center justify-center rounded text-[10px] font-bold text-muted-foreground bg-muted">
          {index + 1}
        </span>

        {/* Icon */}
        <Icon className="h-3.5 w-3.5 flex-shrink-0 text-muted-foreground" />

        {/* Label */}
        <span className="min-w-0 flex-1 truncate text-sm font-medium text-foreground">
          {section.label}
        </span>

        {/* Kind badge */}
        <span
          className={cn(
            'flex-shrink-0 rounded-full border px-2 py-0.5 text-[10px] font-medium',
            meta.color,
          )}
        >
          {meta.badgeLabel}
        </span>

        {/* Char count */}
        <span className="flex-shrink-0 text-[10px] tabular-nums text-muted-foreground">
          {section.content.length.toLocaleString()}
        </span>

        {/* Chevron */}
        {isCollapsed ? (
          <ChevronRight className="h-3.5 w-3.5 flex-shrink-0 text-muted-foreground" />
        ) : (
          <ChevronDown className="h-3.5 w-3.5 flex-shrink-0 text-muted-foreground" />
        )}
      </button>

      {/* Content */}
      {!isCollapsed && (
        <div className="border-t border-border">
          <div className="max-h-80 overflow-y-auto px-4 py-3 text-sm">
            {viewMode === 'markdown' ? (
              <MarkdownRenderer content={section.content} />
            ) : (
              <pre className="whitespace-pre-wrap text-xs leading-relaxed text-foreground">
                {section.content}
              </pre>
            )}
          </div>

          {/* Footer: navigate link for agent files */}
          {section.kind === 'agent-file' && onNavigateToFile && (
            <div className="border-t border-border px-3 py-1.5">
              <button
                type="button"
                onClick={() => onNavigateToFile(section.label)}
                className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
              >
                <ExternalLink className="h-3 w-3" />
                Edit in Files
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
