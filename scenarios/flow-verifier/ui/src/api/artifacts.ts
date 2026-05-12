// API client for the artifacts (codegen lifecycle) surface. Mirrors
// scenarios.ts's Raw → Public type pattern: the wire shape is permissive
// so Go's nil-slice / nil-string quirks don't leak into React, and the
// public type the rest of the app consumes is strictly arrays-only.
import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE, decodeApiError } from "./client";

export type ArtifactStatus = "fresh" | "missing";

export type ArtifactFile = {
  path: string;
  exists: boolean;
  size?: number;
  mtime?: string;
};

export type ArtifactReport = {
  flowId: string;
  scenarioPath: string;
  generatedDir: string;
  status: ArtifactStatus;
  files: ArtifactFile[];
  missing: string[];
};

export type ClearResult = {
  flowId: string;
  removed: string[];
};

type RawArtifactReport = Omit<ArtifactReport, "files" | "missing"> & {
  files?: ArtifactFile[] | null;
  missing?: string[] | null;
};

type RawClearResult = Omit<ClearResult, "removed"> & {
  removed?: string[] | null;
};

async function getJson<T>(path: string): Promise<T> {
  const res = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), {
    method: "GET",
    cache: "no-store",
  });
  if (!res.ok) throw await decodeApiError(res);
  return (await res.json()) as T;
}

async function postJson<T>(path: string): Promise<T> {
  const res = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), {
    method: "POST",
    cache: "no-store",
  });
  if (!res.ok) throw await decodeApiError(res);
  return (await res.json()) as T;
}

async function deleteJson<T>(path: string): Promise<T> {
  const res = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), {
    method: "DELETE",
    cache: "no-store",
  });
  if (!res.ok) throw await decodeApiError(res);
  return (await res.json()) as T;
}

function qs(params: Record<string, string | undefined>): string {
  const out = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) if (v) out.set(k, v);
  const s = out.toString();
  return s ? `?${s}` : "";
}

function normaliseReport(raw: RawArtifactReport): ArtifactReport {
  return {
    ...raw,
    files: raw.files ?? [],
    missing: raw.missing ?? [],
  };
}

function normaliseClear(raw: RawClearResult): ClearResult {
  return { ...raw, removed: raw.removed ?? [] };
}

export async function fetchArtifactsStatus(
  flowId: string,
  opts: { scenarioId?: string; root?: string } = {},
): Promise<ArtifactReport> {
  const raw = await getJson<RawArtifactReport>(
    `/api/v1/flows/${encodeURIComponent(flowId)}/artifacts${qs({
      scenario: opts.scenarioId,
      root: opts.root,
    })}`,
  );
  return normaliseReport(raw);
}

export async function generateArtifacts(
  flowId: string,
  opts: { scenarioId?: string; root?: string } = {},
): Promise<ArtifactReport> {
  const raw = await postJson<RawArtifactReport>(
    `/api/v1/flows/${encodeURIComponent(flowId)}/artifacts:generate${qs({
      scenario: opts.scenarioId,
      root: opts.root,
    })}`,
  );
  return normaliseReport(raw);
}

export async function clearArtifacts(
  flowId: string,
  opts: { scenarioId?: string; root?: string } = {},
): Promise<ClearResult> {
  const raw = await deleteJson<RawClearResult>(
    `/api/v1/flows/${encodeURIComponent(flowId)}/artifacts${qs({
      scenario: opts.scenarioId,
      root: opts.root,
    })}`,
  );
  return normaliseClear(raw);
}

export type ScenarioArtifactsResult = {
  scenarioId: string;
  flows: ArtifactReport[];
};

export type ScenarioClearResult = {
  scenarioId: string;
  flows: ClearResult[];
};

export async function generateScenarioArtifacts(scenarioId: string): Promise<ScenarioArtifactsResult> {
  const raw = await postJson<{ scenarioId: string; flows?: RawArtifactReport[] | null }>(
    `/api/v1/scenarios/${encodeURIComponent(scenarioId)}/artifacts:generate`,
  );
  return { scenarioId: raw.scenarioId, flows: (raw.flows ?? []).map(normaliseReport) };
}

export async function clearScenarioArtifacts(scenarioId: string): Promise<ScenarioClearResult> {
  const raw = await deleteJson<{ scenarioId: string; flows?: RawClearResult[] | null }>(
    `/api/v1/scenarios/${encodeURIComponent(scenarioId)}/artifacts`,
  );
  return { scenarioId: raw.scenarioId, flows: (raw.flows ?? []).map(normaliseClear) };
}
