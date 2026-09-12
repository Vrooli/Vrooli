// Constants for Test Genie UI

import type { DashboardTab, RunsSubtab } from "../types";

// REPO_ROOT is now fetched dynamically from the API via fetchAppConfig()
// This placeholder is kept for backward compatibility but should not be used directly.
// Instead, use the useAppConfig hook or fetchAppConfig() to get the actual value.
export const REPO_ROOT_PLACEHOLDER = "${REPO_ROOT}";

export const DASHBOARD_TABS: DashboardTab[] = [
  { key: "dashboard", label: "Dashboard", description: "Quick actions and health overview" },
  { key: "runs", label: "Runs", description: "Scenarios and test history" },
  { key: "docs", label: "Docs", description: "Documentation browser" },
  { key: "health", label: "Self-Health", description: "Test Genie's own reliability, conformance, and performance" }
];

export const RUNS_SUBTABS: RunsSubtab[] = [
  { key: "scenarios", label: "Scenarios" },
  { key: "history", label: "History" }
];

export const EXECUTION_PRESETS = ["quick", "smoke", "comprehensive"] as const;
