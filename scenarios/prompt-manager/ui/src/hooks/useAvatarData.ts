/**
 * useAvatarData - Data fetching hook for avatars.
 *
 * Handles:
 * - Fetching all avatars via react-query
 * - CRUD operations with cache invalidation
 * - Loading and error states
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as avatarService from '@/services/avatarService'
import type { Avatar, CreateAvatarRequest, UpdateAvatarRequest } from '@/types/avatar'

// Query key constants
const QUERY_KEYS = {
  avatars: ['avatars'] as const,
}

interface UseAvatarDataReturn {
  // Data
  avatars: Avatar[]

  // Loading/error states
  isLoading: boolean
  isError: boolean
  error: Error | null

  // Mutations
  createAvatar: (request: CreateAvatarRequest) => Promise<Avatar>
  updateAvatar: (id: string, updates: UpdateAvatarRequest) => Promise<Avatar>
  deleteAvatar: (id: string) => Promise<void>

  // Mutation states
  isCreating: boolean
  isUpdating: boolean
  isDeleting: boolean

  // Utilities
  refetch: () => void
}

/**
 * Hook for fetching and mutating avatar data.
 */
export function useAvatarData(): UseAvatarDataReturn {
  const queryClient = useQueryClient()

  // Query for all avatars
  const {
    data: avatars = [],
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: QUERY_KEYS.avatars,
    queryFn: () => avatarService.getAvatars(true), // Force refresh on query
    staleTime: 5000, // Match service cache TTL
  })

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (request: CreateAvatarRequest) => avatarService.createAvatar(request),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.avatars })
    },
  })

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: UpdateAvatarRequest }) =>
      avatarService.updateAvatar(id, updates),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.avatars })
    },
  })

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => avatarService.deleteAvatar(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.avatars })
    },
  })

  return {
    // Data
    avatars,

    // Loading/error states
    isLoading,
    isError,
    error: error ?? null,

    // Mutations
    createAvatar: createMutation.mutateAsync,
    updateAvatar: (id: string, updates: UpdateAvatarRequest) =>
      updateMutation.mutateAsync({ id, updates }),
    deleteAvatar: deleteMutation.mutateAsync,

    // Mutation states
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending,
    isDeleting: deleteMutation.isPending,

    // Utilities
    refetch: () => void refetch(),
  }
}
