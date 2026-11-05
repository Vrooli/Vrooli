import { useQuery, type UseQueryResult } from "@tanstack/react-query";

import { fetchHealth, type HealthResponse } from "../../api/health";

/** Poll cadence for the sidecar status indicator (ms). */
const SIDECAR_POLL_MS = 10_000;

/**
 * Poll /health for the Node ts-morph sidecar's lifecycle. Shares the same
 * `fetchHealth` client as HealthCard but on its own cache key + a refresh
 * interval so the persistent indicator stays live without a manual refresh.
 */
export function useSidecarHealth(): UseQueryResult<HealthResponse> {
  return useQuery({
    queryKey: ["health", "sidecar"],
    queryFn: fetchHealth,
    refetchInterval: SIDECAR_POLL_MS,
  });
}
