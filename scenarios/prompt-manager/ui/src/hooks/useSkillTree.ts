/**
 * useSkillTree - Tree navigation state management.
 *
 * Handles:
 * - Expanded/collapsed node state
 * - Selected item tracking
 * - Search/filter/sort/view state
 * - Auto-expand to selected item
 */

import { useState, useCallback, useMemo, useEffect, useRef } from 'react'
import type { Skill } from '@/types'
import type { TreeNode } from '@/types/editor'
import type { FilterState, SortConfig, ViewMode } from '@/types/filterSort'
import { DEFAULT_FILTER_STATE, DEFAULT_SORT_CONFIG, DEFAULT_VIEW_MODE } from '@/types/filterSort'
import {
  buildTree,
  filterTree,
  filterTreeBySkillIds,
  getPathsToItem,
  getAllTags,
  getAllFolders,
} from '@/services/treeService'
import { applyFilters, sortSkills } from '@/services/filterSortService'
// AI_CHECK: TREE_EXPAND_STATE_CHURN=1 | LAST: 2026-02-17

interface UseSkillTreeProps {
  skills: Skill[]
  initialSelectedId?: string | null
  /** Initial collapsed state (for persistence) */
  initialIsCollapsed?: boolean
  /** Initial expanded nodes (for persistence) */
  initialExpandedNodes?: string[]
  /** Initial filter state (for persistence) */
  initialFilterState?: FilterState
  /** Initial sort config (for persistence) */
  initialSortConfig?: SortConfig
  /** Initial view mode (for persistence) */
  initialViewMode?: ViewMode
  /** Initial search query (for persistence) */
  initialSearchQuery?: string
}

interface UseSkillTreeReturn {
  // Tree data
  treeNodes: TreeNode[]
  filteredTreeNodes: TreeNode[]

  // Flat data (for list/card views)
  filteredSortedSkills: Skill[]

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

  // Filter state
  filterState: FilterState
  setFilterState: (state: FilterState) => void
  availableTags: string[]
  availableFolders: string[]

  // Sort state
  sortConfig: SortConfig
  setSortConfig: (config: SortConfig) => void

  // View mode
  viewMode: ViewMode
  setViewMode: (mode: ViewMode) => void

  // Sidebar collapse
  isCollapsed: boolean
  toggleCollapse: () => void
}

function areStringSetsEqual(a: Set<string>, b: Set<string>): boolean {
  if (a.size !== b.size) return false
  for (const value of a) {
    if (!b.has(value)) return false
  }
  return true
}

/**
 * Hook for managing skill tree navigation state.
 */
export function useSkillTree({
  skills,
  initialSelectedId = null,
  initialIsCollapsed = false,
  initialExpandedNodes = [],
  initialFilterState = DEFAULT_FILTER_STATE,
  initialSortConfig = DEFAULT_SORT_CONFIG,
  initialViewMode = DEFAULT_VIEW_MODE,
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

  // Filter/sort/view state (initialized from persistence)
  const [filterState, setFilterState] = useState<FilterState>(initialFilterState)
  const [sortConfig, setSortConfig] = useState<SortConfig>(initialSortConfig)
  const [viewMode, setViewMode] = useState<ViewMode>(initialViewMode)

  // Sidebar collapse state (initialized from persistence)
  const [isCollapsed, setIsCollapsed] = useState(initialIsCollapsed)
  const hasHydratedSearchExpansion = useRef(false)

  // Get all available tags from skills (unfiltered)
  const availableTags = useMemo(() => getAllTags(skills), [skills])

  // Get all available folders from skills (unfiltered)
  const availableFolders = useMemo(() => getAllFolders(skills), [skills])

  // Step 1: Apply filters → flat Skill[]
  // Step 2: Apply search query → flat Skill[]
  // Step 3: Sort → flat Skill[]
  const filteredSortedSkills = useMemo(() => {
    let result = applyFilters(skills, filterState)

    // Apply search query (same fields as filterTree uses for the tree view)
    const query = searchQuery.trim().toLowerCase()
    if (query) {
      result = result.filter((s) =>
        s.name.toLowerCase().includes(query) ||
        s.description.toLowerCase().includes(query) ||
        s.content.toLowerCase().includes(query) ||
        s.tags.some((t) => t.toLowerCase().includes(query)) ||
        s.modes.some((m) => m.toLowerCase().includes(query))
      )
    }

    return sortSkills(result, sortConfig)
  }, [skills, filterState, searchQuery, sortConfig])

  // Step 3: For tree view — filter tree by matching skill IDs
  // Sorting is not applied in tree view (sort dropdown is hidden for tree mode).
  const filteredTreeNodes = useMemo(() => {
    const matchingIds = new Set(filteredSortedSkills.map((s) => s.id))
    let filtered = matchingIds.size === skills.length
      ? treeNodes
      : filterTreeBySkillIds(treeNodes, matchingIds)

    // Apply search filter on top
    if (searchQuery.trim()) {
      filtered = filterTree(filtered, searchQuery, skills)
    }

    return filtered
  }, [treeNodes, filteredSortedSkills, searchQuery, skills])

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
        let changed = false
        const next = new Set(prev)
        for (const path of paths) {
          if (!next.has(path)) {
            next.add(path)
            changed = true
          }
        }
        return changed ? next : prev
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
    // Preserve persisted folder expansion on first mount even when a search query is restored.
    if (!hasHydratedSearchExpansion.current) {
      hasHydratedSearchExpansion.current = true
      return
    }

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
      setExpandedNodes((prev) => (areStringSetsEqual(prev, nodesToExpand) ? prev : nodesToExpand))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchQuery])

  return {
    // Tree data
    treeNodes,
    filteredTreeNodes,

    // Flat data
    filteredSortedSkills,

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

    // Filter state
    filterState,
    setFilterState,
    availableTags,
    availableFolders,

    // Sort state
    sortConfig,
    setSortConfig,

    // View mode
    viewMode,
    setViewMode,

    // Sidebar collapse
    isCollapsed,
    toggleCollapse,
  }
}
