import type {
  ProgressionStage,
  Sector,
  SectorValueSummary,
  StrategicMilestone,
  StrategicStageReference,
  StrategicValueBreakdown,
  StrategicValuePreset,
  StrategicValueSettings
} from '../types/techTree'

const clamp01 = (value: number) => Math.min(1, Math.max(0, value))

export const DEFAULT_STRATEGIC_VALUE_SETTINGS: StrategicValueSettings = {
  completionWeight: 45,
  readinessWeight: 35,
  influenceWeight: 20,
  dependencyPenalty: 0.25
}

export const BUILT_IN_STRATEGIC_VALUE_PRESETS: StrategicValuePreset[] = [
  {
    id: 'balanced-insight',
    name: 'Balanced insight',
    description: 'Default mix of completion, readiness, and leverage.',
    settings: { ...DEFAULT_STRATEGIC_VALUE_SETTINGS },
    builtIn: true
  },
  {
    id: 'execution-focus',
    name: 'Execution focus',
    description: 'Maximize realized value by emphasizing completion and penalties.',
    settings: {
      completionWeight: 60,
      readinessWeight: 25,
      influenceWeight: 15,
      dependencyPenalty: 0.4
    },
    builtIn: true
  },
  {
    id: 'expansion-focus',
    name: 'Expansion focus',
    description: 'Favor connectivity and network effects for long-term bets.',
    settings: {
      completionWeight: 30,
      readinessWeight: 30,
      influenceWeight: 40,
      dependencyPenalty: 0.15
    },
    builtIn: true
  },
  {
    id: 'risk-aware',
    name: 'Risk aware',
    description: 'Discount unfinished dependencies aggressively.',
    settings: {
      completionWeight: 40,
      readinessWeight: 30,
      influenceWeight: 30,
      dependencyPenalty: 0.55
    },
    builtIn: true
  }
]

export const DEFAULT_STRATEGIC_VALUE_PRESET_ID = BUILT_IN_STRATEGIC_VALUE_PRESETS[0]?.id || 'balanced-insight'

const normalizeWeights = (settings: StrategicValueSettings) => {
  const completion = Math.max(0, settings.completionWeight)
  const readiness = Math.max(0, settings.readinessWeight)
  const influence = Math.max(0, settings.influenceWeight)
  const total = completion + readiness + influence
  if (total <= 0) {
    return {
      completion: 1 / 3,
      readiness: 1 / 3,
      influence: 1 / 3
    }
  }
  return {
    completion: completion / total,
    readiness: readiness / total,
    influence: influence / total
  }
}

type StageLookupEntry = {
  sectorId: string
  sectorName: string
  stage: ProgressionStage
  color?: string
}

const buildStageLookup = (sectors: Sector[]): Map<string, StageLookupEntry> => {
  const lookup = new Map<string, StageLookupEntry>()
  sectors.forEach((sector) => {
    ;(sector.stages || []).forEach((stage) => {
      lookup.set(stage.id, {
        sectorId: sector.id,
        sectorName: sector.name,
        stage,
        color: sector.color
      })
    })
  })
  return lookup
}

const buildSectorSummaries = (
  sectors: Sector[],
  milestones: StrategicMilestone[],
  stageLookup: Map<string, StageLookupEntry>
): SectorValueSummary[] => {
  const milestoneCounts = new Map<string, number>()
  const incrementCount = (sectorId?: string) => {
    if (!sectorId) {
      return
    }
    milestoneCounts.set(sectorId, (milestoneCounts.get(sectorId) || 0) + 1)
  }

  milestones.forEach((milestone) => {
    milestone.target_sector_ids?.forEach((sectorId) => {
      incrementCount(sectorId)
    })
    milestone.target_stage_ids?.forEach((stageId) => {
      const entry = stageLookup.get(stageId)
      if (entry) {
        incrementCount(entry.sectorId)
      }
    })
  })

  return sectors.map((sector) => {
    const stageCount = sector.stages?.length || 0
    const stageProgressTotal = (sector.stages || []).reduce(
      (sum, stage) => sum + (stage.progress_percentage || 0),
      0
    )
    const stageAverage = stageCount ? stageProgressTotal / stageCount : sector.progress_percentage || 0
    const scenarioLinks = (sector.stages || []).reduce(
      (sum, stage) => sum + (stage.scenario_mappings?.length || 0),
      0
    )

    const readinessScore = clamp01((sector.progress_percentage || stageAverage || 0) / 100)
    const influenceScore = clamp01(
      0.35 +
        (scenarioLinks ? Math.min(scenarioLinks / 10, 0.35) : 0) +
        (stageAverage ? (stageAverage / 100) * 0.3 : 0)
    )

    return {
      sectorId: sector.id,
      name: sector.name,
      color: sector.color,
      readinessScore,
      influenceScore,
      progressPercentage: sector.progress_percentage || 0,
      scenarioLinks,
      stageAverage,
      milestoneCount: milestoneCounts.get(sector.id) || 0
    }
  })
}

