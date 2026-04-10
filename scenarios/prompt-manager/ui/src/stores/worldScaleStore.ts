/**
 * World scale store — manages per-category scale multipliers.
 * Data persists on disk via the API (store/world-scale.json), not localStorage.
 */

import { create } from 'zustand'
import { api } from '@/lib/api'
import type { WorldScaleConfig } from '@/types/worldScale'
import { DEFAULT_WORLD_SCALE } from '@/types/worldScale'

type ScaleKey = keyof WorldScaleConfig

interface WorldScaleState extends WorldScaleConfig {
  loaded: boolean
}

interface WorldScaleActions {
  fetchScales: () => Promise<void>
  setScale: (key: ScaleKey, value: number) => void
  resetAll: () => void
}

type WorldScaleStore = WorldScaleState & WorldScaleActions

let saveTimer: ReturnType<typeof setTimeout> | null = null
const DEBOUNCE_MS = 500

function debouncedSave(config: WorldScaleConfig) {
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => {
    api.setWorldScale(config).catch((err: unknown) => {
      console.error('[worldScaleStore] Failed to save scales:', err)
    })
  }, DEBOUNCE_MS)
}

export const useWorldScaleStore = create<WorldScaleStore>()((set, get) => ({
  ...DEFAULT_WORLD_SCALE,
  loaded: false,

  fetchScales: async () => {
    if (get().loaded) return
    try {
      const config = await api.getWorldScale()
      set({ ...config, loaded: true })
    } catch (err) {
      console.error('[worldScaleStore] Failed to fetch scales:', err)
      set({ loaded: true })
    }
  },

  setScale: (key, value) => {
    set({ [key]: value })
    const state = get()
    debouncedSave({
      agent: state.agent,
      furniture: state.furniture,
      decoration: state.decoration,
      overlay: state.overlay,
    })
  },

  resetAll: () => {
    set({ ...DEFAULT_WORLD_SCALE })
    debouncedSave({ ...DEFAULT_WORLD_SCALE })
  },
}))

// Auto-fetch on first import so scales are applied before the settings popup opens
void useWorldScaleStore.getState().fetchScales()
