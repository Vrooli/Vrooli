export interface SectorFormState {
  name: string
  category: string
  description: string
  color: string
  positionX: string
  positionY: string
}

export interface StageFormState {
  sectorId: string
  stageType: string
  stageOrder: string
  name: string
  description: string
  progress: string
  positionX: string
  positionY: string
  examples: string
  parentStageId?: string
}

export interface ScenarioFormState {
  stageId: string
  scenarioName: string
  status: string
  contributionWeight: string
  priority: string
  estimatedImpact: string
  notes: string
}

export interface MilestoneFormState {
  name: string
  description: string
  milestoneType: string
  completion: string
  businessValue: string
  confidence: string
  estimatedDate: string
  targetSectors: string[]
  targetStages: string[]
}
