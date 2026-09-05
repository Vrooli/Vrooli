// Mirrors the Phase 3 operational stats response shape.
//
// Source of truth on the Go side:
//   scenarios/agent-manager/api/internal/stats/types.go

import type { HistoryWindow } from "../../../components/stats/HistoryWindow";

export type { HistoryWindow };

export interface FallbackPair {
  from: string;
  to: string;
  reason: string;
  count: number;
}

export interface FallbackInsights {
  generated_at: string;
  history: HistoryWindow;
  event_count: number;
  runner_attempts: number;
  runner_exhausted: number;
  runner_by_reason: Record<string, number>;
  runner_by_pair: FallbackPair[];
  runner_chain_depth: Record<string, number>;
  model_attempts: number;
  model_exhausted: number;
  model_by_reason: Record<string, number>;
  model_by_pair: FallbackPair[];
  model_chain_depth: Record<string, number>;
  model_by_preset: Record<string, number>;
}

export interface ModelHealthEntry {
  runner: string;
  model: string;
  status: string;
  reason?: string;
  message?: string;
  observed_at: string;
  transitions_observed: number;
}

export interface RunnerHealthEntry {
  runner: string;
  status: string;
  reason?: string;
  message?: string;
  observed_at: string;
  transitions_observed: number;
}

export interface HealthSummary {
  generated_at: string;
  history: HistoryWindow;
  models: ModelHealthEntry[];
  runners: RunnerHealthEntry[];
  failing_last_hour: ModelHealthEntry[];
}
