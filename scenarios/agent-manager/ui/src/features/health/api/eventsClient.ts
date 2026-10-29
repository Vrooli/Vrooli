// Fetch helpers for the typed-operational events endpoint.
//
// API: scenarios/agent-manager/api/internal/handlers/events.go

import { apiRequest } from "../../../hooks/useApiTransport";

export interface TypedEventRow {
  id: string;
  run_id: string;
  sequence: number;
  event_type: string;
  schema_version: number;
  timestamp: string;
  payload: unknown;
}

export interface TypedEventsResponse {
  events: TypedEventRow[];
  limit: number;
}

export async function fetchEventsForRun(runId: string, limit = 200): Promise<TypedEventsResponse> {
  const params = new URLSearchParams({ run: runId, limit: String(limit) });
  return apiRequest<TypedEventsResponse>(`/events?${params.toString()}`, { cache: "no-store" });
}

export const eventsQueryKeys = {
  all: ["events"] as const,
  forRun: (runId: string) => [...eventsQueryKeys.all, "run", runId] as const,
};
