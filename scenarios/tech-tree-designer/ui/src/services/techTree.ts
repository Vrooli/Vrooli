import { apiUrl } from '../utils/api'
import type {
  ApiStrategicMilestone,
  ProgressionStage,
  ScenarioCatalogSnapshot,
  ScenarioFormPayload,
  ScenarioVisibilityPayload,
  Sector,
  SectorFormPayload,
  StageDependency,
  StageFormPayload,
  StageIdea,
  TechTreeSummary
} from '../types/techTree'

type ApiResponse<T> = T & { message?: string }

type SectorResponse = { sectors: Sector[] }
type MilestoneResponse = { milestones: ApiStrategicMilestone[] }
type DependencyResponse = { dependencies: StageDependency[] }

const appendTreeId = (path: string, treeId?: string) => {
  if (!treeId) {
    return path
  }
  const separator = path.includes('?') ? '&' : '?'
  return `${path}${separator}tree_id=${encodeURIComponent(treeId)}`
}

const apiRequest = async <T>(path: string, init?: RequestInit): Promise<T> => {
  const response = await fetch(apiUrl(path), init)
  if (!response.ok) {
    const text = await response.text()
    throw new Error(text || `Request failed (${response.status})`)
  }
  return (await response.json()) as T
}

export const fetchTechTrees = () => apiRequest<{ trees: TechTreeSummary[] }>('/tech-trees')

export const fetchSectors = (treeId?: string) =>
  apiRequest<SectorResponse>(appendTreeId('/tech-tree/sectors', treeId))

export const fetchMilestones = (treeId?: string) =>
  apiRequest<MilestoneResponse>(appendTreeId('/milestones', treeId))

export const fetchDependencies = (treeId?: string) =>
  apiRequest<DependencyResponse>(appendTreeId('/dependencies', treeId))

export const fetchScenarioCatalog = () => apiRequest<ScenarioCatalogSnapshot>('/tech-tree/scenario-catalog')

export const refreshScenarioCatalog = () =>
  apiRequest<ApiResponse<{ last_synced?: string }>>('/tech-tree/scenario-catalog/refresh', { method: 'POST' })

export const updateScenarioVisibility = (payload: ScenarioVisibilityPayload) =>
  apiRequest<ApiResponse<ScenarioCatalogSnapshot>>('/tech-tree/scenario-catalog/visibility', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  })

export const createSector = (payload: SectorFormPayload, treeId?: string) =>
  apiRequest<ApiResponse<unknown>>(appendTreeId('/tech-tree/sectors', treeId), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  })

export const createStage = (payload: StageFormPayload, treeId?: string) =>
  apiRequest<ApiResponse<unknown>>(appendTreeId('/tech-tree/stages', treeId), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  })

export const linkScenario = (payload: ScenarioFormPayload, treeId?: string) =>
  apiRequest<ApiResponse<unknown>>(appendTreeId('/progress/scenarios', treeId), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  })

export const requestStageIdeas = (
  payload: { sector_id: string; prompt: string; count: number },
  treeId?: string
) =>
  apiRequest<{ ideas: StageIdea[] }>(appendTreeId('/tech-tree/ai/stage-ideas', treeId), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  })

export const saveGraph = (
  treeId: string | null,
  body: { stages: Array<{ id: string; position_x: number; position_y: number }>; dependencies: unknown }
) =>
  apiRequest<ApiResponse<{ dependencies?: unknown }>>(appendTreeId('/tech-tree/graph', treeId || undefined), {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  })

export const updateSector = (sectorId: string, payload: Partial<SectorFormPayload>, treeId?: string) =>
  apiRequest<ApiResponse<unknown>>(appendTreeId(`/tech-tree/sectors/${sectorId}`, treeId), {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  })

export const deleteSector = (sectorId: string, treeId?: string) =>
  apiRequest<ApiResponse<unknown>>(appendTreeId(`/tech-tree/sectors/${sectorId}`, treeId), {
    method: 'DELETE'
  })

export const updateStage = (stageId: string, payload: Partial<StageFormPayload>, treeId?: string) =>
  apiRequest<ApiResponse<unknown>>(appendTreeId(`/tech-tree/stages/${stageId}`, treeId), {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  })

export const deleteStage = (stageId: string, treeId?: string) =>
  apiRequest<ApiResponse<unknown>>(appendTreeId(`/tech-tree/stages/${stageId}`, treeId), {
    method: 'DELETE'
  })

export const deleteScenarioMapping = (mappingId: string, treeId?: string) =>
  apiRequest<ApiResponse<unknown>>(appendTreeId(`/progress/scenarios/${mappingId}`, treeId), {
    method: 'DELETE'
  })

// Fetch children of a stage (lazy loading for hierarchical trees)
export const fetchStageChildren = (stageId: string, treeId?: string) =>
  apiRequest<{ children: ProgressionStage[]; count: number }>(
    appendTreeId(`/tech-tree/stages/${stageId}/children`, treeId)
  )

// Tree management operations
export interface CreateTreePayload {
  name: string
  slug: string
  description: string
  tree_type: string
  status?: string
  version?: string
  parent_tree_id?: string
  is_active?: boolean
}

export interface CloneTreePayload {
  name: string
  slug: string
  description?: string
  tree_type?: string
  status?: string
  is_active?: boolean
}

export const createTechTree = (payload: CreateTreePayload) =>
  apiRequest<ApiResponse<{ tree: TechTreeSummary }>>('/tech-trees', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  })

export const cloneTechTree = (sourceTreeId: string, payload: CloneTreePayload) =>
  apiRequest<ApiResponse<{ tree: TechTreeSummary }>>(`/tech-trees/${sourceTreeId}/clone`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  })

export const updateTechTree = (
  treeId: string,
  payload: Partial<Omit<CreateTreePayload, 'parent_tree_id'>>
) =>
  apiRequest<ApiResponse<{ tree: TechTreeSummary }>>(`/tech-trees/${treeId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  })

// Export graph in various formats
export const exportGraphDOT = async (treeId?: string): Promise<string> => {
  const response = await fetch(apiUrl(appendTreeId('/tech-tree/graph/dot', treeId)))
  if (!response.ok) {
    throw new Error(`Export failed (${response.status})`)
  }
  return await response.text()
}

export const exportGraphJSON = async (treeId?: string) => {
  const [sectors, dependencies] = await Promise.all([
    fetchSectors(treeId),
    fetchDependencies(treeId)
  ])
  return {
    tree_id: treeId,
    sectors: sectors.sectors,
    dependencies: dependencies.dependencies,
    exported_at: new Date().toISOString()
  }
}

export { appendTreeId }
