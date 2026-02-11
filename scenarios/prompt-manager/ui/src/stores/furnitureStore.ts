/**
 * Furniture store for managing furniture instances in the 3D world.
 * State is per-scene: each SceneType has its own furniture and seating maps.
 *
 * - `undefined` = scene never visited (seed defaults on first visit)
 * - `[]` = user explicitly cleared the scene
 */

import { useCallback } from 'react'
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { FurnitureInstance, FurnitureType, SeatPosition } from '@/types/furniture'
import type { LightMode } from '@/types/decoration'
import { DEFAULT_FURNITURE_COLORS } from '@/types/furniture'
import { getSeats } from '@/stores/worldSeatsStore'
import type { SceneType } from '@/types/environment'
import { useEnvironmentStore } from './environmentStore'
import { getSceneDefaults } from '@/config/sceneDefaults'
import type { SceneGeneratorContext } from '@/config/sceneDefaults'

interface FurnitureState {
  /** Per-scene furniture arrays. `undefined` = never visited; `[]` = cleared. */
  scenes: Partial<Record<SceneType, FurnitureInstance[]>>
  /** Per-scene agent seating maps. */
  seatedAgentsByScene: Partial<Record<SceneType, Record<string, { furnitureId: string; seatIndex: number }>>>
}

interface FurnitureActions {
  /** Add new furniture to the active scene */
  addFurniture: (
    type: FurnitureType,
    position: [number, number, number],
    rotation?: number,
    color?: string
  ) => string
  /** Remove furniture from the active scene */
  removeFurniture: (id: string) => void
  /** Move furniture to new position */
  moveFurniture: (id: string, position: [number, number, number]) => void
  /** Rotate furniture */
  rotateFurniture: (id: string, rotation: number) => void
  /** Seat an agent at furniture in the active scene */
  seatAgent: (agentId: string, furnitureId: string, seatIndex?: number) => boolean
  /** Unseat an agent in the active scene */
  unseatAgent: (agentId: string) => void
  /** Get available seats for furniture in the active scene */
  getAvailableSeats: (furnitureId: string) => SeatPosition[]
  /** Get seat position for an agent (if seated) in the active scene */
  getAgentSeatPosition: (agentId: string) => { position: [number, number, number]; rotation: number } | null
  /** Check if furniture has available seats */
  hasAvailableSeats: (furnitureId: string) => boolean
  /** Get furniture by ID from the active scene */
  getFurniture: (id: string) => FurnitureInstance | undefined
  /** Set light mode for light-emitting furniture (e.g. campfire) */
  setLightMode: (id: string, mode: LightMode) => void
  /** Clear the active scene's furniture and seating (sets to `[]`/`{}`). */
  reset: () => void
  /** Clear and re-seed from scene generator. If no type given, uses the active scene. */
  resetToDefaults: (sceneType?: SceneType, ctx?: SceneGeneratorContext) => void
  /** Seed a scene with an array of items (used by useWorldDefaults). */
  seedScene: (sceneType: SceneType, items: Omit<FurnitureInstance, 'id'>[]) => void
}

type FurnitureStore = FurnitureState & FurnitureActions

const initialState: FurnitureState = {
  scenes: {},
  seatedAgentsByScene: {},
}

/** Stable empty array for selector fallback — avoids new references on every evaluation. */
const EMPTY_FURNITURE: FurnitureInstance[] = []

/** Stable empty object for seated agents fallback — avoids new references on every evaluation. */
const EMPTY_SEATED: Record<string, { furnitureId: string; seatIndex: number }> = {}

let furnitureIdCounter = 0

function generateFurnitureId(): string {
  furnitureIdCounter++
  return `furniture-${furnitureIdCounter}-${Date.now()}`
}

/** Read the active scene type from the environment store. */
function activeScene(): SceneType {
  return useEnvironmentStore.getState().current.type
}

/**
 * Zustand store for furniture management with persistence
 */
