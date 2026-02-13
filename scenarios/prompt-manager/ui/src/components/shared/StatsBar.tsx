/**
 * StatsBar - Displays team/agent/skill counts and selection badge.
 *
 * Self-contained: reads data from hooks and stores directly.
 */

import { Users, Bot, FileText } from 'lucide-react'
import { useTeamData } from '@/hooks/useTeamData'
import { useAgentData } from '@/hooks/useAgentData'
import { useSkillsData } from '@/hooks/useSkillsData'
import { useSelectionStore } from '@/stores/selectionStore'
import { selectors } from '@/constants/selectors'
import { cn } from '@/lib/utils'

interface StatsBarProps {
  compact?: boolean
}

export function StatsBar({ compact = false }: StatsBarProps) {
  const { teams } = useTeamData()
  const { agents } = useAgentData()
  const { skills } = useSkillsData()
  const selectionCount = useSelectionStore((s) => s.selectedSkillIds.length)

  return (
    <div className={cn('flex items-center', compact ? 'gap-2' : 'gap-3')}>
      <div
        className={cn(
          'bg-slate-800/80 border border-slate-700 rounded-md text-xs text-slate-300',
          compact ? 'px-2 py-1 text-[11px]' : 'px-3 py-1.5',
        )}
        data-testid={selectors.viewOverlay.stats}
      >
        <span className="inline-flex items-center gap-1">{teams.length} <Users className="h-3.5 w-3.5" /></span>
        {' \u2022 '}
        <span className="inline-flex items-center gap-1">{agents.length} <Bot className="h-3.5 w-3.5" /></span>
        {' \u2022 '}
        <span className="inline-flex items-center gap-1">{skills.length} <FileText className="h-3.5 w-3.5" /></span>
      </div>
      {selectionCount > 0 && (
        <div className={cn(
          'bg-amber-500/20 border border-amber-500/30 rounded-md text-xs text-amber-300',
          compact ? 'px-2 py-1 text-[11px]' : 'px-3 py-1.5',
        )}
        >
          {selectionCount} selected
        </div>
      )}
    </div>
  )
}
