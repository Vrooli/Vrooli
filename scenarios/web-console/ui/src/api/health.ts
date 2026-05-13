// Non-Connect health probe. Kept on plain HTTP because the /health
// endpoint is the liveness check for the API binary itself and must
// answer before Connect-RPC routing is wired up.

import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
}

export async function fetchHealth(): Promise<HealthResponse> {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) throw new Error(`API health check failed: ${res.status}`);
  return res.json() as Promise<HealthResponse>;
}
