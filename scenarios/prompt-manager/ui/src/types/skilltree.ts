/**
 * Type definitions for the 3D Skill Tree visualization.
 */

import type { Prompt } from '@/types'

/**
 * Represents a node in the 3D skill tree.
 */
export interface SkillTreeNode {
  id: string
  promptId: string
  name: string
  description: string
  /** 3D position in world space [x, y, z] */
  position: [number, number, number]
  /** Parent node ID for hierarchy */
  parentId: string | null
  /** Child node IDs */
  children: string[]
  /** Selection state */
  isSelected: boolean
  /** Depth level in the tree (0 = root) */
  depth: number
  /** Original prompt data */
  prompt: Prompt
  /** Node color based on category/tags */
  color: string
  /** Node size based on usage/importance */
  size: number
  /** Whether this is a mode/category node rather than a prompt node */
  isModeNode?: boolean
}

/**
 * Connection between two nodes in the skill tree.
 */
export interface SkillTreeConnection {
  id: string
  sourceId: string
  targetId: string
  /** Source position [x, y, z] */
  source: [number, number, number]
  /** Target position [x, y, z] */
  target: [number, number, number]
  /** Connection strength for visual weight */
  strength: number
}

/**
 * Complete skill tree data structure.
 */
export interface SkillTreeData {
  nodes: SkillTreeNode[]
  connections: SkillTreeConnection[]
  /** Root node IDs (nodes without parents) */
  roots: string[]
  /** Maximum depth of the tree */
  maxDepth: number
}

/**
 * Props for avatar components.
 */
export interface AvatarProps {
  /** Avatar position in 3D space */
  position: [number, number, number]
  /** Current cursor/pointer position for look-at behavior */
  cursorPosition: { x: number; y: number } | null
  /** Currently selected node IDs */
  selectedNodes: string[]
  /** Whether the avatar is in an animation state */
  isAnimating: boolean
  /** Callback when avatar animation completes */
  onAnimationComplete?: () => void
  /** Callback when avatar is clicked */
  onAvatarClick?: () => void
  /** Avatar ID for identification */
  avatarId?: string
  /** Custom colors for the avatar */
  colors?: {
    body: string
    head: string
    accent: string
  }
}

/**
 * Avatar behavior states.
 */
export type AvatarState =
  | 'idle'
  | 'looking'
  | 'waving'
  | 'celebrating'
  | 'thinking'

/**
 * Configuration for avatar dependency injection.
 */
export interface AvatarConfig {
  /** The avatar component to render */
  Component: React.ComponentType<AvatarProps>
  /** Optional function to preload avatar assets */
  preloadAssets?: () => Promise<void>
  /** Display name for the avatar */
  displayName: string
  /** Description of the avatar */
  description?: string
}

/**
 * Registry of available avatars.
 */
export type AvatarRegistry = Record<string, AvatarConfig>

/**
 * Selection mode for multi-select behavior.
 */
export type SelectionMode = 'single' | 'multi' | 'toggle'

/**
 * Selection state for prompt combination.
 */
export interface SelectionState {
  selectedIds: string[]
  mode: SelectionMode
  anchorId: string | null
}

/**
 * Combined prompts output format.
 */
export type CombineFormat = 'xml' | 'markdown' | 'json'

/**
 * Request to combine multiple prompts.
 */
export interface CombineRequest {
  promptIds: string[]
  format: CombineFormat
}

/**
 * Response from combining prompts.
 */
export interface CombineResponse {
  combined: string
  promptCount: number
  totalTokens: number
  format: CombineFormat
}

/**
 * Camera state for the skill tree scene.
 */
export interface CameraState {
  position: [number, number, number]
  target: [number, number, number]
  zoom: number
}

/**
 * Options for skill tree layout algorithm.
 */
export interface LayoutOptions {
  /** Horizontal spacing between nodes */
  horizontalSpacing: number
  /** Vertical spacing between depth levels */
  verticalSpacing: number
  /** Radial spread angle for children */
  radialSpread: number
  /** Whether to use 3D depth or keep nodes on a plane */
  use3DDepth: boolean
}

/**
 * Event types for skill tree interactions.
 */
export interface SkillTreeEvents {
  onNodeClick: (nodeId: string, event: MouseEvent) => void
  onNodeHover: (nodeId: string | null) => void
  onSelectionChange: (selectedIds: string[]) => void
  onCombineRequest: (selectedIds: string[]) => void
}
