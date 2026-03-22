/**
 * TeamActivityTab - Container with sub-tabs for Handoffs, Tasks, and Decisions.
 */

import { useState, useEffect, useCallback } from 'react'
import * as Tabs from '@radix-ui/react-tabs'
import { Clock, ListTodo, Scale } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamMember } from '@/types/team'
import type { Agent } from '@/types/agent'
import { TabList, TabTrigger } from '@/components/shared/TabTrigger'
import { HandoffTimeline } from './HandoffTimeline'
import { TaskBoardView } from './TaskBoardView'
import { DecisionLogView } from './DecisionLogView'

interface TeamActivityTabProps {
  teamId: string
  members: TeamMember[]
  allAgents?: Agent[]
  decisionMode?: string
  /** Externally-requested sub-tab (e.g. from URL deep-link) */
  initialSubTab?: string | null
  /** Called when the active sub-tab changes */
  onSubTabChange?: (subTab: string) => void
  className?: string
}

export function TeamActivityTab({ teamId, members, allAgents, decisionMode, initialSubTab, onSubTabChange, className }: TeamActivityTabProps) {
  const [activeSubTab, setActiveSubTab] = useState('handoffs')

  // Respond to external sub-tab navigation requests
  useEffect(() => {
    if (initialSubTab) {
      setActiveSubTab(initialSubTab)
    }
  }, [initialSubTab])

  const handleSubTabChange = useCallback((value: string) => {
    setActiveSubTab(value)
    onSubTabChange?.(value)
  }, [onSubTabChange])

  return (
    <div className={cn('flex flex-col', className)}>
      <Tabs.Root
        value={activeSubTab}
        onValueChange={handleSubTabChange}
        className="flex-1 flex flex-col min-h-0"
      >
        <TabList>
          <TabTrigger compact value="handoffs" icon={<Clock className="h-3.5 w-3.5" />} label="Handoffs" />
          <TabTrigger compact value="tasks" icon={<ListTodo className="h-3.5 w-3.5" />} label="Tasks" />
          <TabTrigger compact value="decisions" icon={<Scale className="h-3.5 w-3.5" />} label="Decisions" />
        </TabList>

        <div className="flex-1 min-h-0 overflow-y-auto">
          <Tabs.Content value="handoffs" className="p-4 data-[state=inactive]:hidden">
            <HandoffTimeline teamId={teamId} members={members} allAgents={allAgents} />
          </Tabs.Content>

          <Tabs.Content value="tasks" className="p-4 data-[state=inactive]:hidden">
            <TaskBoardView teamId={teamId} members={members} allAgents={allAgents} />
          </Tabs.Content>

          <Tabs.Content value="decisions" className="p-4 data-[state=inactive]:hidden">
            <DecisionLogView teamId={teamId} members={members} allAgents={allAgents} decisionMode={decisionMode} />
          </Tabs.Content>
        </div>
      </Tabs.Root>
    </div>
  )
}
