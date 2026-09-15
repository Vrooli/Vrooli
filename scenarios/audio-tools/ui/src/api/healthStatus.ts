/**
 * HealthStatusService Connect-RPC wrapper.
 *
 * Thin adapter over the generated client: no hand-typed shapes — every
 * field surfaces directly from the proto module. Callers (React Query
 * hooks, the StatusPage) consume the typed messages from
 * `@vrooli/proto-types/audio-tools/v1/health_status/health_status_pb`.
 */
import { createClient } from "@connectrpc/connect";
import { HealthStatusService } from "@vrooli/proto-types/audio-tools/v1/health_status/health_status_pb";
import type {
  GetProviderHealthResponse,
  ProviderHealthEvent,
  RefreshProviderHealthResponse,
} from "@vrooli/proto-types/audio-tools/v1/health_status/health_status_pb";

import { transport } from "./client";

const healthStatusClient = createClient(HealthStatusService, transport);

export async function getProviderHealth(): Promise<GetProviderHealthResponse> {
  return healthStatusClient.getProviderHealth({});
}

export async function refreshProviderHealth(): Promise<RefreshProviderHealthResponse> {
  return healthStatusClient.refreshProviderHealth({});
}

export function streamProviderHealth(
  signal: AbortSignal,
): AsyncIterable<ProviderHealthEvent> {
  return healthStatusClient.streamProviderHealth({}, { signal });
}

export type {
  GetProviderHealthResponse,
  ProviderHealthEvent,
  RefreshProviderHealthResponse,
};
