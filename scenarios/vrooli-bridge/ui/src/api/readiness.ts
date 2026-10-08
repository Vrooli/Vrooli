import { buildApiUrl } from "@vrooli/api-base";

import { authedFetch, decodeApiError, REST_API_BASE } from "./client";

export interface BridgeReadiness {
  status: "ready" | "not_ready" | "candidate_blocked";
  endpoint: string;
  port: number;
  endpoint_source: "configured" | "tunnel" | "derived";
  reachability_mode: "lan" | "tunnel" | "manual";
  local_api: boolean;
  last_candidate?: { host: string; endpoint: string; mode: string; state: string; category?: string; source_ip?: string };
  firewall?: { available: boolean; inspectable: boolean; active: boolean; rule_found: boolean; privileged: boolean; broker_available: boolean; broker_status?: string };
}

export interface BridgeFirewallActionResult {
  status: string;
  code?: string;
  changed?: boolean;
  evidence?: { available: boolean; active: boolean; rule_found: boolean; managed: boolean };
}

export async function fetchBridgeReadiness(): Promise<BridgeReadiness> {
  const response = await authedFetch(buildApiUrl("/readiness", { baseUrl: REST_API_BASE }), { cache: "no-store" });
  if (!response.ok) throw await decodeApiError(response);
  return response.json() as Promise<BridgeReadiness>;
}

export async function performBridgeFirewallAction(action: "preview" | "inspect" | "verify" | "allow" | "revoke", candidateIP: string, confirm = false): Promise<BridgeFirewallActionResult> {
  const response = await authedFetch(buildApiUrl("/readiness/firewall", { baseUrl: REST_API_BASE }), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ action, candidate_ip: candidateIP, confirm }),
  });
  if (!response.ok) throw await decodeApiError(response);
  return response.json() as Promise<BridgeFirewallActionResult>;
}
