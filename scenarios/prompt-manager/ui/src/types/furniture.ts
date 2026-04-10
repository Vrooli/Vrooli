/**
 * Furniture types for the 3D world.
 * Furniture can be placed in the world and members can interact with them.
 */

/** Types of furniture available */
export type FurnitureType =
  | 'chair'
  | 'bench'
  | 'stool'
  | 'armchair'
  | 'desk'
  | 'table'
  | 'picnic-table'
  | 'coffee-table'
  | 'campfire'

/** Furniture placement in the world */
export interface FurnitureInstance {
  id: string
  type: FurnitureType
  position: [number, number, number]
  rotation: number // Y-axis rotation in radians
  color?: string
  /** ID of member currently seated (null if unoccupied) */
  occupiedBy?: string | null
  /** Light behavior for light-emitting furniture (e.g. campfire) */
  lightMode?: import('@/types/decoration').LightMode
}

/** Seat position relative to furniture */
export interface SeatPosition {
  /** Position offset from furniture center */
  position: [number, number, number]
  /** Rotation offset for seated member */
  rotation: number
}

/** Furniture configuration */
export interface FurnitureConfig {
  type: FurnitureType
  displayName: string
  /** Whether furniture can be moved */
  movable: boolean
  /** Size for collision/placement */
  size: [number, number, number]
  /** Whether this furniture emits light (e.g. campfire) */
  emitsLight?: boolean
}

/**
 * Configuration for each furniture type.
 * Seat positions are stored in world-seats.json — use getSeats(type) from worldSeatsStore.
 */
export const FURNITURE_CONFIGS: Record<FurnitureType, FurnitureConfig> = {
  chair: {
    type: 'chair',
    displayName: 'Chair',
    movable: true,
    size: [0.5, 0.8, 0.5],
  },
  bench: {
    type: 'bench',
    displayName: 'Bench',
    movable: true,
    size: [1.4, 0.85, 0.5],
  },
  stool: {
    type: 'stool',
    displayName: 'Stool',
    movable: true,
    size: [0.3, 0.5, 0.3],
  },
  armchair: {
    type: 'armchair',
    displayName: 'Armchair',
    movable: true,
    size: [0.7, 0.9, 0.7],
  },
  desk: {
    type: 'desk',
    displayName: 'Desk',
    movable: true,
    size: [1.2, 0.75, 0.6],
  },
  table: {
    type: 'table',
    displayName: 'Table',
    movable: true,
    size: [1.0, 0.75, 1.0],
  },
  'picnic-table': {
    type: 'picnic-table',
    displayName: 'Picnic Table',
    movable: false,
    size: [1.1, 0.7, 1.0],
  },
  'coffee-table': {
    type: 'coffee-table',
    displayName: 'Coffee Table',
    movable: true,
    size: [0.8, 0.4, 0.5],
  },
  campfire: {
    type: 'campfire',
    displayName: 'Campfire',
    movable: false,
    size: [2, 1.5, 2],
    emitsLight: true,
  },
}

/**
 * Default furniture colors
 */
export const DEFAULT_FURNITURE_COLORS: Record<FurnitureType, string> = {
  chair: '#8B4513',
  bench: '#654321',
  stool: '#A0522D',
  armchair: '#6B4423',
  desk: '#D2691E',
  table: '#8B4513',
  'picnic-table': '#654321',
  'coffee-table': '#A0522D',
  campfire: '#8b4513',
}
