/**
 * Seat store for managing furniture seating state.
 * Tracks which agents are seated on which furniture.
 */

import { create } from 'zustand'

/** Seat assignment record */
export interface SeatAssignment {
  furnitureId: string
  seatId: string
  agentId: string
  /** Timestamp when seated */
  seatedAt: number
}

/** Seat availability info */
export interface SeatInfo {
  furnitureId: string
  seatId: string
  isOccupied: boolean
  occupiedBy: string | null
  position: [number, number, number]
  rotation: [number, number, number]
}

interface SeatState {
  /** All current seat assignments */
  assignments: SeatAssignment[]
  /** Agents currently transitioning to/from seats */
  transitioning: Set<string>
}

interface SeatActions {
  /** Assign a agent to a seat */
  seatAgent: (agentId: string, furnitureId: string, seatId: string) => boolean
  /** Remove a agent from their seat */
  unseatAgent: (agentId: string) => void
  /** Check if a seat is occupied */
  isSeatOccupied: (furnitureId: string, seatId: string) => boolean
  /** Get the agent seated at a position */
  getAgentAtSeat: (furnitureId: string, seatId: string) => string | null
  /** Get the seat a agent is in */
  getAgentSeat: (agentId: string) => SeatAssignment | null
  /** Get all available seats for a furniture item */
  getAvailableSeats: (furnitureId: string, allSeatIds: string[]) => string[]
  /** Set agent as transitioning (moving to/from seat) */
  setTransitioning: (agentId: string, isTransitioning: boolean) => void
  /** Check if a agent is currently transitioning */
  isTransitioning: (agentId: string) => boolean
  /** Clear all assignments for a furniture item */
  clearFurnitureSeats: (furnitureId: string) => void
  /** Reset all seating state */
  reset: () => void
}

type SeatStore = SeatState & SeatActions

const initialState: SeatState = {
  assignments: [],
  transitioning: new Set(),
}

/**
 * Zustand store for seating management
 */
export const useSeatStore = create<SeatStore>((set, get) => ({
  ...initialState,

  seatAgent: (agentId, furnitureId, seatId) => {
    const { assignments, transitioning } = get()

    // Check if agent is transitioning
    if (transitioning.has(agentId)) {
      return false
    }

    // Check if seat is already occupied
    const isOccupied = assignments.some(
      (a) => a.furnitureId === furnitureId && a.seatId === seatId
    )
    if (isOccupied) {
      return false
    }

    // Check if agent is already seated elsewhere
    const existingSeat = assignments.find((a) => a.agentId === agentId)
    if (existingSeat) {
      // Remove from current seat first
      set({
        assignments: assignments.filter((a) => a.agentId !== agentId),
      })
    }

    // Add new assignment
    set({
      assignments: [
        ...get().assignments,
        {
          furnitureId,
          seatId,
          agentId,
          seatedAt: Date.now(),
        },
      ],
    })

    return true
  },

  unseatAgent: (agentId) => {
    set({
      assignments: get().assignments.filter((a) => a.agentId !== agentId),
    })
  },

  isSeatOccupied: (furnitureId, seatId) => {
    return get().assignments.some(
      (a) => a.furnitureId === furnitureId && a.seatId === seatId
    )
  },

  getAgentAtSeat: (furnitureId, seatId) => {
    const assignment = get().assignments.find(
      (a) => a.furnitureId === furnitureId && a.seatId === seatId
    )
    return assignment?.agentId ?? null
  },

  getAgentSeat: (agentId) => {
    return get().assignments.find((a) => a.agentId === agentId) ?? null
  },

  getAvailableSeats: (furnitureId, allSeatIds) => {
    const occupied = get()
      .assignments.filter((a) => a.furnitureId === furnitureId)
      .map((a) => a.seatId)

    return allSeatIds.filter((seatId) => !occupied.includes(seatId))
  },

  setTransitioning: (agentId, isTransitioning) => {
    const { transitioning } = get()
    const newTransitioning = new Set(transitioning)

    if (isTransitioning) {
      newTransitioning.add(agentId)
    } else {
      newTransitioning.delete(agentId)
    }

    set({ transitioning: newTransitioning })
  },

  isTransitioning: (agentId) => {
    return get().transitioning.has(agentId)
  },

  clearFurnitureSeats: (furnitureId) => {
    set({
      assignments: get().assignments.filter((a) => a.furnitureId !== furnitureId),
    })
  },

  reset: () => set(initialState),
}))

/**
 * Selector for getting all seated agents
 */
export const selectSeatedAgents = (state: SeatStore) =>
  state.assignments.map((a) => a.agentId)

/**
 * Selector for seat count by furniture
 */
export const selectSeatCountByFurniture = (state: SeatStore, furnitureId: string) =>
  state.assignments.filter((a) => a.furnitureId === furnitureId).length
