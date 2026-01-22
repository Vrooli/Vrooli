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
  const combined = [...prompt.modes, ...prompt.tags]

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
 * Build a hierarchical tree structure from modes[].
 * Mode paths like "coding/typescript/debugging" become tree branches.
 */
interface ModeTreeNode {
  mode: string
  fullPath: string
  children: Map<string, ModeTreeNode>
  prompts: Prompt[]
}

function buildModeTree(prompts: Prompt[]): ModeTreeNode {
  const root: ModeTreeNode = {
    mode: 'root',
    fullPath: '',
    children: new Map(),
    prompts: [],
  }

  for (const prompt of prompts) {
    // Use first mode as primary for tree placement
    const primaryMode = prompt.modes[0]
    if (!primaryMode) {
      // Ungrouped prompts go to root
      root.prompts.push(prompt)
      continue
    }

    // Split mode path: "coding/typescript" -> ["coding", "typescript"]
    const modeParts = primaryMode.split('/')
    let currentNode = root
    let currentPath = ''

    for (const part of modeParts) {
      currentPath = currentPath ? `${currentPath}/${part}` : part

      if (!currentNode.children.has(part)) {
        currentNode.children.set(part, {
          mode: part,
          fullPath: currentPath,
          children: new Map(),
          prompts: [],
        })
      }

      const childNode = currentNode.children.get(part)
      if (childNode) {
        currentNode = childNode
      }
    }

    // Add prompt to the deepest mode node
    currentNode.prompts.push(prompt)
  }

  return root
}

/**
 * Calculate positions for a hierarchical tree layout.
 */
function layoutModeTree(
  modeTree: ModeTreeNode,
  nodes: SkillTreeNode[],
  connections: SkillTreeConnection[],
  roots: string[],
  options: LayoutOptions,
  parentId: string | null = null,
  parentPosition: [number, number, number] = [0, 0, 0],
  depth: number = 0,
  angleStart: number = 0,
  angleSpan: number = Math.PI * 2
): number {
  let maxDepthSeen = depth

  // Calculate how many children/prompts we need to position
  const childModes = Array.from(modeTree.children.values())
  const totalItems = childModes.length + modeTree.prompts.length

  if (totalItems === 0) return maxDepthSeen

  let itemIndex = 0

  // Position child mode nodes (category nodes)
  for (const childMode of childModes) {
    const angle = angleStart + (itemIndex / totalItems) * angleSpan
    const radius = options.horizontalSpacing * (depth + 1) * 1.5
    const position: [number, number, number] = [
      parentPosition[0] + Math.cos(angle) * radius,
      depth * options.verticalSpacing,
      parentPosition[2] + Math.sin(angle) * radius,
    ]

    // Create mode node (category node)
    const modeNodeId = `mode-${childMode.fullPath}`
    const modeNode: SkillTreeNode = {
      id: modeNodeId,
      promptId: '', // Mode nodes don't have a prompt
      name: childMode.mode,
      description: `Category: ${childMode.fullPath}`,
      position,
      parentId,
      children: [],
      isSelected: false,
      depth,
      prompt: null as unknown as Prompt, // Mode nodes don't have prompts
      color: CATEGORY_COLORS[childMode.mode.toLowerCase()] ?? CATEGORY_COLORS.default ?? '#64748b',
      size: 0.6, // Mode nodes are slightly larger
      isModeNode: true,
    }

    nodes.push(modeNode)

    if (parentId === null) {
      roots.push(modeNodeId)
    } else {
      // Connect to parent
      const parentNode = nodes.find((n) => n.id === parentId)
      if (parentNode) {
        parentNode.children.push(modeNodeId)
        connections.push({
          id: `conn-${parentId}-${modeNodeId}`,
          sourceId: parentId,
          targetId: modeNodeId,
          source: parentNode.position,
          target: position,
          strength: 0.7,
        })
      }
    }

    // Recursively layout children
    const childAngleSpan = angleSpan / totalItems
    const childDepth = layoutModeTree(
      childMode,
      nodes,
      connections,
      roots,
      options,
      modeNodeId,
      position,
      depth + 1,
      angle - childAngleSpan / 2,
      childAngleSpan
    )
    maxDepthSeen = Math.max(maxDepthSeen, childDepth)

    itemIndex++
  }

  // Position prompt nodes (leaf nodes)
  for (const prompt of modeTree.prompts) {
    const angle = angleStart + (itemIndex / totalItems) * angleSpan
    const radius = options.horizontalSpacing * (depth + 1) * 1.5
    const position: [number, number, number] = [
      parentPosition[0] + Math.cos(angle) * radius,
      depth * options.verticalSpacing,
      parentPosition[2] + Math.sin(angle) * radius,
    ]

    const promptNodeId = `node-${prompt.id}`
    const promptNode: SkillTreeNode = {
      id: promptNodeId,
      promptId: prompt.id,
      name: prompt.name,
      description: prompt.description || '',
      position,
      parentId,
      children: [],
      isSelected: false,
      depth,
      prompt,
      color: getPromptColor(prompt),
      size: getNodeSize(prompt),
    }

    nodes.push(promptNode)

    if (parentId === null) {
      roots.push(promptNodeId)
    } else {
      // Connect to parent
      const parentNode = nodes.find((n) => n.id === parentId)
      if (parentNode) {
        parentNode.children.push(promptNodeId)
        connections.push({
          id: `conn-${parentId}-${promptNodeId}`,
          sourceId: parentId,
          targetId: promptNodeId,
          source: parentNode.position,
          target: position,
          strength: 0.5,
        })
      }
    }

    itemIndex++
  }

  return maxDepthSeen
}

/**
 * Transform an array of prompts into a hierarchical 3D skill tree structure.
 * Uses modes[] to create tree hierarchy (e.g., "coding/typescript" -> coding -> typescript -> prompt)
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

  // Build mode tree from prompts
  const modeTree = buildModeTree(prompts)

  // Layout the tree hierarchically
  const maxDepth = layoutModeTree(
    modeTree,
    nodes,
    connections,
    roots,
    layoutOptions
  )

  return {
    nodes,
    connections,
    roots,
    maxDepth,
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
