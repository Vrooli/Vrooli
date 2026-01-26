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
  /** Whether the member is seated on furniture */
  isSeated?: boolean
  /** Rotation when seated (radians) */
  seatRotation?: number
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
 * Selection state for skill display.
 */
export interface SelectionState {
  selectedIds: string[]
  mode: SelectionMode
  anchorId: string | null
}

/**
 * Displayed skills output format.
 */
export type DisplayFormat = 'xml' | 'markdown' | 'json'

/**
 * Request to display multiple skills.
 */
export interface DisplayRequest {
  identifiers: string[]
  format: DisplayFormat
}

/**
 * Response from displaying skills.
 */
export interface DisplayResponse {
  combined: string
  skillCount: number
  totalTokens: number
  format: DisplayFormat
}
