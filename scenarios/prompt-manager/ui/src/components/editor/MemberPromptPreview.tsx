/**
 * MemberPromptPreview - Shows the assembled prompt for one team member.
 *
 * Simpler than PromptTab: no team selector (team is fixed from context),
 * no unsaved-changes warning. Reuses SectionCard for display.
 */

import { useCallback, useEffect, useMemo, useState } from 'react'
import { copyToClipboard } from '@/lib/clipboard'
import { Copy, Code, FileText, RefreshCw } from 'lucide-react'
import { cn } from '@/lib/utils'
import { toast } from '@/hooks/use-toast'
import type { PromptSection } from '@/lib/schemas'
import * as agentService from '@/services/agentService'
import { SectionCard } from './tabs/SectionCard'

interface MemberPromptPreviewProps {
  teamId: string
  agentId: string
  onNavigateToFile?: (filePath: string) => void
}

export function MemberPromptPreview({
  teamId,
  agentId,
  onNavigateToFile,
}: MemberPromptPreviewProps) {
  const [sections, setSections] = useState<PromptSection[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [viewMode, setViewMode] = useState<'markdown' | 'raw'>('markdown')
  const [collapsedSections, setCollapsedSections] = useState<Set<number>>(
    () => new Set(),
  )

  const loadSections = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const response = await agentService.previewAgentPromptStructured(agentId, teamId)
      setSections(response.sections)
    } catch {
      setSections([])
      setError('Unable to build prompt preview. Check the API and try again.')
    } finally {
      setIsLoading(false)
    }
  }, [agentId, teamId])

  useEffect(() => {
    void loadSections()
  }, [loadSections])

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

  return (
    <div className="space-y-3">
      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-2">
        <div className="flex-1" />

        {sections.length > 0 && (
          <span className="text-[11px] text-muted-foreground tabular-nums">
            {totalChars.toLocaleString()} chars
          </span>
        )}

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

      {/* Section cards */}
      <div className="space-y-2">
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
