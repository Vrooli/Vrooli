import type { StatsFilter } from "../api/types";

export interface RunsNavigationFilter extends StatsFilter {
  workloadKey?: string;
  toolName?: string;
  errorCode?: string;
}

export function runsLink(filter: RunsNavigationFilter = {}): string {
  const now = new Date();
  const hours = { "6h": 6, "12h": 12, "24h": 24, "7d": 168, "30d": 720 }[filter.preset ?? "7d"] ?? 168;
  const from = filter.start ?? new Date(now.getTime() - hours * 60 * 60 * 1000).toISOString();
  const to = filter.end ?? now.toISOString();
  const params = new URLSearchParams({ from, to });
  for (const [key, value] of Object.entries({ model: filter.model, runnerType: filter.runnerType, profileId: filter.profileId, tagPrefix: filter.tagPrefix, workloadKey: filter.workloadKey, toolName: filter.toolName, errorCode: filter.errorCode })) {
    if (value) params.set(key, value);
  }
  return `/runs?${params.toString()}`;
}
