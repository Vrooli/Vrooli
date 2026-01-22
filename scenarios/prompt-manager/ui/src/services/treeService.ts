/**
 * Tree Service - Pure functions for building and manipulating tree structures.
 *
 * Builds hierarchical trees from prompt modes[] arrays.
 * Extracted from agent-inbox ItemTreeSidebar for reusability.
 */

import type { Prompt } from '@/types'
import type { TreeNode } from '@/types/editor'

/**
 * Build a tree structure from prompts based on their modes[] arrays.
 *
 * @param prompts - Array of prompts to build tree from
 * @returns Root-level tree nodes
 */
export function buildTree(prompts: Prompt[]): TreeNode[] {
  const root: TreeNode[] = []
  const nodeMap = new Map<string, TreeNode>()

  for (const prompt of prompts) {
    const modes = prompt.modes ?? []
    const isReadonly = prompt.folder === 'core'

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
        id: `item-${prompt.id}`,
        label: prompt.name,
        isCategory: false,
        children: [],
        itemId: prompt.id,
        depth: 1,
        isReadonly,
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

    // Add prompt as leaf node
    currentChildren.push({
      id: `item-${prompt.id}`,
      label: prompt.name,
      isCategory: false,
      children: [],
      itemId: prompt.id,
      depth: modes.length,
      isReadonly,
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
 * @param dirtyItemIds - Set of dirty prompt IDs
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
 * @param prompts - All prompts to search
 * @param itemId - The prompt ID to find paths for
 * @returns Array of category node IDs that contain this item
 */
export function getPathsToItem(prompts: Prompt[], itemId: string): string[] {
  const prompt = prompts.find((p) => p.id === itemId)
  if (!prompt?.modes || prompt.modes.length === 0) {
    return []
  }

  const paths: string[] = []
  let currentPath = ''
  for (const mode of prompt.modes) {
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
 * @param prompts - All prompts for content search
 * @returns Filtered tree nodes
 */
export function filterTree(nodes: TreeNode[], query: string, prompts: Prompt[]): TreeNode[] {
  if (!query.trim()) return nodes

  const lowerQuery = query.toLowerCase()

  // Find all matching prompt IDs
  const matchingIds = new Set(
    prompts
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
 * Get all unique modes from prompts at a specific level.
 * Used for mode suggestions in the category path editor.
 *
 * @param prompts - All prompts to extract modes from
 * @param level - Zero-based level in the mode hierarchy
 * @param parentPath - Parent path to filter by (modes that come before this level)
 * @returns Array of unique mode values at this level
 */
export function getModesAtLevel(prompts: Prompt[], level: number, parentPath: string[]): string[] {
  const modes = new Set<string>()

  for (const prompt of prompts) {
    const promptModes = prompt.modes ?? []

    // Check if this prompt matches the parent path
    let matches = true
    for (let i = 0; i < parentPath.length; i++) {
      if (promptModes[i] !== parentPath[i]) {
        matches = false
        break
      }
    }

    // If matches and has a mode at the requested level, add it
    if (matches && promptModes[level]) {
      modes.add(promptModes[level])
    }
  }

  return Array.from(modes).sort()
}
