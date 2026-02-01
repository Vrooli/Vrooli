/**
 * Agent Provider - Dependency injection for agent components.
 * Allows easy swapping of agent implementations.
 */
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#dependency-injection-pattern
// DOC: docs/SEAMS.md#1-agentprovider-dependency-injection

/* eslint-disable react-refresh/only-export-components */
import React, { createContext, useContext, useMemo } from 'react'
import type { AgentConfig, AgentRegistry, AgentProps } from '@/types/world'
import { GeometricAgent } from './agents/GeometricAgent'

// Agent registry - add new agents here
const AGENT_REGISTRY: AgentRegistry = {
  geometric: {
    Component: GeometricAgent,
    displayName: 'Geometric Agent',
    description: 'Abstract geometric agent built with Three.js primitives',
  },
  // Future agents can be added here:
  // mixamo: { Component: MixamoAgent, preloadAssets: () => loadModel(), displayName: 'Animated' },
  // rive: { Component: RiveAgent, displayName: '2.5D Character' },
}

// Default agent
const DEFAULT_AGENT = 'geometric'

// Context
const AgentContext = createContext<AgentConfig | null>(null)

interface AgentProviderProps {
  agent?: string
  children: React.ReactNode
}

/**
 * Provider component for agent dependency injection.
 */
export function AgentProvider({ agent = DEFAULT_AGENT, children }: AgentProviderProps) {
  const config = useMemo((): AgentConfig => {
    const selectedConfig = AGENT_REGISTRY[agent]
    if (!selectedConfig) {
      console.warn(`Agent "${agent}" not found, falling back to ${DEFAULT_AGENT}`)
      return AGENT_REGISTRY[DEFAULT_AGENT] as AgentConfig
    }
    return selectedConfig
  }, [agent])

  return <AgentContext.Provider value={config}>{children}</AgentContext.Provider>
}

/**
 * Hook to access the current agent configuration.
 */
export function useAgent(): AgentConfig {
  const context = useContext(AgentContext)
  if (!context) {
    throw new Error('useAgent must be used within an AgentProvider')
  }
  return context
}

/**
 * Hook to get the agent component.
 */
export function useAgentComponent(): React.ComponentType<AgentProps> {
  const { Component } = useAgent()
  return Component
}

/**
 * Get list of available agents.
 */
export function getAvailableAgents(): Array<{ id: string; displayName: string; description?: string }> {
  return Object.entries(AGENT_REGISTRY).map(([id, config]) => ({
    id,
    displayName: config.displayName,
    description: config.description,
  }))
}

/**
 * Register a new agent at runtime.
 */
export function registerAgent(id: string, config: AgentConfig): void {
  if (AGENT_REGISTRY[id]) {
    console.warn(`Agent "${id}" already exists and will be overwritten`)
  }
  AGENT_REGISTRY[id] = config
}

export { AGENT_REGISTRY }
