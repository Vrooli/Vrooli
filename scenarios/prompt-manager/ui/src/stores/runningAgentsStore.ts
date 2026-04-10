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
    const currentAgents = get().agents
    if (currentAgents.length === agents.length) {
      let isSame = true
      for (let i = 0; i < agents.length; i++) {
        const current = currentAgents[i]
        const next = agents[i]
        if (!current || !next) {
          isSame = false
          break
        }
        if (
          current.agentId !== next.agentId ||
          current.teamId !== next.teamId ||
          current.agentName !== next.agentName ||
          current.teamName !== next.teamName ||
          current.runId !== next.runId ||
          current.duration !== next.duration ||
          current.startedAt !== next.startedAt
        ) {
          isSame = false
          break
        }
      }
      if (isSame) {
        return
      }
    }

    const agentMap = new Map<string, RunningAgentEntry>()
    for (const agent of agents) {
      agentMap.set(agent.agentId, agent)
    }
    set({ agents, agentMap })
  },

  isAgentRunning: (agentId) => get().agentMap.has(agentId),
  getRunningAgent: (agentId) => get().agentMap.get(agentId),
}))
