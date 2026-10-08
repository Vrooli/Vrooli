import { createClient } from "@connectrpc/connect";
import {
  ExposureService,
  LeaseStatus,
  type Lease,
  type Exposure,
} from "@vrooli/proto-types/tunnel-manager/v1/exposure/exposure_pb";

import { transport } from "./client";

// exposureClient is the generated Connect-Web client for ExposureService —
// the tiered exposure broker (CORE always-on + LEASED TTL). It backs the
// primary Exposure operations surface under ui/src/features/exposure/.
export const exposureClient = createClient(ExposureService, transport);

/** listExposures returns the reconciled exposure state per scenario. */
export async function listExposures(): Promise<Exposure[]> {
  const resp = await exposureClient.listExposures({});
  return resp.exposures;
}

/** listLeases returns leases, optionally filtered by status. */
export async function listLeases(
  status: LeaseStatus = LeaseStatus.UNSPECIFIED,
): Promise<Lease[]> {
  const resp = await exposureClient.listLeases({ status });
  return resp.leases;
}

/** expose grants on-demand exposure to a scenario; returns the reachable URL. */
export async function expose(
  scenario: string,
  ttlSeconds = 0n,
  requestedBy = "",
): Promise<{ lease?: Lease; publicUrl: string }> {
  const resp = await exposureClient.expose({ scenario, ttlSeconds, requestedBy });
  return { lease: resp.lease, publicUrl: resp.publicUrl };
}

export { LeaseStatus };
export type { Lease, Exposure };
