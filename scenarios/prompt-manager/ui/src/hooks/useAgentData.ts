/**
 * useAgentData - Data fetching hook for agents.
 *
 * Handles:
 * - Fetching all agents via react-query
 * - CRUD operations with cache invalidation
 * - Loading and error states
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as agentService from '@/services/agentService'
import type { Agent, CreateAgentRequest, UpdateAgentRequest } from '@/types/agent'
import { DEFAULT_AGENT_COLORS } from '@/types/agent'

// Query key constants
const QUERY_KEYS = {
  agents: ['agents'] as const,
}

// Stable empty array to avoid new reference on every render
const EMPTY_AGENTS: Agent[] = []

interface UseAgentDataReturn {
  // Data
  agents: Agent[]

  // Loading/error states
  isLoading: boolean
  isError: boolean
  error: Error | null

  // Mutations
  createAgent: (request: CreateAgentRequest) => Promise<Agent>
  updateAgent: (id: string, updates: UpdateAgentRequest) => Promise<Agent>
  deleteAgent: (id: string) => Promise<void>

  // Mutation states
  isCreating: boolean
  isUpdating: boolean
  isDeleting: boolean

  // Utilities
  refetch: () => void
}

/**
 * Hook for fetching and mutating agent data.
 */
export function useAgentData(): UseAgentDataReturn {
  const queryClient = useQueryClient()

  // Query for all agents
  const {
    data: agents = EMPTY_AGENTS,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: QUERY_KEYS.agents,
    queryFn: async () => {
      return agentService.getAgents(true) // Force refresh on query
    },
    staleTime: 5000, // Match service cache TTL
  })

  // Create mutation
  const createMutation = useMutation({
    mutationFn: async (request: CreateAgentRequest) => {
      return agentService.createAgent({
        id: request.id,
        displayName: request.displayName,
        description: request.description,
        appearance: request.appearance ?? {
          body: DEFAULT_AGENT_COLORS.body,
          head: DEFAULT_AGENT_COLORS.head,
          accent: DEFAULT_AGENT_COLORS.accent,
        },
        capabilities: request.capabilities,
        connectors: request.connectors,
        defaultProfileRef: request.defaultProfileRef,
        heartbeat: request.heartbeat,
        tags: request.tags,
        fileOrder: request.fileOrder,
      })
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.agents })
    },
  })

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: async ({ id, updates }: { id: string; updates: UpdateAgentRequest }) => {
      return agentService.updateAgent(id, updates)
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.agents })
    },
  })

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => agentService.deleteAgent(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.agents })
    },
  })

  return {
    // Data
    agents,

    // Loading/error states
    isLoading,
    isError,
    error: error ?? null,

    // Mutations
    createAgent: createMutation.mutateAsync,
    updateAgent: (id: string, updates: UpdateAgentRequest) =>
      updateMutation.mutateAsync({ id, updates }),
    deleteAgent: deleteMutation.mutateAsync,

    // Mutation states
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending,
    isDeleting: deleteMutation.isPending,

    // Utilities
    refetch: () => void refetch(),
  }
}
