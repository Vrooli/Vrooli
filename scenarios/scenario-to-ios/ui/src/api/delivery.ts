import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE } from "./client";

export type DeliveryState = string;

export interface ReadinessRung {
  id: string;
  title: string;
  state: DeliveryState;
  next_action: string;
  missing_capability?: string;
}

export interface ReadinessReport {
  rungs: ReadinessRung[];
}

export interface TargetReport {
  id: string;
  label: string;
  available: boolean;
  reason?: string;
  missing_capability?: string;
  next_action?: string;
}

export interface TargetInventory {
  targets: TargetReport[];
}

export interface ConformancePlan {
  chapters: Array<{ id: string; purpose?: string; expected?: string }>;
}

async function fetchReport<T>(path: string): Promise<T> {
  const response = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), {
    method: "GET",
    cache: "no-store",
  });
  if (!response.ok) {
    throw new Error(`iOS delivery report failed with HTTP ${response.status}`);
  }
  return (await response.json()) as T;
}

export const fetchReadiness = () => fetchReport<ReadinessReport>("/ios/readiness");
export const fetchTargets = () => fetchReport<TargetInventory>("/ios/targets");
export const fetchConformancePlan = () => fetchReport<ConformancePlan>("/ios/conformance-plan");
