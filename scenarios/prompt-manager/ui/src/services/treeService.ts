/**
 * Tree Service - Pure functions for building and manipulating tree structures.
 *
 * Builds hierarchical trees from skill modes[] arrays.
 * Extracted from agent-inbox ItemTreeSidebar for reusability.
 */

import type { Skill } from '@/types'
import type { TreeNode } from '@/types/editor'

/**
 * Build a tree structure from skills based on their modes[] arrays.
 *
 * @param skills - Array of skills to build tree from
 * @returns Root-level tree nodes
 */
export function buildTree(skills: Skill[]): TreeNode[] {
  const root: TreeNode[] = []
  const nodeMap = new Map<string, TreeNode>()

  for (const skill of skills) {
    const modes = skill.modes

    if (modes.length === 0) {
      // Items without modes go to "Other" category
      let other = root.find((n) => n.id === '__other__')
      if (!other) {
        other = {
          id: '__other__',
          label: 'Other',
          isCategory: true,
          children: [],
          depth: 0,
        }
        root.push(other)
      }
      other.children.push({
        id: `item-${skill.id}`,
        label: skill.name,
        isCategory: false,
        children: [],
        itemId: skill.id,
        depth: 1,
      })
      continue
    }

    // Build category path
    let currentPath = ''
    let currentChildren = root

    for (let i = 0; i < modes.length; i++) {
      const mode = modes[i]
      if (!mode) continue
      currentPath = currentPath ? `${currentPath}/${mode}` : mode

      let node = nodeMap.get(currentPath)
      if (!node) {
        node = {
          id: currentPath,
          label: mode,
          isCategory: true,
          children: [],
          depth: i,
        }
        nodeMap.set(currentPath, node)
        currentChildren.push(node)
      }
      currentChildren = node.children
    }

    // Add skill as leaf node
    currentChildren.push({
      id: `item-${skill.id}`,
      label: skill.name,
      isCategory: false,
      children: [],
      itemId: skill.id,
      depth: modes.length,
    })
  }

  // Sort: categories first, then alphabetically
  return sortNodes(root)
}

/**
 * Recursively sort tree nodes: categories first, then alphabetically by label.
 */
function sortNodes(nodes: TreeNode[]): TreeNode[] {
  return nodes
    .map((n) => ({ ...n, children: sortNodes(n.children) }))
    .sort((a, b) => {
      if (a.isCategory !== b.isCategory) return a.isCategory ? -1 : 1
      return a.label.localeCompare(b.label)
    })
}

/**
 * Count dirty items in a subtree.
 *
 * @param node - Root node of subtree to count
 * @param dirtyItemIds - Set of dirty skill IDs
 * @returns Number of dirty items in subtree
 */
export function countDirtyInSubtree(node: TreeNode, dirtyItemIds: Set<string>): number {
  if (!node.isCategory && node.itemId) {
    return dirtyItemIds.has(node.itemId) ? 1 : 0
  }
  return node.children.reduce((acc, child) => acc + countDirtyInSubtree(child, dirtyItemIds), 0)
}

/**
 * Find all node IDs that contain a specific item.
 * Used for auto-expanding to a selected item.
 *
 * @param skills - All skills to search
 * @param itemId - The skill ID to find paths for
 * @returns Array of category node IDs that contain this item
 */
export function getPathsToItem(skills: Skill[], itemId: string): string[] {
  const skill = skills.find((p) => p.id === itemId)
  if (!skill?.modes || skill.modes.length === 0) {
    return []
  }

  const paths: string[] = []
  let currentPath = ''
  for (const mode of skill.modes) {
    currentPath = currentPath ? `${currentPath}/${mode}` : mode
    paths.push(currentPath)
  }
  return paths
}

/**
 * Filter tree nodes by search query.
 * Returns a new tree with only matching items and their parent categories.
 *
 * @param nodes - Tree nodes to filter
 * @param query - Search query (case-insensitive)
 * @param skills - All skills for content search
 * @returns Filtered tree nodes
 */
