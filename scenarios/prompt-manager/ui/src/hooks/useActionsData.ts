/**
 * useActionsData - Data fetching hook for Actions.
 *
 * Handles Action CRUD/validation data wiring while keeping command-contract
 * validation in the API/domain service.
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as actionService from '@/services/actionService'
import { useGraphStore } from '@/stores/graphStore'
import type {
  Action,
  ActionFilters,
  CreateActionRequest,
  UpdateActionRequest,
  ActionMutationResponse,
  ActionValidationResponse,
} from '@/types'

const QUERY_KEYS = {
  actions: ['actions'] as const,
}

const EMPTY_ACTIONS: Action[] = []

interface UseActionsDataReturn {
  actions: Action[]
  isLoading: boolean
  isError: boolean
  error: Error | null
  createAction: (request: CreateActionRequest) => Promise<ActionMutationResponse>
  updateAction: (id: string, updates: UpdateActionRequest) => Promise<ActionMutationResponse>
  deleteAction: (id: string, hard?: boolean) => Promise<void>
  validateAction: (id: string) => Promise<ActionValidationResponse>
  isCreating: boolean
  isUpdating: boolean
  isDeleting: boolean
  isValidating: boolean
  refetch: () => void
}

export function useActionsData(filters?: ActionFilters): UseActionsDataReturn {
  const queryClient = useQueryClient()

  const {
    data: actions = EMPTY_ACTIONS,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: filters ? [...QUERY_KEYS.actions, filters] : QUERY_KEYS.actions,
    queryFn: () => actionService.getActions(filters),
    staleTime: 5000,
  })

  const invalidateActions = () => {
    void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.actions })
    void useGraphStore.getState().fetchHealthScores()
  }

  const createMutation = useMutation({
    mutationFn: (request: CreateActionRequest) => actionService.createAction(request),
    onSuccess: invalidateActions,
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: UpdateActionRequest }) =>
      actionService.updateAction(id, updates),
    onSuccess: invalidateActions,
  })

  const deleteMutation = useMutation({
    mutationFn: ({ id, hard }: { id: string; hard?: boolean }) =>
      actionService.deleteAction(id, hard),
    onSuccess: invalidateActions,
  })

  const validateMutation = useMutation({
    mutationFn: (id: string) => actionService.validateAction(id),
  })

  return {
    actions,
    isLoading,
    isError,
    error: error ?? null,
    createAction: createMutation.mutateAsync,
    updateAction: (id: string, updates: UpdateActionRequest) =>
      updateMutation.mutateAsync({ id, updates }),
    deleteAction: (id: string, hard?: boolean) =>
      deleteMutation.mutateAsync({ id, hard }),
    validateAction: validateMutation.mutateAsync,
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending,
    isDeleting: deleteMutation.isPending,
    isValidating: validateMutation.isPending,
    refetch: () => void refetch(),
  }
}