export const useFurnitureStore = create<FurnitureStore>()(
  persist(
    (set, get) => ({
      ...initialState,

      addFurniture: (type, position, rotation = 0, color) => {
        const id = generateFurnitureId()
        const scene = activeScene()
        const newFurniture: FurnitureInstance = {
          id,
          type,
          position,
          rotation,
          color: color ?? DEFAULT_FURNITURE_COLORS[type],
          occupiedBy: null,
        }

        set((state) => ({
          scenes: {
            ...state.scenes,
            [scene]: [...(state.scenes[scene] ?? []), newFurniture],
          },
        }))

        return id
      },

      removeFurniture: (id) => {
        const scene = activeScene()
        const seatedAgents = get().seatedAgentsByScene[scene] ?? {}

        // Unseat any agents on this furniture
        const updatedSeated = Object.fromEntries(
          Object.entries(seatedAgents).filter(([, info]) => info.furnitureId !== id)
        )

        set((state) => ({
          scenes: {
            ...state.scenes,
            [scene]: (state.scenes[scene] ?? []).filter((f) => f.id !== id),
          },
          seatedAgentsByScene: {
            ...state.seatedAgentsByScene,
            [scene]: updatedSeated,
          },
        }))
      },

      moveFurniture: (id, position) => {
        const scene = activeScene()
        set((state) => ({
          scenes: {
            ...state.scenes,
            [scene]: (state.scenes[scene] ?? []).map((f) =>
              f.id === id ? { ...f, position } : f
            ),
          },
        }))
      },

      rotateFurniture: (id, rotation) => {
        const scene = activeScene()
        set((state) => ({
          scenes: {
            ...state.scenes,
            [scene]: (state.scenes[scene] ?? []).map((f) =>
              f.id === id ? { ...f, rotation } : f
            ),
          },
        }))
      },

      seatAgent: (agentId, furnitureId, seatIndex) => {
        const scene = activeScene()
        const furnitureList = get().scenes[scene] ?? []
        const seatedAgents = get().seatedAgentsByScene[scene] ?? {}
        const furn = furnitureList.find((f) => f.id === furnitureId)
        if (!furn) return false

        const seats = getSeats(furn.type)
        if (seats.length === 0) return false

        // Find occupied seat indices for this furniture
        const occupiedIndices = new Set(
          Object.values(seatedAgents)
            .filter((info) => info.furnitureId === furnitureId)
            .map((info) => info.seatIndex)
        )

        // Determine which seat to use
        let targetSeatIndex = seatIndex ?? 0
        if (seatIndex === undefined) {
          const availableSeat = seats.findIndex(
            (_, idx) => !occupiedIndices.has(idx)
          )
          if (availableSeat === -1) return false
          targetSeatIndex = availableSeat
        } else {
          if (occupiedIndices.has(seatIndex)) return false
        }

        if (targetSeatIndex >= seats.length) return false

        // Unseat from previous furniture if any
        if (seatedAgents[agentId]) {
          get().unseatAgent(agentId)
        }

        set((state) => {
          const current = state.seatedAgentsByScene[scene] ?? {}
          return {
            seatedAgentsByScene: {
              ...state.seatedAgentsByScene,
              [scene]: {
                ...current,
                [agentId]: { furnitureId, seatIndex: targetSeatIndex },
              },
            },
          }
        })

        return true
      },

      unseatAgent: (agentId) => {
        const scene = activeScene()
        set((state) => {
          const current = state.seatedAgentsByScene[scene] ?? {}
          const { [agentId]: _, ...rest } = current
          void _
          return {
            seatedAgentsByScene: {
              ...state.seatedAgentsByScene,
              [scene]: rest,
            },
          }
        })
      },

      getAvailableSeats: (furnitureId) => {
        const scene = activeScene()
        const furnitureList = get().scenes[scene] ?? []
        const seatedAgents = get().seatedAgentsByScene[scene] ?? {}
        const furn = furnitureList.find((f) => f.id === furnitureId)
        if (!furn) return []

        const seats = getSeats(furn.type)
        const occupiedIndices = new Set(
          Object.values(seatedAgents)
            .filter((info) => info.furnitureId === furnitureId)
            .map((info) => info.seatIndex)
        )

        return seats.filter((_, idx) => !occupiedIndices.has(idx))
      },

      getAgentSeatPosition: (agentId) => {
        const scene = activeScene()
        const furnitureList = get().scenes[scene] ?? []
        const seatedAgents = get().seatedAgentsByScene[scene] ?? {}
        const seatInfo = seatedAgents[agentId]
        if (!seatInfo) return null

        const furn = furnitureList.find((f) => f.id === seatInfo.furnitureId)
        if (!furn) return null

        const seats = getSeats(furn.type)
        const seat = seats[seatInfo.seatIndex]
        if (!seat) return null

        // Calculate world position (furniture position + rotated seat offset)
        // Uses Three.js Ry(θ) convention: [[cos,sin],[-sin,cos]] in XZ
        const cos = Math.cos(furn.rotation)
        const sin = Math.sin(furn.rotation)
        const [sx, sy, sz] = seat.position

        return {
          position: [
            furn.position[0] + sx * cos + sz * sin,
            furn.position[1] + sy,
            furn.position[2] - sx * sin + sz * cos,
          ] as [number, number, number],
          rotation: furn.rotation + seat.rotation,
        }
      },

      hasAvailableSeats: (furnitureId) => {
        return get().getAvailableSeats(furnitureId).length > 0
      },

      getFurniture: (id) => {
        const scene = activeScene()
        return (get().scenes[scene] ?? []).find((f) => f.id === id)
      },

      setLightMode: (id, mode) => {
        const scene = activeScene()
        set((state) => ({
          scenes: {
            ...state.scenes,
            [scene]: (state.scenes[scene] ?? []).map((f) =>
              f.id === id ? { ...f, lightMode: mode } : f
            ),
          },
        }))
      },

      reset: () => {
        const scene = activeScene()
        set((state) => ({
          scenes: { ...state.scenes, [scene]: [] },
          seatedAgentsByScene: { ...state.seatedAgentsByScene, [scene]: {} },
        }))
      },

      resetToDefaults: (sceneType?, ctx?) => {
        const scene = sceneType ?? activeScene()
        const defaults = getSceneDefaults(scene, ctx)
        const items: FurnitureInstance[] = defaults.furniture.map((f) => ({
          ...f,
          id: generateFurnitureId(),
        }))
        set((state) => ({
          scenes: { ...state.scenes, [scene]: items },
          seatedAgentsByScene: { ...state.seatedAgentsByScene, [scene]: {} },
        }))
      },

      seedScene: (sceneType, rawItems) => {
        const items: FurnitureInstance[] = rawItems.map((f) => ({
          ...f,
          id: generateFurnitureId(),
        }))
        set((state) => ({
          scenes: { ...state.scenes, [sceneType]: items },
        }))
      },
    }),
    {
      name: 'world-furniture',
      partialize: (state) => ({
        scenes: state.scenes,
        seatedAgentsByScene: state.seatedAgentsByScene,
      }),
    }
  )
)

