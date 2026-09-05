import { buildApiUrl } from '@vrooli/api-base';

import { API_BASE, decodeApiError } from './client';

/**
 * HealthResponse mirrors the /health readiness endpoint. Health is a plain
 * REST probe (not a Connect RPC) so infra and load balancers can poll it
 * without a proto client.
 */
export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
}

/** Fetches API readiness from the REST /health probe. */
export async function fetchHealth(): Promise<HealthResponse> {
  const res = await fetch(buildApiUrl('/health', { baseUrl: API_BASE }), {
    method: 'GET',
    cache: 'no-store',
  });
  if (!res.ok) {
    throw await decodeApiError(res);
  }
  return (await res.json()) as HealthResponse;
}
