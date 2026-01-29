/**
 * Furniture store for managing furniture instances in the 3D world.
 * Handles furniture placement, removal, and member seating.
 */

import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { FurnitureInstance, FurnitureType, SeatPosition } from '@/types/furniture'
import { FURNITURE_CONFIGS, DEFAULT_FURNITURE_COLORS } from '@/types/furniture'

interface FurnitureState {
  /** All furniture instances in the world */
  furniture: FurnitureInstance[]
  /** Map of member IDs to their seated furniture ID and seat index */
  seatedMembers: Record<string, { furnitureId: string; seatIndex: number }>
}

interface FurnitureActions {
  /** Add new furniture to the world */
  addFurniture: (
    type: FurnitureType,
    position: [number, number, number],
    rotation?: number,
    color?: string
  ) => string
  /** Remove furniture from the world */
  removeFurniture: (id: string) => void
  /** Move furniture to new position */
  moveFurniture: (id: string, position: [number, number, number]) => void
  /** Rotate furniture */
  rotateFurniture: (id: string, rotation: number) => void
  /** Seat a member at furniture */
  seatMember: (memberId: string, furnitureId: string, seatIndex?: number) => boolean
  /** Unseat a member */
  unseatMember: (memberId: string) => void
  /** Get available seats for furniture */
  getAvailableSeats: (furnitureId: string) => SeatPosition[]
  /** Get seat position for a member (if seated) */
  getMemberSeatPosition: (memberId: string) => { position: [number, number, number]; rotation: number } | null
  /** Check if furniture has available seats */
  hasAvailableSeats: (furnitureId: string) => boolean
  /** Get furniture by ID */
  getFurniture: (id: string) => FurnitureInstance | undefined
  /** Clear all furniture */
  reset: () => void
}

type FurnitureStore = FurnitureState & FurnitureActions

const initialState: FurnitureState = {
  furniture: [],
  seatedMembers: {},
}

let furnitureIdCounter = 0

/**
 * Generate unique furniture ID
 */
function generateFurnitureId(): string {
  furnitureIdCounter++
  return `furniture-${furnitureIdCounter}-${Date.now()}`
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
        const newFurniture: FurnitureInstance = {
          id,
          type,
          position,
          rotation,
          color: color ?? DEFAULT_FURNITURE_COLORS[type],
          occupiedBy: null,
        }

        set((state) => ({
          furniture: [...state.furniture, newFurniture],
        }))

        return id
      },

      removeFurniture: (id) => {
        const { seatedMembers } = get()

        // Unseat any members on this furniture
        const updatedSeated = Object.fromEntries(
          Object.entries(seatedMembers).filter(([, info]) => info.furnitureId !== id)
        )

        set((state) => ({
          furniture: state.furniture.filter((f) => f.id !== id),
          seatedMembers: updatedSeated,
        }))
      },

      moveFurniture: (id, position) => {
        set((state) => ({
          furniture: state.furniture.map((f) =>
            f.id === id ? { ...f, position } : f
          ),
        }))
      },

      rotateFurniture: (id, rotation) => {
        set((state) => ({
          furniture: state.furniture.map((f) =>
            f.id === id ? { ...f, rotation } : f
          ),
        }))
      },

      seatMember: (memberId, furnitureId, seatIndex) => {
        const { furniture, seatedMembers } = get()
        const furn = furniture.find((f) => f.id === furnitureId)
        if (!furn) return false

        const config = FURNITURE_CONFIGS[furn.type]
        if (config.seats.length === 0) return false

        // Find occupied seat indices for this furniture
        const occupiedIndices = new Set(
          Object.values(seatedMembers)
            .filter((info) => info.furnitureId === furnitureId)
            .map((info) => info.seatIndex)
        )

        // Determine which seat to use
        let targetSeatIndex = seatIndex ?? 0
        if (seatIndex === undefined) {
          // Find first available seat
          const availableSeat = config.seats.findIndex(
            (_, idx) => !occupiedIndices.has(idx)
          )
          if (availableSeat === -1) return false
          targetSeatIndex = availableSeat
        } else {
          if (occupiedIndices.has(seatIndex)) return false
        }

        if (targetSeatIndex >= config.seats.length) return false

        // Unseat from previous furniture if any
        const current = seatedMembers[memberId]
        if (current) {
          get().unseatMember(memberId)
        }

        set((state) => ({
          seatedMembers: {
            ...state.seatedMembers,
            [memberId]: { furnitureId, seatIndex: targetSeatIndex },
          },
        }))

        return true
      },

      unseatMember: (memberId) => {
        set((state) => {
          const { [memberId]: _, ...rest } = state.seatedMembers
          void _
          return { seatedMembers: rest }
        })
      },

      getAvailableSeats: (furnitureId) => {
        const { furniture, seatedMembers } = get()
        const furn = furniture.find((f) => f.id === furnitureId)
        if (!furn) return []

        const config = FURNITURE_CONFIGS[furn.type]

        const occupiedIndices = new Set(
          Object.values(seatedMembers)
            .filter((info) => info.furnitureId === furnitureId)
            .map((info) => info.seatIndex)
        )

        return config.seats.filter((_, idx) => !occupiedIndices.has(idx))
      },

      getMemberSeatPosition: (memberId) => {
        const { furniture, seatedMembers } = get()
        const seatInfo = seatedMembers[memberId]
        if (!seatInfo) return null

        const furn = furniture.find((f) => f.id === seatInfo.furnitureId)
        if (!furn) return null

        const config = FURNITURE_CONFIGS[furn.type]
        const seat = config.seats[seatInfo.seatIndex]
        if (!seat) return null

        // Calculate world position (furniture position + rotated seat offset)
        const cos = Math.cos(furn.rotation)
        const sin = Math.sin(furn.rotation)
        const [sx, sy, sz] = seat.position

        return {
          position: [
            furn.position[0] + sx * cos - sz * sin,
            furn.position[1] + sy,
            furn.position[2] + sx * sin + sz * cos,
          ] as [number, number, number],
          rotation: furn.rotation + seat.rotation,
        }
      },

      hasAvailableSeats: (furnitureId) => {
        return get().getAvailableSeats(furnitureId).length > 0
      },

      getFurniture: (id) => {
        return get().furniture.find((f) => f.id === id)
      },

      reset: () => set(initialState),
    }),
    {
      name: 'world-furniture',
      partialize: (state) => ({
        furniture: state.furniture,
        seatedMembers: state.seatedMembers,
      }),
    }
  )
)

/**
 * Hook to get all furniture instances
 */
export function useFurnitureList(): FurnitureInstance[] {
  return useFurnitureStore((state) => state.furniture)
}

/**
 * Hook to check if a member is seated
 */
export function useIsMemberSeated(memberId: string): boolean {
  return useFurnitureStore((state) => memberId in state.seatedMembers)
}
