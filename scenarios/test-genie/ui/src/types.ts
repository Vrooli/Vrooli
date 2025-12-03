/**
 * Shared type definitions for the Test Genie UI.
 */

import type { SuiteRequest, SuiteExecutionResult } from "./lib/api";

// Navigation types
export type DashboardTabKey = "dashboard" | "runs" | "generate" | "docs";
export type RunsSubtabKey = "scenarios" | "history";
export type ScenarioDetailTabKey = "overview" | "requirements" | "history";

// Tab/subtab configuration types
export interface DashboardTab {
  key: DashboardTabKey;
  label: string;
  description: string;
}

export interface RunsSubtab {
  key: RunsSubtabKey;
  label: string;
}

// Form state types
export interface QueueFormState {
  scenarioName: string;
  requestedTypes: string[];
  coverageTarget: number;
  priority: string;
  notes: string;
}

export interface ExecutionFormState {
  scenarioName: string;
  preset: string;
  failFast: boolean;
  suiteRequestId: string;
}

// Preset and phase types for generation
export interface PresetDetail {
  label: string;
  description: string;
  phases: string[];
}

export interface PhaseForGeneration {
  key: string;
  label: string;
  docsPath: string;
}

// Stats types
export interface CatalogStats {
  tracked: number;
  failing: number;
  pending: number;
  idle: number;
}

export interface FocusQueueStats {
  actionableCount: number;
  totalCount: number;
  mostRecentRequest: SuiteRequest | null;
  recentExecution: SuiteExecutionResult | null;
  failedExecution: SuiteExecutionResult | null;
  nextRequest: SuiteRequest | null;
}
