/**
 * Zustand store for graph visualization state.
 *
 * Manages:
 * - Full graph data (nodes + edges)
 * - Filter state (type toggles, health threshold)
 * - Highlighted node IDs (from query results)
 * - Loading/error state
 *
 * IMPORTANT: Derived selectors that return new references (arrays, objects)
 * must be consumed with `useShallow` to prevent infinite re-render loops.
 * See selectFilteredNodes for details.
 */

import { create } from 'zustand'
import type { GraphResponse, GraphNode, HealthScore } from '@/lib/schemas'
import { getGraph, getGraphHealth, regenerateGraph as regenerateGraphService } from '@/services/graphService'

const GRAPH_VIEWPORT_STORAGE_KEY = 'pm.graphViewport'
const GRAPH_VIEW_SETTINGS_STORAGE_KEY = 'pm.graphViewSettings.v1'

export interface GraphViewport {
  x: number
  y: number
  zoom: number
}

function loadGraphViewport(): GraphViewport | null {
  if (typeof window === 'undefined') return null

  try {
    const raw = localStorage.getItem(GRAPH_VIEWPORT_STORAGE_KEY)
    if (!raw) return null

    const parsed: unknown = JSON.parse(raw) as unknown
    if (
      typeof parsed !== 'object' ||
      parsed === null
    ) return null
    const record = parsed as Record<string, unknown>
    if (
      Number.isFinite(record.x) &&
      Number.isFinite(record.y) &&
      Number.isFinite(record.zoom)
    ) {
      return {
        x: record.x as number,
        y: record.y as number,
        zoom: record.zoom as number,
      }
    }
  } catch {
    // Ignore malformed localStorage payloads.
  }

  return null
}

function saveGraphViewport(viewport: GraphViewport): void {
  if (typeof window === 'undefined') return
  try {
    localStorage.setItem(GRAPH_VIEWPORT_STORAGE_KEY, JSON.stringify(viewport))
  } catch {
    // Ignore quota errors.
  }
}

interface GraphFilters {
  showTeams: boolean
  showAgents: boolean
  showSkills: boolean
  showActions?: boolean
  showCLIs: boolean
  collapseCLIs: boolean
  showLowSignalEdges: boolean
  autoFitOnChange: boolean
  healthThreshold: number
}

export type GraphLayoutMode = 'hierarchical' | 'compact' | 'grouped'
export type GraphQueryDisplayMode = 'highlight' | 'dim-others' | 'hide-others'

interface GraphViewSettingsSnapshot {
  filters: GraphFilters
  layoutDirection: 'TB' | 'LR'
  layoutMode: GraphLayoutMode
  queryDisplayMode: GraphQueryDisplayMode
}

function getDefaultFilters(): GraphFilters {
  return {
    showTeams: true,
    showAgents: true,
    showSkills: true,
    showActions: true,
    showCLIs: true,
    collapseCLIs: false,
    showLowSignalEdges: true,
    autoFitOnChange: true,
    healthThreshold: 0,
  }
}

function isValidQueryDisplayMode(value: unknown): value is GraphQueryDisplayMode {
  return value === 'highlight' || value === 'dim-others' || value === 'hide-others'
}

