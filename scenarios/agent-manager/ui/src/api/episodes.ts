import { getApiBaseUrl } from "../lib/utils";
import { measuresClient } from "../features/stats/api/measuresClient";

export type Availability = { state: string; reason?: string };
export type Episode = { episodeId: string; runId: string; pattern: string; causeScope: string; severity: string; turns: number; tokens: number; wallClockMs: number; suspectedOwnerScenario?: string; suspectedOwnerCommand?: string; ownerConfidence: string; evidenceEventIds: string[] };
export type EpisodeSignal = { fingerprint: string; occurrences: number; distinctRuns: number; summedCostMs: number; confidence: string; representativeRunIds: string[] };
export type EpisodeCohort = { availability: Availability; signals: EpisodeSignal[] };
export type Ledger = { ledgerAvailability: Availability; projectionAvailability: Availability; ledgerTargetRollups: Array<{ targetScenario: string; calls: number; failures: number; totalDurationMs: number; medianDurationMs: number }> };

async function request<T>(path: string): Promise<T> {
  const response = await fetch(`${getApiBaseUrl()}${path}`);
  if (!response.ok) throw new Error(`Request failed: ${response.status}`);
  return response.json() as Promise<T>;
}

// Recurring episodes come from the Measures EpisodeCohort procedure (this week's runs).
export const getEpisodeCohort = async (): Promise<EpisodeCohort> => {
  const response = await measuresClient.episodeCohort({ limit: 100 });
  return {
    availability: { state: response.availabilityState, reason: response.availabilityReason || undefined },
    signals: response.signals.map((signal) => ({ fingerprint: signal.fingerprint, occurrences: Number(signal.occurrences), distinctRuns: Number(signal.distinctRuns), summedCostMs: Number(signal.summedCostMs), confidence: signal.confidence, representativeRunIds: [...signal.representativeRunIds] })),
  };
};
export const getEpisodes = (runId: string) => request<{ episodes: Episode[] }>(`/runs/${encodeURIComponent(runId)}/episodes`);
export const getLedger = (runId: string) => request<Ledger>(`/runs/${encodeURIComponent(runId)}/ledger`);
