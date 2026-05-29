import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { conflictsClient } from "../../../api/conflicts";

/**
 * Stable React Query cache keys for the conflicts feature. One key-builder
 * per surface so cache-key drift can't sneak in via inline string assembly.
 *
 * The conflicts domain is detection-only: detect, list, get, validate. The
 * per-finding lifecycle (assign / resolve / reopen) moved to the migration
 * feature.
 */
export const conflictsKeys = {
  all: () => ["conflicts"] as const,
  list: (scenario: string) => [...conflictsKeys.all(), "list", scenario] as const,
  detail: (id: string) => [...conflictsKeys.all(), "detail", id] as const,
};

export interface UseListConflictsArgs {
  scenario: string;
  enabled?: boolean;
}

/** Paginated conflict listing for one scenario. */
export function useListConflicts({ scenario, enabled = true }: UseListConflictsArgs) {
  return useQuery({
    queryKey: conflictsKeys.list(scenario),
    queryFn: () =>
      conflictsClient.listConflicts({
        scenario,
        types: [],
        pageSize: 50,
        pageToken: "",
      }),
    enabled: enabled && scenario.length > 0,
  });
}

export interface UseGetConflictArgs {
  id: string;
  enabled?: boolean;
}

/** Fetch a single conflict by id. */
export function useGetConflict({ id, enabled = true }: UseGetConflictArgs) {
  return useQuery({
    queryKey: conflictsKeys.detail(id),
    queryFn: () => conflictsClient.getConflict({ id }),
    enabled: enabled && id.length > 0,
  });
}

/** Run all detectors against the current snapshot for a scenario. */
export function useDetectConflicts(scenario: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      conflictsClient.detectConflicts({
        scenario,
        snapshotId: "",
        idempotencyKey: "",
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: conflictsKeys.list(scenario) });
    },
  });
}

/** Re-list the current detected conflicts and report the clean/dirty gate. */
export function useValidateConflicts(scenario: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => conflictsClient.validateConflicts({ scenario }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: conflictsKeys.list(scenario) });
    },
  });
}