interface BuildBreakdownParams {
  milestones: StrategicMilestone[]
  sectors: Sector[]
  settings?: StrategicValueSettings
}

export const calculateStrategicValueBreakdown = ({
  milestones,
  sectors,
  settings = DEFAULT_STRATEGIC_VALUE_SETTINGS
}: BuildBreakdownParams): StrategicValueBreakdown => {
  const stageLookup = buildStageLookup(sectors)
  const sectorSummaries = buildSectorSummaries(sectors, milestones, stageLookup)
  const normalizedWeights = normalizeWeights(settings)
  const dependencyPenalty = clamp01(settings.dependencyPenalty)
  const sectorSummaryMap = new Map(sectorSummaries.map((summary) => [summary.sectorId, summary]))

  const defaultReadiness =
    sectorSummaries.length
      ? sectorSummaries.reduce((sum, summary) => sum + summary.readinessScore, 0) /
        sectorSummaries.length
      : 0.4
  const defaultInfluence =
    sectorSummaries.length
      ? sectorSummaries.reduce((sum, summary) => sum + summary.influenceScore, 0) /
        sectorSummaries.length
      : 0.5

  let fullPotentialValue = 0
  let adjustedValue = 0

  const contributions = milestones.map((milestone) => {
    const baseValue = milestone.business_value_estimate || 0
    fullPotentialValue += baseValue

    const completionScore = clamp01((milestone.completion_percentage || 0) / 100)
    const linkedSectorIds = new Set<string>()
    milestone.target_sector_ids?.forEach((sectorId) => {
      linkedSectorIds.add(sectorId)
    })
    milestone.target_stage_ids?.forEach((stageId) => {
      const entry = stageLookup.get(stageId)
      if (entry) {
        linkedSectorIds.add(entry.sectorId)
      }
    })

    const linkedSummaries = Array.from(linkedSectorIds)
      .map((sectorId) => sectorSummaryMap.get(sectorId))
      .filter((summary): summary is SectorValueSummary => Boolean(summary))

    const readinessScore = linkedSummaries.length
      ? linkedSummaries.reduce((sum, summary) => sum + summary.readinessScore, 0) /
        linkedSummaries.length
      : defaultReadiness

    const influenceScore = linkedSummaries.length
      ? linkedSummaries.reduce((sum, summary) => sum + summary.influenceScore, 0) /
        linkedSummaries.length
      : defaultInfluence

    const weightedScore =
      completionScore * normalizedWeights.completion +
      readinessScore * normalizedWeights.readiness +
      influenceScore * normalizedWeights.influence

    const penaltyFactor = 1 - (1 - completionScore) * dependencyPenalty
    const adjustedContribution = baseValue * weightedScore * penaltyFactor
    adjustedValue += adjustedContribution

    const linkedStages: StrategicStageReference[] = (milestone.target_stage_ids || [])
      .map((stageId) => {
        const entry = stageLookup.get(stageId)
        if (!entry) {
          return null
        }
        const reference: StrategicStageReference = {
          stageId,
          stageName: entry.stage.name,
          sectorId: entry.sectorId,
          sectorName: entry.sectorName
        }
        return reference
      })
      .filter((entry): entry is StrategicStageReference => Boolean(entry))

    return {
      id: milestone.id,
      name: milestone.name,
      baseValue,
      adjustedValue: adjustedContribution,
      completionScore,
      readinessScore,
      influenceScore,
      linkedSectors: Array.from(linkedSectorIds),
      linkedStages
    }
  })

  return {
    fullPotentialValue,
    adjustedValue,
    lockedValue: Math.max(fullPotentialValue - adjustedValue, 0),
    contributions,
    sectorSummaries
  }
}
