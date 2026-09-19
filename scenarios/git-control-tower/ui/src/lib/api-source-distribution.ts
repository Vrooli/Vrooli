import { API_BASE, buildApiUrl, buildRepoHeaders, handleResponse } from "./api-internals";
import type { SourceDistributionDetailResponse, SourceDistributionListResponse } from "./api-types-source-distribution";

export async function fetchSourceDistributions(repoId?: string, scenario?: string): Promise<SourceDistributionListResponse> {
  const params = new URLSearchParams();
  if (scenario) params.set("scenario", scenario);
  if (repoId) params.set("repository_context", repoId);
  const query = params.toString();
  const response = await fetch(buildApiUrl(`/source-distributions${query ? `?${query}` : ""}`, { baseUrl: API_BASE }), { headers: buildRepoHeaders(repoId), cache: "no-store" });
  return handleResponse<SourceDistributionListResponse>(response);
}

export async function fetchSourceDistribution(id: string): Promise<SourceDistributionDetailResponse> {
  const response = await fetch(buildApiUrl(`/source-distributions/${encodeURIComponent(id)}`, { baseUrl: API_BASE }), { headers: buildRepoHeaders(), cache: "no-store" });
  return handleResponse<SourceDistributionDetailResponse>(response);
}
