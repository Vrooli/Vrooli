/**
 * useVariants - Data fetching hooks for skill variants.
 *
 * Handles:
 * - Listing variants via react-query
 * - Create and delete mutations with cache invalidation
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { Variant, CreateVariantRequest } from '@/lib/schemas'

const QUERY_KEYS = {
  variants: (skillId: string) => ['variants', skillId] as const,
}

/**
 * Fetch variants for a skill.
 */
export function useVariantList(skillId: string | null) {
  return useQuery<Variant[]>({
    queryKey: QUERY_KEYS.variants(skillId ?? ''),
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion -- guarded by enabled: !!skillId
    queryFn: () => api.listVariants(skillId!),
    enabled: !!skillId,
    staleTime: 10_000,
  })
}

/**
 * Mutation to create a variant.
 */
export function useCreateVariant() {
  const queryClient = useQueryClient()

  return useMutation<Variant, Error, { skillId: string; req: CreateVariantRequest }>({
    mutationFn: ({ skillId, req }) => api.createVariant(skillId, req),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.variants(variables.skillId) })
    },
  })
}

/**
 * Mutation to delete a variant.
 */
export function useDeleteVariant() {
  const queryClient = useQueryClient()

  return useMutation<undefined, Error, { skillId: string; variantId: string }>({
    mutationFn: async ({ skillId, variantId }) => { await api.deleteVariant(skillId, variantId); return undefined },
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.variants(variables.skillId) })
    },
  })
}
