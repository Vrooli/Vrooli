/**
 * useMemberData - Data fetching hook for members/agents.
 *
 * This hook provides a backward-compatible Member interface while
 * using the Agent API under the hood. UI components can continue
 * using Member types while the API uses agents.
 *
 * Handles:
 * - Fetching all agents via react-query
 * - Converting Agent responses to Member format
 * - CRUD operations with cache invalidation
 * - Loading and error states
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as agentService from '@/services/agentService'
import type { Member, CreateMemberRequest, UpdateMemberRequest } from '@/types/member'
import { agentToMember, DEFAULT_AGENT_COLORS } from '@/types/agent'

// Query key constants
const QUERY_KEYS = {
  members: ['members'] as const,
}

interface UseMemberDataReturn {
  // Data
  members: Member[]

  // Loading/error states
  isLoading: boolean
  isError: boolean
  error: Error | null

  // Mutations
  createMember: (request: CreateMemberRequest) => Promise<Member>
  updateMember: (id: string, updates: UpdateMemberRequest) => Promise<Member>
  deleteMember: (id: string) => Promise<void>

  // Mutation states
  isCreating: boolean
  isUpdating: boolean
  isDeleting: boolean

  // Utilities
  refetch: () => void
}

/**
 * Hook for fetching and mutating member data.
 * Uses Agent API internally but returns Member format for compatibility.
 */
export function useMemberData(): UseMemberDataReturn {
  const queryClient = useQueryClient()

  // Query for all agents, convert to members
  const {
    data: members = [],
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: QUERY_KEYS.members,
    queryFn: async () => {
      const agents = await agentService.getAgents(true) // Force refresh on query
      return agents.map(agentToMember)
    },
    staleTime: 5000, // Match service cache TTL
  })

  // Create mutation - convert Member request to Agent request
  const createMutation = useMutation({
    mutationFn: async (request: CreateMemberRequest) => {
      const agent = await agentService.createAgent({
        id: request.id,
        displayName: request.name,
        appearance: {
          body: request.bodyColor,
          head: request.headColor,
          accent: request.accentColor,
        },
        skills: request.skills,
      })
      return agentToMember(agent)
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.members })
    },
  })

  // Update mutation - convert Member updates to Agent updates
  const updateMutation = useMutation({
    mutationFn: async ({ id, updates }: { id: string; updates: UpdateMemberRequest }) => {
      const agentUpdates: Parameters<typeof agentService.updateAgent>[1] = {}

      if (updates.name !== undefined) {
        agentUpdates.displayName = updates.name
      }

      // Build appearance if any color is updated
      if (updates.bodyColor !== undefined || updates.headColor !== undefined || updates.accentColor !== undefined) {
        agentUpdates.appearance = {
          body: updates.bodyColor ?? DEFAULT_AGENT_COLORS.body,
          head: updates.headColor ?? DEFAULT_AGENT_COLORS.head,
          accent: updates.accentColor ?? DEFAULT_AGENT_COLORS.accent,
        }
      }

      if (updates.skills !== undefined) {
        agentUpdates.skills = updates.skills
      }

      const agent = await agentService.updateAgent(id, agentUpdates)
      return agentToMember(agent)
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.members })
    },
  })

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => agentService.deleteAgent(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.members })
    },
  })

  return {
    // Data
    members,

    // Loading/error states
    isLoading,
    isError,
    error: error ?? null,

    // Mutations
    createMember: createMutation.mutateAsync,
    updateMember: (id: string, updates: UpdateMemberRequest) =>
      updateMutation.mutateAsync({ id, updates }),
    deleteMember: deleteMutation.mutateAsync,

    // Mutation states
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending,
    isDeleting: deleteMutation.isPending,

    // Utilities
    refetch: () => void refetch(),
  }
}
