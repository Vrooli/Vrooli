/**
 * Tests for the world seats store.
 *
 * Covers fetchSeats, setSeats, updateSeat, addSeat, removeSeat, getSeats,
 * and debounced save behavior.
 */

import { describe, it, expect, beforeEach, afterEach, vi, type Mock, type MockInstance } from 'vitest'

// Mock the API module before importing the store (the store auto-fetches on import).
vi.mock('@/lib/api', () => ({
  api: {
    getWorldSeats: vi.fn().mockResolvedValue({}),
    setWorldSeats: vi.fn().mockResolvedValue({}),
  },
}))

// Must import AFTER the mock is set up.
import { useWorldSeatsStore, getSeats } from './worldSeatsStore'
import { api } from '@/lib/api'

let consoleErrorSpy: MockInstance

function resetStore() {
  useWorldSeatsStore.setState({ seats: {}, loaded: false })
}

describe('worldSeatsStore', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.useFakeTimers()
    resetStore()
  })

  afterEach(() => {
    consoleErrorSpy.mockRestore()
    vi.useRealTimers()
  })

  // ---------------------------------------------------------------------------
  // fetchSeats
  // ---------------------------------------------------------------------------
  describe('fetchSeats', () => {
    it('should load seats from API and set loaded=true', async () => {
      const mockConfig = { chair: [{ position: [0, 0, 0], rotation: 0 }] }
      ;(api.getWorldSeats as Mock).mockResolvedValueOnce(mockConfig)

      await useWorldSeatsStore.getState().fetchSeats()

      expect(api.getWorldSeats).toHaveBeenCalledOnce()
      expect(useWorldSeatsStore.getState().seats).toEqual(mockConfig)
      expect(useWorldSeatsStore.getState().loaded).toBe(true)
    })

    it('should skip re-fetch when already loaded', async () => {
      useWorldSeatsStore.setState({ loaded: true })

      await useWorldSeatsStore.getState().fetchSeats()

      expect(api.getWorldSeats).not.toHaveBeenCalled()
    })

    it('should set loaded=true even on error', async () => {
      ;(api.getWorldSeats as Mock).mockRejectedValueOnce(new Error('network'))

      await useWorldSeatsStore.getState().fetchSeats()

      expect(useWorldSeatsStore.getState().loaded).toBe(true)
      expect(useWorldSeatsStore.getState().seats).toEqual({})
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        '[worldSeatsStore] Failed to fetch seats:',
        expect.any(Error)
      )
    })
  })

  // ---------------------------------------------------------------------------
  // setSeats
  // ---------------------------------------------------------------------------
  describe('setSeats', () => {
    it('should replace seats for a furniture type', () => {
      const positions = [
        { position: [1, 0, 0] as [number, number, number], rotation: 0 },
        { position: [2, 0, 0] as [number, number, number], rotation: 0.5 },
      ]

      useWorldSeatsStore.getState().setSeats('chair', positions)

      expect(useWorldSeatsStore.getState().seats['chair']).toEqual(positions)
    })

    it('should trigger debounced save', () => {
      useWorldSeatsStore.getState().setSeats('chair', [])

      // Not called yet — debounce hasn't fired
      expect(api.setWorldSeats).not.toHaveBeenCalled()

      vi.advanceTimersByTime(500)

      expect(api.setWorldSeats).toHaveBeenCalledOnce()
    })
  })

  // ---------------------------------------------------------------------------
  // updateSeat
  // ---------------------------------------------------------------------------
  describe('updateSeat', () => {
    it('should update a seat at a valid index', () => {
      useWorldSeatsStore.setState({
        seats: {
          chair: [
            { position: [0, 0, 0], rotation: 0 },
            { position: [1, 0, 0], rotation: 0 },
          ],
        },
      })

      const updated = { position: [5, 0, 5] as [number, number, number], rotation: 1 }
      useWorldSeatsStore.getState().updateSeat('chair', 1, updated)

      expect(useWorldSeatsStore.getState().seats['chair']?.[1]).toEqual(updated)
      // First seat unchanged
      expect(useWorldSeatsStore.getState().seats['chair']?.[0]?.position).toEqual([0, 0, 0])
    })

    it('should no-op for out-of-bounds index', () => {
      useWorldSeatsStore.setState({
        seats: { chair: [{ position: [0, 0, 0], rotation: 0 }] },
      })

      useWorldSeatsStore.getState().updateSeat('chair', 5, { position: [1, 1, 1], rotation: 0 })

      expect(useWorldSeatsStore.getState().seats['chair']).toHaveLength(1)
    })

    it('should no-op for negative index', () => {
      useWorldSeatsStore.setState({
        seats: { chair: [{ position: [0, 0, 0], rotation: 0 }] },
      })

      useWorldSeatsStore.getState().updateSeat('chair', -1, { position: [1, 1, 1], rotation: 0 })

      expect(useWorldSeatsStore.getState().seats['chair']).toHaveLength(1)
    })
  })

  // ---------------------------------------------------------------------------
  // addSeat
  // ---------------------------------------------------------------------------
  describe('addSeat', () => {
    it('should append to an existing type', () => {
      useWorldSeatsStore.setState({
        seats: { chair: [{ position: [0, 0, 0], rotation: 0 }] },
      })

      useWorldSeatsStore.getState().addSeat('chair', { position: [1, 0, 0], rotation: 0 })

      expect(useWorldSeatsStore.getState().seats['chair']).toHaveLength(2)
    })

    it('should create entry if type is missing', () => {
      useWorldSeatsStore.getState().addSeat('bench', { position: [0, 0, 0], rotation: 0 })

      expect(useWorldSeatsStore.getState().seats['bench']).toHaveLength(1)
    })
  })

  // ---------------------------------------------------------------------------
  // removeSeat
  // ---------------------------------------------------------------------------
  describe('removeSeat', () => {
    it('should remove a seat at a valid index', () => {
      useWorldSeatsStore.setState({
        seats: {
          chair: [
            { position: [0, 0, 0], rotation: 0 },
            { position: [1, 0, 0], rotation: 0 },
          ],
        },
      })

      useWorldSeatsStore.getState().removeSeat('chair', 0)

      const seats = useWorldSeatsStore.getState().seats['chair']
      expect(seats).toHaveLength(1)
      expect(seats?.[0]?.position).toEqual([1, 0, 0])
    })

    it('should no-op for out-of-bounds index', () => {
      useWorldSeatsStore.setState({
        seats: { chair: [{ position: [0, 0, 0], rotation: 0 }] },
      })

      useWorldSeatsStore.getState().removeSeat('chair', 5)

      expect(useWorldSeatsStore.getState().seats['chair']).toHaveLength(1)
    })
  })

  // ---------------------------------------------------------------------------
  // getSeats (standalone function)
  // ---------------------------------------------------------------------------
  describe('getSeats', () => {
    it('should return seats for a known type', () => {
      useWorldSeatsStore.setState({
        seats: { chair: [{ position: [1, 2, 3], rotation: 0 }] },
      })

      const result = getSeats('chair')

      expect(result).toHaveLength(1)
      expect(result[0]?.position).toEqual([1, 2, 3])
    })

    it('should return empty array for unknown type', () => {
      expect(getSeats('bench')).toEqual([])
    })
  })

  // ---------------------------------------------------------------------------
  // Debounce behavior
  // ---------------------------------------------------------------------------
  describe('debounce', () => {
    it('should batch multiple rapid changes into a single save', () => {
      useWorldSeatsStore.getState().setSeats('chair', [{ position: [1, 0, 0], rotation: 0 }])
      vi.advanceTimersByTime(200)
      useWorldSeatsStore.getState().setSeats('chair', [{ position: [2, 0, 0], rotation: 0 }])
      vi.advanceTimersByTime(200)
      useWorldSeatsStore.getState().setSeats('chair', [{ position: [3, 0, 0], rotation: 0 }])

      // Not yet fired
      expect(api.setWorldSeats).not.toHaveBeenCalled()

      // Let debounce fire
      vi.advanceTimersByTime(500)

      expect(api.setWorldSeats).toHaveBeenCalledOnce()
      // Should be called with the final state
      const savedConfig = (api.setWorldSeats as Mock).mock.calls[0]?.[0]
      expect(savedConfig?.['chair']?.[0]?.position).toEqual([3, 0, 0])
    })

    it('should reset timer on new change', () => {
      useWorldSeatsStore.getState().setSeats('chair', [{ position: [1, 0, 0], rotation: 0 }])
      vi.advanceTimersByTime(400) // 100ms left on first timer

      useWorldSeatsStore.getState().setSeats('bench', [{ position: [0, 0, 0], rotation: 0 }])
      vi.advanceTimersByTime(400) // New timer started; still 100ms left

      expect(api.setWorldSeats).not.toHaveBeenCalled()

      vi.advanceTimersByTime(100) // Now 500ms from last change

      expect(api.setWorldSeats).toHaveBeenCalledOnce()
    })
  })
})
