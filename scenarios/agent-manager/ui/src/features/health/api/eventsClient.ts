// Fetch helpers for the typed-operational events endpoint.
//
// API: scenarios/agent-manager/api/internal/handlers/events.go

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

async function fetchJson<T>(url: string): Promise<T> {
  const res = await fetch(url, { headers: { "Content-Type": "application/json" }, cache: "no-store" });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`Events API error ${res.status}: ${body}`);
  }
  return (await res.json()) as T;
}

export async function fetchEventsForRun(runId: string, limit = 200): Promise<TypedEventsResponse> {
  const params = new URLSearchParams({ run: runId, limit: String(limit) });
  return fetchJson<TypedEventsResponse>(`/api/v1/events?${params.toString()}`);
}

export const eventsQueryKeys = {
  all: ["events"] as const,
  forRun: (runId: string) => [...eventsQueryKeys.all, "run", runId] as const,
};
