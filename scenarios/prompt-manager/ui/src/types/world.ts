/**
 * Type definitions for the World visualization and member system.
 *
 * Note: 3D skill tree types (SkillTreeNode, SkillTreeConnection, SkillTreeData)
 * have been removed. Skill selection now uses the 2D overlay with types defined
 * in skillLayoutService.ts.
 */

/**
 * Props for member components.
 */
export interface MemberProps {
  /** Member position in 3D space */
  position: [number, number, number]
  /** Current cursor/pointer position for look-at behavior */
  cursorPosition: { x: number; y: number } | null
  /** Currently selected node IDs (may be undefined during initialization) */
  selectedNodes?: string[]
  /** Whether the member is in an animation state */
  isAnimating: boolean
  /** Callback when member animation completes */
  onAnimationComplete?: () => void
  /** Callback when member is clicked */
  onMemberClick?: () => void
  /** Member ID for identification */
  memberId?: string
  /** Custom colors for the member */
  colors?: {
    body: string
    head: string
    accent: string
  }
}

/**
 * Member behavior states.
 */
export type MemberState =
  | 'idle'
  | 'looking'
  | 'waving'
  | 'celebrating'
  | 'thinking'

/**
 * Configuration for member dependency injection.
 */
export interface MemberConfig {
  /** The member component to render */
  Component: React.ComponentType<MemberProps>
  /** Optional function to preload member assets */
  preloadAssets?: () => Promise<void>
  /** Display name for the member */
  displayName: string
  /** Description of the member */
  description?: string
}

/**
 * Registry of available members.
 */
export type MemberRegistry = Record<string, MemberConfig>

/**
 * Selection mode for multi-select behavior.
 */
export type SelectionMode = 'single' | 'multi' | 'toggle'

/**
 * Selection state for skill combination.
 */
export interface SelectionState {
  selectedIds: string[]
  mode: SelectionMode
  anchorId: string | null
}

/**
 * Combined skills output format.
 */
export type CombineFormat = 'xml' | 'markdown' | 'json'

/**
 * Request to combine multiple skills.
 */
export interface CombineRequest {
  skillIds: string[]
  format: CombineFormat
}

/**
 * Response from combining skills.
 */
export interface CombineResponse {
  combined: string
  skillCount: number
  totalTokens: number
  format: CombineFormat
}
