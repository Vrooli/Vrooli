import React, { createContext, useContext, type ReactNode } from 'react'
import type {
  ProgressionStage,
  ScenarioCatalogSnapshot,
  Sector,
  StageDependency
} from '../types/techTree'
import type { ScenarioEntryMap } from '../types/graph'

/**
 * Context for graph-related data and operations.
 * Reduces prop drilling through deeply nested graph components.
 */
export interface GraphContextValue {
  // Data
  sectors: Sector[]
  dependencies: StageDependency[]
  scenarioCatalog: ScenarioCatalogSnapshot
  scenarioEntryMap: ScenarioEntryMap
  selectedTreeId: string | null

  // Display modes
  showLiveScenarios: boolean
  scenarioOnlyMode: boolean
  showHiddenScenarios: boolean

  // Fullscreen state
  isFullscreen: boolean
  isLayoutFullscreen: boolean
  canFullscreen: boolean

  // Callbacks - Graph operations
  onGraphPersist: (payload: { notice?: string; message?: string }) => void
  setGraphNotice: (notice: string) => void
  buildTreeAwarePath: (path: string) => string
  toggleFullscreen: () => void
  refreshTreeData?: () => void

  // Callbacks - Node operations
  onLinkScenario?: (stage: ProgressionStage) => void
  onCreateSector?: () => void
  onCreateStage?: (sectorId?: string, stageType?: string, parentStageId?: string) => void
  onGenerateAIStageIdeas?: (sectorId: string) => void
  onEditStage?: (stage: ProgressionStage) => void
  onEditSector?: (sector: Sector) => void
  onDeleteStage?: (stageId: string, stageName: string) => void
  onDeleteSector?: (sectorId: string, sectorName: string) => void
  onUnlinkScenario?: (scenarioName: string, mappingId?: string) => void

  // Callbacks - Scenario visibility
  handleScenarioVisibility: (scenarioName: string, hidden: boolean) => void
}

const GraphContext = createContext<GraphContextValue | null>(null)

export interface GraphProviderProps extends GraphContextValue {
  children: ReactNode
}

/**
 * Provider component for GraphContext.
 * Wraps the graph view and makes all graph-related data/operations available via context.
 */
export const GraphProvider: React.FC<GraphProviderProps> = ({ children, ...value }) => {
  return <GraphContext.Provider value={value}>{children}</GraphContext.Provider>
}

/**
 * Hook to access graph context.
 * Throws error if used outside of GraphProvider.
 */
export const useGraphContext = (): GraphContextValue => {
  const context = useContext(GraphContext)
  if (!context) {
    throw new Error('useGraphContext must be used within a GraphProvider')
  }
  return context
}
