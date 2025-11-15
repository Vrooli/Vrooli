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

export type ScenarioVisibilityPayload = {
  scenario: string
  hidden: boolean
}

export type TreeViewMode = 'overview' | 'designer'
