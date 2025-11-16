import type { ProgressionStage, Sector, StrategicMilestone } from '../types/techTree'

type DemoStage = Omit<ProgressionStage, 'id' | 'sector_id'> & {
  id?: string
  sector_id?: string
}

type DemoSector = Omit<Sector, 'stages' | 'tree_id'> & {
  tree_id?: string
  stages: DemoStage[]
}

const DEMO_TREE_ID = 'demo-tree'

const buildStage = (sectorId: string, name: string, progress: number, stageType: string): DemoStage => ({
  id: `${sectorId}-${name.toLowerCase().replace(/[^a-z0-9]+/g, '-')}`,
  sector_id: sectorId,
  name,
  progress_percentage: progress,
  stage_type: stageType,
  description: '',
  examples: []
})

const baseSectors: DemoSector[] = [
  {
    id: 'sector-1',
    tree_id: DEMO_TREE_ID,
    name: 'Personal Productivity',
    category: 'individual',
    description: 'Individual tools for task management, note-taking, time tracking',
    progress_percentage: 75,
    color: '#10B981',
    position_x: 100,
    position_y: 100,
    stages: [
      buildStage('sector-1', 'Task Management', 85, 'foundation'),
      buildStage('sector-1', 'Personal Automation', 70, 'operational'),
      buildStage('sector-1', 'Self-Analytics', 60, 'analytics'),
      buildStage('sector-1', 'Life Integration', 45, 'integration'),
      buildStage('sector-1', 'Personal Digital Twin', 30, 'digital_twin')
    ]
  },
  {
    id: 'sector-2',
    tree_id: DEMO_TREE_ID,
    name: 'Software Engineering',
    category: 'software',
    description: 'Development tools and platforms that accelerate all other domains',
    progress_percentage: 45,
    color: '#EC4899',
    position_x: 100,
    position_y: 300,
    stages: [
      buildStage('sector-2', 'Development Tools', 65, 'foundation'),
      buildStage('sector-2', 'DevOps', 50, 'operational'),
      buildStage('sector-2', 'Code Intelligence', 25, 'analytics'),
      buildStage('sector-2', 'Platform Integration', 15, 'integration'),
      buildStage('sector-2', 'Engineering Digital Twin', 5, 'digital_twin')
    ]
  },
  {
    id: 'sector-3',
    tree_id: DEMO_TREE_ID,
    name: 'Manufacturing Systems',
    category: 'manufacturing',
    description: 'Complete manufacturing ecosystem from design to production',
    progress_percentage: 25,
    color: '#3B82F6',
    position_x: 300,
    position_y: 200,
    stages: [
      buildStage('sector-3', 'Product Lifecycle Management', 40, 'foundation'),
      buildStage('sector-3', 'Manufacturing Execution', 20, 'operational'),
      buildStage('sector-3', 'Production Analytics', 15, 'analytics'),
      buildStage('sector-3', 'Smart Factory Integration', 10, 'integration'),
      buildStage('sector-3', 'Manufacturing Digital Twin', 5, 'digital_twin')
    ]
  },
  {
    id: 'sector-4',
    tree_id: DEMO_TREE_ID,
    name: 'Healthcare Systems',
    category: 'healthcare',
    description: 'Healthcare technology from records to population health twins',
    progress_percentage: 20,
    color: '#EF4444',
    position_x: 500,
    position_y: 200,
    stages: [
      buildStage('sector-4', 'Electronic Health Records', 35, 'foundation'),
      buildStage('sector-4', 'Clinical Operations', 15, 'operational'),
      buildStage('sector-4', 'Clinical Decision Support', 10, 'analytics'),
      buildStage('sector-4', 'Health Information Exchange', 8, 'integration'),
      buildStage('sector-4', 'Population Health Twin', 4, 'digital_twin')
    ]
  }
]

export const getDemoSectors = (augmenter?: (sectors: Sector[]) => Sector[]): Sector[] => {
  const normalized = baseSectors.map((sector) => ({
    ...sector,
    tree_id: sector.tree_id ?? DEMO_TREE_ID,
    stages: sector.stages.map((stage, index) => ({
      id: stage.id ?? `${sector.id}-stage-${index + 1}`,
      sector_id: stage.sector_id ?? sector.id,
      stage_type: stage.stage_type,
      stage_order: index + 1,
      name: stage.name,
      description: stage.description ?? '',
      progress_percentage: stage.progress_percentage,
      position_x: sector.position_x,
      position_y: sector.position_y,
      examples: stage.examples ?? []
    }))
  })) as Sector[]

  return typeof augmenter === 'function' ? augmenter(normalized) : normalized
}

export const getDemoMilestones = (): StrategicMilestone[] => [
  {
    id: 'milestone-1',
    tree_id: DEMO_TREE_ID,
    name: 'Individual Productivity Mastery',
    description: 'Complete personal productivity and self-optimization capabilities',
    completion_percentage: 75,
    business_value_estimate: 10_000_000,
    milestone_type: 'sector_complete',
    target_sector_ids: ['sector-1']
  },
  {
    id: 'milestone-2',
    tree_id: DEMO_TREE_ID,
    name: 'Core Sector Digital Twins',
    description: 'Manufacturing, Healthcare, Finance digital twins operational',
    completion_percentage: 25,
    business_value_estimate: 1_000_000_000,
    milestone_type: 'cross_sector_integration',
    target_sector_ids: ['sector-2', 'sector-3', 'sector-4']
  },
  {
    id: 'milestone-3',
    tree_id: DEMO_TREE_ID,
    name: 'Civilization Digital Twin',
    description: 'Complete society-scale simulation integrating all sectors',
    completion_percentage: 5,
    business_value_estimate: 100_000_000_000,
    milestone_type: 'civilization_twin',
    target_sector_ids: ['sector-1', 'sector-2', 'sector-3', 'sector-4']
  }
]