function loadGraphViewSettings(): GraphViewSettingsSnapshot {
  const defaults: GraphViewSettingsSnapshot = {
    filters: getDefaultFilters(),
    layoutDirection: 'TB',
    layoutMode: 'compact',
    queryDisplayMode: 'dim-others',
  }

  if (typeof window === 'undefined') return defaults

  try {
    const raw = localStorage.getItem(GRAPH_VIEW_SETTINGS_STORAGE_KEY)
    if (!raw) return defaults
    const parsed: unknown = JSON.parse(raw)
    if (typeof parsed !== 'object' || parsed === null) return defaults
    const record = parsed as Record<string, unknown>

    const filtersRaw = (typeof record.filters === 'object' && record.filters !== null)
      ? (record.filters as Record<string, unknown>)
      : {}
    const defaultFilters = getDefaultFilters()
    const filters: GraphFilters = {
      showTeams: typeof filtersRaw.showTeams === 'boolean' ? filtersRaw.showTeams : defaultFilters.showTeams,
      showAgents: typeof filtersRaw.showAgents === 'boolean' ? filtersRaw.showAgents : defaultFilters.showAgents,
      showSkills: typeof filtersRaw.showSkills === 'boolean' ? filtersRaw.showSkills : defaultFilters.showSkills,
      showActions: typeof filtersRaw.showActions === 'boolean' ? filtersRaw.showActions : defaultFilters.showActions,
      showCLIs: typeof filtersRaw.showCLIs === 'boolean' ? filtersRaw.showCLIs : defaultFilters.showCLIs,
      collapseCLIs: typeof filtersRaw.collapseCLIs === 'boolean' ? filtersRaw.collapseCLIs : defaultFilters.collapseCLIs,
      showLowSignalEdges: typeof filtersRaw.showLowSignalEdges === 'boolean' ? filtersRaw.showLowSignalEdges : defaultFilters.showLowSignalEdges,
      autoFitOnChange: typeof filtersRaw.autoFitOnChange === 'boolean' ? filtersRaw.autoFitOnChange : defaultFilters.autoFitOnChange,
      healthThreshold: Number.isFinite(filtersRaw.healthThreshold) ? Number(filtersRaw.healthThreshold) : defaultFilters.healthThreshold,
    }

    const layoutDirection = record.layoutDirection === 'LR' ? 'LR' : 'TB'
    const layoutMode = record.layoutMode === 'hierarchical' || record.layoutMode === 'grouped'
      ? record.layoutMode
      : 'compact'
    const queryDisplayMode = isValidQueryDisplayMode(record.queryDisplayMode)
      ? record.queryDisplayMode
      : defaults.queryDisplayMode

    return {
      filters,
      layoutDirection,
      layoutMode,
      queryDisplayMode,
    }
  } catch {
    return defaults
  }
}

function saveGraphViewSettings(settings: GraphViewSettingsSnapshot): void {
  if (typeof window === 'undefined') return
  try {
    localStorage.setItem(GRAPH_VIEW_SETTINGS_STORAGE_KEY, JSON.stringify(settings))
  } catch {
    // Ignore quota errors.
  }
}

export type HighlightSource = 'query' | 'focus' | null

interface GraphStore {
  graph: GraphResponse | null
  standaloneHealthScores: HealthScore[] | null
  healthScoreOverride: HealthScore[] | null
  loading: boolean
  error: string | null
  filters: GraphFilters
  highlightedNodeIds: Set<string>
  highlightSource: HighlightSource
  queryDisplayMode: GraphQueryDisplayMode
  layoutDirection: 'TB' | 'LR'
  layoutMode: GraphLayoutMode
  fitViewRequested: number
  viewport: GraphViewport | null

  fetchGraph: (forceRefresh?: boolean) => Promise<void>
  fetchHealthScores: () => Promise<void>
  regenerateGraph: () => Promise<void>
  setFilter: <K extends keyof GraphFilters>(key: K, value: GraphFilters[K]) => void
  highlightNodes: (ids: string[]) => void
  focusNodes: (ids: string[]) => void
  clearHighlights: () => void
  setQueryDisplayMode: (mode: GraphQueryDisplayMode) => void
  setLayoutDirection: (dir: 'TB' | 'LR') => void
  setLayoutMode: (mode: GraphLayoutMode) => void
  requestFitView: () => void
  setViewport: (viewport: GraphViewport) => void
  setHealthScoreOverride: (scores: HealthScore[] | null) => void
  clearHealthScoreOverride: () => void
}

