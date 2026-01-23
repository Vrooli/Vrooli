/**
 * Type definitions for the Skill Tree visualization and avatar system.
 *
 * Note: 3D skill tree types (SkillTreeNode, SkillTreeConnection, SkillTreeData)
 * have been removed. Skill selection now uses the 2D overlay with types defined
 * in skillLayoutService.ts.
 */

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

