import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { graphClient } from "../../../api/graph";

/**
 * Stable React Query cache keys for the targets feature. One key-builder per
 * surface so cache-key drift can't sneak in via inline string assembly.
 */
export const targetsKeys = {
  all: () => ["targets"] as const,
  snapshots: (scenario?: string) =>
    [...targetsKeys.all(), "snapshots", scenario ?? "_all"] as const,
};

const DEFAULT_LIST_PAGE_SIZE = 25;

/** Read recent snapshots across all scenarios (or filtered to one). */
export function useListSnapshots({
  scenario,
  pageSize = DEFAULT_LIST_PAGE_SIZE,
  enabled = true,
}: {
  scenario?: string;
  pageSize?: number;
  enabled?: boolean;
} = {}) {
  return useQuery({
    queryKey: targetsKeys.snapshots(scenario),
    queryFn: () =>
      graphClient.listGraphSnapshots({
        scenario: scenario ?? "",
        pageSize,
        pageToken: "",
      }),
    enabled,
  });
}

export interface ExtractGraphArgs {
  scenario: string;
  /** Optional caller-supplied idempotency key (rarely needed from the UI). */
  idempotencyKey?: string;
}

/**
 * Trigger an `ExtractGraph` RPC. On success the snapshot list cache is
 * invalidated so the overview page refreshes the active snapshots panel.
 */
export function useExtractGraph() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ scenario, idempotencyKey }: ExtractGraphArgs) =>
      graphClient.extractGraph({
        scenario,
        languages: [],
        idempotencyKey: idempotencyKey ?? "",
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: targetsKeys.all() });
    },
  });
}
