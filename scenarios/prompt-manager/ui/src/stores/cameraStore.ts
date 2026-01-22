/**
 * Zustand store for 3D camera control state.
 *
 * Manages camera position, target, mode, and history for smooth transitions.
 */

import { create } from 'zustand'

export type CameraMode = 'freeform' | 'zoomed-avatar' | 'top-down'

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
  focusedAvatarId: string | null

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
  zoomToAvatar: (avatarId: string, position: [number, number, number]) => void
  exitZoom: () => void
  setTopDown: () => void
  setFreeform: () => void

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
  focusedAvatarId: null,
  history: [],
  isAnimating: false,

  setPosition: (position) => set({ position }),
  setTarget: (target) => set({ target }),
  setZoom: (zoom) => set({ zoom }),
  setMode: (mode) => set({ mode }),
  setIsAnimating: (isAnimating) => set({ isAnimating }),

  zoomToAvatar: (avatarId, avatarPosition) => {
    const state = get()

    // Push current state to history
    state.pushHistory()

    // Calculate camera position to focus on avatar
    // Position camera in front of and above avatar
    const cameraOffset: [number, number, number] = [0, 2, 5]
    const newPosition: [number, number, number] = [
      avatarPosition[0] + cameraOffset[0],
      avatarPosition[1] + cameraOffset[1],
      avatarPosition[2] + cameraOffset[2],
    ]

    set({
      position: newPosition,
      target: avatarPosition,
      zoom: 2,
      mode: 'zoomed-avatar',
      focusedAvatarId: avatarId,
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
        focusedAvatarId: null,
        isAnimating: true,
      })
    } else {
      // No history, return to default
      set({
        position: DEFAULT_POSITION,
        target: DEFAULT_TARGET,
        zoom: DEFAULT_ZOOM,
        mode: 'freeform',
        focusedAvatarId: null,
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
      focusedAvatarId: null,
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
      focusedAvatarId: null,
      isAnimating: true,
    })

    setTimeout(() => {
      set({ isAnimating: false })
    }, 1000)
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
