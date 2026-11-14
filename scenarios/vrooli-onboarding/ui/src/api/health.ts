import { buildApiUrl } from "@vrooli/api-base";
import { REST_API_BASE } from "./base";

export interface HealthResponse { status: string; service: string; timestamp: string }

export async function fetchHealth(): Promise<HealthResponse> {
  const response = await fetch(buildApiUrl("/health", { baseUrl: REST_API_BASE }), { cache: "no-store" });
  if (!response.ok) throw new Error(`API request failed: ${response.status}`);
  return response.json() as Promise<HealthResponse>;
}
