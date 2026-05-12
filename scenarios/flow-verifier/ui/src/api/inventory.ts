// API client for the flow-verifier inventory: discovered flows + their
// most recent verification run, and POST /api/v1/verifications to kick
// one off. These endpoints are plain JSON (not proto-typed), so this
// module fetches directly rather than going through connect/proto.
import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE, decodeApiError } from "./client";

export type FlowSummary = {
  flowId: string;
  contractPath: string;
  language: string;
  schemaVersion: number;
};

export type RunRow = {
  id: string;
  flowId: string;
  flowPath: string;
  root: string;
  mode: "run" | "check";
  status: "passed" | "failed" | "error";
  errorMessage?: string;
  output?: string;
  startedAt: string;
  finishedAt: string;
  durationMs: number;
};

export type VerifyResponse = {
  status: "passed" | "failed";
  error?: string;
  runs: RunRow[];
};

async function getJson<T>(path: string): Promise<T> {
  const res = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), {
    method: "GET",
    cache: "no-store",
  });
  if (!res.ok) throw await decodeApiError(res);
  return (await res.json()) as T;
}

export async function fetchFlows(root: string): Promise<FlowSummary[]> {
  const q = new URLSearchParams({ root });
  const body = await getJson<{ flows: FlowSummary[] }>(`/api/v1/flows?${q.toString()}`);
  return body.flows ?? [];
}

export async function fetchRuns(opts: { flowId?: string; limit?: number } = {}): Promise<RunRow[]> {
  const q = new URLSearchParams();
  if (opts.flowId) q.set("flowId", opts.flowId);
  if (opts.limit !== undefined) q.set("limit", String(opts.limit));
  const suffix = q.toString();
  const body = await getJson<{ runs: RunRow[] }>(`/api/v1/runs${suffix ? "?" + suffix : ""}`);
  return body.runs ?? [];
}

export async function verifyFlow(root: string, flowId?: string): Promise<VerifyResponse> {
  const res = await fetch(buildApiUrl("/api/v1/verifications", { baseUrl: API_BASE }), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
    body: JSON.stringify({ root, flowId, mode: "check" }),
  });
  if (!res.ok) throw await decodeApiError(res);
  return (await res.json()) as VerifyResponse;
}
