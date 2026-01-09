import { useCallback, useEffect, useRef, useState } from 'react'
import type { EdgeChange, NodeChange } from 'react-flow-renderer'
import { apiUrl } from '../utils/api'
import { isTemporaryEdgeId } from '../utils/graph'
import type { DesignerEdge, StageNode } from '../types/graph'

interface UseGraphEditorParams {
  stageNodes: StageNode[]
  designerEdges: DesignerEdge[]
  scenarioOnlyMode: boolean
  buildTreeAwarePath: (path: string) => string
  setNodes: (nodes: StageNode[] | ((prev: StageNode[]) => StageNode[])) => void
  setEdges: (edges: DesignerEdge[] | ((prev: DesignerEdge[]) => DesignerEdge[])) => void
  onGraphPersist: (payload: { notice?: string; message?: string }) => void
  setGraphNotice: (notice: string) => void
}

interface UseGraphEditorResult {
  isEditMode: boolean
  hasGraphChanges: boolean
  isPersisting: boolean
  editError: string | null
  setIsEditMode: (value: boolean) => void
  setHasGraphChanges: (value: boolean) => void
  setEditError: (error: string | null) => void
  handleToggleEditMode: () => void
  persistGraph: () => Promise<boolean>
  canUndo: boolean
  canRedo: boolean
  handleUndo: () => void
  handleRedo: () => void
  handleCancelEdit: () => void
}

interface GraphSnapshot {
  nodes: StageNode[]
  edges: DesignerEdge[]
}

const cloneStageNodes = (nodes: StageNode[]): StageNode[] =>
  nodes.map((node) => ({
    ...node,
    position: { ...node.position },
    data: { ...node.data }
  }))

const cloneDesignerEdges = (edges: DesignerEdge[]): DesignerEdge[] =>
  edges.map((edge) => ({
    ...edge,
    data: edge.data ? { ...edge.data } : edge.data
  }))

const buildNodeMap = (nodes: StageNode[]) => {
  const map = new Map<string, { x: number; y: number }>()
  nodes.forEach((node) => {
    map.set(node.id, {
      x: node?.position?.x ?? 0,
      y: node?.position?.y ?? 0
    })
  })
  return map
}

const buildEdgeSet = (edges: DesignerEdge[]) => {
  const set = new Set<string>()
  edges.forEach((edge) => {
    set.add(`${edge.source}->${edge.target}`)
  })
  return set
}

const areSnapshotsEqual = (first: GraphSnapshot, second: GraphSnapshot) => {
  if (first.nodes.length !== second.nodes.length || first.edges.length !== second.edges.length) {
    return false
  }

  const nodeMap = buildNodeMap(second.nodes)
  for (const node of first.nodes) {
    const comparison = nodeMap.get(node.id)
    if (!comparison) {
      return false
    }
    const x = node?.position?.x ?? 0
    const y = node?.position?.y ?? 0
    if (comparison.x !== x || comparison.y !== y) {
      return false
    }
  }

  const edgeSet = buildEdgeSet(second.edges)
  for (const edge of first.edges) {
    if (!edgeSet.has(`${edge.source}->${edge.target}`)) {
      return false
    }
  }

  return true
}

const createSnapshot = (nodes: StageNode[], edges: DesignerEdge[]): GraphSnapshot => ({
  nodes: cloneStageNodes(nodes),
  edges: cloneDesignerEdges(edges)
})

/**
 * Custom hook for managing graph edit mode, dirty state tracking, and persistence.
 * Separates edit concerns from the main canvas component.
 */
