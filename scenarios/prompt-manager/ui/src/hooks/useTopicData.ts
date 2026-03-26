/**
 * useTopicData - Data fetching hook for topics.
 *
 * Handles:
 * - Fetching all topics via react-query
 * - CRUD operations with cache invalidation
 * - Loading and error states
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as topicService from '@/services/topicService'
import type { Topic, CreateTopicRequest, UpdateTopicRequest } from '@/lib/schemas'

// Query key constants
const QUERY_KEYS = {
  topics: ['topics'] as const,
  topic: (id: string) => ['topics', id] as const,
}

// Stable empty array to avoid new reference on every render
const EMPTY_TOPICS: Topic[] = []

interface UseTopicsReturn {
  // Data
  topics: Topic[]

  // Loading/error states
  isLoading: boolean
  isError: boolean
  error: Error | null

  // Mutations
  createTopic: (request: CreateTopicRequest) => Promise<Topic>
  updateTopic: (id: string, updates: UpdateTopicRequest) => Promise<Topic>
  deleteTopic: (id: string) => Promise<void>

  // Mutation states
  isCreating: boolean
  isUpdating: boolean
  isDeleting: boolean

  // Utilities
  refetch: () => void
}

/**
 * Hook for fetching and mutating topics data.
 */
export function useTopics(): UseTopicsReturn {
  const queryClient = useQueryClient()

  // Query for all topics
  const {
    data: topics = EMPTY_TOPICS,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: QUERY_KEYS.topics,
    queryFn: () => topicService.getTopics(),
    staleTime: 5000, // Match service cache TTL
  })

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (request: CreateTopicRequest) => topicService.createTopic(request),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.topics })
    },
  })

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: UpdateTopicRequest }) =>
      topicService.updateTopic(id, updates),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.topics })
    },
  })

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => topicService.deleteTopic(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.topics })
    },
  })

  return {
    // Data
    topics,

    // Loading/error states
    isLoading,
    isError,
    error: error ?? null,

    // Mutations
    createTopic: createMutation.mutateAsync,
    updateTopic: (id: string, updates: UpdateTopicRequest) =>
      updateMutation.mutateAsync({ id, updates }),
    deleteTopic: deleteMutation.mutateAsync,

    // Mutation states
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending,
    isDeleting: deleteMutation.isPending,

    // Utilities
    refetch: () => void refetch(),
  }
}

interface UseTopicReturn {
  topic: Topic | undefined
  isLoading: boolean
  isError: boolean
  error: Error | null
}

/**
 * Hook for fetching a single topic by ID.
 */
export function useTopic(id: string | null): UseTopicReturn {
  const {
    data: topic,
    isLoading,
    isError,
    error,
  } = useQuery({
    queryKey: QUERY_KEYS.topic(id ?? ''),
    queryFn: () => topicService.getTopic(id ?? ''),
    enabled: !!id,
    staleTime: 5000,
  })

  return {
    topic,
    isLoading,
    isError,
    error: error ?? null,
  }
}
