/**
 * usePromptTree - Tree navigation state management.
 *
 * Handles:
 * - Expanded/collapsed node state
 * - Selected item tracking
 * - Search/filter state
 * - Auto-expand to selected item
 */

import { useState, useCallback, useMemo, useEffect } from 'react'
import type { Prompt } from '@/types'
import type { TreeNode } from '@/types/editor'
import { buildTree, filterTree, getPathsToItem } from '@/services/treeService'

interface UsePromptTreeProps {
  prompts: Prompt[]
  initialSelectedId?: string | null
}

interface UsePromptTreeReturn {
  // Tree data
  treeNodes: TreeNode[]
  filteredTreeNodes: TreeNode[]

  // Selection state
  selectedItemId: string | null
  setSelectedItemId: (id: string | null) => void

  // Expansion state
  expandedNodes: Set<string>
  toggleNode: (nodeId: string) => void
  expandAll: () => void
  collapseAll: () => void
  expandToItem: (itemId: string) => void

  // Search state
  searchQuery: string
  setSearchQuery: (query: string) => void

  // Sidebar collapse
  isCollapsed: boolean
  toggleCollapse: () => void
}

/**
 * Hook for managing prompt tree navigation state.
 */
export function usePromptTree({ prompts, initialSelectedId = null }: UsePromptTreeProps): UsePromptTreeReturn {
  // Build tree from prompts
  const treeNodes = useMemo(() => buildTree(prompts), [prompts])

  // Selection state
  const [selectedItemId, setSelectedItemId] = useState<string | null>(initialSelectedId)

  // Expanded nodes state
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(() => new Set())

  // Search state
  const [searchQuery, setSearchQuery] = useState('')

  // Sidebar collapse state
  const [isCollapsed, setIsCollapsed] = useState(false)

  // Filter tree based on search query
  const filteredTreeNodes = useMemo(() => {
    if (!searchQuery.trim()) return treeNodes
    return filterTree(treeNodes, searchQuery, prompts)
  }, [treeNodes, searchQuery, prompts])

  // Toggle a single node's expanded state
  const toggleNode = useCallback((nodeId: string) => {
    setExpandedNodes((prev) => {
      const next = new Set(prev)
      if (next.has(nodeId)) {
        next.delete(nodeId)
      } else {
        next.add(nodeId)
      }
      return next
    })
  }, [])

  // Expand all category nodes
  const expandAll = useCallback(() => {
    const allCategoryIds = new Set<string>()

    const collectCategoryIds = (nodes: TreeNode[]) => {
      for (const node of nodes) {
        if (node.isCategory) {
          allCategoryIds.add(node.id)
          collectCategoryIds(node.children)
        }
      }
    }

    collectCategoryIds(treeNodes)
    setExpandedNodes(allCategoryIds)
  }, [treeNodes])

  // Collapse all nodes
  const collapseAll = useCallback(() => {
    setExpandedNodes(new Set())
  }, [])

  // Expand nodes to reveal a specific item
  const expandToItem = useCallback(
    (itemId: string) => {
      const paths = getPathsToItem(prompts, itemId)
      if (paths.length === 0) return

      setExpandedNodes((prev) => {
        const next = new Set(prev)
        for (const path of paths) {
          next.add(path)
        }
        return next
      })
    },
    [prompts]
  )

  // Toggle sidebar collapse
  const toggleCollapse = useCallback(() => {
    setIsCollapsed((prev) => !prev)
  }, [])

  // Auto-expand to selected item when selection changes
  useEffect(() => {
    if (selectedItemId) {
      expandToItem(selectedItemId)
    }
  }, [selectedItemId, expandToItem])

  // When search query changes, expand all matching nodes
  useEffect(() => {
    if (searchQuery.trim()) {
      // Expand all nodes in filtered tree
      const nodesToExpand = new Set<string>()

      const collectExpandable = (nodes: TreeNode[]) => {
        for (const node of nodes) {
          if (node.isCategory) {
            nodesToExpand.add(node.id)
            collectExpandable(node.children)
          }
        }
      }

      collectExpandable(filteredTreeNodes)
      setExpandedNodes(nodesToExpand)
    }
  }, [searchQuery, filteredTreeNodes])

  return {
    // Tree data
    treeNodes,
    filteredTreeNodes,

    // Selection state
    selectedItemId,
    setSelectedItemId,

    // Expansion state
    expandedNodes,
    toggleNode,
    expandAll,
    collapseAll,
    expandToItem,

    // Search state
    searchQuery,
    setSearchQuery,

    // Sidebar collapse
    isCollapsed,
    toggleCollapse,
  }
}