export const useGraphEditor = ({
  stageNodes,
  designerEdges,
  scenarioOnlyMode,
  buildTreeAwarePath,
  setNodes,
  setEdges,
  onGraphPersist,
  setGraphNotice
}: UseGraphEditorParams): UseGraphEditorResult => {
  const [isEditMode, setIsEditMode] = useState(false)
  const [hasGraphChanges, setHasGraphChanges] = useState(false)
  const [isPersisting, setIsPersisting] = useState(false)
  const [editError, setEditError] = useState<string | null>(null)
  const [canUndo, setCanUndo] = useState(false)
  const [canRedo, setCanRedo] = useState(false)
  const historyRef = useRef<{
    past: GraphSnapshot[]
    future: GraphSnapshot[]
    baseline: GraphSnapshot | null
    isRestoring: boolean
  }>({ past: [], future: [], baseline: null, isRestoring: false })

  // Clear dirty state and errors when exiting edit mode
  useEffect(() => {
    if (!isEditMode) {
      setHasGraphChanges(false)
      setEditError(null)
      historyRef.current = { past: [], future: [], baseline: null, isRestoring: false }
      setCanUndo(false)
      setCanRedo(false)
      return
    }

    if (!historyRef.current.baseline) {
      const baselineSnapshot = createSnapshot(stageNodes, designerEdges)
      historyRef.current = {
        past: [baselineSnapshot],
        future: [],
        baseline: baselineSnapshot,
        isRestoring: false
      }
      setCanUndo(false)
      setCanRedo(false)
    }
  }, [designerEdges, isEditMode, stageNodes, setHasGraphChanges])

  // Prevent edit mode in scenario-only view
  useEffect(() => {
    if (scenarioOnlyMode && isEditMode) {
      setIsEditMode(false)
    }
  }, [scenarioOnlyMode, isEditMode])

  useEffect(() => {
    if (!isEditMode) {
      return
    }
    const history = historyRef.current
    if (history.isRestoring || !history.baseline) {
      history.isRestoring = false
      return
    }

    const snapshot = createSnapshot(stageNodes, designerEdges)
    const lastSnapshot = history.past[history.past.length - 1]
    if (lastSnapshot && areSnapshotsEqual(lastSnapshot, snapshot)) {
      return
    }

    history.past = [...history.past, snapshot]
    history.future = []
    setCanUndo(history.past.length > 1)
    setCanRedo(false)
    setHasGraphChanges(!areSnapshotsEqual(history.baseline, snapshot))
  }, [designerEdges, isEditMode, stageNodes, setHasGraphChanges])

  const persistGraph = useCallback(async (): Promise<boolean> => {
    const stagePositions = stageNodes.map((node) => ({
      id: node.id,
      position_x: node?.position?.x ?? 0,
      position_y: node?.position?.y ?? 0
    }))

    const dependencyPayload = designerEdges.map((edge) => ({
      id: isTemporaryEdgeId(edge.id) ? undefined : edge.id,
      dependent_stage_id: edge.target,
      prerequisite_stage_id: edge.source,
      dependency_type: edge.data?.dependencyType || 'required',
      dependency_strength:
        typeof edge.data?.dependencyStrength === 'number' ? edge.data.dependencyStrength : 1,
      description: edge.data?.description || ''
    }))

    setIsPersisting(true)
    setEditError(null)

    try {
      const response = await fetch(apiUrl(buildTreeAwarePath('/tech-tree/graph')), {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          stages: stagePositions,
          dependencies: dependencyPayload
        })
      })

      if (!response.ok) {
        throw new Error(`Failed to update graph (status ${response.status})`)
      }

      const payload = await response.json()

      onGraphPersist(payload)

      // Update edges with server response
      if (Array.isArray(payload?.dependencies)) {
        const { createGraphEdge } = await import('../utils/graph')
        const normalizedEdges = payload.dependencies
          .map((item: any) => {
            const dep = item?.dependency || item
            if (!dep) {
              return null
            }

            return createGraphEdge({
              id: dep.id,
              source: dep.prerequisite_stage_id,
              target: dep.dependent_stage_id,
              dependencyType: dep.dependency_type,
              dependencyStrength: dep.dependency_strength,
              description: dep.description
            })
          })
          .filter(Boolean) as DesignerEdge[]

        setEdges(normalizedEdges)
      }

      setHasGraphChanges(false)
      setEditError(null)
      if (isEditMode) {
        const snapshot = createSnapshot(stageNodes, designerEdges)
        historyRef.current = {
          past: [snapshot],
          future: [],
          baseline: snapshot,
          isRestoring: false
        }
        setCanUndo(false)
        setCanRedo(false)
      }
      return true
    } catch (error) {
      console.error('Failed to persist graph edits:', error)
      setEditError(
        error instanceof Error
          ? error.message
          : 'Unable to save graph changes. Please try again.'
      )
      return false
    } finally {
      setIsPersisting(false)
    }
  }, [buildTreeAwarePath, designerEdges, isEditMode, onGraphPersist, setEdges, setHasGraphChanges, stageNodes])

  const handleUndo = useCallback(() => {
    if (!isEditMode) {
      return
    }
    const history = historyRef.current
    if (history.past.length <= 1) {
      return
    }
    const current = history.past.pop()
    if (current) {
      history.future.unshift(current)
    }
    const previous = history.past[history.past.length - 1]
    if (!previous) {
      return
    }
    history.isRestoring = true
    setNodes(cloneStageNodes(previous.nodes))
    setEdges(cloneDesignerEdges(previous.edges))
    setCanUndo(history.past.length > 1)
    setCanRedo(history.future.length > 0)
    if (history.baseline) {
      setHasGraphChanges(!areSnapshotsEqual(history.baseline, previous))
    } else {
      setHasGraphChanges(history.past.length > 1)
    }
  }, [isEditMode, setEdges, setHasGraphChanges, setNodes])

  const handleRedo = useCallback(() => {
    if (!isEditMode) {
      return
    }
    const history = historyRef.current
    if (history.future.length === 0) {
      return
    }
    const nextSnapshot = history.future.shift()
    if (!nextSnapshot) {
      return
    }
    history.past.push(nextSnapshot)
    history.isRestoring = true
    setNodes(cloneStageNodes(nextSnapshot.nodes))
    setEdges(cloneDesignerEdges(nextSnapshot.edges))
    setCanUndo(history.past.length > 1)
    setCanRedo(history.future.length > 0)
    if (history.baseline) {
      setHasGraphChanges(!areSnapshotsEqual(history.baseline, nextSnapshot))
    } else {
      setHasGraphChanges(true)
    }
  }, [isEditMode, setEdges, setHasGraphChanges, setNodes])

  const handleCancelEdit = useCallback(() => {
    const history = historyRef.current
    const baseline = history.baseline
    if (baseline) {
      history.isRestoring = true
      setNodes(cloneStageNodes(baseline.nodes))
      setEdges(cloneDesignerEdges(baseline.edges))
    }
    setHasGraphChanges(false)
    setCanUndo(false)
    setCanRedo(false)
    setIsEditMode(false)
  }, [setEdges, setHasGraphChanges, setNodes])

  const handleToggleEditMode = useCallback(() => {
    if (scenarioOnlyMode) {
      setGraphNotice(
        'Scenario-only view is read-only. Switch to the combined view to edit the strategic graph.'
      )
      return
    }

    if (!isEditMode) {
      setHasGraphChanges(false)
      setEditError(null)
      setIsEditMode(true)
      return
    }

    if (isPersisting) {
      return
    }

    if (!hasGraphChanges) {
      setIsEditMode(false)
      return
    }

    void persistGraph().then((success) => {
      if (success) {
        setIsEditMode(false)
      }
    })
  }, [hasGraphChanges, isEditMode, isPersisting, persistGraph, scenarioOnlyMode, setGraphNotice])

  return {
    isEditMode,
    hasGraphChanges,
    isPersisting,
    editError,
    setIsEditMode,
    setHasGraphChanges,
    setEditError,
    handleToggleEditMode,
    persistGraph,
    canUndo,
    canRedo,
    handleUndo,
    handleRedo,
    handleCancelEdit
  }
}
