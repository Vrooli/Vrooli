import { getApiBaseUrl } from "../lib/utils";

export type Availability = { state: string; detail?: string };
export type Episode = { episodeId: string; runId: string; pattern: string; causeScope: string; severity: string; turns: number; tokens: number; wallClockMs: number; suspectedOwnerScenario?: string; suspectedOwnerCommand?: string; ownerConfidence: string; evidenceEventIds: string[] };
export type EpisodeSignal = { fingerprint: string; occurrences: number; distinctRuns: number; summedCostMs: number; confidence: string; representativeRunIds: string[] };
export type EpisodeCohort = { availability: Availability; signals: EpisodeSignal[] };
export type Ledger = { ledgerAvailability: Availability; projectionAvailability: Availability; ledgerTargetRollups: Array<{ targetScenario: string; calls: number; failures: number; totalDurationMs: number; medianDurationMs: number }> };

async function request<T>(path: string): Promise<T> {
  const response = await fetch(`${getApiBaseUrl()}${path}`);
  if (!response.ok) throw new Error(`Request failed: ${response.status}`);
  return response.json() as Promise<T>;
}

export const getEpisodeCohort = () => request<EpisodeCohort>("/runs/episode-cohort?limit=100");
export const getEpisodes = (runId: string) => request<{ episodes: Episode[] }>(`/runs/${encodeURIComponent(runId)}/episodes`);
export const getLedger = (runId: string) => request<Ledger>(`/runs/${encodeURIComponent(runId)}/ledger`);
