/**
 * Accessory types for agent customization.
 * Accessories are visual items attached to agents.
 */

/** Types of head accessories */
export type HeadAccessoryType = 'none' | 'hat' | 'glasses' | 'crown' | 'headphones' | 'halo'

/** Types of back accessories */
export type BackAccessoryType = 'none' | 'paper' | 'folder' | 'briefcase' | 'backpack'

/** Types of held items */
export type HeldItemType = 'none' | 'book' | 'tool' | 'orb' | 'wand'

/**
 * Head accessory configuration
 */
export interface HeadAccessory {
  type: HeadAccessoryType
  /** Variant name for different styles of same type */
  variant?: string
  /** Custom color override */
  color?: string
}

/**
 * Back accessory configuration.
 */
export interface BackAccessory {
  type: BackAccessoryType
  /** Scale modifier (0.5-1.5) */
  scale?: number
}

/**
 * Held item configuration
 */
export interface HeldAccessory {
  type: HeldItemType
  /** Which hand holds the item */
  hand?: 'left' | 'right' | 'both'
  /** Custom color override */
  color?: string
}

/**
 * Complete accessory configuration for an agent
 */
export interface AgentAccessories {
  head?: HeadAccessory
  back?: BackAccessory
  held?: HeldAccessory
}

/**
 * Agent status indicator types
 */
export type AgentStatusType = 'normal' | 'warning' | 'error' | 'info' | 'thinking' | 'speaking' | 'pending-decision'

/**
 * Agent status configuration
 */
export interface AgentStatus {
  type: AgentStatusType
  /** Optional message for speech bubbles or tooltips */
  message?: string
  /** Auto-hide duration in ms (0 = never) */
  duration?: number
  /** Source of status, for priority resolution when clearing */
  source?: 'heartbeat' | 'user' | 'system' | 'decision'
}

/**
 * Accessory slot position offsets
 */
export interface AccessoryOffset {
  position: [number, number, number]
  rotation: [number, number, number]
  scale: number
}

/** All accessory slot types */
export type AccessorySlot = 'head' | 'back' | 'leftHand' | 'rightHand'

/**
 * Default accessory offsets for each slot relative to agent origin.
 *
 * SlimeAgent anatomy (relative to agent origin at Y=0):
 * - Body sphere: center at [0, 0, 0], radius 0.4, slight Y squash (0.82-0.88)
 * - Top of dome: Y ≈ 0.35
 * - Eyes: at Y=0.1, Z=0.3
 * - Body extends to X/Z ≈ ±0.4
 */
export const ACCESSORY_OFFSETS: Record<AccessorySlot, AccessoryOffset> = {
  // Hat sits on top of dome
  head: {
    position: [0, 0.4, 0],
    rotation: [0, 0, 0],
    scale: 1,
  },
  // Backpack attaches to back of body
  back: {
    position: [0, 0, -0.35],
    rotation: [0, 0, 0],
    scale: 1,
  },
  // Left hand floats beside body
  leftHand: {
    position: [-0.4, 0, 0.1],
    rotation: [0, 0, 0],
    scale: 0.8,
  },
  // Right hand floats beside body
  rightHand: {
    position: [0.4, 0, 0.1],
    rotation: [0, 0, 0],
    scale: 0.8,
  },
}

/**
 * Skill count thresholds for backpack type
 */
export const SKILL_BACKPACK_THRESHOLDS = {
  none: 0,
  paper: 1,
  folder: 3,
  briefcase: 6,
  backpack: 11,
} as const