/**
 * Hook to get the furniture list for the current scene.
 */
export function useFurnitureList(): FurnitureInstance[] {
  const sceneType = useEnvironmentStore((s) => s.current.type)
  return useFurnitureStore(
    useCallback((state) => state.scenes[sceneType] ?? EMPTY_FURNITURE, [sceneType])
  )
}

/**
 * Hook to get the seated agents map for the current scene.
 */
export function useSeatedAgents(): Record<string, { furnitureId: string; seatIndex: number }> {
  const sceneType = useEnvironmentStore((s) => s.current.type)
  return useFurnitureStore(
    useCallback(
      (state) => state.seatedAgentsByScene[sceneType] ?? EMPTY_SEATED,
      [sceneType]
    )
  )
}

/**
 * Hook to add furniture at a position.
 */
export function useAddFurniture() {
  return useFurnitureStore((state) => state.addFurniture)
}

/**
 * Hook to remove furniture.
 */
export function useRemoveFurniture() {
  return useFurnitureStore((state) => state.removeFurniture)
}

/**
 * Hook to check if an agent is seated in the current scene
 */
export function useIsAgentSeated(agentId: string): boolean {
  const sceneType = useEnvironmentStore((s) => s.current.type)
  return useFurnitureStore(
    useCallback(
      (state) => agentId in (state.seatedAgentsByScene[sceneType] ?? EMPTY_SEATED),
      [sceneType, agentId]
    )
  )
}
