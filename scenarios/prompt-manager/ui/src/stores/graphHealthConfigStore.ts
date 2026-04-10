import { create } from 'zustand'
import type { GraphHealthConfigResponse } from '@/lib/schemas'
import { getGraphHealthConfig, setGraphHealthConfig } from '@/services/graphService'

const FALLBACK_CONFIG: GraphHealthConfigResponse = {
  team: {
    outgoingEdges: 1,
    incomingEdges: 1,
    codeUsage: 0.5,
    recentActivity: 0.5,
    skillContentLength: 0,
    agentContextLoad: 0,
    teamMemberCountBalance: 0.75,
    teamRoleCoverage: 0.75,
  },
  agent: {
    outgoingEdges: 1,
    incomingEdges: 1,
    codeUsage: 0.5,
    recentActivity: 0.5,
    skillContentLength: 0,
    agentContextLoad: 0.75,
    teamMemberCountBalance: 0,
    teamRoleCoverage: 0,
  },
  skill: {
    outgoingEdges: 1,
    incomingEdges: 1,
    codeUsage: 0.5,
    recentActivity: 0.5,
    skillContentLength: 0.75,
    agentContextLoad: 0,
    teamMemberCountBalance: 0,
    teamRoleCoverage: 0,
  },
  cli: { neutralCommands: ['vrooli'], externalToolScore: 0, scenarioFallbackScore: 0 },
}

type EntityKey = 'team' | 'agent' | 'skill'
type WeightKey =
  | 'outgoingEdges'
  | 'incomingEdges'
  | 'codeUsage'
  | 'recentActivity'
  | 'skillContentLength'
  | 'agentContextLoad'
  | 'teamMemberCountBalance'
  | 'teamRoleCoverage'

interface GraphHealthConfigStore {
  config: GraphHealthConfigResponse
  savedConfig: GraphHealthConfigResponse
  dirty: boolean
  loaded: boolean
  loading: boolean
  saving: boolean
  error: string | null

  loadConfig: () => Promise<void>
  setEntityWeight: (entity: EntityKey, key: WeightKey, value: number) => void
  setCLIField: (key: 'externalToolScore' | 'scenarioFallbackScore', value: number) => void
  setNeutralCommandsText: (value: string) => void
  saveConfig: () => Promise<boolean>
  resetToDefault: () => void
}

export const useGraphHealthConfigStore = create<GraphHealthConfigStore>((set, get) => ({
  config: FALLBACK_CONFIG,
  savedConfig: FALLBACK_CONFIG,
  dirty: false,
  loaded: false,
  loading: false,
  saving: false,
  error: null,

  loadConfig: async () => {
    if (get().loaded || get().loading) return
    set({ loading: true, error: null })
    try {
      const loaded = await getGraphHealthConfig()
      set({
        config: loaded ?? FALLBACK_CONFIG,
        savedConfig: loaded ?? FALLBACK_CONFIG,
        dirty: false,
        loaded: true,
        loading: false,
      })
    } catch (error) {
      set({
        error: error instanceof Error ? error.message : 'Failed to load graph health config',
        loaded: true,
        loading: false,
      })
    }
  },

  setEntityWeight: (entity, key, value) => {
    set((state) => {
      const nextConfig = {
        ...state.config,
        [entity]: {
          ...state.config[entity],
          [key]: Number.isFinite(value) ? value : 0,
        },
      }
      return {
        config: nextConfig,
        dirty: !configsEqual(nextConfig, state.savedConfig),
      }
    })
  },

  setCLIField: (key, value) => {
    set((state) => {
      const nextConfig = {
        ...state.config,
        cli: {
          ...state.config.cli,
          [key]: Number.isFinite(value) ? value : 0,
        },
      }
      return {
        config: nextConfig,
        dirty: !configsEqual(nextConfig, state.savedConfig),
      }
    })
  },

  setNeutralCommandsText: (value) => {
    const normalized = value
      .split(',')
      .map((item) => item.trim())
      .filter((item) => item.length > 0)

    set((state) => {
      const nextConfig = {
        ...state.config,
        cli: {
          ...state.config.cli,
          neutralCommands: normalized.length > 0 ? normalized : ['vrooli'],
        },
      }
      return {
        config: nextConfig,
        dirty: !configsEqual(nextConfig, state.savedConfig),
      }
    })
  },

  saveConfig: async () => {
    if (get().saving) return false
    set({ saving: true, error: null })
    try {
      const saved = await setGraphHealthConfig(get().config)
      if (!saved) {
        set({ saving: false, error: 'Health config save returned invalid payload' })
        return false
      }
      set({ config: saved, savedConfig: saved, dirty: false, saving: false })
      return true
    } catch (error) {
      set({
        saving: false,
        error: error instanceof Error ? error.message : 'Failed to save graph health config',
      })
      return false
    }
  },

  resetToDefault: () => {
    set((state) => ({
      config: FALLBACK_CONFIG,
      dirty: !configsEqual(FALLBACK_CONFIG, state.savedConfig),
    }))
  },
}))

function configsEqual(a: GraphHealthConfigResponse, b: GraphHealthConfigResponse): boolean {
  return JSON.stringify(a) === JSON.stringify(b)
}
