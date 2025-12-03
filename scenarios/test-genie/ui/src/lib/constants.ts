// Constants for Test Genie UI

import type { DashboardTab, RunsSubtab, PresetDetail, PhaseForGeneration } from "../types";

export const REPO_ROOT = "/home/matthalloran8/Vrooli";

export const DASHBOARD_TABS: DashboardTab[] = [
  { key: "dashboard", label: "Dashboard", description: "Quick actions and health overview" },
  { key: "runs", label: "Runs", description: "Scenarios and test history" },
  { key: "generate", label: "Generate", description: "AI-powered test generation" },
  { key: "docs", label: "Docs", description: "Documentation browser" }
];

export const RUNS_SUBTABS: RunsSubtab[] = [
  { key: "scenarios", label: "Scenarios" },
  { key: "history", label: "History" }
];

export const REQUESTED_TYPE_OPTIONS = ["unit", "integration", "performance", "vault", "regression"] as const;
export const PRIORITY_OPTIONS = ["low", "normal", "high", "urgent"] as const;
export const EXECUTION_PRESETS = ["quick", "smoke", "comprehensive"] as const;

export const PRESET_DETAILS: Record<string, PresetDetail> = {
  quick: {
    label: "Quick",
    description: "Structure + unit phases for quick sanity checks.",
    phases: ["structure", "unit"]
  },
  smoke: {
    label: "Smoke",
    description: "Structure + integration to verify core functionality.",
    phases: ["structure", "integration"]
  },
  comprehensive: {
    label: "Comprehensive",
    description: "Full coverage with all test phases.",
    phases: ["structure", "dependencies", "unit", "integration", "business", "performance"]
  }
};

export const PHASE_LABELS: Record<string, string> = {
  structure: "Structure validation",
  dependencies: "Dependency audit",
  unit: "Unit tests",
  integration: "Integration suite",
  playbooks: "E2E playbooks",
  business: "Business validation",
  performance: "Performance checks"
};

export const PHASES_FOR_GENERATION: PhaseForGeneration[] = [
  { key: "unit", label: "Unit Tests", docsPath: "/docs/phases/unit.md" },
  { key: "integration", label: "Integration Tests", docsPath: "/docs/phases/integration.md" },
  { key: "playbooks", label: "E2E Playbooks", docsPath: "/docs/phases/playbooks.md" },
  { key: "business", label: "Business Validation", docsPath: "/docs/phases/business.md" }
];

export const GENERATION_PRESETS = [
  {
    key: "bootstrap",
    label: "New scenario bootstrap",
    description: "Generate initial test suite for a new scenario"
  },
  {
    key: "coverage",
    label: "Add coverage for feature",
    description: "Add tests for a specific feature or module"
  },
  {
    key: "fix-failing",
    label: "Fix failing tests",
    description: "Generate fixes for failing test cases"
  }
];
