/**
 * Accessory component types and interfaces.
 */

import type {
  HeadAccessoryType,
  BackAccessoryType,
  HeldItemType,
  AccessoryOffset,
} from '@/types/accessory'

/**
 * Props for accessory components
 */
export interface AccessoryBaseProps {
  /** Position relative to member */
  position?: [number, number, number]
  /** Rotation in radians */
  rotation?: [number, number, number]
  /** Scale factor */
  scale?: number
  /** Primary color */
  color?: string
  /** Whether to cast shadows */
  castShadow?: boolean
}

/**
 * Props for head accessories
 */
export interface HeadAccessoryProps extends AccessoryBaseProps {
  type: HeadAccessoryType
  variant?: string
}

/**
 * Props for back accessories
 */
export interface BackAccessoryProps extends AccessoryBaseProps {
  type: BackAccessoryType
  /** Number of skills (for visual variation) */
  skillCount?: number
}

/**
 * Props for held items
 */
export interface HeldItemProps extends AccessoryBaseProps {
  type: HeldItemType
  hand?: 'left' | 'right' | 'both'
}

/**
 * Accessory render info for batching
 */
export interface AccessoryRenderInfo {
  type: string
  offset: AccessoryOffset
  color?: string
  metadata?: Record<string, unknown>
}

/**
 * Default accessory offsets
 */
const DEFAULT_OFFSETS: Record<'head' | 'back' | 'leftHand' | 'rightHand', AccessoryOffset> = {
  head: { position: [0, 0.55, 0], rotation: [0, 0, 0], scale: 1 },
  back: { position: [0, -0.2, -0.3], rotation: [0, 0, 0], scale: 1 },
  leftHand: { position: [-0.45, -0.2, 0.1], rotation: [0, 0, 0], scale: 0.8 },
  rightHand: { position: [0.45, -0.2, 0.1], rotation: [0, 0, 0], scale: 0.8 },
}

/**
 * Get default offset for accessory slot
 */
export function getDefaultOffset(
  slot: 'head' | 'back' | 'leftHand' | 'rightHand'
): AccessoryOffset {
  return DEFAULT_OFFSETS[slot]
}
