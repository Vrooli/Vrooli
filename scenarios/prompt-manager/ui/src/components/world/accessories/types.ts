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
 * GeometricAgent anatomy (relative to agent origin at Y=0):
 * - Head sphere: center at [0, 0.4, 0], radius 0.3 -> top at Y=0.7
 * - Body capsule: center at [0, -0.3, 0], radius 0.25, height 0.5 -> extends from Y=-0.55 to Y=+0.2
 * - Arms: positioned at X=±0.35, Y=-0.1
 *
 * Note: Agent origin is typically at Y=0.8 to place feet on ground.
 */
const DEFAULT_OFFSETS: Record<'head' | 'back' | 'leftHand' | 'rightHand', AccessoryOffset> = {
  // Hat sits on top of head (head top at Y=0.7, add small gap)
  head: { position: [0, 0.75, 0], rotation: [0, 0, 0], scale: 1 },
  // Backpack attaches to back of body (body extends to Z≈-0.25, add small gap)
  back: { position: [0, -0.15, -0.35], rotation: [0, 0, 0], scale: 1 },
  // Left hand position (arm is at X=-0.35, Y=-0.1)
  leftHand: { position: [-0.4, -0.3, 0.15], rotation: [0, 0, 0], scale: 0.8 },
  // Right hand position (arm is at X=0.35, Y=-0.1)
  rightHand: { position: [0.4, -0.3, 0.15], rotation: [0, 0, 0], scale: 0.8 },
}

/**
 * Get default offset for accessory slot
 */
export function getDefaultOffset(
  slot: 'head' | 'back' | 'leftHand' | 'rightHand'
): AccessoryOffset {
  return DEFAULT_OFFSETS[slot]
}
