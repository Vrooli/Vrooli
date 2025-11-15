import React from 'react'
import StageDialog from '../techTree/StageDialog'
import GraphSidebar from '../techTree/GraphSidebar'
import type { ScenarioCatalogEntry, ProgressionStage } from '../../types/techTree'
import type { ScenarioEntryMap, StageLookupEntry } from '../../types/graph'

interface GraphSidePanelProps {
  selectedStage: StageLookupEntry | null
  selectedScenario: ScenarioCatalogEntry | null
  scenarioEntryMap: ScenarioEntryMap
  shouldShowDialog: boolean
  scenarioTitleId?: string
  stageTitleId?: string
  onToggleScenarioVisibility: (scenarioName: string, hidden: boolean) => void
  onLinkScenario?: (stage: ProgressionStage) => void
  onCloseDetail: () => void
}

/**
 * GraphSidePanel - Manages the detail dialogs and sidebar for selected nodes.
 * Extracted from TechTreeCanvas to simplify component hierarchy.
 */
const GraphSidePanel: React.FC<GraphSidePanelProps> = ({
  selectedStage,
  selectedScenario,
  scenarioEntryMap,
  shouldShowDialog,
  scenarioTitleId,
  stageTitleId,
  onToggleScenarioVisibility,
  onLinkScenario,
  onCloseDetail
}) => {
  return (
    <>
      <StageDialog
        isOpen={shouldShowDialog}
        selectedStage={selectedStage}
        selectedScenario={selectedScenario}
        scenarioEntryMap={scenarioEntryMap}
        onClose={onCloseDetail}
        scenarioTitleId={scenarioTitleId}
        stageTitleId={stageTitleId}
        onToggleScenarioVisibility={onToggleScenarioVisibility}
        onLinkScenario={onLinkScenario}
      />

      <GraphSidebar
        hidden={shouldShowDialog}
        selectedStage={selectedStage}
        selectedScenario={selectedScenario}
        scenarioEntryMap={scenarioEntryMap}
        onToggleScenarioVisibility={onToggleScenarioVisibility}
        onLinkScenario={onLinkScenario}
        scenarioTitleId={scenarioTitleId}
        stageTitleId={stageTitleId}
      />
    </>
  )
}

export default GraphSidePanel
