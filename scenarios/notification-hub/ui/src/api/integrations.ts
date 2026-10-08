import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE } from "./client";

export interface EventIntegrationConfig {
  events_api_base: string;
  webhook_url: string;
  pattern: string;
  templates?: Record<string, { title: string; body: string }>;
  sensitivity_by_severity?: Record<string, string>;
}

export async function fetchEventIntegrationConfig(): Promise<EventIntegrationConfig> {
  const response = await fetch(buildApiUrl("/config/event-integration", { baseUrl: API_BASE }), { cache: "no-store" });
  if (!response.ok) throw new Error(`Event integration config request failed (${response.status})`);
  return (await response.json()) as EventIntegrationConfig;
}

export async function updateEventIntegrationConfig(config: EventIntegrationConfig): Promise<EventIntegrationConfig> {
  const response = await fetch(buildApiUrl("/config/event-integration", { baseUrl: API_BASE }), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  });
  if (!response.ok) throw new Error(`Event integration config update failed (${response.status})`);
  return (await response.json()) as EventIntegrationConfig;
}
