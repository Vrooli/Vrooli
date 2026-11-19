import { apiUrl, decodeApiError } from "./client";

export interface DecompositionSummary {
  track_sha256: string;
  analysis_job_id: string;
  title?: string;
  artist?: string;
  uploader?: string;
  received_at: string;
  review_count: number;
}
export interface DecompositionPage { items: DecompositionSummary[]; next_cursor?: string }
export interface Correction { start_seconds: number; end_seconds: number; label: string; comment?: string }
export interface Decomposition {
  track_sha256: string;
  analysis_job_id: string;
  source?: { title: string; artist?: string; uploader?: string };
  tool_result: { analysis?: Record<string, unknown>; [key: string]: unknown };
  reviews?: Array<{ verdict: string; note: string; corrections: Correction[]; created_at: string }>;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(apiUrl(path), { cache: "no-store", ...init });
  if (!response.ok) throw await decodeApiError(response);
  return response.json() as Promise<T>;
}
export function listDecompositions(cursor = ""): Promise<DecompositionPage> {
  const query = new URLSearchParams({ limit: "20" });
  if (cursor) query.set("cursor", cursor);
  return request(`/decompositions?${query}`);
}
export function getDecomposition(sha: string, jobID = ""): Promise<Decomposition> {
  const query = jobID ? `?analysis_job_id=${encodeURIComponent(jobID)}` : "";
  return request(`/decompositions/${encodeURIComponent(sha)}${query}`);
}
export function submitReview(sha: string, review: { analysis_job_id: string; verdict: string; note: string; timestamped_corrections: Correction[] }) {
  return request(`/decompositions/${encodeURIComponent(sha)}/reviews`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(review) });
}
export function sourceAudioUrl(sha: string): string {
  return apiUrl(`/decompositions/${encodeURIComponent(sha)}/audio`);
}
