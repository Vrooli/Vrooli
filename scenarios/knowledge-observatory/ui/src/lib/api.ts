import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

// Use /api prefix without /v1 suffix since our health endpoint is at /health (not /api/v1/health)
// The /api prefix ensures requests go through the UI server proxy to the API server
const API_BASE = resolveApiBase({ appendSuffix: false });

export async function fetchHealth() {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`API health check failed: ${res.status}`);
  }

  return res.json() as Promise<{ status: string; service: string; timestamp: string }>;
}
