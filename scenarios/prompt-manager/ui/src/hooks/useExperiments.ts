/**
 * useExperiments - Data fetching hooks for experiments.
 *
 * Handles:
 * - Listing experiments by skill via react-query
 * - Getting a single experiment
 * - Start, conclude, and create mutations
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type {
  Experiment,
  CreateExperimentRequest,
  ConcludeExperimentRequest,
} from '@/lib/schemas'

const QUERY_KEYS = {
  experimentsBySkill: (skillId: string) => ['experiments', 'bySkill', skillId] as const,
  experiment: (id: string) => ['experiments', id] as const,
}

/**
 * Fetch experiments for a skill.
 */
export function useExperimentsBySkill(skillId: string | null) {
  return useQuery<Experiment[]>({
    queryKey: QUERY_KEYS.experimentsBySkill(skillId ?? ''),
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion -- guarded by enabled: !!skillId
    queryFn: () => api.listExperimentsBySkill(skillId!),
    enabled: !!skillId,
    staleTime: 10_000,
  })
}

/**
 * Fetch a single experiment.
 */
export function useExperiment(experimentId: string | null) {
  return useQuery<Experiment>({
    queryKey: QUERY_KEYS.experiment(experimentId ?? ''),
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion -- guarded by enabled: !!experimentId
    queryFn: () => api.getExperiment(experimentId!),
    enabled: !!experimentId,
    staleTime: 10_000,
  })
}

/**
 * Mutation to create an experiment.
 */
export function useCreateExperiment() {
  const queryClient = useQueryClient()

  return useMutation<Experiment, Error, CreateExperimentRequest>({
    mutationFn: (req) => api.createExperiment(req),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.experimentsBySkill(variables.skillId),
      })
    },
  })
}

/**
 * Mutation to start an experiment.
 */
export function useStartExperiment() {
  const queryClient = useQueryClient()

  return useMutation<Experiment, Error, { experimentId: string; skillId: string }>({
    mutationFn: ({ experimentId }) => api.startExperiment(experimentId),
    onSuccess: (data, variables) => {
      void queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.experimentsBySkill(variables.skillId),
      })
      void queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.experiment(data.id),
      })
    },
  })
}

/**
 * Mutation to conclude an experiment.
 */
export function useConcludeExperiment() {
  const queryClient = useQueryClient()

  return useMutation<
    Experiment,
    Error,
    { experimentId: string; skillId: string; req: ConcludeExperimentRequest }
  >({
    mutationFn: ({ experimentId, req }) => api.concludeExperiment(experimentId, req),
    onSuccess: (data, variables) => {
      void queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.experimentsBySkill(variables.skillId),
      })
      void queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.experiment(data.id),
      })
      // Skill content may have changed if winner was promoted
      void queryClient.invalidateQueries({ queryKey: ['skills'] })
    },
  })
}
