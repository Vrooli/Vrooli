/**
 * Accessory types for member customization.
 * Accessories are visual items attached to members.
 */

/** Types of head accessories */
export type HeadAccessoryType = 'none' | 'hat' | 'glasses' | 'crown' | 'headphones' | 'halo'

/** Types of back accessories (auto-computed from skill count) */
export type BackAccessoryType = 'none' | 'paper' | 'folder' | 'briefcase' | 'backpack'

/** Types of held items */
export type HeldItemType = 'none' | 'book' | 'tool' | 'orb' | 'wand'

/** Types of clothing - tops */
export type ClothingTopType = 'none' | 'tshirt' | 'hoodie' | 'jacket' | 'vest' | 'dress'

/** Types of clothing - bottoms */
export type ClothingBottomType = 'none' | 'pants' | 'shorts' | 'skirt'

/** Types of footwear */
export type FootwearType = 'none' | 'shoes' | 'boots' | 'sneakers' | 'sandals'

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
 * Usually auto-computed from skill count.
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
 * Clothing top configuration
 */
export interface ClothingTop {
  type: ClothingTopType
  /** Primary color */
  color?: string
  /** Secondary/accent color for patterns */
  accentColor?: string
}

/**
 * Clothing bottom configuration
 */
export interface ClothingBottom {
  type: ClothingBottomType
  /** Primary color */
  color?: string
}

/**
 * Footwear configuration
 */
export interface Footwear {
  type: FootwearType
  /** Primary color */
  color?: string
}

/**
 * Complete accessory configuration for a member
 */
export interface MemberAccessories {
  head?: HeadAccessory
  back?: BackAccessory
  held?: HeldAccessory
  clothingTop?: ClothingTop
  clothingBottom?: ClothingBottom
  footwear?: Footwear
}

/**
 * Member status indicator types
 */
export type MemberStatusType = 'normal' | 'warning' | 'error' | 'info' | 'thinking' | 'speaking'

/**
 * Member status configuration
 */
export interface MemberStatus {
  type: MemberStatusType
  /** Optional message for speech bubbles or tooltips */
  message?: string
  /** Auto-hide duration in ms (0 = never) */
  duration?: number
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
export type AccessorySlot = 'head' | 'back' | 'leftHand' | 'rightHand' | 'torso' | 'legs' | 'feet'

/**
 * Default accessory offsets for each slot relative to member origin.
 *
 * GeometricMember anatomy (relative to member origin at Y=0):
 * - Head sphere: center at [0, 0.4, 0], radius 0.3 -> top at Y=0.7
 * - Body capsule: center at [0, -0.3, 0], radius 0.25, height 0.5 -> extends from Y=-0.55 to Y=+0.2
 * - Arms: positioned at X=±0.35, Y=-0.1
 *
 * Note: Member origin is typically at Y=0.8 to place feet on ground.
 */
export const ACCESSORY_OFFSETS: Record<AccessorySlot, AccessoryOffset> = {
  // Hat sits on top of head (head top at Y=0.7, add small gap)
  head: {
    position: [0, 0.75, 0],
    rotation: [0, 0, 0],
    scale: 1,
  },
  // Backpack attaches to back of body (body extends to Z≈-0.25, add small gap)
  back: {
    position: [0, -0.15, -0.35],
    rotation: [0, 0, 0],
    scale: 1,
  },
  // Left hand position (arm is at X=-0.35, Y=-0.1)
  leftHand: {
    position: [-0.4, -0.3, 0.15],
    rotation: [0, 0, 0],
    scale: 0.8,
  },
  // Right hand position (arm is at X=0.35, Y=-0.1)
  rightHand: {
    position: [0.4, -0.3, 0.15],
    rotation: [0, 0, 0],
    scale: 0.8,
  },
  // Torso clothing wraps around body (body center at Y=-0.3)
  // ClothingTop adds +0.05 internal offset, so offset of -0.35 puts it at body center
  torso: {
    position: [0, -0.35, 0],
    rotation: [0, 0, 0],
    scale: 1,
  },
  // Pants/shorts - ClothingBottom adds +0.25 for waist position
  // Body bottom is at Y=-0.8, waist should be around Y=-0.5
  // offset = -0.5 - 0.25 = -0.75
  legs: {
    position: [0, -0.75, 0],
    rotation: [0, 0, 0],
    scale: 1,
  },
  // Shoes at the bottom (member bottom at Y=-0.8)
  // FootwearAccessory geometry base is at Y=0 relative to offset
  feet: {
    position: [0, -0.8, 0],
    rotation: [0, 0, 0],
    scale: 1,
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
