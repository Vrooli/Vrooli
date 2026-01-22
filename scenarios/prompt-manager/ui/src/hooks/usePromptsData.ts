/**
 * usePromptsData - Data fetching hook for prompts.
 *
 * Handles:
 * - Fetching all prompts via react-query
 * - CRUD operations with cache invalidation
 * - Loading and error states
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as promptService from '@/services/promptService'
import type { Prompt, CreatePromptRequest, UpdatePromptRequest } from '@/types'

// Query key constants
const QUERY_KEYS = {
  prompts: ['prompts'] as const,
}

interface UsePromptsDataReturn {
  // Data
  prompts: Prompt[]

  // Loading/error states
  isLoading: boolean
  isError: boolean
  error: Error | null

  // Mutations
  createPrompt: (request: CreatePromptRequest) => Promise<Prompt>
  updatePrompt: (id: string, updates: UpdatePromptRequest) => Promise<Prompt>
  updatePrompts: (updates: Map<string, UpdatePromptRequest>) => Promise<Map<string, Prompt | Error>>
  deletePrompt: (id: string) => Promise<void>

  // Mutation states
  isCreating: boolean
  isUpdating: boolean
  isDeleting: boolean

  // Utilities
  refetch: () => void
}

/**
 * Hook for fetching and mutating prompts data.
 */
export function usePromptsData(): UsePromptsDataReturn {
  const queryClient = useQueryClient()

  // Query for all prompts
  const {
    data: prompts = [],
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: QUERY_KEYS.prompts,
    queryFn: () => promptService.getPrompts(true), // Force refresh on query
    staleTime: 5000, // Match service cache TTL
  })

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (request: CreatePromptRequest) => promptService.createPrompt(request),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.prompts })
    },
  })

  // Update single mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: UpdatePromptRequest }) =>
      promptService.updatePrompt(id, updates),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.prompts })
    },
  })

  // Batch update mutation
  const batchUpdateMutation = useMutation({
    mutationFn: (updates: Map<string, UpdatePromptRequest>) => promptService.updatePrompts(updates),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.prompts })
    },
  })

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => promptService.deletePrompt(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.prompts })
    },
  })

  return {
    // Data
    prompts,

    // Loading/error states
    isLoading,
    isError,
    error: error ?? null,

    // Mutations
    createPrompt: createMutation.mutateAsync,
    updatePrompt: (id: string, updates: UpdatePromptRequest) =>
      updateMutation.mutateAsync({ id, updates }),
    updatePrompts: batchUpdateMutation.mutateAsync,
    deletePrompt: deleteMutation.mutateAsync,

    // Mutation states
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending || batchUpdateMutation.isPending,
    isDeleting: deleteMutation.isPending,

    // Utilities
    refetch: () => void refetch(),
  }
}
