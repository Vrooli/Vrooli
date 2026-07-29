import { buildApiUrl } from "@vrooli/api-base";
import { API_BASE, decodeApiError } from "./client";

const root = "/api/v1/channel-manager";
async function request<T>(path: string, body?: unknown): Promise<T> {
  const response = await fetch(buildApiUrl(`${root}${path}`, { baseUrl: API_BASE }), {
    method: body === undefined ? "GET" : "POST",
    headers: body === undefined ? undefined : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!response.ok) throw await decodeApiError(response);
  return response.json() as Promise<T>;
}
export type Identity = { id: string; platform_id: string; purpose: string; environment_ref: string; vault_ref: string; status: string; lane_grants?: string[] };
export type Action = { id: string; identity_id: string; kind: string; window: string; status: string; rolled_count: number };
export type Program = { id: string; platform_id: string; provenance: { source_kind: string; confidence: string; revisit_trigger: string } };
export type Flag = { identity_id?: string; metric?: string; message?: string };
export type Overview = { identities: Record<string, Identity>; actions: Record<string, Action>; programs?: Record<string, Program>; flags?: Record<string, Flag[]> };
export const overview = () => request<Overview>("/overview");
export const createIdentity = (identity: Identity & { attestations: Record<string, boolean> }) => request<Identity>("/identities", identity);
export const startProgram = (id: string, program_id: string) => request<{ status: string }>(`/identities/${id}/start`, { program_id });
export const enqueueAction = (identity_id: string, kind: string) => request<Action>("/actions", { identity_id, kind, seed: Date.now() });
export const completeAction = (id: string, evidence: string) => request<{ status: string }>(`/actions/${id}/complete`, { evidence });
export const recordObservation = (id: string, value: number) => request<{ flag: unknown }>(`/identities/${id}/observations`, { metric: "reach", value });
