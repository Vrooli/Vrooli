/**
 * Scene object types for the 3D world.
 * Objects include furniture, decorations, and interactive items.
 */

/** Types of objects in the scene */
export type ObjectType = 'avatar' | 'furniture' | 'decoration' | 'interactive'

/** Furniture subcategories */
export type FurnitureType = 'chair' | 'bench' | 'table' | 'desk' | 'shelf'

/** Decoration subcategories */
export type DecorationType = 'plant' | 'lamp' | 'rug' | 'art' | 'statue'

/**
 * Base scene object interface
 */
export interface SceneObject {
  id: string
  type: ObjectType
  position: [number, number, number]
  rotation: [number, number, number]
  scale: [number, number, number]
  /** Whether the object is visible */
  visible?: boolean
  /** Optional name for display */
  name?: string
  /** Custom metadata */
  metadata?: Record<string, unknown>
}

/**
 * Capabilities that objects can have
 */
export interface ObjectCapabilities {
  /** Can be dragged to new position */
  draggable: boolean
  /** Highlights on hover */
  hoverable: boolean
  /** Can be clicked/selected */
  clickable: boolean
  /** Members can sit on this object */
  sittable: boolean
  /** Participates in collision detection */
  collidable: boolean
  /** Can be deleted by user */
  deletable: boolean
}

/**
 * Furniture object with seating capabilities
 */
export interface FurnitureObject extends SceneObject {
  type: 'furniture'
  furnitureType: FurnitureType
  capabilities: ObjectCapabilities
  /** Seat positions relative to object center */
  seats?: SeatPosition[]
}

/**
 * Seat position within furniture
 */
export interface SeatPosition {
  id: string
  /** Position offset from furniture center */
  offset: [number, number, number]
  /** Rotation when seated */
  rotation: [number, number, number]
  /** ID of member currently seated (null if empty) */
  occupiedBy?: string | null
}

/**
 * Decoration object (non-interactive visual elements)
 */
export interface DecorationObject extends SceneObject {
  type: 'decoration'
  decorationType: DecorationType
  capabilities: Pick<ObjectCapabilities, 'draggable' | 'hoverable' | 'clickable' | 'deletable'>
}

/**
 * Interactive object (triggers actions on click)
 */
export interface InteractiveObject extends SceneObject {
  type: 'interactive'
  /** Action to trigger when clicked */
  action: string
  /** Action parameters */
  actionParams?: Record<string, unknown>
  capabilities: ObjectCapabilities
}

/**
 * AABB (Axis-Aligned Bounding Box) for collision detection
 */
export interface BoundingBox {
  min: [number, number, number]
  max: [number, number, number]
}

/**
 * Object placement validation result
 */
export interface PlacementValidation {
  valid: boolean
  reason?: string
  suggestedPosition?: [number, number, number]
}

/**
 * Default capabilities for each object type
 */
export const DEFAULT_CAPABILITIES: Record<ObjectType, ObjectCapabilities> = {
  avatar: {
    draggable: false,
    hoverable: true,
    clickable: true,
    sittable: false,
    collidable: true,
    deletable: false,
  },
  furniture: {
    draggable: true,
    hoverable: true,
    clickable: true,
    sittable: true,
    collidable: true,
    deletable: true,
  },
  decoration: {
    draggable: true,
    hoverable: true,
    clickable: true,
    sittable: false,
    collidable: false,
    deletable: true,
  },
  interactive: {
    draggable: false,
    hoverable: true,
    clickable: true,
    sittable: false,
    collidable: false,
    deletable: false,
  },
}