export const useGraphStore = create<GraphStore>((set, get) => ({
  ...loadGraphViewSettings(),
  graph: null,
  standaloneHealthScores: null,
  healthScoreOverride: null,
  loading: false,
  error: null,
  highlightedNodeIds: new Set(),
  highlightSource: null,
  fitViewRequested: 0,
  viewport: loadGraphViewport(),

  fetchGraph: async (forceRefresh = false) => {
    if (get().loading) return
    set({ loading: true, error: null })
    try {
      const data = await getGraph(forceRefresh)
      set({ graph: data, loading: false })
    } catch (error) {
      set({
        error: error instanceof Error ? error.message : 'Failed to load graph',
        loading: false,
      })
    }
  },

  fetchHealthScores: async () => {
    try {
      const scores = await getGraphHealth()
      if (scores) {
        set({ standaloneHealthScores: scores })
      }
    } catch {
      // Non-critical — sidebar will just show no health badges
    }
  },

  regenerateGraph: async () => {
    set({ loading: true, error: null })
    try {
      await regenerateGraphService()
      const data = await getGraph(true)
      set({ graph: data, loading: false })
    } catch (error) {
      set({
        error: error instanceof Error ? error.message : 'Failed to regenerate graph',
        loading: false,
      })
    }
  },

  setFilter: (key, value) => {
    set((state) => {
      const filters = { ...state.filters, [key]: value }
      saveGraphViewSettings({
        filters,
        layoutDirection: state.layoutDirection,
        layoutMode: state.layoutMode,
        queryDisplayMode: state.queryDisplayMode,
      })
      return { filters }
    })
  },

  highlightNodes: (ids) => {
    set({ highlightedNodeIds: new Set(ids), highlightSource: 'query' })
  },

  focusNodes: (ids) => {
    set({ highlightedNodeIds: new Set(ids), highlightSource: 'focus' })
  },

  clearHighlights: () => {
    set({ highlightedNodeIds: new Set(), highlightSource: null })
  },

  setQueryDisplayMode: (mode) => {
    set((state) => {
      saveGraphViewSettings({
        filters: state.filters,
        layoutDirection: state.layoutDirection,
        layoutMode: state.layoutMode,
        queryDisplayMode: mode,
      })
      return { queryDisplayMode: mode }
    })
  },

  setLayoutDirection: (dir) => {
    set((state) => {
      saveGraphViewSettings({
        filters: state.filters,
        layoutDirection: dir,
        layoutMode: state.layoutMode,
        queryDisplayMode: state.queryDisplayMode,
      })
      return { layoutDirection: dir }
    })
  },

  setLayoutMode: (mode) => {
    set((state) => {
      saveGraphViewSettings({
        filters: state.filters,
        layoutDirection: state.layoutDirection,
        layoutMode: mode,
        queryDisplayMode: state.queryDisplayMode,
      })
      return { layoutMode: mode }
    })
  },

  requestFitView: () => {
    set((state) => ({ fitViewRequested: state.fitViewRequested + 1 }))
  },

  setViewport: (viewport) => {
    set({ viewport })
    saveGraphViewport(viewport)
  },

  setHealthScoreOverride: (scores) => {
    set({ healthScoreOverride: scores })
  },

  clearHealthScoreOverride: () => {
    set({ healthScoreOverride: null })
  },
}))

/**
 * Selector: get filtered nodes based on current filter state.
 */
export function selectFilteredNodes(state: GraphStore): GraphNode[] {
  const { graph, filters } = state
  if (!graph) return []
  const effectiveScores = state.healthScoreOverride ?? graph.graph.healthScores

  // Build health map for threshold filtering
  const healthMap = new Map<string, number>()
  for (const hs of effectiveScores) {
    healthMap.set(hs.nodeId, hs.score)
  }

  return graph.graph.nodes.filter((node) => {
    // Type filter
    if (node.type === 'team' && !filters.showTeams) return false
    if (node.type === 'agent' && !filters.showAgents) return false
    if (node.type === 'skill' && !filters.showSkills) return false
    if (node.type === 'action' && filters.showActions === false) return false
    if (node.type === 'cli' && !filters.showCLIs) return false

    // Health threshold filter
    if (filters.healthThreshold > 0) {
      const score = healthMap.get(node.id)
      if (score !== undefined && score < filters.healthThreshold) return false
    }

    return true
  })
}

export function selectEffectiveHealthScores(state: GraphStore): HealthScore[] {
  return state.healthScoreOverride
    ?? state.graph?.graph.healthScores
    ?? state.standaloneHealthScores
    ?? []
}
