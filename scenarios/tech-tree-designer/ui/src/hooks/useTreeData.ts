import { useCallback, useEffect, useState } from 'react'
import { getDemoMilestones, getDemoSectors } from '../utils/demoData'
import type {
  ApiStrategicMilestone,
  Sector,
  StageDependency,
  StrategicMilestone
} from '../types/techTree'
import { fetchDependencies, fetchMilestones, fetchSectors } from '../services/techTree'

const attachStageMetadata = (sectorList: Sector[]): Sector[] => {
  if (!Array.isArray(sectorList)) {
    return []
  }

  return sectorList.map((sector, sectorIndex) => {
    const baseX = typeof sector.position_x === 'number' ? sector.position_x : 220 + sectorIndex * 240
    const baseY = typeof sector.position_y === 'number' ? sector.position_y : 80

    const stages = (sector.stages || []).map((stage, stageIndex) => ({
      ...stage,
      id: stage.id || `${sector.id}-stage-${stageIndex + 1}`,
      position_x: typeof stage.position_x === 'number' ? stage.position_x : baseX,
      position_y: typeof stage.position_y === 'number' ? stage.position_y : baseY + stageIndex * 110,
      stage_type: stage.stage_type || (stage as { stageType?: string }).stageType || 'foundation'
    }))

    return {
      ...sector,
      stages,
      position_x: baseX,
      position_y: baseY
    }
  })
}

const generateLinearDependenciesFromSectors = (sectorList: Sector[]): StageDependency[] => {
  const inferred: StageDependency[] = []

  sectorList.forEach((sector) => {
    const stages = Array.isArray(sector.stages) ? sector.stages : []
    for (let index = 1; index < stages.length; index += 1) {
      const current = stages[index]
      const previous = stages[index - 1]
      if (!current || !previous) {
        continue
      }

      const currentId = current.id || `${sector.id}-stage-${index + 1}`
      const previousId = previous.id || `${sector.id}-stage-${index}`

      inferred.push({
        dependency: {
          id: `${previousId}->${currentId}`,
          dependent_stage_id: currentId,
          prerequisite_stage_id: previousId,
          dependency_type: 'progression',
          dependency_strength: Math.min(1, 0.5 + index * 0.1),
          description: `${current.name} builds on ${previous.name}`
        },
        dependent_name: current.name,
        prerequisite_name: previous.name
      })
    }
  })

  return inferred
}

interface UseTreeDataParams {
  selectedTreeId: string | null
  treeLoading: boolean
}

/**
 * Hook for fetching and managing tech tree data (sectors, dependencies, milestones)
 * for the currently selected tree.
 */
const parseTargetIds = (raw: unknown): string[] => {
  if (Array.isArray(raw)) {
    return raw.map((value) => `${value}`).filter(Boolean)
  }
  if (typeof raw === 'string') {
    try {
      const parsed = JSON.parse(raw)
      if (Array.isArray(parsed)) {
        return parsed.map((value) => `${value}`).filter(Boolean)
      }
    } catch (error) {
      // Ignore JSON parsing errors and fall through to fallback.
    }
    return raw ? raw.split(',').map((value) => value.trim()).filter(Boolean) : []
  }
  return []
}

const normalizeMilestones = (payload: ApiStrategicMilestone[]): StrategicMilestone[] =>
  payload.map((milestone) => {
    const { required_sectors, required_stages, target_sector_ids, target_stage_ids, ...rest } = milestone
    return {
      ...rest,
      target_sector_ids: target_sector_ids?.length
        ? target_sector_ids
        : parseTargetIds(required_sectors),
      target_stage_ids: target_stage_ids?.length ? target_stage_ids : parseTargetIds(required_stages)
    }
  })

const useTreeData = ({ selectedTreeId, treeLoading }: UseTreeDataParams) => {
  const [sectors, setSectors] = useState<Sector[]>([])
  const [milestones, setMilestones] = useState<StrategicMilestone[]>([])
  const [dependencies, setDependencies] = useState<StageDependency[]>([])
  const [loading, setLoading] = useState(true)
  const [graphNotice, setGraphNotice] = useState<string | null>(null)

  const fetchData = useCallback(
    async (treeId: string | null) => {
      if (!treeId) {
        setLoading(false)
        return
      }

      setLoading(true)
      try {
        const [sectorData, milestoneData, dependencyData] = await Promise.all([
          fetchSectors(treeId),
          fetchMilestones(treeId),
          fetchDependencies(treeId)
        ])

        const normalizedSectors = attachStageMetadata((sectorData.sectors || []) as Sector[])
        const fallbackSectors = getDemoSectors(attachStageMetadata) as Sector[]
        const sectorPayload = normalizedSectors.length ? normalizedSectors : fallbackSectors
        setSectors(sectorPayload)

        const milestonePayload = normalizeMilestones(
          (milestoneData.milestones || []) as ApiStrategicMilestone[]
        )
        setMilestones(
          milestonePayload.length ? milestonePayload : (getDemoMilestones() as StrategicMilestone[])
        )

        const dependencyPayload = (dependencyData.dependencies || []) as StageDependency[]
        if (dependencyPayload.length) {
          setDependencies(dependencyPayload)
          setGraphNotice(null)
        } else {
          setDependencies(generateLinearDependenciesFromSectors(sectorPayload))
          setGraphNotice(
            'Live dependency data unavailable. Showing inferred progression links so the designer remains usable.'
          )
        }
      } catch (error) {
        console.error('Failed to fetch tech tree data:', error)
        const demoSectors = getDemoSectors(attachStageMetadata) as Sector[]
        setSectors(demoSectors)
        setMilestones(getDemoMilestones() as StrategicMilestone[])
        setDependencies(generateLinearDependenciesFromSectors(demoSectors))
        setGraphNotice('Operating in demo mode due to API connectivity issues. Graph interactions use sample data.')
      } finally {
        setLoading(false)
      }
    },
    []
  )

  useEffect(() => {
    if (!selectedTreeId) {
      if (!treeLoading) {
        setLoading(false)
      }
      return
    }

    const controller = { cancelled: false }
    fetchData(selectedTreeId)
    return () => {
      controller.cancelled = true
    }
  }, [fetchData, selectedTreeId, treeLoading])

  const refreshTreeData = useCallback(() => {
    if (!selectedTreeId) {
      return
    }
    fetchData(selectedTreeId)
  }, [fetchData, selectedTreeId])

  return {
    sectors,
    milestones,
    dependencies,
    loading,
    graphNotice,
    setGraphNotice,
    refreshTreeData
  }
}

export default useTreeData
