import type { Edge, Node } from 'react-flow-renderer'
import type { ProgressionStage, ScenarioCatalogEntry, Sector } from './techTree'

export interface StageNodeData {
  label: string
  progress: number
  type: string
  sectorName: string
  sectorColor?: string
  sectorId?: string
  stageId: string
  hasChildren?: boolean
  childrenLoaded?: boolean
  isExpanded?: boolean
  isLoading?: boolean
  parentStageId?: string | null
  scenarioCount?: number // Number of linked scenarios
  onToggleExpand?: (stageId: string, hasChildren: boolean, childrenLoaded: boolean) => void
}

export interface ScenarioNodeData {
  label: string
  scenarioName: string
  description: string
  tags: string[]
  hidden: boolean
}

export interface SectorGroupNodeData {
  sectorId: string
  sectorName: string
  sectorColor?: string
  stageCount: number
  avgProgress: number
  collapsed: boolean
}

export type DesignerNodeData = StageNodeData | ScenarioNodeData | SectorGroupNodeData

export interface GraphEdgeData {
  dependencyType: string
  dependencyStrength: number
  description: string
  persistedId?: string
}

export type StageNode = Node<StageNodeData>
export type DesignerNode = Node<DesignerNodeData>
export type DesignerEdge = Edge<GraphEdgeData>

export interface StageLookupEntry extends ProgressionStage {
  sector: Sector
}

export type ScenarioEntryMap = Map<string, ScenarioCatalogEntry>

// Re-export for backward compatibility
export type { ScenarioCatalogEntry } from './techTree'
