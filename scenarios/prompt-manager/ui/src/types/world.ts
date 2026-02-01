/**
 * Type definitions for the World visualization and agent system.
 *
 * Note: 3D skill tree types (SkillTreeNode, SkillTreeConnection, SkillTreeData)
 * have been removed. Skill selection now uses the 2D overlay with types defined
 * in skillLayoutService.ts.
 */

/**
 * Props for agent components.
 */
export interface AgentProps {
  /** Agent position in 3D space */
  position: [number, number, number]
  /** Current cursor/pointer position for look-at behavior */
  cursorPosition: { x: number; y: number } | null
  /** Currently selected node IDs (may be undefined during initialization) */
  selectedNodes?: string[]
  /** Whether the agent is in an animation state */
  isAnimating: boolean
  /** Callback when agent animation completes */
  onAnimationComplete?: () => void
  /** Callback when agent is clicked */
  onAgentClick?: () => void
  /** Agent ID for identification */
  agentId?: string
  /** Custom colors for the agent */
  colors?: {
    body: string
    head: string
    accent: string
  }
  /** Whether the agent is seated on furniture */
  isSeated?: boolean
  /** Rotation when seated (radians) */
  seatRotation?: number
}

/**
 * Agent behavior states.
 */
export type AgentState =
  | 'idle'
  | 'looking'
  | 'waving'
  | 'celebrating'
  | 'thinking'

/**
 * Configuration for agent dependency injection.
 */
export interface AgentConfig {
  /** The agent component to render */
  Component: React.ComponentType<AgentProps>
  /** Optional function to preload agent assets */
  preloadAssets?: () => Promise<void>
  /** Display name for the agent */
  displayName: string
  /** Description of the agent */
  description?: string
}

/**
 * Registry of available agents.
 */
export type AgentRegistry = Record<string, AgentConfig>

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
export type DisplayFormat = 'xml' | 'markdown' | 'json' | 'cli'

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
