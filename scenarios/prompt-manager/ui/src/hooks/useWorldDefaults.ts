/**
 * useWorldDefaults - Hook to seed the world with default furniture and decorations.
 * Only seeds if the stores are empty (first-time setup).
 */

import { useEffect, useRef } from 'react'
import { useFurnitureStore } from '@/stores/furnitureStore'
import { useDecorationStore } from '@/stores/decorationStore'

/**
 * Default furniture items to seed
 */
const DEFAULT_FURNITURE = [
  { type: 'bench' as const, position: [3, 0, -3] as [number, number, number], rotation: Math.PI / 4 },
  { type: 'chair' as const, position: [-3, 0, 2] as [number, number, number], rotation: -Math.PI / 6 },
  { type: 'coffee-table' as const, position: [0, 0, -4] as [number, number, number], rotation: 0 },
]

/**
 * Default decorations to seed
 */
const DEFAULT_DECORATIONS = [
  { type: 'potted-plant' as const, position: [-4, 0, -4] as [number, number, number] },
  { type: 'tall-plant' as const, position: [4, 0, -3] as [number, number, number] },
  { type: 'floor-lamp' as const, position: [-4, 0, 3] as [number, number, number] },
  { type: 'globe' as const, position: [3, 0, 4] as [number, number, number] },
]

/**
 * Hook to seed the world with default furniture and decorations.
 * Only runs once on initial mount when stores are empty.
 */
export function useWorldDefaults() {
  const furniture = useFurnitureStore((state) => state.furniture)
  const addFurniture = useFurnitureStore((state) => state.addFurniture)

  const decorations = useDecorationStore((state) => state.decorations)
  const addDecoration = useDecorationStore((state) => state.addDecoration)

  const hasSeeded = useRef(false)

  useEffect(() => {
    // Only seed once and only if both stores are empty
    if (hasSeeded.current) return
    if (furniture.length > 0 || decorations.length > 0) {
      hasSeeded.current = true
      return
    }

    hasSeeded.current = true

    // Seed furniture
    for (const item of DEFAULT_FURNITURE) {
      addFurniture(item.type, item.position, item.rotation)
    }

    // Seed decorations
    for (const item of DEFAULT_DECORATIONS) {
      addDecoration(item.type, item.position)
    }
  }, [furniture.length, decorations.length, addFurniture, addDecoration])
}

/**
 * Reset world to defaults by clearing and re-seeding.
 */
export function useResetWorldToDefaults() {
  const resetFurniture = useFurnitureStore((state) => state.reset)
  const resetDecorations = useDecorationStore((state) => state.reset)
  const addFurniture = useFurnitureStore((state) => state.addFurniture)
  const addDecoration = useDecorationStore((state) => state.addDecoration)

  return () => {
    // Clear existing
    resetFurniture()
    resetDecorations()

    // Re-seed
    for (const item of DEFAULT_FURNITURE) {
      addFurniture(item.type, item.position, item.rotation)
    }

    for (const item of DEFAULT_DECORATIONS) {
      addDecoration(item.type, item.position)
    }
  }
}
