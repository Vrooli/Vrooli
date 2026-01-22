/**
 * Service for transforming prompts into 3D skill tree positions.
 * Handles layout algorithms and tree construction.
 */

import type { Prompt } from '@/types'
import type {
  SkillTreeNode,
  SkillTreeConnection,
  SkillTreeData,
  LayoutOptions,
} from '@/types/skilltree'

// Default layout options
const DEFAULT_LAYOUT: LayoutOptions = {
  horizontalSpacing: 3,
  verticalSpacing: 2.5,
  radialSpread: Math.PI * 0.6,
  use3DDepth: true,
}

// Color palette for different categories
const CATEGORY_COLORS: Record<string, string> = {
  coding: '#3b82f6', // blue
  writing: '#22c55e', // green
  analysis: '#f59e0b', // amber
  creative: '#ec4899', // pink
  general: '#8b5cf6', // purple
  system: '#06b6d4', // cyan
  default: '#64748b', // slate
}

/**
 * Get color for a prompt based on its modes/tags.
 */
function getPromptColor(prompt: Prompt): string {
  const modes = prompt.modes || []
  const tags = prompt.tags || []
  const combined = [...modes, ...tags]

  for (const item of combined) {
    const category = item.split('/')[0]?.toLowerCase() || ''
    if (CATEGORY_COLORS[category]) {
      return CATEGORY_COLORS[category]
    }
  }

  return CATEGORY_COLORS.default ?? '#64748b'
}

/**
 * Calculate node size based on usage count and other metrics.
 */
function getNodeSize(prompt: Prompt): number {
  const baseSize = 0.4
  const usageBonus = Math.min((prompt.usageCount || 0) * 0.02, 0.3)
  const ratingBonus = ((prompt.effectivenessRating || 3) - 3) * 0.05
  return baseSize + usageBonus + ratingBonus
}


/**
 * Transform an array of prompts into a 3D skill tree structure.
 */
export function buildSkillTree(
  prompts: Prompt[],
  options: Partial<LayoutOptions> = {}
): SkillTreeData {
  const layoutOptions = { ...DEFAULT_LAYOUT, ...options }
  const nodes: SkillTreeNode[] = []
  const connections: SkillTreeConnection[] = []
  const roots: string[] = []

  if (prompts.length === 0) {
    return { nodes, connections, roots, maxDepth: 0 }
  }

  // Group prompts by primary mode for clustering
  const modeGroups = new Map<string, Prompt[]>()
  const ungrouped: Prompt[] = []

  prompts.forEach((prompt) => {
    const primaryMode = (prompt.modes || [])[0]?.split('/')[0]
    if (primaryMode) {
      if (!modeGroups.has(primaryMode)) {
        modeGroups.set(primaryMode, [])
      }
      modeGroups.get(primaryMode)!.push(prompt)
    } else {
      ungrouped.push(prompt)
    }
  })

  // All grouped prompts become group roots
  const allGroups = Array.from(modeGroups.entries())
  let nodeIndex = 0

  // Position each group as a cluster
  allGroups.forEach(([_mode, groupPrompts], groupIndex) => {
    const groupAngle = (groupIndex / allGroups.length) * Math.PI * 2
    const groupRadius = layoutOptions.horizontalSpacing * 3
    const groupCenter: [number, number, number] = [
      Math.cos(groupAngle) * groupRadius,
      0,
      Math.sin(groupAngle) * groupRadius,
    ]

    // Position prompts within the group
    groupPrompts.forEach((prompt, promptIndex) => {
      const localAngle = (promptIndex / groupPrompts.length) * Math.PI * 2
      const localRadius = layoutOptions.horizontalSpacing * 0.8
      const position: [number, number, number] = [
        groupCenter[0] + Math.cos(localAngle) * localRadius,
        groupCenter[1],
        groupCenter[2] + Math.sin(localAngle) * localRadius,
      ]

      const node: SkillTreeNode = {
        id: `node-${prompt.id}`,
        promptId: prompt.id,
        name: prompt.name,
        description: prompt.description || '',
        position,
        parentId: null,
        children: [],
        isSelected: false,
        depth: 0,
        prompt,
        color: getPromptColor(prompt),
        size: getNodeSize(prompt),
      }

      nodes.push(node)
      roots.push(node.id)
      nodeIndex++
    })

    // Create connections between nodes in the same group
    if (groupPrompts.length > 1) {
      for (let i = 0; i < groupPrompts.length; i++) {
        const j = (i + 1) % groupPrompts.length
        const sourceNode = nodes.find(
          (n) => n.promptId === groupPrompts[i]?.id
        )
        const targetNode = nodes.find(
          (n) => n.promptId === groupPrompts[j]?.id
        )

        if (sourceNode && targetNode) {
          connections.push({
            id: `conn-${sourceNode.id}-${targetNode.id}`,
            sourceId: sourceNode.id,
            targetId: targetNode.id,
            source: sourceNode.position,
            target: targetNode.position,
            strength: 0.3,
          })
        }
      }
    }
  })

  // Position ungrouped prompts in the center
  ungrouped.forEach((prompt, index) => {
    const angle = (index / Math.max(ungrouped.length, 1)) * Math.PI * 2
    const radius = layoutOptions.horizontalSpacing * 0.5
    const position: [number, number, number] = [
      Math.cos(angle) * radius,
      layoutOptions.verticalSpacing,
      Math.sin(angle) * radius,
    ]

    const node: SkillTreeNode = {
      id: `node-${prompt.id}`,
      promptId: prompt.id,
      name: prompt.name,
      description: prompt.description || '',
      position,
      parentId: null,
      children: [],
      isSelected: false,
      depth: 0,
      prompt,
      color: getPromptColor(prompt),
      size: getNodeSize(prompt),
    }

    nodes.push(node)
    roots.push(node.id)
  })

  return {
    nodes,
    connections,
    roots,
    maxDepth: 0, // Flat structure for now
  }
}

