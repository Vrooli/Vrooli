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
 * Default accessory offsets for each slot
 */
export const ACCESSORY_OFFSETS: Record<AccessorySlot, AccessoryOffset> = {
  head: {
    position: [0, 0.55, 0],
    rotation: [0, 0, 0],
    scale: 1,
  },
  back: {
    position: [0, -0.2, -0.3],
    rotation: [0, 0, 0],
    scale: 1,
  },
  leftHand: {
    position: [-0.45, -0.2, 0.1],
    rotation: [0, 0, 0],
    scale: 0.8,
  },
  rightHand: {
    position: [0.45, -0.2, 0.1],
    rotation: [0, 0, 0],
    scale: 0.8,
  },
  torso: {
    position: [0, -0.3, 0],
    rotation: [0, 0, 0],
    scale: 1,
  },
  legs: {
    position: [0, -0.7, 0],
    rotation: [0, 0, 0],
    scale: 1,
  },
  feet: {
    position: [0, -1.0, 0],
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
