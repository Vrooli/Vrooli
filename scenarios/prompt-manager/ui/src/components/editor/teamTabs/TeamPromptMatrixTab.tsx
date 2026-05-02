/**
 * TeamPromptMatrixTab - Bird's-eye view of what every team member receives.
 *
 * Renders a table: rows = members, columns = section kinds.
 * Cells show checkmark + char count, or warning icons for gaps.
 * Click a cell to expand and see the section content.
 */

import { useCallback, useEffect, useMemo, useState } from 'react'
import { AlertTriangle, Check, RefreshCw, XCircle } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamPromptMatrixEntry } from '@/lib/schemas'
import * as agentService from '@/services/agentService'
import { SectionCard } from '../tabs/SectionCard'

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface TeamPromptMatrixTabProps {
  teamId: string
  onNavigateToMember?: (agentId: string) => void
  className?: string
}

const PROMPT_SIZE_RANGES = [
  { label: 'Missing', range: '0 chars', tone: 'text-amber-500', icon: <AlertTriangle className="h-3 w-3" /> },
  { label: 'Healthy', range: '1-12,000 chars', tone: 'text-green-500', icon: <Check className="h-3 w-3" /> },
  { label: 'Large', range: '12,001-18,000 chars', tone: 'text-amber-500', icon: <AlertTriangle className="h-3 w-3" /> },
  { label: 'Too large', range: '>18,000 chars', tone: 'text-destructive', icon: <XCircle className="h-3 w-3" /> },
]

