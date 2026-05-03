/**
 * TopicsValidationPanel - Sidebar listing topics-graph validation findings.
 *
 * Renders findings grouped by severity, then by member. Clicking a finding
 * focuses the corresponding member node in the graph.
 *
 * DOC: docs/agent-system/TOPICS_SCHEMA.md
 */

import { useMemo, useState, useCallback } from 'react'
import { AlertTriangle, AlertCircle, CheckCircle2, ChevronDown, ChevronRight, FileText } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TopicFinding, TopicValidation } from '@/types/topicsGraph'

interface TopicsValidationPanelProps {
  validation: TopicValidation
  onSelectMember?: (team: string, member: string) => void
  /** Open the team/member's file in the Files tab (used for the "Open topics.json" CTA). */
  onOpenMemberFile?: (team: string, member: string, fileName: string) => void
  className?: string
}

interface GroupedFindings {
  errors: TopicFinding[]
  warnings: TopicFinding[]
}

function groupFindings(findings: TopicFinding[]): GroupedFindings {
  const errors: TopicFinding[] = []
  const warnings: TopicFinding[] = []
  for (const f of findings) {
    if (f.severity === 'error') errors.push(f)
    else warnings.push(f)
  }
  return { errors, warnings }
}

function FindingRow({
  finding,
  expanded,
  onToggle,
  onSelect,
  onOpenFile,
}: {
  finding: TopicFinding
  expanded: boolean
  onToggle: () => void
  onSelect?: () => void
  onOpenFile?: () => void
}) {
  const Icon = finding.severity === 'error' ? AlertCircle : AlertTriangle
  const Chevron = expanded ? ChevronDown : ChevronRight
  const tone =
    finding.severity === 'error'
      ? 'text-rose-300 hover:bg-rose-500/10'
      : 'text-amber-300 hover:bg-amber-500/10'

  return (
    <div
      className={cn(
        'w-full p-2 rounded-md border border-transparent transition-colors',
        'hover:border-border',
        tone,
      )}
      data-expanded={expanded ? 'true' : 'false'}
    >
      <button
        type="button"
        onClick={() => {
          onToggle()
          onSelect?.()
        }}
        className="w-full text-left flex items-start gap-2"
        aria-expanded={expanded}
        data-testid={`topics-finding-${finding.severity}-${finding.rule}`}
      >
        <Chevron className="h-3.5 w-3.5 mt-0.5 flex-shrink-0 text-muted-foreground" />
        <Icon className="h-3.5 w-3.5 mt-0.5 flex-shrink-0" />
        <div className="min-w-0 flex-1">
          <p className="text-xs font-mono">{finding.rule}</p>
          <p className="text-[11px] text-muted-foreground truncate">
            {finding.member.team}/{finding.member.member}
          </p>
          {finding.prefix && (
            <p className="text-[10px] text-muted-foreground/80 truncate font-mono">
              {finding.prefix}
            </p>
          )}
          <p
            className={cn(
              'text-[10px] text-muted-foreground/90 mt-1',
              !expanded && 'line-clamp-2',
            )}
          >
            {finding.detail}
          </p>
        </div>
      </button>
      {expanded && onOpenFile && (
        <div className="mt-2 pl-7">
          <button
            type="button"
            onClick={onOpenFile}
            className={cn(
              'inline-flex items-center gap-1.5 px-2 py-0.5 text-[10px] font-medium rounded-md',
              'bg-card border border-border text-foreground hover:bg-muted transition-colors',
            )}
            data-testid={`topics-finding-open-file-${finding.rule}`}
          >
            <FileText className="h-3 w-3" />
            Open topics.json
          </button>
        </div>
      )}
    </div>
  )
}

export function TopicsValidationPanel({
  validation,
  onSelectMember,
  onOpenMemberFile,
  className,
}: TopicsValidationPanelProps) {
  const grouped = useMemo(() => groupFindings(validation.findings), [validation.findings])
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set())
  const toggleExpanded = useCallback((key: string) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }, [])

  const findingKey = useCallback((f: TopicFinding, prefix: string) =>
    `${prefix}-${f.rule}-${f.member.team}-${f.member.member}-${f.prefix ?? ''}`,
  [])

  return (
    <div
      className={cn('h-full flex flex-col bg-background', className)}
      data-testid="topics-validation-panel"
    >
      <div className="p-3 border-b border-border flex items-center justify-between">
        <p className="text-sm font-medium">Validation</p>
        <p className="text-[10px] text-muted-foreground font-mono">
          {validation.errors}E · {validation.warnings}W
        </p>
      </div>

      <div className="flex-1 overflow-y-auto p-2 space-y-3">
        {validation.findings.length === 0 && (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <CheckCircle2 className="h-8 w-8 text-emerald-500 mb-2" />
            <p className="text-xs text-muted-foreground">No findings</p>
          </div>
        )}

        {grouped.errors.length > 0 && (
          <section>
            <p className="text-[10px] uppercase tracking-wide text-muted-foreground mb-1 px-1">
              Errors ({grouped.errors.length})
            </p>
            <div className="space-y-1">
              {grouped.errors.map((f, i) => {
                const key = findingKey(f, `err-${i}`)
                return (
                  <FindingRow
                    key={key}
                    finding={f}
                    expanded={expanded.has(key)}
                    onToggle={() => toggleExpanded(key)}
                    onSelect={() => onSelectMember?.(f.member.team, f.member.member)}
                    onOpenFile={
                      onOpenMemberFile
                        ? () => onOpenMemberFile(f.member.team, f.member.member, 'topics.json')
                        : undefined
                    }
                  />
                )
              })}
            </div>
          </section>
        )}

        {grouped.warnings.length > 0 && (
          <section>
            <p className="text-[10px] uppercase tracking-wide text-muted-foreground mb-1 px-1">
              Warnings ({grouped.warnings.length})
            </p>
            <div className="space-y-1">
              {grouped.warnings.map((f, i) => {
                const key = findingKey(f, `warn-${i}`)
                return (
                  <FindingRow
                    key={key}
                    finding={f}
                    expanded={expanded.has(key)}
                    onToggle={() => toggleExpanded(key)}
                    onSelect={() => onSelectMember?.(f.member.team, f.member.member)}
                    onOpenFile={
                      onOpenMemberFile
                        ? () => onOpenMemberFile(f.member.team, f.member.member, 'topics.json')
                        : undefined
                    }
                  />
                )
              })}
            </div>
          </section>
        )}
      </div>
    </div>
  )
}
