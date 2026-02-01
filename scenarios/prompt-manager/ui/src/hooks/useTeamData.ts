/**
 * useTeamData - Data fetching hook for teams.
 *
 * Handles:
 * - Fetching all teams via react-query
 * - Fetching individual team details
 * - CRUD operations with cache invalidation
 * - Member operations (add, update, remove)
 * - Role operations
 * - Loading and error states
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as teamService from '@/services/teamService'
import type {
  Team,
  TeamDetails,
  TeamRole,
  TeamMember,
  CreateTeamRequest,
  UpdateTeamRequest,
  AddMemberRequest,
  UpdateMemberRequest,
} from '@/types/team'

// Query key constants
const QUERY_KEYS = {
  teams: ['teams'] as const,
  teamDetails: (id: string) => ['teams', id] as const,
}

interface UseTeamDataReturn {
  // Data
  teams: Team[]

  // Loading/error states
  isLoading: boolean
  isError: boolean
  error: Error | null

  // Team CRUD mutations
  createTeam: (request: CreateTeamRequest) => Promise<TeamDetails>
  updateTeam: (id: string, updates: UpdateTeamRequest) => Promise<TeamDetails>
  deleteTeam: (id: string) => Promise<void>

  // Member mutations
  addMember: (teamId: string, request: AddMemberRequest) => Promise<TeamMember>
  updateMember: (teamId: string, agentId: string, request: UpdateMemberRequest) => Promise<TeamMember>
  removeMember: (teamId: string, agentId: string) => Promise<void>

  // Role mutations
  setRoles: (teamId: string, roles: TeamRole[]) => Promise<TeamRole[]>

  // Mutation states
  isCreating: boolean
  isUpdating: boolean
  isDeleting: boolean

  // Utilities
  refetch: () => void
}

/**
 * Hook for fetching and mutating team data.
 */
export function useTeamData(): UseTeamDataReturn {
  const queryClient = useQueryClient()

  // Query for all teams
  const {
    data: teams = [],
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: QUERY_KEYS.teams,
    queryFn: async () => {
      return teamService.getTeams(true) // Force refresh on query
    },
    staleTime: 5000, // Match service cache TTL
  })

  // Create mutation
  const createMutation = useMutation({
    mutationFn: async (request: CreateTeamRequest) => {
      return teamService.createTeam(request)
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.teams })
    },
  })

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: async ({ id, updates }: { id: string; updates: UpdateTeamRequest }) => {
      return teamService.updateTeam(id, updates)
    },
    onSuccess: (_, { id }) => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.teams })
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.teamDetails(id) })
    },
  })

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => teamService.deleteTeam(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.teams })
    },
  })

  // Add member mutation
  const addMemberMutation = useMutation({
    mutationFn: async ({ teamId, request }: { teamId: string; request: AddMemberRequest }) => {
      return teamService.addTeamMember(teamId, request)
    },
    onSuccess: (_, { teamId }) => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.teams })
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.teamDetails(teamId) })
    },
  })

  // Update member mutation
  const updateMemberMutation = useMutation({
    mutationFn: async ({
      teamId,
      agentId,
      request,
    }: {
      teamId: string
      agentId: string
      request: UpdateMemberRequest
    }) => {
      return teamService.updateTeamMember(teamId, agentId, request)
    },
    onSuccess: (_, { teamId }) => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.teamDetails(teamId) })
    },
  })

  // Remove member mutation
  const removeMemberMutation = useMutation({
    mutationFn: async ({ teamId, agentId }: { teamId: string; agentId: string }) => {
      return teamService.removeTeamMember(teamId, agentId)
    },
    onSuccess: (_, { teamId }) => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.teams })
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.teamDetails(teamId) })
    },
  })

  // Set roles mutation
  const setRolesMutation = useMutation({
    mutationFn: async ({ teamId, roles }: { teamId: string; roles: TeamRole[] }) => {
      return teamService.setTeamRoles(teamId, roles)
    },
    onSuccess: (_, { teamId }) => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.teamDetails(teamId) })
    },
  })

  return {
    // Data
    teams,

    // Loading/error states
    isLoading,
    isError,
    error: error ?? null,

    // Team CRUD mutations
    createTeam: createMutation.mutateAsync,
    updateTeam: (id: string, updates: UpdateTeamRequest) =>
      updateMutation.mutateAsync({ id, updates }),
    deleteTeam: deleteMutation.mutateAsync,

    // Member mutations
    addMember: (teamId: string, request: AddMemberRequest) =>
      addMemberMutation.mutateAsync({ teamId, request }),
    updateMember: (teamId: string, agentId: string, request: UpdateMemberRequest) =>
      updateMemberMutation.mutateAsync({ teamId, agentId, request }),
    removeMember: (teamId: string, agentId: string) =>
      removeMemberMutation.mutateAsync({ teamId, agentId }),

    // Role mutations
    setRoles: (teamId: string, roles: TeamRole[]) =>
      setRolesMutation.mutateAsync({ teamId, roles }),

    // Mutation states
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending,
    isDeleting: deleteMutation.isPending,

    // Utilities
    refetch: () => void refetch(),
  }
}

interface UseTeamDetailsReturn {
  // Data
  team: TeamDetails | undefined

  // Loading/error states
  isLoading: boolean
  isError: boolean
  error: Error | null

  // Utilities
  refetch: () => void
}

/**
 * Hook for fetching a single team's details.
 */
export function useTeamDetails(teamId: string | null): UseTeamDetailsReturn {
  const {
    data: team,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: QUERY_KEYS.teamDetails(teamId ?? ''),
    queryFn: async () => {
      if (!teamId) return undefined
      return teamService.getTeam(teamId)
    },
    enabled: !!teamId,
    staleTime: 5000,
  })

  return {
    team,
    isLoading,
    isError,
    error: error ?? null,
    refetch: () => void refetch(),
  }
}
