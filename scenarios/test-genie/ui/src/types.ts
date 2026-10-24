/**
 * Shared type definitions for the Test Genie UI.
 */

// Navigation types
export type DashboardTabKey = "dashboard" | "runs" | "docs" | "health";
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

export interface ExecutionFormState {
  scenarioName: string;
  preset: string;
  failFast: boolean;
}

// Stats types
export interface CatalogStats {
  tracked: number;
  failing: number;
  idle: number;
}
