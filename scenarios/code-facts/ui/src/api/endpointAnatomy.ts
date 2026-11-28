import { buildApiUrl } from "@vrooli/api-base";
import { API_BASE, decodeApiError } from "./client";

export interface EndpointSource { path: string; line: number; sha256: string; basis: string }
export interface EndpointRouteCandidate { handler: string; registration: EndpointSource; handler_source: EndpointSource }
export interface EndpointAnatomy { id: string; title: string; summary: string; scenario: string; route_path: string; method: string; handler: string; state: string; declaration: EndpointSource; registration: EndpointSource; handler_source: EndpointSource; transport?: "connect" | "http_json"; connect_service?: string; connect_method?: string; connect_chain?: EndpointSource[]; route_candidates?: EndpointRouteCandidate[]; route_candidate_total?: number; route_candidates_omitted?: number; limits: string[]; score: number }
export interface EndpointAnatomyResponse { query: string; scenario: string; scope: string; results: EndpointAnatomy[]; degradations: string[] }

export async function searchEndpointAnatomy(query: string, scenario: string, limit = 10): Promise<EndpointAnatomyResponse> {
  const response = await fetch(buildApiUrl("/api/v1/endpoint-anatomy/search", { baseUrl: API_BASE }), { method:"POST", headers:{"Content-Type":"application/json"}, body:JSON.stringify({query,scenario,limit}) });
  if (!response.ok) throw await decodeApiError(response);
  return (await response.json()) as EndpointAnatomyResponse;
}
