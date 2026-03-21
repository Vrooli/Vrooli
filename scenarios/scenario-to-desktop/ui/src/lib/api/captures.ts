import { buildUrl, throwIfNotOk } from "./client";

export interface Capture {
  id: string;
  scenario_name: string;
  type: "screenshot" | "recording";
  filename: string;
  file_size_bytes: number;
  width?: number;
  height?: number;
  duration_ms?: number;
  source_session: string;
  created_at: string;
}

export interface CapturesSummary {
  count: number;
  total_bytes: number;
}

export async function listCaptures(scenario: string): Promise<Capture[]> {
  const res = await fetch(buildUrl(`/captures/${encodeURIComponent(scenario)}`));
  await throwIfNotOk(res);
  return res.json();
}

export async function getCapturesSummary(scenario: string): Promise<CapturesSummary> {
  const res = await fetch(buildUrl(`/captures/${encodeURIComponent(scenario)}/summary`));
  await throwIfNotOk(res);
  return res.json();
}

export function buildCaptureFileUrl(scenario: string, captureId: string): string {
  return buildUrl(`/captures/${encodeURIComponent(scenario)}/${encodeURIComponent(captureId)}/file`);
}

export async function deleteCapture(scenario: string, captureId: string): Promise<void> {
  const res = await fetch(
    buildUrl(`/captures/${encodeURIComponent(scenario)}/${encodeURIComponent(captureId)}`),
    { method: "DELETE" },
  );
  await throwIfNotOk(res);
}

export async function deleteAllCaptures(scenario: string): Promise<void> {
  const res = await fetch(buildUrl(`/captures/${encodeURIComponent(scenario)}`), {
    method: "DELETE",
  });
  await throwIfNotOk(res);
}

export function buildCapturesDownloadUrl(scenario: string, ids: string[]): string {
  return buildUrl(`/captures/${encodeURIComponent(scenario)}/download?ids=${ids.map(encodeURIComponent).join(",")}`);
}
