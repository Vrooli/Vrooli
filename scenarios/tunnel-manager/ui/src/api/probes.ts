import { createClient } from "@connectrpc/connect";
import {
  ProbesService,
  ProbeKind,
  ProbeStatus,
  FailureClass,
  type ProbeResult,
  type RouteClassification,
} from "@vrooli/proto-types/tunnel-manager/v1/probes/probes_pb";

import { transport } from "./client";

// probesClient is the generated Connect-Web client for ProbesService —
// internal+external liveness probing, history, and failure classification.
// Backs the per-route probe drill-down under ui/src/features/probes/.
export const probesClient = createClient(ProbesService, transport);

/** runProbes executes one probe cycle across all exposed routes. */
export async function runProbes(): Promise<ProbeResult[]> {
  const resp = await probesClient.runProbes({});
  return resp.results;
}

/** listProbes returns recent probe history, optionally filtered by subdomain. */
export async function listProbes(subdomain = "", limit = 0): Promise<ProbeResult[]> {
  const resp = await probesClient.listProbes({ subdomain, limit });
  return resp.results;
}

/** classify returns the per-route reachability diagnosis. */
export async function classify(): Promise<RouteClassification[]> {
  const resp = await probesClient.classify({});
  return resp.classifications;
}

export { ProbeKind, ProbeStatus, FailureClass };
export type { ProbeResult, RouteClassification };
