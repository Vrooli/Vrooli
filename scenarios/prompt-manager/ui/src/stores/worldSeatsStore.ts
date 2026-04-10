/**
 * World seats store — manages per-furniture-type seat positions.
 * Data persists on disk via the API (store/world-seats.json), not localStorage.
 *
 * `getSeats(type)` is the single source of truth for seat positions.
 */

import { create } from 'zustand'
import { api } from '@/lib/api'
import type { SeatPosition, FurnitureType } from '@/types/furniture'
import type { WorldSeatsConfig } from '@/types/worldSeats'

interface WorldSeatsState {
  seats: WorldSeatsConfig
  loaded: boolean
}

interface WorldSeatsActions {
  fetchSeats: () => Promise<void>
  /** Replace all seats for a furniture type */
  setSeats: (type: FurnitureType, positions: SeatPosition[]) => void
  /** Update a single seat at index */
  updateSeat: (type: FurnitureType, index: number, seat: SeatPosition) => void
  /** Add a seat to a furniture type */
  addSeat: (type: FurnitureType, seat: SeatPosition) => void
  /** Remove a seat by index */
  removeSeat: (type: FurnitureType, index: number) => void
}

type WorldSeatsStore = WorldSeatsState & WorldSeatsActions

let saveTimer: ReturnType<typeof setTimeout> | null = null
const DEBOUNCE_MS = 500

function debouncedSave(config: WorldSeatsConfig) {
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => {
    api.setWorldSeats(config).catch((err: unknown) => {
      console.error('[worldSeatsStore] Failed to save seats:', err)
    })
  }, DEBOUNCE_MS)
}

export const useWorldSeatsStore = create<WorldSeatsStore>()((set, get) => ({
  seats: {},
  loaded: false,

  fetchSeats: async () => {
    if (get().loaded) return
    try {
      const config = await api.getWorldSeats()
      set({ seats: config, loaded: true })
    } catch (err) {
      console.error('[worldSeatsStore] Failed to fetch seats:', err)
      set({ loaded: true })
    }
  },

  setSeats: (type, positions) => {
    const newSeats = { ...get().seats, [type]: positions }
    set({ seats: newSeats })
    debouncedSave(newSeats)
  },

  updateSeat: (type, index, seat) => {
    const current = [...(get().seats[type] ?? [])]
    if (index < 0 || index >= current.length) return
    current[index] = seat
    const newSeats = { ...get().seats, [type]: current }
    set({ seats: newSeats })
    debouncedSave(newSeats)
  },

  addSeat: (type, seat) => {
    const current = [...(get().seats[type] ?? [])]
    current.push(seat)
    const newSeats = { ...get().seats, [type]: current }
    set({ seats: newSeats })
    debouncedSave(newSeats)
  },

  removeSeat: (type, index) => {
    const current = [...(get().seats[type] ?? [])]
    if (index < 0 || index >= current.length) return
    current.splice(index, 1)
    const newSeats = { ...get().seats, [type]: current }
    set({ seats: newSeats })
    debouncedSave(newSeats)
  },
}))

// Auto-fetch on first import
void useWorldSeatsStore.getState().fetchSeats()

/**
 * Get seats for a furniture type — standalone function (not a hook).
 * This is the ONLY way to look up seat positions.
 */
export function getSeats(type: FurnitureType): SeatPosition[] {
  return useWorldSeatsStore.getState().seats[type] ?? []
}
