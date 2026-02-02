/**
 * useSkillTree - Tree navigation state management.
 *
 * Handles:
 * - Expanded/collapsed node state
 * - Selected item tracking
 * - Search/filter state
 * - Tag filtering
 * - Auto-expand to selected item
 */

import { useState, useCallback, useMemo, useEffect, useRef } from 'react'
import type { Skill } from '@/types'
import type { TreeNode } from '@/types/editor'
import {
  buildTree,
  filterTree,
  filterTreeByTags,
  filterTreeByFolders,
  getPathsToItem,
  getAllTags,
  getAllFolders,
} from '@/services/treeService'

interface UseSkillTreeProps {
  skills: Skill[]
  initialSelectedId?: string | null
  /** Initial collapsed state (for persistence) */
  initialIsCollapsed?: boolean
  /** Initial expanded nodes (for persistence) */
  initialExpandedNodes?: string[]
  /** Initial selected tags (for persistence) */
  initialSelectedTags?: string[]
  /** Initial selected folders (for persistence) */
  initialSelectedFolders?: string[]
  /** Initial search query (for persistence) */
  initialSearchQuery?: string
}

interface UseSkillTreeReturn {
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

  // Tag filter state
  selectedTags: string[]
  setSelectedTags: (tags: string[]) => void
  availableTags: string[]

  // Folder filter state
  selectedFolders: string[]
  setSelectedFolders: (folders: string[]) => void
  availableFolders: string[]

  // Sidebar collapse
  isCollapsed: boolean
  toggleCollapse: () => void
}

/**
 * Hook for managing skill tree navigation state.
 */
export function useSkillTree({
  skills,
  initialSelectedId = null,
  initialIsCollapsed = false,
  initialExpandedNodes = [],
  initialSelectedTags = [],
  initialSelectedFolders = [],
  initialSearchQuery = '',
}: UseSkillTreeProps): UseSkillTreeReturn {
  // Ref to hold skills for stable callbacks (avoids re-creating callbacks when skills load)
  const skillsRef = useRef(skills)
  skillsRef.current = skills

  // Build tree from skills
  const treeNodes = useMemo(() => buildTree(skills), [skills])

  // Selection state
  const [selectedItemId, setSelectedItemId] = useState<string | null>(initialSelectedId)

  // Expanded nodes state (initialized from persistence)
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(
    () => new Set(initialExpandedNodes)
  )

  // Search state (persisted)
  const [searchQuery, setSearchQuery] = useState(initialSearchQuery)

  // Tag filter state (initialized from persistence)
  const [selectedTags, setSelectedTags] = useState<string[]>(initialSelectedTags)

  // Folder filter state (initialized from persistence)
  const [selectedFolders, setSelectedFolders] = useState<string[]>(initialSelectedFolders)

  // Sidebar collapse state (initialized from persistence)
  const [isCollapsed, setIsCollapsed] = useState(initialIsCollapsed)

  // Get all available tags from skills
  const availableTags = useMemo(() => getAllTags(skills), [skills])

  // Get all available folders from skills
  const availableFolders = useMemo(() => getAllFolders(skills), [skills])

  // Filter tree based on search query, tags, and folders
  const filteredTreeNodes = useMemo(() => {
    let filtered = treeNodes

    // Apply folder filter
    if (selectedFolders.length > 0) {
      filtered = filterTreeByFolders(filtered, selectedFolders, skills)
    }

    // Apply tag filter
    if (selectedTags.length > 0) {
      filtered = filterTreeByTags(filtered, selectedTags, skills)
    }

    // Apply search filter
    if (searchQuery.trim()) {
      filtered = filterTree(filtered, searchQuery, skills)
    }

    return filtered
  }, [treeNodes, searchQuery, selectedTags, selectedFolders, skills])

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
  // Note: Uses skillsRef instead of skills dependency to keep callback stable
  // when skills array loads/changes. This prevents infinite loops in effects
  // that depend on expandToItem.
  const expandToItem = useCallback(
    (itemId: string) => {
      const paths = getPathsToItem(skillsRef.current, itemId)
      if (paths.length === 0) return

      setExpandedNodes((prev) => {
        const next = new Set(prev)
        for (const path of paths) {
          next.add(path)
        }
        return next
      })
    },
    []
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
  // Note: We intentionally only depend on searchQuery, not filteredTreeNodes,
  // to avoid infinite loops when setExpandedNodes triggers a re-render.
  // We access filteredTreeNodes as a snapshot, not as a reactive dependency.
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchQuery])

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

    // Tag filter state
    selectedTags,
    setSelectedTags,
    availableTags,

    // Folder filter state
    selectedFolders,
    setSelectedFolders,
    availableFolders,

    // Sidebar collapse
    isCollapsed,
    toggleCollapse,
  }
}
