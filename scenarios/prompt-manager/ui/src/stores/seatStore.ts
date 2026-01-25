/**
 * Seat store for managing furniture seating state.
 * Tracks which members are seated on which furniture.
 */

import { create } from 'zustand'

/** Seat assignment record */
export interface SeatAssignment {
  furnitureId: string
  seatId: string
  memberId: string
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
  /** Members currently transitioning to/from seats */
  transitioning: Set<string>
}

interface SeatActions {
  /** Assign a member to a seat */
  seatMember: (memberId: string, furnitureId: string, seatId: string) => boolean
  /** Remove a member from their seat */
  unseatMember: (memberId: string) => void
  /** Check if a seat is occupied */
  isSeatOccupied: (furnitureId: string, seatId: string) => boolean
  /** Get the member seated at a position */
  getMemberAtSeat: (furnitureId: string, seatId: string) => string | null
  /** Get the seat a member is in */
  getMemberSeat: (memberId: string) => SeatAssignment | null
  /** Get all available seats for a furniture item */
  getAvailableSeats: (furnitureId: string, allSeatIds: string[]) => string[]
  /** Set member as transitioning (moving to/from seat) */
  setTransitioning: (memberId: string, isTransitioning: boolean) => void
  /** Check if a member is currently transitioning */
  isTransitioning: (memberId: string) => boolean
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

  seatMember: (memberId, furnitureId, seatId) => {
    const { assignments, transitioning } = get()

    // Check if member is transitioning
    if (transitioning.has(memberId)) {
      return false
    }

    // Check if seat is already occupied
    const isOccupied = assignments.some(
      (a) => a.furnitureId === furnitureId && a.seatId === seatId
    )
    if (isOccupied) {
      return false
    }

    // Check if member is already seated elsewhere
    const existingSeat = assignments.find((a) => a.memberId === memberId)
    if (existingSeat) {
      // Remove from current seat first
      set({
        assignments: assignments.filter((a) => a.memberId !== memberId),
      })
    }

    // Add new assignment
    set({
      assignments: [
        ...get().assignments,
        {
          furnitureId,
          seatId,
          memberId,
          seatedAt: Date.now(),
        },
      ],
    })

    return true
  },

  unseatMember: (memberId) => {
    set({
      assignments: get().assignments.filter((a) => a.memberId !== memberId),
    })
  },

  isSeatOccupied: (furnitureId, seatId) => {
    return get().assignments.some(
      (a) => a.furnitureId === furnitureId && a.seatId === seatId
    )
  },

  getMemberAtSeat: (furnitureId, seatId) => {
    const assignment = get().assignments.find(
      (a) => a.furnitureId === furnitureId && a.seatId === seatId
    )
    return assignment?.memberId ?? null
  },

  getMemberSeat: (memberId) => {
    return get().assignments.find((a) => a.memberId === memberId) ?? null
  },

  getAvailableSeats: (furnitureId, allSeatIds) => {
    const occupied = get()
      .assignments.filter((a) => a.furnitureId === furnitureId)
      .map((a) => a.seatId)

    return allSeatIds.filter((seatId) => !occupied.includes(seatId))
  },

  setTransitioning: (memberId, isTransitioning) => {
    const { transitioning } = get()
    const newTransitioning = new Set(transitioning)

    if (isTransitioning) {
      newTransitioning.add(memberId)
    } else {
      newTransitioning.delete(memberId)
    }

    set({ transitioning: newTransitioning })
  },

  isTransitioning: (memberId) => {
    return get().transitioning.has(memberId)
  },

  clearFurnitureSeats: (furnitureId) => {
    set({
      assignments: get().assignments.filter((a) => a.furnitureId !== furnitureId),
    })
  },

  reset: () => set(initialState),
}))

/**
 * Selector for getting all seated members
 */
export const selectSeatedMembers = (state: SeatStore) =>
  state.assignments.map((a) => a.memberId)

/**
 * Selector for seat count by furniture
 */
export const selectSeatCountByFurniture = (state: SeatStore, furnitureId: string) =>
  state.assignments.filter((a) => a.furnitureId === furnitureId).length
