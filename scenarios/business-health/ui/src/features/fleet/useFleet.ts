import { useQuery } from "@tanstack/react-query";
import type { ScanFleetResponse } from "@vrooli/proto-types/business-health/v1/fleet/fleet_pb";

import { fleetClient } from "../../api/fleet";

/** React Query key for the (single, auto-scanning) fleet grade. */
export const fleetQueryKey = ["fleet"] as const;

/**
 * Statically grade every discovered scenario, worst-first. The scan covers the
 * whole fleet, so there is no scenario picker — the query fires on mount and is
 * re-run by the Rescan button via `refetch()`.
 */
export function useFleet() {
  return useQuery<ScanFleetResponse>({
    queryKey: fleetQueryKey,
    queryFn: ({ signal }) => fleetClient.scanFleet({}, { signal }),
  });
}