function getPromptSizeStatus(chars: number, count: number): 'missing' | 'healthy' | 'large' | 'too-large' {
  if (count === 0) return 'missing'
  if (chars > 18000) return 'too-large'
  if (chars > 12000) return 'large'
  return 'healthy'
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function TeamPromptMatrixTab({
  teamId,
  onNavigateToMember,
  className,
}: TeamPromptMatrixTabProps) {
  const [entries, setEntries] = useState<TeamPromptMatrixEntry[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [expandedCell, setExpandedCell] = useState<{
    agentId: string
    kind: string
  } | null>(null)

  const loadMatrix = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const response = await agentService.getTeamPromptMatrix(teamId)
      setEntries(response.entries)
    } catch {
      setEntries([])
      setError('Unable to load prompt matrix. Check the API and try again.')
    } finally {
      setIsLoading(false)
    }
  }, [teamId])

  useEffect(() => {
    void loadMatrix()
  }, [loadMatrix])

  // Determine which section kinds are present across all entries
  const activeKinds = useMemo(() => {
    const found: string[] = []
    const seen = new Set<string>()
    for (const entry of entries) {
      for (const section of entry.sections) {
        if (seen.has(section.kind)) continue
        seen.add(section.kind)
        found.push(section.kind)
      }
    }
    return found
  }, [entries])

  const labelsByKind = useMemo(() => {
    const labels = new Map<string, string>()
    for (const entry of entries) {
      for (const section of entry.sections) {
        if (!labels.has(section.kind)) {
          labels.set(section.kind, section.label)
        }
      }
    }
    return labels
  }, [entries])

  const totalChars = useMemo(
    () =>
      entries.reduce(
        (sum, e) => sum + e.sections.reduce((s, sec) => s + sec.content.length, 0),
        0,
      ),
    [entries],
  )

  const toggleCell = useCallback((agentId: string, kind: string) => {
    setExpandedCell((prev) =>
      prev?.agentId === agentId && prev.kind === kind ? null : { agentId, kind },
    )
  }, [])

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  if (isLoading) {
    return (
      <div className={cn('flex h-32 items-center justify-center text-sm text-muted-foreground', className)}>
        Loading prompt matrix…
      </div>
    )
  }

  if (error) {
    return (
      <div className={cn('flex h-32 items-center justify-center text-sm text-amber-700 dark:text-amber-200', className)}>
        {error}
      </div>
    )
  }

  if (entries.length === 0) {
    return (
      <div className={cn('flex h-32 items-center justify-center text-sm text-muted-foreground', className)}>
        No team members to display.
      </div>
    )
  }

  return (
    <div className={cn('space-y-3', className)}>
      {/* Toolbar */}
      <div className="flex items-start gap-2">
        <div>
          <span className="text-sm font-medium text-foreground">
            {entries.length} member{entries.length !== 1 ? 's' : ''}
          </span>
          <div className="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-[11px] text-muted-foreground">
            {PROMPT_SIZE_RANGES.map((item) => (
              <span key={item.label} className="inline-flex items-center gap-1">
                <span className={item.tone}>{item.icon}</span>
                <span>{item.label}:</span>
                <span className="tabular-nums">{item.range}</span>
              </span>
            ))}
          </div>
        </div>
        <div className="flex-1" />
        {totalChars > 0 && (
          <span className="shrink-0 text-[11px] text-muted-foreground tabular-nums">
            {totalChars.toLocaleString()} total chars
          </span>
        )}
        <button
          type="button"
          onClick={() => void loadMatrix()}
          disabled={isLoading}
          className={cn(
            'inline-flex items-center gap-1.5 rounded-md border border-border px-2.5 py-1.5 text-xs font-medium',
            'text-foreground transition-colors hover:bg-muted',
          )}
        >
          <RefreshCw className="h-3.5 w-3.5" />
          Refresh
        </button>
      </div>

      {/* Matrix table */}
      <div className="overflow-x-auto rounded-lg border border-border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-muted/50">
              <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">
                Member
              </th>
              {activeKinds.map((kind) => {
                return (
                  <th
                    key={kind}
                    className="px-3 py-2 text-center text-xs font-medium text-muted-foreground"
                  >
                    {labelsByKind.get(kind) ?? kind}
                  </th>
                )
              })}
            </tr>
          </thead>
          <tbody>
            {entries.map((entry) => (
              <MemberRow
                key={entry.agentId}
                entry={entry}
                activeKinds={activeKinds}
                expandedKind={expandedCell?.agentId === entry.agentId ? expandedCell.kind : null}
                onToggleCell={(kind) => toggleCell(entry.agentId, kind)}
                onNavigateToMember={onNavigateToMember}
              />
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// MemberRow
// ---------------------------------------------------------------------------

interface MemberRowProps {
  entry: TeamPromptMatrixEntry
  activeKinds: readonly string[]
  expandedKind: string | null
  onToggleCell: (kind: string) => void
  onNavigateToMember?: (agentId: string) => void
}

function MemberRow({
  entry,
  activeKinds,
  expandedKind,
  onToggleCell,
  onNavigateToMember,
}: MemberRowProps) {
  // Group sections by kind for quick lookup
  const sectionsByKind = useMemo(() => {
    const map = new Map<string, typeof entry.sections>()
    for (const section of entry.sections) {
      const existing = map.get(section.kind) ?? []
      existing.push(section)
      map.set(section.kind, existing)
    }
    return map
  // eslint-disable-next-line react-hooks/exhaustive-deps -- only entry.sections matters, not the full entry object
  }, [entry.sections])

  // Find expanded sections
  const expandedSections = expandedKind ? sectionsByKind.get(expandedKind) : null

  return (
    <>
      <tr className="border-b border-border last:border-b-0 hover:bg-muted/30 transition-colors">
        {/* Member name */}
        <td className="px-3 py-2">
          {onNavigateToMember ? (
            <button
              type="button"
              onClick={() => onNavigateToMember(entry.agentId)}
              className="text-sm font-medium text-primary hover:underline"
            >
              {entry.displayName}
            </button>
          ) : (
            <span className="text-sm font-medium text-foreground">{entry.displayName}</span>
          )}
          {entry.error && (
            <div className="flex items-center gap-1 mt-0.5">
              <XCircle className="h-3 w-3 text-destructive" />
              <span className="text-[10px] text-destructive truncate max-w-[200px]">
                {entry.error}
              </span>
            </div>
          )}
        </td>

        {/* Section cells */}
        {activeKinds.map((kind) => {
          const sections = sectionsByKind.get(kind)
          const totalChars = sections?.reduce((s, sec) => s + sec.content.length, 0) ?? 0
          const count = sections?.length ?? 0
          const isExpanded = expandedKind === kind

          return (
            <td key={kind} className="px-3 py-2 text-center">
              {entry.error ? (
                <span className="text-muted-foreground">—</span>
              ) : count > 0 ? (
                <button
                  type="button"
                  onClick={() => onToggleCell(kind)}
                  data-size-status={getPromptSizeStatus(totalChars, count)}
                  className={cn(
                    'inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] tabular-nums transition-colors',
                    isExpanded
                      ? 'bg-primary/15 text-primary'
                      : getPromptSizeStatus(totalChars, count) === 'too-large'
                        ? 'text-destructive hover:bg-muted'
                        : getPromptSizeStatus(totalChars, count) === 'large'
                          ? 'text-amber-500 hover:bg-muted'
                          : 'text-green-500 hover:bg-muted',
                  )}
                  title={`${count} section${count > 1 ? 's' : ''}, ${totalChars.toLocaleString()} chars`}
                >
                  {getPromptSizeStatus(totalChars, count) === 'too-large' ? (
                    <XCircle className="h-3 w-3" />
                  ) : getPromptSizeStatus(totalChars, count) === 'large' ? (
                    <AlertTriangle className="h-3 w-3" />
                  ) : (
                    <Check className="h-3 w-3" />
                  )}
                  {totalChars.toLocaleString()}
                </button>
              ) : (
                <span title="Missing">
                  <AlertTriangle className="inline h-3 w-3 text-amber-500" />
                </span>
              )}
            </td>
          )
        })}
      </tr>

      {/* Expanded section cards */}
      {expandedSections && expandedSections.length > 0 && (
        <tr>
          <td colSpan={activeKinds.length + 1} className="p-3 bg-muted/20 border-b border-border">
            <div className="space-y-2">
              {expandedSections.map((section, idx) => (
                <SectionCard
                  key={`${section.kind}-${section.label}-${idx}`}
                  index={idx}
                  section={section}
                  isCollapsed={false}
                  onToggle={() => {}}
                  viewMode="markdown"
                />
              ))}
            </div>
          </td>
        </tr>
      )}
    </>
  )
}
