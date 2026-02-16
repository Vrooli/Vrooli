/**
 * Environment store for managing scene backgrounds and lighting.
 * Handles environment presets, transitions, and continuous time control.
 */

import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type {
  EnvironmentConfig,
  EnvironmentTransition,
  LightingPreset,
  DreiEnvironmentPreset,
  THEME_ENVIRONMENTS,
} from '@/types/environment'
import { createEnvironmentConfig, getPresetFromTime } from '@/config/environments'

interface EnvironmentState {
  /** Current environment configuration */
  current: EnvironmentConfig
  /** Active drei environment preset name */
  dreiPreset: DreiEnvironmentPreset
  /** Whether environment is transitioning */
  isTransitioning: boolean
  /** Transition progress (0-1) */
  transitionProgress: number
  /** Previous environment for blending during transition */
  previous: EnvironmentConfig | null
  /** Continuous time value (0-24 hours, e.g., 14.5 = 2:30 PM) */
  timeValue: number
  /** Whether to sync time with system clock */
  realTimeMode: boolean
  /** Whether to sync with system theme */
  syncWithTheme: boolean
}

interface EnvironmentActions {
  /** Set the current environment */
  setEnvironment: (config: EnvironmentConfig) => void
  /** Transition to a new environment */
  transitionTo: (config: EnvironmentConfig, transition?: EnvironmentTransition) => void
  /** Set drei environment preset */
  setDreiPreset: (preset: DreiEnvironmentPreset) => void
  /** Sync drei preset with theme */
  syncWithSystemTheme: (theme: 'light' | 'dark') => void
  /** Update transition progress */
  setTransitionProgress: (progress: number) => void
  /** Complete transition */
  completeTransition: () => void
  /** Cancel ongoing transition */
  cancelTransition: () => void
  /** Set continuous time value (0-24 hours) */
  setTimeValue: (hour: number) => void
  /** Toggle real-time mode (sync with system clock) */
  setRealTimeMode: (enabled: boolean) => void
  /** Toggle theme sync */
  setSyncWithTheme: (sync: boolean) => void
  /** Update lighting preset */
  updateLighting: (lighting: Partial<LightingPreset>) => void
  /** Set scene type directly */
  setSceneType: (type: EnvironmentConfig['type']) => void
  /** Reset to defaults */
  reset: () => void
}

type EnvironmentStore = EnvironmentState & EnvironmentActions

/** Default environment configuration (10 AM = morning park) */
const DEFAULT_ENVIRONMENT: EnvironmentConfig = createEnvironmentConfig(
  'default',
  'Default Environment',
  { sceneType: 'outdoor-park', timeValue: 10 }
)

const initialState: EnvironmentState = {
  current: DEFAULT_ENVIRONMENT,
  dreiPreset: 'studio',
  isTransitioning: false,
  transitionProgress: 0,
  previous: null,
  timeValue: 10, // Default to 10 AM (morning)
  realTimeMode: false,
  syncWithTheme: true,
}

/**
 * Zustand store for environment management with persistence
 */
export const useEnvironmentStore = create<EnvironmentStore>()(
  persist(
    (set, get) => ({
      ...initialState,

      setEnvironment: (config) => {
        set({
          current: config,
          isTransitioning: false,
          transitionProgress: 0,
          previous: null,
        })
      },

      transitionTo: (config, transition = { duration: 1, easing: 'easeInOut' }) => {
        const { current } = get()
        set({
          previous: current,
          current: config,
          isTransitioning: true,
          transitionProgress: 0,
        })

        // Note: Actual transition animation should be handled by the
        // EnvironmentSetup component using useFrame
        void transition
      },

      setDreiPreset: (preset) => {
        set({ dreiPreset: preset })
      },

      syncWithSystemTheme: (theme) => {
        if (!get().syncWithTheme) return

        const preset = (
          { dark: 'night', light: 'studio' } as typeof THEME_ENVIRONMENTS
        )[theme]

        set({ dreiPreset: preset })
      },

      setTransitionProgress: (progress) => {
        set({ transitionProgress: Math.min(1, Math.max(0, progress)) })
      },

      completeTransition: () => {
        set({
          isTransitioning: false,
          transitionProgress: 1,
          previous: null,
        })
      },

      cancelTransition: () => {
        const { previous } = get()
        set({
          current: previous ?? get().current,
          isTransitioning: false,
          transitionProgress: 0,
          previous: null,
        })
      },

      setTimeValue: (hour) => {
        // Normalize to 0-24 range
        const normalized = ((hour % 24) + 24) % 24
        set({ timeValue: normalized })
      },

      setRealTimeMode: (enabled) => {
        set({ realTimeMode: enabled })
      },

      setSyncWithTheme: (sync) => {
        set({ syncWithTheme: sync })
      },

      updateLighting: (lighting) => {
        const { current } = get()

        set({
          current: {
            ...current,
            lighting: {
              ...current.lighting,
              ...lighting,
            },
          },
        })
      },

      setSceneType: (type) => {
        const state = get()
        const newEnv = createEnvironmentConfig(
          `${type}-${state.timeValue}`,
          `${type} environment`,
          { sceneType: type, timeValue: state.timeValue }
        )
        set({ current: newEnv })
        if (!state.syncWithTheme) {
          set({ dreiPreset: getPresetFromTime(state.timeValue) })
        }
      },

      reset: () => set(initialState),
    }),
    {
      name: 'scene-environment',
      partialize: (state) => ({
        dreiPreset: state.dreiPreset,
        timeValue: state.timeValue,
        realTimeMode: state.realTimeMode,
        syncWithTheme: state.syncWithTheme,
        sceneType: state.current.type,
      }),
      onRehydrateStorage: () => (state) => {
        // Always rebuild environment config from persisted values so that
        // skybox, fog, and lighting reflect the persisted time and scene type.
        // Previously this only ran when sceneType differed, leaving stale
        // configs (e.g. solid skybox) that ignored the persisted timeValue.
        if (state) {
          const persisted = state as EnvironmentStore & { sceneType?: string }
          const sceneType = (persisted.sceneType ?? state.current.type) as EnvironmentConfig['type']
          state.current = createEnvironmentConfig(
            `${sceneType}-${state.timeValue}`,
            `${sceneType} environment`,
            { sceneType, timeValue: state.timeValue }
          )
        }
      },
    }
  )
)

/**
 * Selector for current lighting config
 */
export const selectCurrentLighting = (state: EnvironmentStore) =>
  state.current.lighting

/**
 * Selector for fog config
 */
export const selectCurrentFog = (state: EnvironmentStore) =>
  state.current.fog ?? null

/**
 * Hook to get the current fog color (useful for background matching).
 */
export function useFogColor(): string {
  const fog = useEnvironmentStore(selectCurrentFog)
  return fog?.color ?? '#0f172a'
}

/**
 * Helper to get drei preset based on continuous time value
 */
export { getPresetFromTime } from '@/config/environments'
