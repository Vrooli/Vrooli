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
import type { GraphResponse, GraphNode } from '@/lib/schemas'
import { getGraph, regenerateGraph as regenerateGraphService } from '@/services/graphService'

const GRAPH_VIEWPORT_STORAGE_KEY = 'pm.graphViewport'

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

    const parsed = JSON.parse(raw)
    if (
      typeof parsed === 'object' &&
      parsed !== null &&
      Number.isFinite(parsed.x) &&
      Number.isFinite(parsed.y) &&
      Number.isFinite(parsed.zoom)
    ) {
      return {
        x: parsed.x,
        y: parsed.y,
        zoom: parsed.zoom,
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
  showCLIs: boolean
  healthThreshold: number
}

interface GraphStore {
  graph: GraphResponse | null
  loading: boolean
  error: string | null
  filters: GraphFilters
  highlightedNodeIds: Set<string>
  layoutDirection: 'TB' | 'LR'
  fitViewRequested: number
  viewport: GraphViewport | null

  fetchGraph: (forceRefresh?: boolean) => Promise<void>
  regenerateGraph: () => Promise<void>
  setFilter: <K extends keyof GraphFilters>(key: K, value: GraphFilters[K]) => void
  highlightNodes: (ids: string[]) => void
  clearHighlights: () => void
  setLayoutDirection: (dir: 'TB' | 'LR') => void
  requestFitView: () => void
  setViewport: (viewport: GraphViewport) => void
}

export const useGraphStore = create<GraphStore>((set, get) => ({
  graph: null,
  loading: false,
  error: null,
  filters: {
    showTeams: true,
    showAgents: true,
    showSkills: true,
    showCLIs: true,
    healthThreshold: 0,
  },
  highlightedNodeIds: new Set(),
  layoutDirection: 'TB',
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
    set((state) => ({
      filters: { ...state.filters, [key]: value },
    }))
  },

  highlightNodes: (ids) => {
    set({ highlightedNodeIds: new Set(ids) })
  },

  clearHighlights: () => {
    set({ highlightedNodeIds: new Set() })
  },

  setLayoutDirection: (dir) => {
    set({ layoutDirection: dir })
  },

  requestFitView: () => {
    set((state) => ({ fitViewRequested: state.fitViewRequested + 1 }))
  },

  setViewport: (viewport) => {
    set({ viewport })
    saveGraphViewport(viewport)
  },
}))

/**
 * Selector: get filtered nodes based on current filter state.
 */
export function selectFilteredNodes(state: GraphStore): GraphNode[] {
  const { graph, filters } = state
  if (!graph) return []

  // Build health map for threshold filtering
  const healthMap = new Map<string, number>()
  for (const hs of graph.graph.healthScores) {
    healthMap.set(hs.nodeId, hs.score)
  }

  return graph.graph.nodes.filter((node) => {
    // Type filter
    if (node.type === 'team' && !filters.showTeams) return false
    if (node.type === 'agent' && !filters.showAgents) return false
    if (node.type === 'skill' && !filters.showSkills) return false
    if (node.type === 'cli' && !filters.showCLIs) return false

    // Health threshold filter
    if (filters.healthThreshold > 0) {
      const score = healthMap.get(node.id)
      if (score !== undefined && score < filters.healthThreshold) return false
    }

    return true
  })
}
