/**
 * Decoration store for managing decorative objects in the 3D world.
 */

import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { DecorationInstance, DecorationType } from '@/types/decoration'
import { DEFAULT_DECORATION_COLORS, DECORATION_CONFIGS } from '@/types/decoration'

interface DecorationState {
  /** All decoration instances in the world */
  decorations: DecorationInstance[]
}

interface DecorationActions {
  /** Add new decoration to the world */
  addDecoration: (
    type: DecorationType,
    position: [number, number, number],
    rotation?: number,
    color?: string,
    scale?: number
  ) => string
  /** Remove decoration from the world */
  removeDecoration: (id: string) => void
  /** Move decoration to new position */
  moveDecoration: (id: string, position: [number, number, number]) => void
  /** Rotate decoration */
  rotateDecoration: (id: string, rotation: number) => void
  /** Toggle lamp light */
  toggleLight: (id: string) => void
  /** Get decoration by ID */
  getDecoration: (id: string) => DecorationInstance | undefined
  /** Clear all decorations */
  reset: () => void
}

type DecorationStore = DecorationState & DecorationActions

const initialState: DecorationState = {
  decorations: [],
}

let decorationIdCounter = 0

/**
 * Generate unique decoration ID
 */
function generateDecorationId(): string {
  decorationIdCounter++
  return `decoration-${decorationIdCounter}-${Date.now()}`
}

/**
 * Zustand store for decoration management with persistence
 */
export const useDecorationStore = create<DecorationStore>()(
  persist(
    (set, get) => ({
      ...initialState,

      addDecoration: (type, position, rotation = 0, color, scale = 1) => {
        const id = generateDecorationId()
        const config = DECORATION_CONFIGS[type]

        const newDecoration: DecorationInstance = {
          id,
          type,
          position: [position[0], config.defaultY, position[2]],
          rotation,
          scale,
          color: color ?? DEFAULT_DECORATION_COLORS[type],
          lightOn: config.emitsLight ? true : undefined,
        }

        set((state) => ({
          decorations: [...state.decorations, newDecoration],
        }))

        return id
      },

      removeDecoration: (id) => {
        set((state) => ({
          decorations: state.decorations.filter((d) => d.id !== id),
        }))
      },

      moveDecoration: (id, position) => {
        set((state) => ({
          decorations: state.decorations.map((d) =>
            d.id === id ? { ...d, position } : d
          ),
        }))
      },

      rotateDecoration: (id, rotation) => {
        set((state) => ({
          decorations: state.decorations.map((d) =>
            d.id === id ? { ...d, rotation } : d
          ),
        }))
      },

      toggleLight: (id) => {
        set((state) => ({
          decorations: state.decorations.map((d) =>
            d.id === id && d.lightOn !== undefined
              ? { ...d, lightOn: !d.lightOn }
              : d
          ),
        }))
      },

      getDecoration: (id) => {
        return get().decorations.find((d) => d.id === id)
      },

      reset: () => set(initialState),
    }),
    {
      name: 'world-decorations',
      partialize: (state) => ({
        decorations: state.decorations,
      }),
    }
  )
)

/**
 * Hook to get all decoration instances
 */
export function useDecorationList(): DecorationInstance[] {
  return useDecorationStore((state) => state.decorations)
}

/**
 * Hook to get only light-emitting decorations that are on
 */
export function useActiveLights(): DecorationInstance[] {
  return useDecorationStore((state) =>
    state.decorations.filter((d) => d.lightOn === true)
  )
}

/**
 * Hook to add decoration at a position.
 */
export function useAddDecoration() {
  return useDecorationStore((state) => state.addDecoration)
}

/**
 * Hook to remove decoration.
 */
export function useRemoveDecoration() {
  return useDecorationStore((state) => state.removeDecoration)
}

/**
 * Hook to toggle decoration light.
 */
export function useToggleDecorationLight() {
  return useDecorationStore((state) => state.toggleLight)
}
