/**
 * SkillMetadataBadges — Reusable skill metadata display (usage, rating, folder, recency).
 * Shared by SkillListView and SkillCardView.
 */

import { Star } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Skill } from '@/types'

interface SkillMetadataBadgesProps {
  skill: Skill
  className?: string
}

const FOLDER_COLORS: Record<string, string> = {
  core: 'bg-blue-400',
  local: 'bg-green-400',
  drafts: 'bg-amber-400',
}

export function SkillMetadataBadges({ skill, className }: SkillMetadataBadgesProps) {
  return (
    <div className={cn('flex items-center gap-2 text-[10px] text-muted-foreground', className)}>
      {/* Folder dot */}
      <span className="flex items-center gap-1">
        <span className={cn('w-1.5 h-1.5 rounded-full', FOLDER_COLORS[skill.folder] ?? 'bg-muted')} />
        <span className="capitalize">{skill.folder}</span>
      </span>

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
