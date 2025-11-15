import { useCallback, useState } from 'react'
import { fetchStageChildren } from '../services/techTree'
import type { ProgressionStage } from '../types/techTree'

interface UseNodeExpansionParams {
  selectedTreeId?: string | null
  onChildrenLoaded?: (parentStageId: string, children: ProgressionStage[]) => void
  onError?: (error: string) => void
}

interface UseNodeExpansionResult {
  expandedNodes: Set<string>
  loadingNodes: Set<string>
  toggleNodeExpansion: (stageId: string, hasChildren: boolean, childrenLoaded: boolean) => Promise<void>
  isNodeExpanded: (stageId: string) => boolean
  isNodeLoading: (stageId: string) => boolean
}

/**
 * Custom hook for managing hierarchical node expansion state and loading children.
 *
 * Responsibilities:
 * - Track which nodes are expanded/collapsed
 * - Track which nodes are currently loading children
 * - Fetch children from API when expanding a node for the first time
 * - Notify parent components when children are loaded
 */
export const useNodeExpansion = ({
  selectedTreeId,
  onChildrenLoaded,
  onError
}: UseNodeExpansionParams = {}): UseNodeExpansionResult => {
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set())
  const [loadingNodes, setLoadingNodes] = useState<Set<string>>(new Set())

  const toggleNodeExpansion = useCallback(
    async (stageId: string, hasChildren: boolean, childrenLoaded: boolean) => {
      // If node is currently expanded, just collapse it
      if (expandedNodes.has(stageId)) {
        setExpandedNodes((prev) => {
          const next = new Set(prev)
          next.delete(stageId)
          return next
        })
        return
      }

      // If node doesn't have children, nothing to expand
      if (!hasChildren) {
        return
      }

      // Expand the node immediately (optimistic update)
      setExpandedNodes((prev) => new Set(prev).add(stageId))

      // If children already loaded, we're done
      if (childrenLoaded) {
        return
      }

      // Load children from API
      setLoadingNodes((prev) => new Set(prev).add(stageId))

      try {
        const response = await fetchStageChildren(stageId, selectedTreeId || undefined)
        const children = response.children || []

        // Notify parent component to insert children into graph
        if (onChildrenLoaded) {
          onChildrenLoaded(stageId, children)
        }
      } catch (error) {
        // On error, collapse the node back
        setExpandedNodes((prev) => {
          const next = new Set(prev)
          next.delete(stageId)
          return next
        })

        const errorMessage = error instanceof Error ? error.message : 'Failed to load children'
        if (onError) {
          onError(errorMessage)
        }
      } finally {
        setLoadingNodes((prev) => {
          const next = new Set(prev)
          next.delete(stageId)
          return next
        })
      }
    },
    [expandedNodes, selectedTreeId, onChildrenLoaded, onError]
  )

  const isNodeExpanded = useCallback(
    (stageId: string) => expandedNodes.has(stageId),
    [expandedNodes]
  )

  const isNodeLoading = useCallback(
    (stageId: string) => loadingNodes.has(stageId),
    [loadingNodes]
  )

  return {
    expandedNodes,
    loadingNodes,
    toggleNodeExpansion,
    isNodeExpanded,
    isNodeLoading
  }
}
