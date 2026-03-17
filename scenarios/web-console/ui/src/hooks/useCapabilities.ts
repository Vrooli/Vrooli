// DOC: docs/internal/SEAMS.md#capability-registry-seam
import { useQuery } from "@tanstack/react-query";
import { fetchCapabilities, type CapabilitiesResponse } from "../lib/api";

/**
 * React-query hook for polling dependency capabilities.
 * Mirrors git-control-tower's useCapabilities pattern with 30s refetch.
 *
 * @param enabled - Pass false to pause fetching (e.g. when the settings modal is closed).
 */
export function useCapabilities(enabled = true) {
  return useQuery<CapabilitiesResponse, Error>({
    queryKey: ["capabilities"],
    queryFn: fetchCapabilities,
    refetchInterval: 30_000,
    enabled,
  });
}
