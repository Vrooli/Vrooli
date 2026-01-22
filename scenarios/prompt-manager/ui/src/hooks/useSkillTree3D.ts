/**
 * Hook for managing 3D skill tree state and interactions.
 */

import { useCallback, useEffect, useMemo, useState } from 'react'
import type { Prompt } from '@/types'
import type { SkillTreeNode, CameraState } from '@/types/skilltree'
import {
  buildSkillTree,
  updateSelection,
  findNodeByPromptId,
  getSelectedPrompts,
  calculateCameraPosition,
} from '@/services/skillTreeService'

interface UseSkillTree3DOptions {
  prompts: Prompt[]
  selectedPromptIds?: string[]
  onSelectionChange?: (promptIds: string[]) => void
  onNodeClick?: (promptId: string) => void
}

export function useSkillTree3D(options: UseSkillTree3DOptions) {
  const { prompts, selectedPromptIds = [], onSelectionChange, onNodeClick } = options

  // Build tree data from prompts
  const treeData = useMemo(() => {
    const data = buildSkillTree(prompts)
    return updateSelection(data, selectedPromptIds)
  }, [prompts, selectedPromptIds])

  // Camera state
  const [cameraState, setCameraState] = useState<CameraState>(() => ({
    position: calculateCameraPosition(treeData),
    target: [0, 0, 0],
    zoom: 1,
  }))

  // Hovered node
  const [hoveredNodeId, setHoveredNodeId] = useState<string | null>(null)

  // Update camera when tree changes significantly
  useEffect(() => {
    if (prompts.length > 0) {
      const newPosition = calculateCameraPosition(treeData)
      setCameraState((prev) => ({
        ...prev,
        position: newPosition,
      }))
    }
  }, [prompts.length, treeData])

  // Handle node click
  const handleNodeClick = useCallback(
    (nodeId: string, event: MouseEvent) => {
      const node = treeData.nodes.find((n) => n.id === nodeId)
      if (!node) return

      if (event.metaKey || event.ctrlKey) {
        // Toggle selection
        const newSelection = selectedPromptIds.includes(node.promptId)
          ? selectedPromptIds.filter((id) => id !== node.promptId)
          : [...selectedPromptIds, node.promptId]
        onSelectionChange?.(newSelection)
      } else if (event.shiftKey && selectedPromptIds.length > 0) {
        // Add to selection
        if (!selectedPromptIds.includes(node.promptId)) {
          onSelectionChange?.([...selectedPromptIds, node.promptId])
        }
      } else {
        // Single select
        onSelectionChange?.([node.promptId])
        onNodeClick?.(node.promptId)
      }
    },
    [treeData.nodes, selectedPromptIds, onSelectionChange, onNodeClick]
  )

  // Handle node hover
  const handleNodeHover = useCallback((nodeId: string | null) => {
    setHoveredNodeId(nodeId)
  }, [])

  // Get node by prompt ID
  const getNode = useCallback(
    (promptId: string): SkillTreeNode | undefined => {
      return findNodeByPromptId(treeData, promptId)
    },
    [treeData]
  )

  // Get all selected prompts
  const selectedPrompts = useMemo(
    () => getSelectedPrompts(treeData),
    [treeData]
  )

  // Focus camera on a specific node
  const focusOnNode = useCallback(
    (promptId: string) => {
      const node = findNodeByPromptId(treeData, promptId)
      if (node) {
        setCameraState((prev) => ({
          ...prev,
          target: node.position,
          position: [
            node.position[0],
            node.position[1] + 3,
            node.position[2] + 5,
          ],
        }))
      }
    },
    [treeData]
  )

  // Reset camera to show all nodes
  const resetCamera = useCallback(() => {
    setCameraState({
      position: calculateCameraPosition(treeData),
      target: [0, 0, 0],
      zoom: 1,
    })
  }, [treeData])

  // Zoom controls
  const zoomIn = useCallback(() => {
    setCameraState((prev) => ({
      ...prev,
      zoom: Math.min(prev.zoom * 1.2, 3),
    }))
  }, [])

  const zoomOut = useCallback(() => {
    setCameraState((prev) => ({
      ...prev,
      zoom: Math.max(prev.zoom / 1.2, 0.5),
    }))
  }, [])

  return {
    // Data
    treeData,
    nodes: treeData.nodes,
    connections: treeData.connections,
    selectedPrompts,

    // Camera
    cameraState,
    focusOnNode,
    resetCamera,
    zoomIn,
    zoomOut,

    // Interaction
    hoveredNodeId,
    handleNodeClick,
    handleNodeHover,
    getNode,

    // Computed
    nodeCount: treeData.nodes.length,
    hasNodes: treeData.nodes.length > 0,
    selectionCount: selectedPromptIds.length,
  }
}
