/**
 * Zustand store for 3D camera control state.
 *
 * Manages camera position, target, mode, and history for smooth transitions.
 */
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#camera-system
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#state-management-flow
// DOC: docs/SEAMS.md#2-zustand-stores-state-injection

import { create } from 'zustand'

export type CameraMode = 'freeform' | 'zoomed-member' | 'top-down'

interface CameraHistoryEntry {
  position: [number, number, number]
  target: [number, number, number]
  zoom: number
  mode: CameraMode
}

interface CameraStore {
  // Camera state
  position: [number, number, number]
  target: [number, number, number]
  zoom: number
  mode: CameraMode

  // Focus tracking
  focusedMemberId: string | null

  // History for back navigation
  history: CameraHistoryEntry[]

  // Animation state
  isAnimating: boolean

  // Actions
  setPosition: (position: [number, number, number]) => void
  setTarget: (target: [number, number, number]) => void
  setZoom: (zoom: number) => void
  setMode: (mode: CameraMode) => void
  setIsAnimating: (isAnimating: boolean) => void

  // Compound actions
  zoomToMember: (memberId: string, position: [number, number, number]) => void
  exitZoom: () => void
  setTopDown: () => void
  setFreeform: () => void
  cycleCameraMode: (memberId?: string, memberPosition?: [number, number, number]) => void

  // History management
  pushHistory: () => void
  popHistory: () => CameraHistoryEntry | null
  clearHistory: () => void
}

// Default camera position (top-down view of skill tree)
const DEFAULT_POSITION: [number, number, number] = [0, 15, 15]
const DEFAULT_TARGET: [number, number, number] = [0, 0, 0]
const DEFAULT_ZOOM = 1

// Top-down view settings
const TOP_DOWN_POSITION: [number, number, number] = [0, 20, 0.1]
const TOP_DOWN_TARGET: [number, number, number] = [0, 0, 0]

export const useCameraStore = create<CameraStore>((set, get) => ({
  position: DEFAULT_POSITION,
  target: DEFAULT_TARGET,
  zoom: DEFAULT_ZOOM,
  mode: 'freeform',
  focusedMemberId: null,
  history: [],
  isAnimating: false,

  setPosition: (position) => set({ position }),
  setTarget: (target) => set({ target }),
  setZoom: (zoom) => set({ zoom }),
  setMode: (mode) => set({ mode }),
  setIsAnimating: (isAnimating) => set({ isAnimating }),

  zoomToMember: (memberId, memberPosition) => {
    const state = get()

    // Push current state to history
    state.pushHistory()

    // Calculate camera position to focus on member
    // Position camera in front of and above member
    const cameraOffset: [number, number, number] = [0, 2, 5]
    const newPosition: [number, number, number] = [
      memberPosition[0] + cameraOffset[0],
      memberPosition[1] + cameraOffset[1],
      memberPosition[2] + cameraOffset[2],
    ]

    set({
      position: newPosition,
      target: memberPosition,
      zoom: 2,
      mode: 'zoomed-member',
      focusedMemberId: memberId,
      isAnimating: true,
    })

    // Reset animation flag after transition completes
    setTimeout(() => {
      set({ isAnimating: false })
    }, 1000)
  },

  exitZoom: () => {
    const history = get().popHistory()

    if (history) {
      set({
        position: history.position,
        target: history.target,
        zoom: history.zoom,
        mode: history.mode,
        focusedMemberId: null,
        isAnimating: true,
      })
    } else {
      // No history, return to default
      set({
        position: DEFAULT_POSITION,
        target: DEFAULT_TARGET,
        zoom: DEFAULT_ZOOM,
        mode: 'freeform',
        focusedMemberId: null,
        isAnimating: true,
      })
    }

    setTimeout(() => {
      set({ isAnimating: false })
    }, 1000)
  },

  setTopDown: () => {
    const state = get()
    state.pushHistory()

    set({
      position: TOP_DOWN_POSITION,
      target: TOP_DOWN_TARGET,
      zoom: 1,
      mode: 'top-down',
      focusedMemberId: null,
      isAnimating: true,
    })

    setTimeout(() => {
      set({ isAnimating: false })
    }, 1000)
  },

  setFreeform: () => {
    const state = get()
    state.pushHistory()

    set({
      position: DEFAULT_POSITION,
      target: DEFAULT_TARGET,
      zoom: DEFAULT_ZOOM,
      mode: 'freeform',
      focusedMemberId: null,
      isAnimating: true,
    })

    setTimeout(() => {
      set({ isAnimating: false })
    }, 1000)
  },

  cycleCameraMode: (memberId?: string, memberPosition?: [number, number, number]) => {
    const state = get()
    const currentMode = state.mode

    // Cycle: zoomed-member -> freeform -> top-down -> zoomed-member
    if (currentMode === 'zoomed-member') {
      // Go to freeform
      state.setFreeform()
    } else if (currentMode === 'freeform') {
      // Go to top-down
      state.setTopDown()
    } else {
      // top-down -> zoomed-member (if we have a member to zoom to)
      if (memberId && memberPosition) {
        state.zoomToMember(memberId, memberPosition)
      } else {
        // No member available, go back to freeform
        state.setFreeform()
      }
    }
  },

  pushHistory: () => {
    const state = get()
    const entry: CameraHistoryEntry = {
      position: state.position,
      target: state.target,
      zoom: state.zoom,
      mode: state.mode,
    }

    set({
      history: [...state.history, entry].slice(-10), // Keep last 10 entries
    })
  },

  popHistory: () => {
    const state = get()
    if (state.history.length === 0) return null

    const lastEntry = state.history[state.history.length - 1]
    set({
      history: state.history.slice(0, -1),
    })

    return lastEntry ?? null
  },

  clearHistory: () => set({ history: [] }),
}))
