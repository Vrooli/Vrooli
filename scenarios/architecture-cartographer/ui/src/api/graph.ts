import { createClient, type Client } from "@connectrpc/connect";
import { buildApiUrl } from "@vrooli/api-base";
import { GraphService } from "@vrooli/proto-types/architecture-cartographer/v1/graph/graph_pb";

import { API_BASE, decodeApiError, transport } from "./client";

/**
 * Connect-RPC client for the cartographer GraphService. Consumers should
 * call methods (`graphClient.extractGraph(...)`) directly through React
 * Query rather than wrapping them in ad-hoc fetch helpers — the generated
 * client owns proto encoding, error parsing, and cancellation.
 */
export const graphClient: Client<typeof GraphService> = createClient(GraphService, transport);

export interface SliceEvidenceView {
  kind: string;
  value: string;
  source: string;
}

export interface SliceFileView {
  path: string;
  lines: number;
  is_test: boolean;
}

export interface SliceSymbolView {
  name: string;
  kind: string;
  file: string;
}

export interface SliceRungView {
  name: string;
  present: boolean;
  evidence: SliceEvidenceView[];
  files: SliceFileView[];
  symbols: SliceSymbolView[];
}

export interface DomainSliceView {
  scenario: string;
  domain: string;
  snapshot_id: string;
  rungs: SliceRungView[];
  surfaces: string[];
  layer_edges: { from_rung: string; to_rung: string; kind: string }[];
  attestation?: {
    claim: string;
    citations: { locator: string; kind: string; note: string }[];
    basis: number;
    sufficiency: number;
    gaps: string[];
    suggested_follow_ups: string[];
  };
}

export interface RankedSliceHitView {
  id: string;
  title: string;
  score: number;
  text: string;
  path: string;
  paths: string[];
  scenario: string;
  domain: string;
  snapshot_id: string;
  slice: DomainSliceView;
}

export interface SliceSearchResponseView {
  query: string;
  scope: string;
  results: RankedSliceHitView[];
  degradations: { code: string; message: string }[];
}

export async function searchSlices(query: string, scope: string, limit = 5): Promise<SliceSearchResponseView> {
  const response = await fetch(buildApiUrl("/api/v1/slices/search", { baseUrl: API_BASE }), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query, scope, limit }),
  });
  if (!response.ok) throw await decodeApiError(response);
  return (await response.json()) as SliceSearchResponseView;
}
