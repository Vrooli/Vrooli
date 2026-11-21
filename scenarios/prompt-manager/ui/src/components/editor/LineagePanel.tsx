import * as Tabs from '@radix-ui/react-tabs'
import { History, GitBranch, FlaskConical } from 'lucide-react'
import { cn } from '@/lib/utils'
import { TabList, TabTrigger } from '../shared/TabTrigger'
import { VersionHistoryTab } from './tabs/VersionHistoryTab'
import { VariantPanel } from './VariantPanel'
import { ExperimentPanel } from './ExperimentPanel'

export type LineageTab = 'history' | 'variants' | 'experiments'

interface LineagePanelProps {
  skillId: string
  currentContent: string
  activeTab: LineageTab
  onActiveTabChange: (tab: LineageTab) => void
  className?: string
}

export function LineagePanel({
  skillId,
  currentContent,
  activeTab,
  onActiveTabChange,
  className,
}: LineagePanelProps) {
  return (
    <Tabs.Root
      value={activeTab}
      onValueChange={(v) => onActiveTabChange(v as LineageTab)}
      className={cn('flex h-full flex-col min-h-0 overflow-hidden', className)}
    >
      <TabList>
        <TabTrigger
          value="history"
          icon={<History className="h-4 w-4" />}
          label="History"
          alwaysShowLabel
          compact
          testId="lineage-tab-history"
        />
        <TabTrigger
          value="variants"
          icon={<GitBranch className="h-4 w-4" />}
          label="Variants"
          alwaysShowLabel
          compact
          testId="lineage-tab-variants"
        />
        <TabTrigger
          value="experiments"
          icon={<FlaskConical className="h-4 w-4" />}
          label="Experiments"
          alwaysShowLabel
          compact
          testId="lineage-tab-experiments"
        />
      </TabList>

      <div className="flex-1 min-h-0 flex flex-col">
        <Tabs.Content
          value="history"
          forceMount
          className="flex-1 min-h-0 overflow-y-auto data-[state=inactive]:hidden"
        >
          <VersionHistoryTab skillId={skillId} />
        </Tabs.Content>
        <Tabs.Content
          value="variants"
          forceMount
          className="flex-1 min-h-0 overflow-y-auto data-[state=inactive]:hidden"
        >
          <VariantPanel skillId={skillId} currentContent={currentContent} />
        </Tabs.Content>
        <Tabs.Content
          value="experiments"
          forceMount
          className="flex-1 min-h-0 overflow-y-auto data-[state=inactive]:hidden"
        >
          <ExperimentPanel skillId={skillId} />
        </Tabs.Content>
      </div>
    </Tabs.Root>
  )
}
