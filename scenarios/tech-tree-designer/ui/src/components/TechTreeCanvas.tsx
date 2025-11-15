import React, { useCallback, useMemo, useState } from 'react'
import GraphView from './graph/GraphView'
import GraphSidePanel from './graph/GraphSidePanel'
import { GraphProvider } from '../contexts/GraphContext'
import type {
  ProgressionStage,
  ScenarioCatalogSnapshot,
  Sector,
  StageDependency
} from '../types/techTree'
import type {
  ScenarioCatalogEntry,
  ScenarioEntryMap,
  StageLookupEntry
} from '../types/graph'

interface TechTreeCanvasProps {
  sectors: Sector[]
  dependencies: StageDependency[]
  selectedTreeId: string | null
  graphNotice: string | null
  techTreeCanvasRef: React.RefObject<HTMLDivElement>
  isFullscreen: boolean
  isLayoutFullscreen: boolean
  toggleFullscreen: () => void
  canFullscreen: boolean
  onGraphPersist: (payload: { notice?: string; message?: string }) => void
  scenarioCatalog: ScenarioCatalogSnapshot
  scenarioEntryMap: ScenarioEntryMap
  showLiveScenarios: boolean
  scenarioOnlyMode: boolean
  showHiddenScenarios: boolean
  handleScenarioVisibility: (scenarioName: string, hidden: boolean) => void
  setGraphNotice: (notice: string) => void
  buildTreeAwarePath: (path: string) => string
  onLinkScenario?: (stage: ProgressionStage) => void
  onCreateSector?: () => void
  onCreateStage?: (sectorId?: string, stageType?: string, parentStageId?: string) => void
  onGenerateAIStageIdeas?: (sectorId: string) => void
  onEditStage?: (stage: ProgressionStage) => void
  onEditSector?: (sector: any) => void
  onDeleteStage?: (stageId: string, stageName: string) => void
  onDeleteSector?: (sectorId: string, sectorName: string) => void
  onUnlinkScenario?: (scenarioName: string, mappingId?: string) => void
  refreshTreeData?: () => void
}

/**
 * Tech Tree Canvas - Main orchestrator component for the interactive graph.
 *
 * REFACTORED: Previously 418 lines handling all graph concerns.
 * Now delegates to focused components:
 * - GraphView: Graph interaction, edit mode, context menu (all hook orchestration)
 * - GraphSidePanel: Detail dialogs and sidebar for selected nodes
 *
 * This component now only handles selection state and dialog visibility logic.
 */
