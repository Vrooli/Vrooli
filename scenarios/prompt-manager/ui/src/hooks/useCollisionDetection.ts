/**
 * useCollisionDetection - Hook for AABB collision detection.
 * Checks if objects overlap in 3D space.
 */

import { useMemo, useCallback } from 'react'
import type { BoundingBox, PlacementValidation } from '@/types/objects'

/**
 * Check if two bounding boxes overlap
 */
function boxesOverlap(a: BoundingBox, b: BoundingBox): boolean {
  return (
    a.min[0] <= b.max[0] &&
    a.max[0] >= b.min[0] &&
    a.min[1] <= b.max[1] &&
    a.max[1] >= b.min[1] &&
    a.min[2] <= b.max[2] &&
    a.max[2] >= b.min[2]
  )
}

/**
 * Create a bounding box from position and size
 */
function createBoundingBox(
  position: [number, number, number],
  size: [number, number, number]
): BoundingBox {
  const halfSize: [number, number, number] = [size[0] / 2, size[1] / 2, size[2] / 2]

  return {
    min: [position[0] - halfSize[0], position[1] - halfSize[1], position[2] - halfSize[2]],
    max: [position[0] + halfSize[0], position[1] + halfSize[1], position[2] + halfSize[2]],
  }
}

/**
 * Calculate distance between two points
 */
function distance(
  a: [number, number, number],
  b: [number, number, number]
): number {
  return Math.sqrt(
    (a[0] - b[0]) ** 2 + (a[1] - b[1]) ** 2 + (a[2] - b[2]) ** 2
  )
}

interface CollidableObject {
  id: string
  position: [number, number, number]
  size: [number, number, number]
}

interface CollisionDetectionResult {
  /** Check if a position collides with any object */
  checkCollision: (
    position: [number, number, number],
    size: [number, number, number],
    excludeIds?: string[]
  ) => string[]
  /** Validate placement at a position */
  validatePlacement: (
    position: [number, number, number],
    size: [number, number, number],
    excludeIds?: string[]
  ) => PlacementValidation
  /** Find nearest valid position */
  findValidPosition: (
    position: [number, number, number],
    size: [number, number, number],
    excludeIds?: string[]
  ) => [number, number, number] | null
  /** Get all objects near a position */
  getNearbyObjects: (
    position: [number, number, number],
    radius: number
  ) => CollidableObject[]
}

/**
 * Hook for collision detection among 3D objects.
 *
 * @param objects - Array of objects with position and size
 *
 * @example
 * ```tsx
 * function DraggableObject({ id, initialPosition }) {
 *   const objects = useObjectStore(state => state.getCollidables())
 *   const { validatePlacement } = useCollisionDetection(objects)
 *
 *   const handleDragEnd = (newPos) => {
 *     const result = validatePlacement(newPos, [1, 1, 1], [id])
 *     if (result.valid) {
 *       setPosition(newPos)
 *     } else {
 *       // Snap to suggested position or reject
 *     }
 *   }
 * }
 * ```
 */
export function useCollisionDetection(
  objects: CollidableObject[]
): CollisionDetectionResult {
  // Create bounding boxes for all objects
  const boundingBoxes = useMemo(() => {
    return objects.map((obj) => ({
      id: obj.id,
      box: createBoundingBox(obj.position, obj.size),
    }))
  }, [objects])

  const checkCollision = useCallback(
    (
      position: [number, number, number],
      size: [number, number, number],
      excludeIds: string[] = []
    ): string[] => {
      const testBox = createBoundingBox(position, size)
      const collisions: string[] = []

      for (const { id, box } of boundingBoxes) {
        if (excludeIds.includes(id)) continue
        if (boxesOverlap(testBox, box)) {
          collisions.push(id)
        }
      }

      return collisions
    },
    [boundingBoxes]
  )

  const validatePlacement = useCallback(
    (
      position: [number, number, number],
      size: [number, number, number],
      excludeIds: string[] = []
    ): PlacementValidation => {
      const collisions = checkCollision(position, size, excludeIds)

      if (collisions.length === 0) {
        return { valid: true }
      }

      // Try to find a nearby valid position
      const suggested = findNearbyValidPosition(
        position,
        size,
        boundingBoxes.filter((b) => !excludeIds.includes(b.id)),
        0.5
      )

      return {
        valid: false,
        reason: `Collides with ${collisions.length} object(s)`,
        suggestedPosition: suggested ?? undefined,
      }
    },
    [checkCollision, boundingBoxes]
  )

  const findValidPosition = useCallback(
    (
      position: [number, number, number],
      size: [number, number, number],
      excludeIds: string[] = []
    ): [number, number, number] | null => {
      // First check if current position is valid
      if (checkCollision(position, size, excludeIds).length === 0) {
        return position
      }

      // Search in expanding circles
      return findNearbyValidPosition(
        position,
        size,
        boundingBoxes.filter((b) => !excludeIds.includes(b.id)),
        0.5
      )
    },
    [checkCollision, boundingBoxes]
  )

  const getNearbyObjects = useCallback(
    (
      position: [number, number, number],
      radius: number
    ): CollidableObject[] => {
      return objects.filter(
        (obj) => distance(obj.position, position) <= radius
      )
    },
    [objects]
  )

  return {
    checkCollision,
    validatePlacement,
    findValidPosition,
    getNearbyObjects,
  }
}

/**
 * Find a valid position near the given position
 */
function findNearbyValidPosition(
  position: [number, number, number],
  size: [number, number, number],
  boundingBoxes: { id: string; box: BoundingBox }[],
  step: number
): [number, number, number] | null {
  const maxRadius = 5

  // Search in expanding rings
  for (let radius = step; radius <= maxRadius; radius += step) {
    const angles = Math.ceil((2 * Math.PI * radius) / step)

    for (let i = 0; i < angles; i++) {
      const angle = (i / angles) * 2 * Math.PI
      const testPos: [number, number, number] = [
        position[0] + Math.cos(angle) * radius,
        position[1],
        position[2] + Math.sin(angle) * radius,
      ]

      const testPosBox = createBoundingBox(testPos, size)
      let valid = true

      for (const { box } of boundingBoxes) {
        if (boxesOverlap(testPosBox, box)) {
          valid = false
          break
        }
      }

      if (valid) {
        return testPos
      }
    }
  }

  return null
}

/**
 * Simple hook for checking if a point is inside a bounding box
 */
export function usePointInBox(
  point: [number, number, number],
  boxMin: [number, number, number],
  boxMax: [number, number, number]
): boolean {
  return (
    point[0] >= boxMin[0] &&
    point[0] <= boxMax[0] &&
    point[1] >= boxMin[1] &&
    point[1] <= boxMax[1] &&
    point[2] >= boxMin[2] &&
    point[2] <= boxMax[2]
  )
}