export function filterTree(nodes: TreeNode[], query: string, skills: Skill[]): TreeNode[] {
  if (!query.trim()) return nodes

  const lowerQuery = query.toLowerCase()

  // Find all matching skill IDs
  const matchingIds = new Set(
    skills
      .filter((p) =>
        p.name.toLowerCase().includes(lowerQuery) ||
        p.description.toLowerCase().includes(lowerQuery) ||
        p.content.toLowerCase().includes(lowerQuery) ||
        p.tags.some((t) => t.toLowerCase().includes(lowerQuery)) ||
        p.modes.some((m) => m.toLowerCase().includes(lowerQuery))
      )
      .map((p) => p.id)
  )

  // Recursively filter tree, keeping categories that have matching descendants
  function filterNode(node: TreeNode): TreeNode | null {
    if (!node.isCategory) {
      // Leaf node - include if it matches
      return node.itemId && matchingIds.has(node.itemId) ? node : null
    }

    // Category node - include if any children match
    const filteredChildren = node.children
      .map(filterNode)
      .filter((n): n is TreeNode => n !== null)

    if (filteredChildren.length === 0) return null

    return { ...node, children: filteredChildren }
  }

  return nodes.map(filterNode).filter((n): n is TreeNode => n !== null)
}

/**
 * Get all unique tags from skills.
 *
 * @param skills - All skills to extract tags from
 * @returns Sorted array of unique tag values
 */
export function getAllTags(skills: Skill[]): string[] {
  const tags = new Set<string>()

  for (const skill of skills) {
    for (const tag of skill.tags) {
      tags.add(tag)
    }
  }

  return Array.from(tags).sort()
}

/**
 * Filter tree nodes by selected tags.
 * Returns a new tree with only items that have at least one of the selected tags.
 *
 * @param nodes - Tree nodes to filter
 * @param selectedTags - Tags to filter by (items must have at least one)
 * @param skills - All skills for tag lookup
 * @returns Filtered tree nodes
 */
export function filterTreeByTags(
  nodes: TreeNode[],
  selectedTags: string[],
  skills: Skill[]
): TreeNode[] {
  if (selectedTags.length === 0) return nodes

  const tagSet = new Set(selectedTags)

  // Find all matching skill IDs
  const matchingIds = new Set(
    skills
      .filter((p) => p.tags.some((tag) => tagSet.has(tag)))
      .map((p) => p.id)
  )

  // Recursively filter tree, keeping categories that have matching descendants
  function filterNode(node: TreeNode): TreeNode | null {
    if (!node.isCategory) {
      // Leaf node - include if it matches
      return node.itemId && matchingIds.has(node.itemId) ? node : null
    }

    // Category node - include if any children match
    const filteredChildren = node.children
      .map(filterNode)
      .filter((n): n is TreeNode => n !== null)

    if (filteredChildren.length === 0) return null

    return { ...node, children: filteredChildren }
  }

  return nodes.map(filterNode).filter((n): n is TreeNode => n !== null)
}

/**
 * Get the count of selected items in a subtree.
 *
 * @param node - Root node of subtree to count
 * @param selectedIds - Set of selected skill IDs
 * @returns Number of selected items in subtree
 */
export function countSelectedInSubtree(node: TreeNode, selectedIds: Set<string>): number {
  if (!node.isCategory && node.itemId) {
    return selectedIds.has(node.itemId) ? 1 : 0
  }
  return node.children.reduce((acc, child) => acc + countSelectedInSubtree(child, selectedIds), 0)
}

/**
 * Get all skill IDs in a subtree.
 *
 * @param node - Root node of subtree
 * @returns Array of all skill IDs in the subtree
 */
export function getAllItemIdsInSubtree(node: TreeNode): string[] {
  if (!node.isCategory && node.itemId) {
    return [node.itemId]
  }
  return node.children.flatMap(getAllItemIdsInSubtree)
}

/**
 * Extract the modes path from a tree node.
 * Category nodes have IDs like "Writing/Blog/Technical", leaf nodes have IDs like "item-{id}".
 *
 * @param node - The tree node
 * @returns The modes array for this node's path, empty for leaf nodes or "Other"
 */
export function getModesPathFromNode(node: TreeNode): string[] {
  if (!node.isCategory || node.id === '__other__') {
    return []
  }
  return node.id.split('/')
}

/**
 * Get all unique modes from skills at a specific level.
 * Used for mode suggestions in the category path editor.
 *
 * @param skills - All skills to extract modes from
 * @param level - Zero-based level in the mode hierarchy
 * @param parentPath - Parent path to filter by (modes that come before this level)
 * @returns Array of unique mode values at this level
 */
export function getModesAtLevel(skills: Skill[], level: number, parentPath: string[]): string[] {
  const modes = new Set<string>()

  for (const skill of skills) {
    const skillModes = skill.modes

    // Check if this skill matches the parent path
    let matches = true
    for (let i = 0; i < parentPath.length; i++) {
      if (skillModes[i] !== parentPath[i]) {
        matches = false
        break
      }
    }

    // If matches and has a mode at the requested level, add it
    if (matches && skillModes[level]) {
      modes.add(skillModes[level])
    }
  }

  return Array.from(modes).sort()
}