/**
 * Update node selection state.
 */
export function updateSelection(
  treeData: SkillTreeData,
  selectedIds: string[]
): SkillTreeData {
  return {
    ...treeData,
    nodes: treeData.nodes.map((node) => ({
      ...node,
      isSelected: selectedIds.includes(node.promptId),
    })),
  }
}

/**
 * Find a node by prompt ID.
 */
export function findNodeByPromptId(
  treeData: SkillTreeData,
  promptId: string
): SkillTreeNode | undefined {
  return treeData.nodes.find((n) => n.promptId === promptId)
}

/**
 * Get all selected prompts from the tree.
 */
export function getSelectedPrompts(treeData: SkillTreeData): Prompt[] {
  return treeData.nodes.filter((n) => n.isSelected).map((n) => n.prompt)
}

/**
 * Calculate camera position to view the entire tree.
 */
export function calculateCameraPosition(
  treeData: SkillTreeData
): [number, number, number] {
  if (treeData.nodes.length === 0) {
    return [0, 5, 10]
  }

  // Find bounding box
  let minX = Infinity,
    maxX = -Infinity
  let minY = Infinity,
    maxY = -Infinity
  let minZ = Infinity,
    maxZ = -Infinity

  treeData.nodes.forEach((node) => {
    const [x, y, z] = node.position
    minX = Math.min(minX, x)
    maxX = Math.max(maxX, x)
    minY = Math.min(minY, y)
    maxY = Math.max(maxY, y)
    minZ = Math.min(minZ, z)
    maxZ = Math.max(maxZ, z)
  })

  // Calculate center and distance
  const centerX = (minX + maxX) / 2
  const centerY = (minY + maxY) / 2
  const centerZ = (minZ + maxZ) / 2

  const width = maxX - minX
  const height = maxY - minY
  const depth = maxZ - minZ

  const maxDimension = Math.max(width, height, depth)
  const distance = maxDimension * 1.5 + 5

  return [centerX, centerY + distance * 0.5, centerZ + distance]
}
