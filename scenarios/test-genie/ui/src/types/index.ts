// Shared TypeScript types for Test Genie UI
import type { SuiteExecutionResult, SuiteRequest } from "../lib/api";

export type DashboardTabKey = "dashboard" | "runs" | "generate" | "docs";
export type RunsSubtabKey = "scenarios" | "history";
export type ScenarioDetailTabKey = "overview" | "requirements" | "history";

export interface DashboardTab {
  key: DashboardTabKey;
  label: string;
  description: string;
}

export interface RunsSubtab {
  key: RunsSubtabKey;
  label: string;
}

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

export interface PresetDetail {
  label: string;
  description: string;
  phases: string[];
}

export interface PhaseForGeneration {
  key: string;
  label: string;
  description: string;
  docsPath: string;
}

export interface QuickNavItem {
  key: string;
  eyebrow: string;
  title: string;
  description: string;
  statValue: string;
  statLabel: string;
  actionLabel: string;
  onClick: () => void;
}

export interface SignalNavItem {
  key: string;
  label: string;
  anchor: string;
}

export interface FocusQueueStats {
  actionableCount: number;
  totalCount: number;
  mostRecentRequest: SuiteRequest | null;
  recentExecution: SuiteExecutionResult | null;
  failedExecution: SuiteExecutionResult | null;
  nextRequest: SuiteRequest | null;
}

export interface CatalogStats {
  tracked: number;
  pending: number;
  failing: number;
  idle: number;
}

// Re-export API types for convenience
export type {
  QueueSnapshot,
  PhaseSummary,
  PhaseExecutionResult,
  SuiteExecutionResult,
  ApiHealthResponse,
  SuiteRequest,
  QueueSuiteRequestInput,
  ExecuteSuiteInput,
  ScenarioSummary
} from "../lib/api";
