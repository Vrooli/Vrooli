import { useMemo } from 'react'
import type { Sector, StageDependency } from '../types/techTree'
import type { DesignerEdge, StageLookupEntry, StageNode } from '../types/graph'
import { createGraphEdge } from '../utils/graph'

interface GraphModel {
  initialNodes: StageNode[]
  initialEdges: DesignerEdge[]
  stageLookup: Map<string, StageLookupEntry>
}

const DEFAULT_STAGE_TYPE = 'foundation'

export const useGraphModel = (sectors: Sector[], dependencies: StageDependency[]): GraphModel => {
  return useMemo(() => {
    console.log('[useGraphModel] Building graph from sectors:', sectors.length, 'dependencies:', dependencies.length)
    const stageMap = new Map<string, StageLookupEntry>()
    const stageNodes: StageNode[] = []
    const edgeList: DesignerEdge[] = []

    sectors.forEach((sector, sectorIndex) => {
      const baseX = typeof sector.position_x === 'number' ? sector.position_x : sectorIndex * 260
      const baseY = typeof sector.position_y === 'number' ? sector.position_y : 120 + sectorIndex * 45

      ;(sector.stages || []).forEach((stage, stageIndex) => {
        const stageId = stage.id || `${sector.id}-stage-${stageIndex + 1}`
        const positionX = typeof stage.position_x === 'number' ? stage.position_x : baseX + stageIndex * 48
        const positionY = typeof stage.position_y === 'number' ? stage.position_y : baseY + stageIndex * 90
        const progress = typeof stage.progress_percentage === 'number' ? stage.progress_percentage : 0
        const stageType = stage.stage_type || DEFAULT_STAGE_TYPE

        const enrichedStage: StageLookupEntry = {
          ...stage,
          id: stageId,
          stage_type: stageType,
          progress_percentage: progress,
          position_x: positionX,
          position_y: positionY,
          sector
        }

        stageMap.set(stageId, enrichedStage)

        stageNodes.push({
          id: stageId,
          position: { x: positionX, y: positionY },
          data: {
            label: stage.name,
            progress,
            type: stageType,
            sectorName: sector.name,
            sectorColor: sector.color,
            sectorId: sector.id,
            stageId: stageId,
            hasChildren: stage.has_children || false,
            childrenLoaded: stage.children_loaded || false,
            parentStageId: stage.parent_stage_id || null,
            isExpanded: false,
            isLoading: false
          },
          type: 'stageNode'
        })
      })
    })

    ;(dependencies || []).forEach((dependencyItem) => {
      const dep = dependencyItem?.dependency || dependencyItem
      const source = dep?.prerequisite_stage_id || (dep as any)?.prerequisiteStageId
      const target = dep?.dependent_stage_id || (dep as any)?.dependentStageId

      if (!source || !target || !stageMap.has(source) || !stageMap.has(target)) {
        return
      }

      edgeList.push(
        createGraphEdge({
          id: dep?.id || `${source}->${target}`,
          source,
          target,
          dependencyType: dep?.dependency_type,
          dependencyStrength: dep?.dependency_strength,
          description: dep?.description
        })
      )
    })

    console.log('[useGraphModel] Built nodes:', stageNodes.length, 'edges:', edgeList.length)
    return { initialNodes: stageNodes, initialEdges: edgeList, stageLookup: stageMap }
  }, [sectors, dependencies])
}
