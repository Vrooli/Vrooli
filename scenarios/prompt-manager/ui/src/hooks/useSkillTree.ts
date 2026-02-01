/**
 * useSkillTree - Tree navigation state management.
 *
 * Handles:
 * - Expanded/collapsed node state
 * - Selected item tracking
 * - Search/filter state
 * - Tag filtering
 * - Skill selection mode for agents
 * - Auto-expand to selected item
 */

import { useState, useCallback, useMemo, useEffect } from 'react'
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
  countSelectedInSubtree,
  getAllItemIdsInSubtree,
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

  // Skill selection mode
  skillSelectionMode: boolean
  skillSelectedIds: Set<string>
  currentAgentId: string | null
  enterSkillSelectionMode: (agentId: string, currentSkills: string[]) => void
  exitSkillSelectionMode: () => void
  toggleSkillSelection: (skillId: string) => void
  toggleFolderSkillSelection: (node: TreeNode) => void
  getSkillSelectionState: (node: TreeNode) => 'none' | 'partial' | 'all'

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
}: UseSkillTreeProps): UseSkillTreeReturn {
  // Build tree from skills
  const treeNodes = useMemo(() => buildTree(skills), [skills])

  // Selection state
  const [selectedItemId, setSelectedItemId] = useState<string | null>(initialSelectedId)

  // Expanded nodes state (initialized from persistence)
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(
    () => new Set(initialExpandedNodes)
  )

  // Search state (intentionally not persisted - transient)
  const [searchQuery, setSearchQuery] = useState('')

  // Tag filter state (initialized from persistence)
  const [selectedTags, setSelectedTags] = useState<string[]>(initialSelectedTags)

  // Folder filter state (initialized from persistence)
  const [selectedFolders, setSelectedFolders] = useState<string[]>(initialSelectedFolders)

  // Skill selection mode state
  const [skillSelectionMode, setSkillSelectionMode] = useState(false)
  const [skillSelectedIds, setSkillSelectedIds] = useState<Set<string>>(new Set())
  const [currentAgentId, setCurrentMemberId] = useState<string | null>(null)

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
  const expandToItem = useCallback(
    (itemId: string) => {
      const paths = getPathsToItem(skills, itemId)
      if (paths.length === 0) return

      setExpandedNodes((prev) => {
        const next = new Set(prev)
        for (const path of paths) {
          next.add(path)
        }
        return next
      })
    },
    [skills]
  )

  // Toggle sidebar collapse
  const toggleCollapse = useCallback(() => {
    setIsCollapsed((prev) => !prev)
  }, [])

  // Skill selection mode functions
  const enterSkillSelectionMode = useCallback((agentId: string, currentSkills: string[]) => {
    setSkillSelectionMode(true)
    setCurrentMemberId(agentId)
    setSkillSelectedIds(new Set(currentSkills))
  }, [])

  const exitSkillSelectionMode = useCallback(() => {
    setSkillSelectionMode(false)
    setCurrentMemberId(null)
    setSkillSelectedIds(new Set())
  }, [])

  const toggleSkillSelection = useCallback((skillId: string) => {
    setSkillSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(skillId)) {
        next.delete(skillId)
      } else {
        next.add(skillId)
      }
      return next
    })
  }, [])

  const toggleFolderSkillSelection = useCallback((node: TreeNode) => {
    const allIds = getAllItemIdsInSubtree(node)
    setSkillSelectedIds((prev) => {
      const next = new Set(prev)
      // Check if all are selected
      const allSelected = allIds.every((id) => prev.has(id))

      if (allSelected) {
        // Deselect all
        for (const id of allIds) {
          next.delete(id)
        }
      } else {
        // Select all
        for (const id of allIds) {
          next.add(id)
        }
      }
      return next
    })
  }, [])

  const getSkillSelectionState = useCallback(
    (node: TreeNode): 'none' | 'partial' | 'all' => {
      if (!node.isCategory && node.itemId) {
        return skillSelectedIds.has(node.itemId) ? 'all' : 'none'
      }

      const allIds = getAllItemIdsInSubtree(node)
      if (allIds.length === 0) return 'none'

      const selectedCount = countSelectedInSubtree(node, skillSelectedIds)
      if (selectedCount === 0) return 'none'
      if (selectedCount === allIds.length) return 'all'
      return 'partial'
    },
    [skillSelectedIds]
  )

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

    // Tag filter state
    selectedTags,
    setSelectedTags,
    availableTags,

    // Folder filter state
    selectedFolders,
    setSelectedFolders,
    availableFolders,

    // Skill selection mode
    skillSelectionMode,
    skillSelectedIds,
    currentAgentId,
    enterSkillSelectionMode,
    exitSkillSelectionMode,
    toggleSkillSelection,
    toggleFolderSkillSelection,
    getSkillSelectionState,

    // Sidebar collapse
    isCollapsed,
    toggleCollapse,
  }
}
