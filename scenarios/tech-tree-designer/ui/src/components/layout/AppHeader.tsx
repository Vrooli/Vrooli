import React from 'react'
import type { ScenarioCatalogEntry, TechTreeSummary } from '../../types/techTree'
import type { ScenarioEntryMap } from '../../types/graph'
import TreeSelector from './TreeSelector'
import ScenarioControls from './ScenarioControls'
import HiddenScenariosBanner from './HiddenScenariosBanner'

interface AppHeaderProps {
  techTrees: TechTreeSummary[]
  selectedTreeId: string | null
  treeBadgeClass: string
  treeBadgeLabel: string
  treeStatsSummary: string | null
  isTreeSelectorDisabled: boolean
  onTreeSelectionChange: (value: string | null) => void
  onCreateSector: () => void
  onCreateTree?: () => void
  onCloneTree?: () => void
  showLiveScenarios: boolean
  scenarioOnlyMode: boolean
  showHiddenScenarios: boolean
  onToggleLiveScenarios: (value: boolean) => void
  onToggleScenarioOnly: (value: boolean) => void
  onToggleHiddenScenarios: (value: boolean) => void
  onSyncScenarios: () => void
  isSyncingScenarios: boolean
  lastSyncedLabel: string
  hiddenScenarios: string[]
  scenarioEntryMap: ScenarioEntryMap
  onToggleScenarioVisibility: (name: string, hidden: boolean) => void
}

const AppHeader: React.FC<AppHeaderProps> = ({
  techTrees,
  selectedTreeId,
  treeBadgeClass,
  treeBadgeLabel,
  treeStatsSummary,
  isTreeSelectorDisabled,
  onTreeSelectionChange,
  onCreateSector,
  onCreateTree,
  onCloneTree,
  showLiveScenarios,
  scenarioOnlyMode,
  showHiddenScenarios,
  onToggleLiveScenarios,
  onToggleScenarioOnly,
  onToggleHiddenScenarios,
  onSyncScenarios,
  isSyncingScenarios,
  lastSyncedLabel,
  hiddenScenarios,
  scenarioEntryMap,
  onToggleScenarioVisibility
}) => (
  <header className="app-header">
    <div className="app-heading">
      <h1 className="app-title">Tech Tree Designer</h1>
      <p className="app-subtitle">
        Civilizational technology intelligence that maps sector momentum to strategic milestones.
      </p>
    </div>
    <TreeSelector
      techTrees={techTrees}
      selectedTreeId={selectedTreeId}
      disabled={isTreeSelectorDisabled}
      badgeClassName={treeBadgeClass}
      badgeLabel={treeBadgeLabel}
      statsSummary={treeStatsSummary}
      onChange={onTreeSelectionChange}
      onCreateSector={onCreateSector}
      onCreateTree={onCreateTree}
      onCloneTree={onCloneTree}
    />
    <ScenarioControls
      showLiveScenarios={showLiveScenarios}
      scenarioOnlyMode={scenarioOnlyMode}
      showHiddenScenarios={showHiddenScenarios}
      onToggleLive={onToggleLiveScenarios}
      onToggleScenarioOnly={onToggleScenarioOnly}
      onToggleHidden={onToggleHiddenScenarios}
      onSync={onSyncScenarios}
      isSyncing={isSyncingScenarios}
      lastSyncedLabel={lastSyncedLabel}
    />
    <HiddenScenariosBanner
      hidden={hiddenScenarios}
      scenarioEntryMap={scenarioEntryMap}
      onToggleVisibility={onToggleScenarioVisibility}
    />
  </header>
)

export default AppHeader
