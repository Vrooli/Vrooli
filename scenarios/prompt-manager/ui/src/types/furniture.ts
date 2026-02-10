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
  /** Seat positions relative to furniture origin */
  seats: SeatPosition[]
  /** Whether furniture can be moved */
  movable: boolean
  /** Size for collision/placement */
  size: [number, number, number]
  /** Whether this furniture emits light (e.g. campfire) */
  emitsLight?: boolean
}

/**
 * Configuration for each furniture type
 * Note: Seat Y positions are absolute world Y values where seated members should be placed.
 * Members at these Y values will appear seated (with their isSeated flag set).
 */
export const FURNITURE_CONFIGS: Record<FurnitureType, FurnitureConfig> = {
  chair: {
    type: 'chair',
    displayName: 'Chair',
    seats: [{ position: [0, 1.1, 0], rotation: 0 }], // Seat height 0.3 + member ground offset 0.8
    movable: true,
    size: [0.5, 0.8, 0.5],
  },
  bench: {
    type: 'bench',
    displayName: 'Bench',
    seats: [
      { position: [-0.4, 1.1, 0], rotation: 0 },
      { position: [0, 1.1, 0], rotation: 0 },
      { position: [0.4, 1.1, 0], rotation: 0 },
    ],
    movable: true,
    size: [1.4, 0.85, 0.5],
  },
  stool: {
    type: 'stool',
    displayName: 'Stool',
    seats: [{ position: [0, 1.2, 0], rotation: 0 }], // Seat height 0.4 + member ground offset 0.8
    movable: true,
    size: [0.3, 0.5, 0.3],
  },
  armchair: {
    type: 'armchair',
    displayName: 'Armchair',
    seats: [{ position: [0, 1.05, 0], rotation: 0 }], // Seat height 0.25 + member ground offset 0.8
    movable: true,
    size: [0.7, 0.9, 0.7],
  },
  desk: {
    type: 'desk',
    displayName: 'Desk',
    seats: [{ position: [0, 0.8, 0.5], rotation: Math.PI }], // Seat in front of desk (standing height)
    movable: true,
    size: [1.2, 0.75, 0.6],
  },
  table: {
    type: 'table',
    displayName: 'Table',
    seats: [
      { position: [0, 0.8, 0.7], rotation: Math.PI },
      { position: [0, 0.8, -0.7], rotation: 0 },
      { position: [0.7, 0.8, 0], rotation: -Math.PI / 2 },
      { position: [-0.7, 0.8, 0], rotation: Math.PI / 2 },
    ],
    movable: true,
    size: [1.0, 0.75, 1.0],
  },
  'picnic-table': {
    type: 'picnic-table',
    displayName: 'Picnic Table',
    seats: [
      { position: [-0.25, 1.10, 0.40], rotation: Math.PI },
      { position: [0.25, 1.10, 0.40], rotation: Math.PI },
      { position: [-0.25, 1.10, -0.40], rotation: 0 },
      { position: [0.25, 1.10, -0.40], rotation: 0 },
    ],
    movable: false,
    size: [1.1, 0.7, 1.0],
  },
  'coffee-table': {
    type: 'coffee-table',
    displayName: 'Coffee Table',
    seats: [], // No seats, just decorative
    movable: true,
    size: [0.8, 0.4, 0.5],
  },
  campfire: {
    type: 'campfire',
    displayName: 'Campfire',
    seats: Array.from({ length: 6 }, (_, i) => {
      const angle = (i / 6) * Math.PI * 2
      return {
        position: [
          Math.cos(angle) * 0.8,
          0.2, // ground-level: body bottom at Y=0 after -0.2 seated offset
          Math.sin(angle) * 0.8,
        ] as [number, number, number],
        rotation: angle + Math.PI, // face center
      }
    }),
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
