/**
 * useMemberData - Data fetching hook for members.
 *
 * Handles:
 * - Fetching all members via react-query
 * - CRUD operations with cache invalidation
 * - Loading and error states
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as memberService from '@/services/memberService'
import type { Member, CreateMemberRequest, UpdateMemberRequest } from '@/types/member'

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
 */
export function useMemberData(): UseMemberDataReturn {
  const queryClient = useQueryClient()

  // Query for all members
  const {
    data: members = [],
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: QUERY_KEYS.members,
    queryFn: () => memberService.getMembers(true), // Force refresh on query
    staleTime: 5000, // Match service cache TTL
  })

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (request: CreateMemberRequest) => memberService.createMember(request),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.members })
    },
  })

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: UpdateMemberRequest }) =>
      memberService.updateMember(id, updates),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.members })
    },
  })

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => memberService.deleteMember(id),
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
