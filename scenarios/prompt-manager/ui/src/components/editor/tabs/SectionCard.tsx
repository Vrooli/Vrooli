/**
 * SectionCard - Reusable card for displaying a single prompt section.
 *
 * Used by PromptTab, MemberPromptPreview, and TeamPromptMatrixTab.
 */

import {
  ChevronDown,
  ChevronRight,
  ExternalLink,
  FileText,
  GitBranch,
  Heart,
  Inbox,
  Users,
  Map,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import type { PromptSection } from '@/lib/schemas'
import { MarkdownRenderer } from '@/components/markdown/MarkdownRenderer'

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
  'active-task-brief': {
    icon: Heart,
    color: 'bg-sky-500/15 text-sky-400 border-sky-500/25',
    badgeLabel: 'Brief',
  },
  'team-operating-policy': {
    icon: FileText,
    color: 'bg-slate-500/15 text-slate-400 border-slate-500/25',
    badgeLabel: 'Policy',
  },
  'team-responsibilities': {
    icon: Users,
    color: 'bg-green-500/15 text-green-400 border-green-500/25',
    badgeLabel: 'Team',
  },
  'team-org-context': {
    icon: GitBranch,
    color: 'bg-purple-500/15 text-purple-400 border-purple-500/25',
    badgeLabel: 'Team',
  },
  'team-inbox': {
    icon: Inbox,
    color: 'bg-yellow-500/15 text-yellow-400 border-yellow-500/25',
    badgeLabel: 'Team',
  },
  'team-storage-map': {
    icon: Map,
    color: 'bg-cyan-500/15 text-cyan-400 border-cyan-500/25',
    badgeLabel: 'Storage',
  },
  'last-handoff': {
    icon: GitBranch,
    color: 'bg-indigo-500/15 text-indigo-400 border-indigo-500/25',
    badgeLabel: 'History',
  },
  'heartbeat-task': {
    icon: Heart,
    color: 'bg-red-500/15 text-red-400 border-red-500/25',
    badgeLabel: 'Team',
  },
  'task-reminder': {
    icon: Heart,
    color: 'bg-rose-500/15 text-rose-400 border-rose-500/25',
    badgeLabel: 'Reminder',
  },
}

const FALLBACK_META: { icon: typeof FileText; color: string; badgeLabel: string } = {
  icon: FileText,
  color: 'bg-blue-500/15 text-blue-400 border-blue-500/25',
  badgeLabel: 'Agent file',
}

// ---------------------------------------------------------------------------
// SectionCard
// ---------------------------------------------------------------------------

export interface SectionCardProps {
  index: number
  section: PromptSection
  isCollapsed: boolean
  onToggle: () => void
  viewMode: 'markdown' | 'raw'
  onNavigateToFile?: (filePath: string) => void
}

export function SectionCard({
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
