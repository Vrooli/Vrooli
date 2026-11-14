/**
 * useSkillsData - Data fetching hook for skills.
 *
 * Handles:
 * - Fetching all skills via react-query
 * - CRUD operations with cache invalidation
 * - Loading and error states
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as skillService from '@/services/skillService'
import { useGraphStore } from '@/stores/graphStore'
import type { Skill, CreateSkillRequest, UpdateSkillRequest } from '@/types'

// Query key constants
const QUERY_KEYS = {
  skills: ['skills'] as const,
}

// Stable empty array to avoid new reference on every render
const EMPTY_SKILLS: Skill[] = []

interface UseSkillsDataReturn {
  // Data
  skills: Skill[]

  // Loading/error states
  isLoading: boolean
  isError: boolean
  error: Error | null

  // Mutations
  createSkill: (request: CreateSkillRequest) => Promise<Skill>
  updateSkill: (id: string, updates: UpdateSkillRequest) => Promise<Skill>
  updateSkills: (updates: Map<string, UpdateSkillRequest>) => Promise<Map<string, Skill | Error>>
  deleteSkill: (id: string) => Promise<void>

  // Mutation states
  isCreating: boolean
  isUpdating: boolean
  isDeleting: boolean

  // Utilities
  refetch: () => void
}

/**
 * Hook for fetching and mutating skills data.
 */
export function useSkillsData(): UseSkillsDataReturn {
  const queryClient = useQueryClient()

  // Query for all skills
  const {
    data: skills = EMPTY_SKILLS,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: QUERY_KEYS.skills,
    queryFn: () => skillService.getSkills(),
    staleTime: 5000, // Match service cache TTL
  })

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (request: CreateSkillRequest) => skillService.createSkill(request),
    onSuccess: async (createdSkill) => {
      // Reconcile the server list first, then commit the mutation response as
      // the read-after-write authority. A newly created skill can otherwise
      // disappear between the create response and a list projection refresh,
      // leaving callers that immediately navigate to its ID on an empty route.
      await queryClient.invalidateQueries({ queryKey: QUERY_KEYS.skills })
      queryClient.setQueryData<Skill[]>(QUERY_KEYS.skills, (current = []) => {
        const withoutCreated = current.filter((skill) => skill.id !== createdSkill.id)
        return [...withoutCreated, createdSkill]
      })
      void useGraphStore.getState().fetchHealthScores()
    },
  })

  // Update single mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: UpdateSkillRequest }) =>
      skillService.updateSkill(id, updates),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.skills })
      void useGraphStore.getState().fetchHealthScores()
    },
  })

  // Batch update mutation
  const batchUpdateMutation = useMutation({
    mutationFn: (updates: Map<string, UpdateSkillRequest>) => skillService.updateSkills(updates),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.skills })
      void useGraphStore.getState().fetchHealthScores()
    },
  })

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => skillService.deleteSkill(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.skills })
      void useGraphStore.getState().fetchHealthScores()
    },
  })

  return {
    // Data
    skills,

    // Loading/error states
    isLoading,
    isError,
    error: error ?? null,

    // Mutations
    createSkill: createMutation.mutateAsync,
    updateSkill: (id: string, updates: UpdateSkillRequest) =>
      updateMutation.mutateAsync({ id, updates }),
    updateSkills: batchUpdateMutation.mutateAsync,
    deleteSkill: deleteMutation.mutateAsync,

    // Mutation states
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending || batchUpdateMutation.isPending,
    isDeleting: deleteMutation.isPending,

    // Utilities
    refetch: () => void refetch(),
  }
}
