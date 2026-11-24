import { cn } from '@/lib/utils'
import { VersionHistoryTab } from './tabs/VersionHistoryTab'

interface LineagePanelProps {
  skillId: string
  className?: string
}

export function LineagePanel({ skillId, className }: LineagePanelProps) {
  return (
    <div className={cn('flex h-full flex-col min-h-0 overflow-hidden', className)}>
      <div className="flex-1 min-h-0 overflow-y-auto">
        <VersionHistoryTab skillId={skillId} />
      </div>
    </div>
  )
}
