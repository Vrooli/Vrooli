import { useCallback, useEffect, useState } from 'react'
import type { EdgeChange, NodeChange } from 'react-flow-renderer'
import { apiUrl } from '../utils/api'
import { isTemporaryEdgeId } from '../utils/graph'
import type { DesignerEdge, StageNode } from '../types/graph'

interface UseGraphEditorParams {
  stageNodes: StageNode[]
  designerEdges: DesignerEdge[]
  scenarioOnlyMode: boolean
  buildTreeAwarePath: (path: string) => string
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
}

/**
 * Custom hook for managing graph edit mode, dirty state tracking, and persistence.
 * Separates edit concerns from the main canvas component.
 */
export const useGraphEditor = ({
  stageNodes,
  designerEdges,
  scenarioOnlyMode,
  buildTreeAwarePath,
  setEdges,
  onGraphPersist,
  setGraphNotice
}: UseGraphEditorParams): UseGraphEditorResult => {
  const [isEditMode, setIsEditMode] = useState(false)
  const [hasGraphChanges, setHasGraphChanges] = useState(false)
  const [isPersisting, setIsPersisting] = useState(false)
  const [editError, setEditError] = useState<string | null>(null)

  // Clear dirty state and errors when exiting edit mode
  useEffect(() => {
    if (!isEditMode) {
      setHasGraphChanges(false)
      setEditError(null)
    }
  }, [isEditMode])

  // Prevent edit mode in scenario-only view
  useEffect(() => {
    if (scenarioOnlyMode && isEditMode) {
      setIsEditMode(false)
    }
  }, [scenarioOnlyMode, isEditMode])

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
  }, [stageNodes, designerEdges, buildTreeAwarePath, setEdges, onGraphPersist])

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
    persistGraph
  }
}
