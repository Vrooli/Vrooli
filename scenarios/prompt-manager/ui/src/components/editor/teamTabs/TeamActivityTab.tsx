/**
 * TeamActivityTab - Container with sub-tabs for Handoffs, Tasks, and Knowledge.
 */

import { useState, useEffect, useCallback } from 'react'
import * as Tabs from '@radix-ui/react-tabs'
import { Clock, ListTodo, BookOpen } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamMember } from '@/types/team'
import type { Agent } from '@/types/agent'
import { TabList, TabTrigger } from '@/components/shared/TabTrigger'
import { HandoffTimeline } from './HandoffTimeline'
import { TaskKanbanBoard } from './kanban'
import { KnowledgeLogView } from './KnowledgeLogView'

interface TeamActivityTabProps {
  teamId: string
  members: TeamMember[]
  allAgents?: Agent[]
  /** Externally-requested sub-tab (e.g. from URL deep-link) */
  initialSubTab?: string | null
  /** Called when the active sub-tab changes */
  onSubTabChange?: (subTab: string) => void
  className?: string
}

export function TeamActivityTab({ teamId, members, allAgents, initialSubTab, onSubTabChange, className }: TeamActivityTabProps) {
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
          <TabTrigger compact alwaysShowLabel value="handoffs" icon={<Clock className="h-3.5 w-3.5" />} label="Handoffs" />
          <TabTrigger compact alwaysShowLabel value="tasks" icon={<ListTodo className="h-3.5 w-3.5" />} label="Tasks" />
          <TabTrigger compact alwaysShowLabel value="knowledge" icon={<BookOpen className="h-3.5 w-3.5" />} label="Knowledge" />
        </TabList>

        <div className="flex-1 min-h-0 flex flex-col">
          <Tabs.Content value="handoffs" className="flex-1 min-h-0 p-4 data-[state=inactive]:hidden overflow-y-auto">
            <HandoffTimeline teamId={teamId} members={members} allAgents={allAgents} />
          </Tabs.Content>

          <Tabs.Content value="tasks" className="data-[state=inactive]:hidden flex-1 min-h-0 flex flex-col overflow-hidden">
            <TaskKanbanBoard teamId={teamId} members={members} allAgents={allAgents} />
          </Tabs.Content>

          <Tabs.Content value="knowledge" className="flex-1 min-h-0 p-4 data-[state=inactive]:hidden overflow-y-auto">
            <KnowledgeLogView teamId={teamId} members={members} allAgents={allAgents} />
          </Tabs.Content>
        </div>
      </Tabs.Root>
    </div>
  )
}
