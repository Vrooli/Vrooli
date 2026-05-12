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
  scenarioId?: string;
};

// FailureReason narrows a failed run into a typed category. The UI's
// status pill renders the "missing_artifacts" / "stale_artifacts" cases
// as a distinct yellow "Needs generate" state with a one-click
// regenerate CTA. Other reasons fall through to the standard red
// "Failed" pill.
export type FailureReason =
  | ""
  | "missing_artifacts"
  | "stale_artifacts"
  | "counterexample"
  | "lint"
  | "quint_failure"
  | "io";

export type RunRow = {
  id: string;
  flowId: string;
  flowPath: string;
  root: string;
  mode: "run" | "check";
  status: "passed" | "failed" | "error";
  errorMessage?: string;
  failureReason?: FailureReason;
  missingArtifacts?: string[];
  output?: string;
  counterexample?: string;
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

export async function fetchFlows(opts: { scenarioId?: string } = {}): Promise<FlowSummary[]> {
  const q = new URLSearchParams();
  if (opts.scenarioId) q.set("scenario", opts.scenarioId);
  const suffix = q.toString();
  const body = await getJson<{ flows?: FlowSummary[] }>(
    `/api/v1/flows${suffix ? "?" + suffix : ""}`,
  );
  return body.flows ?? [];
}

export async function fetchRuns(opts: { flowId?: string; limit?: number } = {}): Promise<RunRow[]> {
  const q = new URLSearchParams();
  if (opts.flowId) q.set("flowId", opts.flowId);
  if (opts.limit !== undefined) q.set("limit", String(opts.limit));
  const suffix = q.toString();
  const body = await getJson<{ runs?: RunRow[] }>(`/api/v1/runs${suffix ? "?" + suffix : ""}`);
  return body.runs ?? [];
}

export type FlowState = {
  id: string;
  quint: string;
  initial?: boolean;
  terminal?: boolean;
};

export type FlowEvent = {
  id: string;
  quint: string;
};

export type FlowTransition = {
  from: string;
  event: string;
  to: string;
  wantError: boolean;
};

export type FlowTraceStep = {
  event: string;
  want: string;
  wantError: boolean;
};

export type FlowTrace = {
  name: string;
  initial: string;
  steps: FlowTraceStep[];
};

export type FlowInvariant = {
  id: string;
  quint: string;
  description: string;
  expression?: string;
};

export type FlowModelVerify = {
  invariants: string[];
};

export type FlowModel = {
  module: string;
  seed: string;
  maxSteps: number;
  traceCount: number;
  verify: FlowModelVerify;
};

export type FlowGoRuntime = {
  package: string;
  statusType: string;
  eventType: string;
  constantPrefix: string;
};

export type FlowTypeScriptRuntime = {
  statusType: string;
  eventType: string;
  statusesConst: string;
  eventsConst: string;
  formalExpectationConst: string;
  stateUnionType?: string;
  eventUnionType?: string;
};

export type FlowRuntime = {
  go?: FlowGoRuntime;
  typescript?: FlowTypeScriptRuntime;
  sideEffects?: string[];
  staleCompletion?: string;
};

export type FlowDetail = {
  flowId: string;
  domain?: string;
  description?: string;
  contractPath: string;
  language: string;
  schemaVersion: number;
  initialState: string;
  states: FlowState[];
  events: FlowEvent[];
  transitions: FlowTransition[];
  traces: FlowTrace[];
  invariants: FlowInvariant[];
  model: FlowModel;
  runtime: FlowRuntime;
  report: string;
};

export async function fetchFlowDetail(
  flowId: string,
  opts: { scenarioId?: string } = {},
): Promise<FlowDetail> {
  const q = new URLSearchParams();
  if (opts.scenarioId) q.set("scenario", opts.scenarioId);
  const suffix = q.toString();
  return getJson<FlowDetail>(
    `/api/v1/flows/${encodeURIComponent(flowId)}${suffix ? "?" + suffix : ""}`,
  );
}

export async function fetchRun(runId: string): Promise<RunRow> {
  return getJson<RunRow>(`/api/v1/runs/${encodeURIComponent(runId)}`);
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
