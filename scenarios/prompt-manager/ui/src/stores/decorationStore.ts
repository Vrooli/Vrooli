/**
 * Decoration store for managing decorative objects in the 3D world.
 * State is per-scene: each SceneType has its own array of decorations.
 *
 * - `undefined` = scene never visited (seed defaults on first visit)
 * - `[]` = user explicitly cleared the scene
 */

import { useCallback } from 'react'
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { DecorationInstance, DecorationType, LightMode } from '@/types/decoration'
import { DEFAULT_DECORATION_COLORS, DECORATION_CONFIGS } from '@/types/decoration'
import type { SceneType } from '@/types/environment'
import { useEnvironmentStore } from './environmentStore'
import { getSceneDefaults } from '@/config/sceneDefaults'

interface DecorationState {
  /** Per-scene decoration arrays. `undefined` = never visited; `[]` = cleared. */
  scenes: Partial<Record<SceneType, DecorationInstance[]>>
}

interface DecorationActions {
  /** Add new decoration to the active scene */
  addDecoration: (
    type: DecorationType,
    position: [number, number, number],
    rotation?: number,
    color?: string,
    scale?: number
  ) => string
  /** Remove decoration from the active scene */
  removeDecoration: (id: string) => void
  /** Move decoration to new position */
  moveDecoration: (id: string, position: [number, number, number]) => void
  /** Rotate decoration */
  rotateDecoration: (id: string, rotation: number) => void
  /** Set light mode for a decoration */
  setLightMode: (id: string, mode: LightMode) => void
  /** Get decoration by ID from the active scene */
  getDecoration: (id: string) => DecorationInstance | undefined
  /** Clear the active scene's decorations (sets to `[]`). */
  reset: () => void
  /** Clear and re-seed from scene generator. If no type given, uses the active scene. */
  resetToDefaults: (sceneType?: SceneType) => void
  /** Seed a scene with an array of items (used by useWorldDefaults). */
  seedScene: (sceneType: SceneType, items: Omit<DecorationInstance, 'id'>[]) => void
}

type DecorationStore = DecorationState & DecorationActions

const initialState: DecorationState = {
  scenes: {},
}

/** Stable empty array for selector fallback — avoids new references on every evaluation. */
const EMPTY_DECORATIONS: DecorationInstance[] = []

let decorationIdCounter = 0

function generateDecorationId(): string {
  decorationIdCounter++
  return `decoration-${decorationIdCounter}-${Date.now()}`
}

/** Read the active scene type from the environment store. */
function activeScene(): SceneType {
  return useEnvironmentStore.getState().current.type
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
        const scene = activeScene()

        const newDecoration: DecorationInstance = {
          id,
          type,
          position: [position[0], config.defaultY, position[2]],
          rotation,
          scale,
          color: color ?? DEFAULT_DECORATION_COLORS[type],
          lightMode: config.emitsLight ? 'auto' : undefined,
        }

        set((state) => ({
          scenes: {
            ...state.scenes,
            [scene]: [...(state.scenes[scene] ?? []), newDecoration],
          },
        }))

        return id
      },

      removeDecoration: (id) => {
        const scene = activeScene()
        set((state) => ({
          scenes: {
            ...state.scenes,
            [scene]: (state.scenes[scene] ?? []).filter((d) => d.id !== id),
          },
        }))
      },

      moveDecoration: (id, position) => {
        const scene = activeScene()
        set((state) => ({
          scenes: {
            ...state.scenes,
            [scene]: (state.scenes[scene] ?? []).map((d) =>
              d.id === id ? { ...d, position } : d
            ),
          },
        }))
      },

      rotateDecoration: (id, rotation) => {
        const scene = activeScene()
        set((state) => ({
          scenes: {
            ...state.scenes,
            [scene]: (state.scenes[scene] ?? []).map((d) =>
              d.id === id ? { ...d, rotation } : d
            ),
          },
        }))
      },

      setLightMode: (id, mode) => {
        const scene = activeScene()
        set((state) => ({
          scenes: {
            ...state.scenes,
            [scene]: (state.scenes[scene] ?? []).map((d) =>
              d.id === id ? { ...d, lightMode: mode } : d
            ),
          },
        }))
      },

      getDecoration: (id) => {
        const scene = activeScene()
        return (get().scenes[scene] ?? []).find((d) => d.id === id)
      },

      reset: () => {
        const scene = activeScene()
        set((state) => ({
          scenes: { ...state.scenes, [scene]: [] },
        }))
      },

      resetToDefaults: (sceneType?) => {
        const scene = sceneType ?? activeScene()
        const defaults = getSceneDefaults(scene)
        const items: DecorationInstance[] = defaults.decorations.map((d) => ({
          ...d,
          id: generateDecorationId(),
        }))
        set((state) => ({
          scenes: { ...state.scenes, [scene]: items },
        }))
      },

      seedScene: (sceneType, rawItems) => {
        const items: DecorationInstance[] = rawItems.map((d) => ({
          ...d,
          id: generateDecorationId(),
        }))
        set((state) => ({
          scenes: { ...state.scenes, [sceneType]: items },
        }))
      },
    }),
    {
      name: 'world-decorations',
      partialize: (state) => ({
        scenes: state.scenes,
      }),
    }
  )
)

/**
 * Hook to get the decoration list for the current scene.
 * Composes with environmentStore so it re-renders on scene change.
 */
export function useDecorationList(): DecorationInstance[] {
  const sceneType = useEnvironmentStore((s) => s.current.type)
  return useDecorationStore(
    useCallback((state) => state.scenes[sceneType] ?? EMPTY_DECORATIONS, [sceneType])
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
