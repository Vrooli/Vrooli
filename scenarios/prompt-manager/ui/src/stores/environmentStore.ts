/**
 * Environment store for managing scene backgrounds and lighting.
 * Handles environment presets and transitions.
 */

import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type {
  EnvironmentConfig,
  EnvironmentTransition,
  TimeOfDay,
  LightingPreset,
  DreiEnvironmentPreset,
  THEME_ENVIRONMENTS,
} from '@/types/environment'

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
  /** User's preferred time of day */
  preferredTimeOfDay: TimeOfDay
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
  /** Set preferred time of day */
  setPreferredTimeOfDay: (timeOfDay: TimeOfDay) => void
  /** Toggle theme sync */
  setSyncWithTheme: (sync: boolean) => void
  /** Update lighting preset */
  updateLighting: (lighting: Partial<LightingPreset>) => void
  /** Reset to defaults */
  reset: () => void
}

type EnvironmentStore = EnvironmentState & EnvironmentActions

/** Default environment configuration */
const DEFAULT_ENVIRONMENT: EnvironmentConfig = {
  id: 'default',
  name: 'Default Environment',
  type: 'abstract-space',
  timeOfDay: 'night',
  lighting: {
    ambient: { color: '#404040', intensity: 0.4 },
    directional: [
      {
        position: [10, 10, 5],
        color: '#ffffff',
        intensity: 1,
        castShadow: true,
      },
    ],
    point: [
      { position: [-10, 5, -10], color: '#6366f1', intensity: 0.5 },
      { position: [10, -5, 10], color: '#22d3ee', intensity: 0.3 },
    ],
  },
  fog: { color: '#0f172a', near: 10, far: 50 },
  skybox: { type: 'solid', source: '#0f172a' },
  ground: {
    visible: true,
    type: 'grid',
    size: 30,
    divisions: 30,
    position: 0,
  },
  boundary: {
    visible: true,
    shape: 'square',
    size: 60,
    position: 0.01,
    color: '#94a3b8',
    opacity: 0.4,
  },
  placement: {
    snapToGrid: true,
    snapSize: 1,
    clampToBoundary: true,
  },
}

const initialState: EnvironmentState = {
  current: DEFAULT_ENVIRONMENT,
  dreiPreset: 'night',
  isTransitioning: false,
  transitionProgress: 0,
  previous: null,
  preferredTimeOfDay: 'night',
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

      setPreferredTimeOfDay: (timeOfDay) => {
        set({ preferredTimeOfDay: timeOfDay })
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

      reset: () => set(initialState),
    }),
    {
      name: 'scene-environment',
      partialize: (state) => ({
        dreiPreset: state.dreiPreset,
        preferredTimeOfDay: state.preferredTimeOfDay,
        syncWithTheme: state.syncWithTheme,
      }),
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
 * Helper to get time-of-day based environment preset
 */
export function getTimeOfDayPreset(timeOfDay: TimeOfDay): DreiEnvironmentPreset {
  const presetMap: Record<TimeOfDay, DreiEnvironmentPreset> = {
    morning: 'dawn',
    noon: 'studio',
    sunset: 'sunset',
    night: 'night',
  }
  return presetMap[timeOfDay]
}
