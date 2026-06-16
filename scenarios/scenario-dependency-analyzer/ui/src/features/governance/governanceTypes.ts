import type { ApprovedDependencyRecord } from "../../api/governance";

export type GovernanceView = "findings" | "dependencies" | "scenarios" | "records";
export type GovernanceSeverity = "all" | "error" | "warning" | "info";
export type GovernanceState = "all" | "approved" | "approved_with_constraints" | "needs_review" | "denied" | "deprecated" | "unrecorded";

export interface GovernanceFilters {
  query: string;
  ecosystem: string;
  severity: GovernanceSeverity;
  state: GovernanceState;
  scenario: string;
}

export const defaultGovernanceFilters: GovernanceFilters = {
  query: "",
  ecosystem: "all",
  severity: "all",
  state: "all",
  scenario: "all"
};

export function governanceRecordFromDecision(options: {
  ecosystem: string;
  packageName: string;
  state: "approved" | "denied" | "deprecated" | "needs_review";
  versionRange: string;
  rationale: string;
  approvedBy: string;
  replacement?: string;
  allowedScenarios?: string[];
}): ApprovedDependencyRecord {
  const now = new Date().toISOString().slice(0, 10);
  return {
    $typeName: "vrooli.scenario_dependency_analyzer.v1.dependency_governance.ApprovedDependencyRecord",
    ecosystem: options.ecosystem,
    packageName: options.packageName,
    versionRange: options.versionRange,
    state: options.state,
    allowedSurfaces: [],
    useCases: [],
    rationale: options.rationale,
    approvedBy: options.approvedBy,
    approvedDate: now,
    lastReviewed: now,
    reviewExpires: "",
    licenseNotes: "",
    securityNotes: "",
    exampleScenarios: options.allowedScenarios ?? [],
    replacement: options.replacement ?? "",
    keywords: [],
    allowedScenarios: options.allowedScenarios ?? [],
    deniedScenarios: [],
    allowedDependencyGroups: []
  };
}
