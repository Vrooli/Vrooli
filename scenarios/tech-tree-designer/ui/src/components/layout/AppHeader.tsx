import React from 'react'
import type { GraphViewMode, TechTreeSummary, TreeViewMode } from '../../types/techTree'
import type { ScenarioEntryMap } from '../../types/graph'
import TreeSelector from './TreeSelector'
import ViewModeSelector from './ViewModeSelector'
import HiddenScenariosBanner from './HiddenScenariosBanner'
import { RefreshCcw } from 'lucide-react'

interface AppHeaderProps {
  viewMode: TreeViewMode
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
  graphViewMode: GraphViewMode
  onGraphViewModeChange: (mode: GraphViewMode) => void
  showHiddenScenarios: boolean
  showIsolatedScenarios: boolean
  onToggleHiddenScenarios: (value: boolean) => void
  onToggleIsolatedScenarios: (value: boolean) => void
  onSyncScenarios: () => void
  isSyncingScenarios: boolean
  lastSyncedLabel: string
  hiddenScenarios: string[]
  scenarioEntryMap: ScenarioEntryMap
  onToggleScenarioVisibility: (name: string, hidden: boolean) => void
}

const AppHeader: React.FC<AppHeaderProps> = ({
  viewMode,
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
  graphViewMode,
  onGraphViewModeChange,
  showHiddenScenarios,
  showIsolatedScenarios,
  onToggleHiddenScenarios,
  onToggleIsolatedScenarios,
  onSyncScenarios,
  isSyncingScenarios,
  lastSyncedLabel,
  hiddenScenarios,
  scenarioEntryMap,
  onToggleScenarioVisibility
}) => {
  const showDesignerControls = viewMode === 'designer'

  return (
    <header className={`app-header ${showDesignerControls ? '' : 'app-header--minimal'}`}>
      <div className="app-header__primary">
        <div className="app-heading">
          <h1 className="app-title">Tech Tree Designer</h1>
          <p className="app-subtitle">
            Civilizational technology intelligence that maps sector momentum to strategic milestones.
          </p>
        </div>
        {showDesignerControls && (
          <div className="scenario-sync">
            <button
              type="button"
              className="button"
              onClick={onSyncScenarios}
              disabled={isSyncingScenarios}
            >
              <RefreshCcw className={isSyncingScenarios ? 'spin' : ''} aria-hidden="true" />
              {isSyncingScenarios ? 'Syncing…' : 'Sync scenarios'}
            </button>
            <p className="scenario-sync__meta">Last sync: {lastSyncedLabel}</p>
          </div>
        )}
      </div>

      {showDesignerControls && (
        <>
          <div className="app-header__controls">
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
            <ViewModeSelector
              viewMode={graphViewMode}
              onChange={onGraphViewModeChange}
              showHiddenScenarios={showHiddenScenarios}
              showIsolatedScenarios={showIsolatedScenarios}
              onToggleHidden={onToggleHiddenScenarios}
              onToggleIsolated={onToggleIsolatedScenarios}
            />
          </div>
          <HiddenScenariosBanner
            hidden={hiddenScenarios}
            scenarioEntryMap={scenarioEntryMap}
            onToggleVisibility={onToggleScenarioVisibility}
          />
        </>
      )}
    </header>
  )
}

export default AppHeader
