import { useCallback } from 'react'
import type { Connection, EdgeChange, NodeChange } from 'react-flow-renderer'
import { createGraphEdge, generateTempEdgeId } from '../utils/graph'
import type { DesignerEdge } from '../types/graph'

interface UseGraphInteractionParams {
  isEditMode: boolean
  scenarioOnlyMode: boolean
  onNodesStateChange: (changes: NodeChange[]) => void
  onEdgesStateChange: (changes: EdgeChange[]) => void
  setEdges: (edges: DesignerEdge[] | ((prev: DesignerEdge[]) => DesignerEdge[])) => void
  setHasGraphChanges: (value: boolean) => void
  setEditError: (error: string | null) => void
}

interface UseGraphInteractionResult {
  handleNodesChange: (changes: NodeChange[]) => void
  handleEdgesChange: (changes: EdgeChange[]) => void
  handleConnect: (connection: Connection) => void
}

/**
 * Custom hook for managing graph interaction handlers (nodes, edges, connections).
 * Handles edit mode constraints and dirty state tracking.
 */
export const useGraphInteraction = ({
  isEditMode,
  scenarioOnlyMode,
  onNodesStateChange,
  onEdgesStateChange,
  setEdges,
  setHasGraphChanges,
  setEditError
}: UseGraphInteractionParams): UseGraphInteractionResult => {
  const handleNodesChange = useCallback(
    (changes: NodeChange[]) => {
      if (!isEditMode) {
        return
      }

      if (
        changes.some(
          (change) =>
            change.type === 'position' || change.type === 'remove' || change.type === 'add'
        )
      ) {
        setHasGraphChanges(true)
      }

      setEditError(null)
      onNodesStateChange(changes)
    },
    [isEditMode, onNodesStateChange, setHasGraphChanges, setEditError]
  )

  const handleEdgesChange = useCallback(
    (changes: EdgeChange[]) => {
      if (!isEditMode) {
        return
      }

      if (changes.some((change) => change.type === 'remove' || change.type === 'add')) {
        setHasGraphChanges(true)
      }

      setEditError(null)
      onEdgesStateChange(changes)
    },
    [isEditMode, onEdgesStateChange, setHasGraphChanges, setEditError]
  )

  const handleConnect = useCallback(
    (connection: Connection) => {
      if (!isEditMode || !connection?.source || !connection?.target) {
        return
      }

      setEditError(null)
      const sourceId = connection.source as string
      const targetId = connection.target as string

      setEdges((currentEdges) => {
        // Prevent duplicate edges
        const exists = currentEdges.some(
          (edge) => edge.source === sourceId && edge.target === targetId
        )
        if (exists) {
          return currentEdges
        }

        const newEdge = createGraphEdge({
          id: `temp-${generateTempEdgeId()}`,
          source: sourceId,
          target: targetId,
          dependencyType: 'required',
          dependencyStrength: 0.6
        })

        setHasGraphChanges(true)
        return [...currentEdges, newEdge]
      })
    },
    [isEditMode, setEdges, setEditError, setHasGraphChanges]
  )

  return {
    handleNodesChange,
    handleEdgesChange,
    handleConnect
  }
}
