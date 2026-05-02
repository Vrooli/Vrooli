/**
 * TopicsValidationPanel - Sidebar listing topics-graph validation findings.
 *
 * Renders findings grouped by severity, then by member. Clicking a finding
 * focuses the corresponding member node in the graph.
 *
 * DOC: docs/agent-system/drafts/topics-schema.md
 */

import { useMemo } from 'react'
import { AlertTriangle, AlertCircle, CheckCircle2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TopicFinding, TopicValidation } from '@/types/topicsGraph'

interface TopicsValidationPanelProps {
  validation: TopicValidation
  onSelectMember?: (team: string, member: string) => void
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
  onSelect,
}: {
  finding: TopicFinding
  onSelect?: () => void
}) {
  const Icon = finding.severity === 'error' ? AlertCircle : AlertTriangle
  const tone =
    finding.severity === 'error'
      ? 'text-rose-300 hover:bg-rose-500/10'
      : 'text-amber-300 hover:bg-amber-500/10'

  return (
    <button
      type="button"
      onClick={onSelect}
      className={cn(
        'w-full text-left p-2 rounded-md border border-transparent transition-colors',
        'hover:border-border',
        tone,
      )}
      data-testid={`topics-finding-${finding.severity}-${finding.rule}`}
    >
      <div className="flex items-start gap-2">
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
          <p className="text-[10px] text-muted-foreground/90 mt-1 line-clamp-2">
            {finding.detail}
          </p>
        </div>
      </div>
    </button>
  )
}

export function TopicsValidationPanel({
  validation,
  onSelectMember,
  className,
}: TopicsValidationPanelProps) {
  const grouped = useMemo(() => groupFindings(validation.findings), [validation.findings])

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
              {grouped.errors.map((f, i) => (
                <FindingRow
                  key={`err-${i}`}
                  finding={f}
                  onSelect={() => onSelectMember?.(f.member.team, f.member.member)}
                />
              ))}
            </div>
          </section>
        )}

        {grouped.warnings.length > 0 && (
          <section>
            <p className="text-[10px] uppercase tracking-wide text-muted-foreground mb-1 px-1">
              Warnings ({grouped.warnings.length})
            </p>
            <div className="space-y-1">
              {grouped.warnings.map((f, i) => (
                <FindingRow
                  key={`warn-${i}`}
                  finding={f}
                  onSelect={() => onSelectMember?.(f.member.team, f.member.member)}
                />
              ))}
            </div>
          </section>
        )}
      </div>
    </div>
  )
}
