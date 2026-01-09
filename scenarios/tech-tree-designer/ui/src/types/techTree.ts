export interface TechTree {
  id: string
  slug: string
  name: string
  description: string
  version: string
  tree_type: string
  status: string
  is_active: boolean
  parent_tree_id?: string | null
  created_at?: string
  updated_at?: string
}

export interface TechTreeSummary {
  tree: TechTree
  sector_count: number
  stage_count: number
  scenario_mapping_count: number
}

export interface ScenarioMapping {
  id?: string
  scenario_name: string
  completion_status?: string
  contribution_weight?: number
  priority?: number
  estimated_impact?: number
  notes?: string
}

export interface StageDependency {
  dependency: {
    id: string
    dependent_stage_id: string
    prerequisite_stage_id: string
    dependency_type?: string
    dependency_strength?: number
    description?: string
  }
  dependent_name?: string
  prerequisite_name?: string
}

export interface ProgressionStage {
  id: string
  sector_id: string
  parent_stage_id?: string | null // Hierarchical parent (null = root level)
  stage_type: string
  stage_order?: number
  name: string
  description?: string
  progress_percentage?: number
  position_x?: number
  position_y?: number
  has_children?: boolean // Indicates if this stage has child stages
  children_loaded?: boolean // Client-side tracking for lazy loading
  examples?: string[]
  scenario_mappings?: ScenarioMapping[]
  children?: ProgressionStage[] // Lazy-loaded child stages
}

export interface Sector {
  id: string
  tree_id: string
  name: string
  category: string
  description?: string
  progress_percentage?: number
  position_x?: number
  position_y?: number
  color?: string
  stages: ProgressionStage[]
}

export interface StrategicMilestone {
  id: string
  tree_id: string
  name: string
  description: string
  milestone_type: string
  business_value_estimate?: number
  completion_percentage?: number
  estimated_completion_date?: string
  target_sector_ids?: string[]
  target_stage_ids?: string[]
  confidence_level?: number
}

export interface ApiStrategicMilestone extends Omit<StrategicMilestone, 'target_sector_ids' | 'target_stage_ids'> {
  target_sector_ids?: string[]
  target_stage_ids?: string[]
  required_sectors?: string[] | string | null
  required_stages?: string[] | string | null
}

export interface StrategicValueSettings {
  completionWeight: number
  readinessWeight: number
  influenceWeight: number
  dependencyPenalty: number
}

export interface StrategicValueContribution {
  id: string
  name: string
  baseValue: number
  adjustedValue: number
  completionScore: number
  readinessScore: number
  influenceScore: number
  linkedSectors: string[]
  linkedStages: StrategicStageReference[]
}

export interface SectorValueSummary {
  sectorId: string
  name: string
  color?: string
  readinessScore: number
  influenceScore: number
  progressPercentage: number
  scenarioLinks: number
  stageAverage: number
  milestoneCount: number
}

export interface StrategicValueBreakdown {
  fullPotentialValue: number
  adjustedValue: number
  lockedValue: number
  contributions: StrategicValueContribution[]
  sectorSummaries: SectorValueSummary[]
}

export interface StrategicStageReference {
  stageId: string
  stageName?: string
  sectorId?: string
  sectorName?: string
}

export interface StrategicValuePreset {
  id: string
  name: string
  description?: string
  settings: StrategicValueSettings
  builtIn?: boolean
}

export interface ScenarioCatalogEntry {
  name: string
  display_name: string
  description: string
  relative_path: string
  tags: string[]
  dependencies: string[]
  resources: string[]
  hidden: boolean
  last_modified?: string
}

export interface ScenarioDependencyEdge {
  from: string
  to: string
  type: string
}

export interface ScenarioCatalogSnapshot {
  scenarios: ScenarioCatalogEntry[]
  edges: ScenarioDependencyEdge[]
  hidden: string[]
  last_synced: string | null
}

export interface StageIdea {
  name: string
  description: string
  stage_type: string
  suggested_scenarios?: string[]
  confidence?: number
  strategic_rationale?: string
}

export interface ScenarioFormPayload {
  stage_id: string
  scenario_name: string
  completion_status: string
  contribution_weight: number
  priority: number
  estimated_impact: number
  notes: string
}

export interface SectorFormPayload {
  name: string
  category: string
  description: string
  color: string
  position_x?: number
  position_y?: number
}

export interface StageFormPayload {
  sector_id: string
  stage_type: string
  stage_order?: number
  name: string
  description: string
  progress_percentage?: number
  position_x?: number
  position_y?: number
  examples: string[]
}

export interface MilestoneFormPayload {
  name: string
  description: string
  milestone_type: string
  completion_percentage?: number
  business_value_estimate?: number
  confidence_level?: number
  estimated_completion_date?: string
  target_sector_ids?: string[]
  target_stage_ids?: string[]
}

export type ScenarioVisibilityPayload = {
  scenario: string
  hidden: boolean
}

export type TreeViewMode = 'overview' | 'designer'

export type GraphViewMode = 'tech-tree' | 'hybrid' | 'scenarios'

export type SectorSortOption =
  | 'most-progress'
  | 'least-progress'
  | 'most-strategic'
  | 'least-strategic'
  | 'alpha-asc'
  | 'alpha-desc'
