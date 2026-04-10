/**
 * Decoration types for the 3D world.
 * Decorative objects that add visual interest to the scene.
 *
 * DOC: docs/guides/ASSET-GENERATION.md
 */

/** Types of decorations available */
export type DecorationType =
  | 'potted-plant'
  | 'tall-plant'
  | 'cactus'
  | 'flowers'
  | 'oak-tree'
  | 'pine-tree'
  | 'birch-tree'
  | 'floor-lamp'
  | 'desk-lamp'
  | 'hanging-lamp'
  | 'bookshelf'
  | 'rug'
  | 'painting'
  | 'vase'
  | 'globe'
  | 'clock'
  | 'trophy'

/** Light behavior mode for light-emitting decorations */
export type LightMode = 'auto' | 'on' | 'off'

/** Decoration placement in the world */
export interface DecorationInstance {
  id: string
  type: DecorationType
  position: [number, number, number]
  rotation: number
  scale?: number
  color?: string
  /** Light behavior: 'auto' follows day/night, 'on' always on, 'off' always off */
  lightMode?: LightMode
}

/** Decoration configuration */
export interface DecorationConfig {
  type: DecorationType
  displayName: string
  /** Whether this decoration emits light */
  emitsLight: boolean
  /** Whether decoration can be moved */
  movable: boolean
  /** Size for collision/placement */
  size: [number, number, number]
  /** Default Y position (0 = floor level) */
  defaultY: number
}

/**
 * Configuration for each decoration type
 */
export const DECORATION_CONFIGS: Record<DecorationType, DecorationConfig> = {
  'potted-plant': {
    type: 'potted-plant',
    displayName: 'Potted Plant',
    emitsLight: false,
    movable: true,
    size: [0.3, 0.5, 0.3],
    defaultY: 0,
  },
  'tall-plant': {
    type: 'tall-plant',
    displayName: 'Tall Plant',
    emitsLight: false,
    movable: true,
    size: [0.5, 1.2, 0.5],
    defaultY: 0,
  },
  cactus: {
    type: 'cactus',
    displayName: 'Cactus',
    emitsLight: false,
    movable: true,
    size: [0.2, 0.4, 0.2],
    defaultY: 0,
  },
  flowers: {
    type: 'flowers',
    displayName: 'Flowers',
    emitsLight: false,
    movable: true,
    size: [0.25, 0.35, 0.25],
    defaultY: 0,
  },
  'oak-tree': {
    type: 'oak-tree',
    displayName: 'Oak Tree',
    emitsLight: false,
    movable: true,
    size: [3, 4, 3],
    defaultY: 0,
  },
  'pine-tree': {
    type: 'pine-tree',
    displayName: 'Pine Tree',
    emitsLight: false,
    movable: true,
    size: [2.5, 5, 2.5],
    defaultY: 0,
  },
  'birch-tree': {
    type: 'birch-tree',
    displayName: 'Birch Tree',
    emitsLight: false,
    movable: true,
    size: [2, 3.5, 2],
    defaultY: 0,
  },
  'floor-lamp': {
    type: 'floor-lamp',
    displayName: 'Floor Lamp',
    emitsLight: true,
    movable: true,
    size: [0.4, 1.5, 0.4],
    defaultY: 0,
  },
  'desk-lamp': {
    type: 'desk-lamp',
    displayName: 'Desk Lamp',
    emitsLight: true,
    movable: true,
    size: [0.2, 0.4, 0.2],
    defaultY: 0.75,
  },
  'hanging-lamp': {
    type: 'hanging-lamp',
    displayName: 'Hanging Lamp',
    emitsLight: true,
    movable: false,
    size: [0.3, 0.4, 0.3],
    defaultY: 2.5,
  },
  bookshelf: {
    type: 'bookshelf',
    displayName: 'Bookshelf',
    emitsLight: false,
    movable: false,
    size: [0.8, 1.8, 0.3],
    defaultY: 0,
  },
  rug: {
    type: 'rug',
    displayName: 'Rug',
    emitsLight: false,
    movable: true,
    size: [2, 0.02, 1.5],
    defaultY: 0.01,
  },
  painting: {
    type: 'painting',
    displayName: 'Painting',
    emitsLight: false,
    movable: false,
    size: [0.8, 0.6, 0.05],
    defaultY: 1.5,
  },
  vase: {
    type: 'vase',
    displayName: 'Vase',
    emitsLight: false,
    movable: true,
    size: [0.15, 0.3, 0.15],
    defaultY: 0,
  },
  globe: {
    type: 'globe',
    displayName: 'Globe',
    emitsLight: false,
    movable: true,
    size: [0.25, 0.35, 0.25],
    defaultY: 0,
  },
  clock: {
    type: 'clock',
    displayName: 'Wall Clock',
    emitsLight: false,
    movable: false,
    size: [0.3, 0.3, 0.05],
    defaultY: 1.8,
  },
  trophy: {
    type: 'trophy',
    displayName: 'Trophy',
    emitsLight: false,
    movable: true,
    size: [0.1, 0.25, 0.1],
    defaultY: 0,
  },
}

/**
 * Default colors for decorations
 */
export const DEFAULT_DECORATION_COLORS: Partial<Record<DecorationType, string>> = {
  'potted-plant': '#228b22',
  'tall-plant': '#2e8b57',
  cactus: '#3cb371',
  flowers: '#ff69b4',
  'oak-tree': '#2d5a1e',
  'pine-tree': '#1a4d2e',
  'birch-tree': '#5a8f3c',
  'floor-lamp': '#c0c0c0',
  'desk-lamp': '#2f4f4f',
  'hanging-lamp': '#d4af37',
  bookshelf: '#8b4513',
  rug: '#800020',
  vase: '#4169e1',
  globe: '#4682b4',
  trophy: '#ffd700',
}
