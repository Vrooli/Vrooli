import React from 'react'
import type { ScenarioCatalogEntry } from '../../types/techTree'
import type { ScenarioEntryMap, StageLookupEntry } from '../../types/graph'
import StageDetail from './StageDetail'
import ScenarioDetail from './ScenarioDetail'

interface GraphSidebarProps {
  hidden: boolean
  selectedStage: StageLookupEntry | null
  selectedScenario: ScenarioCatalogEntry | null
  scenarioEntryMap: ScenarioEntryMap
  onToggleScenarioVisibility: (scenarioName: string, hidden: boolean) => void
  onLinkScenario?: (stage: StageLookupEntry) => void
  scenarioTitleId?: string
  stageTitleId?: string
}

const GraphSidebar: React.FC<GraphSidebarProps> = ({
  hidden,
  selectedStage,
  selectedScenario,
  scenarioEntryMap,
  onToggleScenarioVisibility,
  onLinkScenario,
  scenarioTitleId,
  stageTitleId
}) => (
  <aside
    className={['tech-tree-sidebar', hidden ? 'tech-tree-sidebar--hidden' : null].filter(Boolean).join(' ')}
    aria-label="Stage details"
    aria-hidden={hidden}
  >
    {selectedScenario ? (
      <ScenarioDetail
        scenario={selectedScenario}
        titleId={scenarioTitleId}
        onToggleVisibility={onToggleScenarioVisibility}
      />
    ) : selectedStage ? (
      <StageDetail
        stage={selectedStage}
        titleId={stageTitleId}
        scenarioEntryMap={scenarioEntryMap}
        onToggleScenarioVisibility={onToggleScenarioVisibility}
        onLinkScenario={onLinkScenario}
      />
    ) : (
      <div className="stage-placeholder">
        <p>Select a node in the graph to inspect sector context and linked scenarios.</p>
        <p className="stage-placeholder__hint">Tip: Use the minimap or scroll wheel to navigate expansive trees.</p>
      </div>
    )}
  </aside>
)

export default GraphSidebar
