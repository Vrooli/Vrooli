// API client for the scenarios aggregate. Scenarios are the primary
// inventory entity in the UI: every flow lives inside one. The list
// endpoint returns one row per scenario (with flow count); detail
// embeds the flows directly so the scenario-detail page renders in
// one round-trip.
import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE, decodeApiError } from "./client";
import type { FlowSummary } from "./inventory";

export type ScenarioSummary = {
  id: string;
  displayName: string;
  description?: string;
  path: string;
  flowCount: number;
  discoveryError?: string;
};

export type ScenariosListResponse = {
  vrooliRoot: string;
  scenarios: ScenarioSummary[];
};

export type ScenarioDetail = ScenarioSummary & {
  flows: FlowSummary[];
};

async function getJson<T>(path: string): Promise<T> {
  const res = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), {
    method: "GET",
    cache: "no-store",
  });
  if (!res.ok) throw await decodeApiError(res);
  return (await res.json()) as T;
}

// The wire shape is permissive — Go may emit `null` for empty slices —
// so we accept partial fields and normalise to the strict client type
// below. Keeping the raw type narrow here is what lets the rest of the
// app trust that `scenarios`/`flows` are arrays.
type RawScenariosListResponse = {
  vrooliRoot?: string;
  scenarios?: ScenarioSummary[] | null;
};

type RawScenarioDetail = Omit<ScenarioDetail, "flows"> & {
  flows?: FlowSummary[] | null;
};

export async function fetchScenarios(): Promise<ScenariosListResponse> {
  const body = await getJson<RawScenariosListResponse>(`/api/v1/scenarios`);
  return {
    vrooliRoot: body.vrooliRoot ?? "",
    scenarios: body.scenarios ?? [],
  };
}

export async function fetchScenarioDetail(id: string): Promise<ScenarioDetail> {
  const body = await getJson<RawScenarioDetail>(
    `/api/v1/scenarios/${encodeURIComponent(id)}`,
  );
  return { ...body, flows: body.flows ?? [] };
}
