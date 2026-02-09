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
  /** Position relative to agent */
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
 * Default accessory offsets relative to agent origin.
 *
 * SlimeAgent anatomy (body center at Y=0.4 via inner offset group):
 * - Body sphere: center at [0, 0.4, 0], radius 0.4, slight Y squash (0.82-0.88)
 * - Top of dome: Y ≈ 0.75
 * - Eyes: at Y=0.5, Z=0.3
 * - Body extends to X/Z ≈ ±0.4, Y from 0 (ground) to ~0.75
 */
const DEFAULT_OFFSETS: Record<'head' | 'back' | 'leftHand' | 'rightHand', AccessoryOffset> = {
  // Hat sits on top of dome
  head: { position: [0, 0.8, 0], rotation: [0, 0, 0], scale: 1 },
  // Backpack attaches to rear surface
  back: { position: [0, 0.4, -0.35], rotation: [0, 0, 0], scale: 1 },
  // Left hand floats beside body
  leftHand: { position: [-0.4, 0.4, 0.1], rotation: [0, 0, 0], scale: 0.8 },
  // Right hand floats beside body
  rightHand: { position: [0.4, 0.4, 0.1], rotation: [0, 0, 0], scale: 0.8 },
}

/**
 * Get default offset for accessory slot
 */
export function getDefaultOffset(
  slot: 'head' | 'back' | 'leftHand' | 'rightHand'
): AccessoryOffset {
  return DEFAULT_OFFSETS[slot]
}
