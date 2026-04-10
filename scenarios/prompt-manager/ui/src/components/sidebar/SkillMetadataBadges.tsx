/**
 * SkillMetadataBadges — Reusable skill metadata display.
 * Shared by SkillListView, SkillCardView, and TreeNode.
 */

import { Star, Activity } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Skill } from '@/types'

interface SkillMetadataBadgesProps {
  skill: Skill
  /** Composite health score 0-1 from graph system. null = not loaded. */
  healthScore?: number | null
  className?: string
}

const FOLDER_COLORS: Record<string, string> = {
  core: 'bg-blue-400',
  local: 'bg-green-400',
  drafts: 'bg-amber-400',
}

function getHealthColor(score: number): string {
  if (score < 0.3) return 'text-red-400'
  if (score < 0.6) return 'text-yellow-400'
  return 'text-emerald-400'
}

function getHealthDotColor(score: number): string {
  if (score < 0.3) return 'bg-red-400'
  if (score < 0.6) return 'bg-yellow-400'
  return 'bg-emerald-400'
}

export function SkillMetadataBadges({ skill, healthScore, className }: SkillMetadataBadgesProps) {
  return (
    <div className={cn('flex items-center gap-2 text-[10px] text-muted-foreground', className)}>
      {/* Folder dot */}
      <span className="flex items-center gap-1">
        <span className={cn('w-1.5 h-1.5 rounded-full', FOLDER_COLORS[skill.folder] ?? 'bg-muted')} />
        <span className="capitalize">{skill.folder}</span>
      </span>

      {/* Draft badge */}
      {skill.draft && (
        <span className="px-1 py-0.5 text-[8px] font-medium bg-amber-500/20 text-amber-400 rounded leading-none">
          Draft
        </span>
      )}

      {/* Health score */}
      {healthScore != null && (
        <span className={cn('flex items-center gap-0.5', getHealthColor(healthScore))}>
          <Activity className="h-2.5 w-2.5" />
          {Math.round(healthScore * 100)}%
        </span>
      )}

      {/* Usage count */}
      {skill.usageCount > 0 && (
        <span>{skill.usageCount} use{skill.usageCount !== 1 ? 's' : ''}</span>
      )}

      {/* Rating */}
      {skill.effectivenessRating != null && (
        <span className="flex items-center gap-0.5">
          <Star className="h-2.5 w-2.5 fill-amber-400 text-amber-400" />
          {skill.effectivenessRating.toFixed(1)}
        </span>
      )}

      {/* Last used */}
      {skill.lastUsed && (
        <span>{formatRelativeTime(skill.lastUsed)}</span>
      )}

      {/* Updated at */}
      {skill.updatedAt && (
        <span>upd {formatRelativeTime(skill.updatedAt)}</span>
      )}
    </div>
  )
}

/** Compact version for tree view — just the health dot, folder, and key stats. */
export function SkillMetadataCompact({ skill, healthScore, className }: SkillMetadataBadgesProps) {
  return (
    <div className={cn('flex items-center gap-1.5 text-[9px] text-muted-foreground', className)}>
      {/* Health dot */}
      {healthScore != null && (
        <span
          className={cn('w-1.5 h-1.5 rounded-full flex-shrink-0', getHealthDotColor(healthScore))}
          title={`Health: ${Math.round(healthScore * 100)}%`}
        />
      )}

      {/* Folder dot */}
      <span className="flex items-center gap-0.5">
        <span className={cn('w-1.5 h-1.5 rounded-full', FOLDER_COLORS[skill.folder] ?? 'bg-muted')} />
        {skill.folder}
      </span>

      {/* Draft badge */}
      {skill.draft && (
        <span className="text-amber-400">draft</span>
      )}

      {/* Usage */}
      {skill.usageCount > 0 && (
        <span>{skill.usageCount} use{skill.usageCount !== 1 ? 's' : ''}</span>
      )}

      {/* Rating */}
      {skill.effectivenessRating != null && (
        <span className="flex items-center gap-0.5">
          <Star className="h-2 w-2 fill-amber-400 text-amber-400" />
          {skill.effectivenessRating.toFixed(1)}
        </span>
      )}
    </div>
  )
}

function formatRelativeTime(isoDate: string): string {
  const now = Date.now()
  const then = new Date(isoDate).getTime()
  const diffMs = now - then

  if (diffMs < 0) return 'just now'

  const minutes = Math.floor(diffMs / 60_000)
  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${minutes}m ago`

  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`

  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}d ago`

  const months = Math.floor(days / 30)
  if (months < 12) return `${months}mo ago`

  const years = Math.floor(months / 12)
  return `${years}y ago`
}