const TechTreeCanvas: React.FC<TechTreeCanvasProps> = ({
  sectors,
  dependencies,
  selectedTreeId,
  graphNotice,
  techTreeCanvasRef,
  isFullscreen,
  isLayoutFullscreen,
  toggleFullscreen,
  canFullscreen,
  onGraphPersist,
  scenarioCatalog,
  scenarioEntryMap,
  showLiveScenarios,
  scenarioOnlyMode,
  showHiddenScenarios,
  handleScenarioVisibility,
  setGraphNotice,
  buildTreeAwarePath,
  onLinkScenario,
  onCreateSector,
  onCreateStage,
  onGenerateAIStageIdeas,
  onEditStage,
  onEditSector,
  onDeleteStage,
  onDeleteSector,
  onUnlinkScenario,
  refreshTreeData
}) => {
  // Selection state for detail panels
  const [selectedStage, setSelectedStage] = useState<StageLookupEntry | null>(null)
  const [selectedScenario, setSelectedScenario] = useState<ScenarioCatalogEntry | null>(null)

  // Determine if detail dialog should be shown (fullscreen or compact layout)
  const [shouldShowDialog, setShouldShowDialog] = useState(false)

  // Handle stage selection from GraphView
  const handleOpenStageDialog = useCallback((stage: ProgressionStage) => {
    setSelectedStage(stage as StageLookupEntry)
    setSelectedScenario(null)
    setShouldShowDialog(isFullscreen || isLayoutFullscreen)
  }, [isFullscreen, isLayoutFullscreen])

  // Handle scenario selection from GraphView
  const handleOpenScenarioDialog = useCallback((scenarioName: string) => {
    const entry =
      scenarioEntryMap.get(scenarioName) ||
      scenarioEntryMap.get(scenarioName.toLowerCase())

    const scenarioEntry = entry || {
      name: scenarioName,
      display_name: scenarioName,
      description: '',
      relative_path: '',
      tags: [],
      dependencies: [],
      resources: [],
      hidden: false,
      last_modified: undefined
    }

    setSelectedScenario(scenarioEntry)
    setSelectedStage(null)
    setShouldShowDialog(isFullscreen || isLayoutFullscreen)
  }, [scenarioEntryMap, isFullscreen, isLayoutFullscreen])

  // Close detail panel
  const handleCloseDetail = useCallback(() => {
    setSelectedStage(null)
    setSelectedScenario(null)
    setShouldShowDialog(false)
  }, [])

  // Generate unique IDs for dialog titles
  const deriveTitleId = (value?: string | null, prefix = 'detail') => {
    if (!value) {
      return undefined
    }
    const normalized = value
      .toLowerCase()
      .replace(/[^a-z0-9-]+/g, '-')
      .replace(/^-+|-+$/g, '')
    return `${prefix}-${normalized || 'item'}`
  }

  const stageDetailTitleId = selectedStage
    ? deriveTitleId(selectedStage.id || selectedStage.name, 'stage-detail')
    : undefined
  const scenarioDetailTitleId = selectedScenario
    ? deriveTitleId(
        selectedScenario.name || selectedScenario.display_name,
        'scenario-detail'
      )
    : undefined

  return (
    <GraphProvider
      sectors={sectors}
      dependencies={dependencies}
      scenarioCatalog={scenarioCatalog}
      scenarioEntryMap={scenarioEntryMap}
      selectedTreeId={selectedTreeId}
      showLiveScenarios={showLiveScenarios}
      scenarioOnlyMode={scenarioOnlyMode}
      showHiddenScenarios={showHiddenScenarios}
      isFullscreen={isFullscreen}
      isLayoutFullscreen={isLayoutFullscreen}
      canFullscreen={canFullscreen}
      onGraphPersist={onGraphPersist}
      setGraphNotice={setGraphNotice}
      buildTreeAwarePath={buildTreeAwarePath}
      toggleFullscreen={toggleFullscreen}
      refreshTreeData={refreshTreeData}
      onLinkScenario={onLinkScenario}
      onCreateSector={onCreateSector}
      onCreateStage={onCreateStage}
      onGenerateAIStageIdeas={onGenerateAIStageIdeas}
      onEditStage={onEditStage}
      onEditSector={onEditSector}
      onDeleteStage={onDeleteStage}
      onDeleteSector={onDeleteSector}
      onUnlinkScenario={onUnlinkScenario}
      handleScenarioVisibility={handleScenarioVisibility}
    >
      <section className="tech-tree-designer" aria-label="Interactive tech tree canvas">
        <GraphView
          graphNotice={graphNotice}
          techTreeCanvasRef={techTreeCanvasRef}
          onOpenStageDialog={handleOpenStageDialog}
          onOpenScenarioDialog={handleOpenScenarioDialog}
        />

        <GraphSidePanel
          selectedStage={selectedStage}
          selectedScenario={selectedScenario}
          scenarioEntryMap={scenarioEntryMap}
          shouldShowDialog={shouldShowDialog}
          scenarioTitleId={scenarioDetailTitleId}
          stageTitleId={stageDetailTitleId}
          onToggleScenarioVisibility={handleScenarioVisibility}
          onLinkScenario={onLinkScenario}
          onCloseDetail={handleCloseDetail}
        />
      </section>
    </GraphProvider>
  )
}

export default TechTreeCanvas
