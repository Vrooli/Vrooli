/**
 * Running agents store - lightweight Zustand store (no persistence)
 * for sharing running agent state across components without prop-drilling.
 */

import { create } from 'zustand'
import type { RunningAgentEntry } from '@/services/heartbeatService'

interface RunningAgentsState {
  agents: RunningAgentEntry[]
  agentMap: Map<string, RunningAgentEntry>
  setAgents: (agents: RunningAgentEntry[]) => void
  isAgentRunning: (agentId: string) => boolean
  getRunningAgent: (agentId: string) => RunningAgentEntry | undefined
}

export const useRunningAgentsStore = create<RunningAgentsState>()((set, get) => ({
  agents: [],
  agentMap: new Map(),

  setAgents: (agents) => {
    const agentMap = new Map<string, RunningAgentEntry>()
    for (const agent of agents) {
      agentMap.set(agent.agentId, agent)
    }
    set({ agents, agentMap })
  },

  isAgentRunning: (agentId) => get().agentMap.has(agentId),
  getRunningAgent: (agentId) => get().agentMap.get(agentId),
}))
