import {
  AlertTriangle,
  CheckCircle2,
  FileCheck2,
  GitBranch,
  PlayCircle,
  Search,
  ShieldCheck,
  Wrench,
  type LucideIcon,
} from "lucide-react";

export type AssetKind = "Case" | "Flow" | "Action" | "Seed";
export type Safety = "Observer" | "Mutating guarded" | "Registry-only";
export type FindingSeverity = "Critical" | "High" | "Medium" | "Low";

export interface Stat {
  label: string;
  value: string;
  detail: string;
  icon: LucideIcon;
}

export interface Asset {
  kind: AssetKind;
  name: string;
  path: string;
  safety: Safety;
  requirements: string;
  status: "Ready" | "Needs metadata" | "Stale registry";
}

export interface SearchResult {
  type: "workflow.flow" | "workflow.test" | "workflow.fragment";
  name: string;
  intent: string;
  safety: Safety;
  rank: string;
}

export interface RunEvent {
  time: string;
  label: string;
  detail: string;
  status: "passed" | "warned" | "blocked";
}

export interface Finding {
  id: string;
  severity: FindingSeverity;
  asset: string;
  summary: string;
  remediation: string;
}

export interface FixPreview {
  rule: string;
  file: string;
  change: string;
  risk: "Mechanical" | "Review required";
}

export const stats: Stat[] = [
  {
    label: "Maturity",
    value: "L3 Safe",
    detail: "Traceable workflows with mutating guards",
    icon: ShieldCheck,
  },
  {
    label: "Assets",
    value: "27",
    detail: "Cases, flows, actions, and seeds indexed",
    icon: GitBranch,
  },
  {
    label: "Findings",
    value: "4",
    detail: "No critical blockers in latest static pass",
    icon: AlertTriangle,
  },
  {
    label: "Last run",
    value: "Passed",
    detail: "Static validation and provider response",
    icon: CheckCircle2,
  },
];

export const assets: Asset[] = [
  {
    kind: "Case",
    name: "Provider validation smoke",
    path: "bas/cases/provider/validate-scenario.json",
    safety: "Observer",
    requirements: "REQ-P4-001, REQ-P5-002",
    status: "Ready",
  },
  {
    kind: "Flow",
    name: "Workflow search inspection",
    path: "bas/flows/search/inspect-result.json",
    safety: "Observer",
    requirements: "REQ-P6-001",
    status: "Ready",
  },
  {
    kind: "Flow",
    name: "Apply deterministic fix",
    path: "bas/flows/fixes/apply-registry-rebuild.json",
    safety: "Mutating guarded",
    requirements: "REQ-P3-004, REQ-P4-003",
    status: "Needs metadata",
  },
  {
    kind: "Action",
    name: "Open scenario dashboard",
    path: "bas/actions/navigation/open-dashboard.json",
    safety: "Observer",
    requirements: "Fragment",
    status: "Ready",
  },
  {
    kind: "Seed",
    name: "Routed workflow fixture",
    path: "bas/seeds/routed-fixture.json",
    safety: "Registry-only",
    requirements: "REQ-P4-003",
    status: "Stale registry",
  },
];

export const searchResults: SearchResult[] = [
  {
    type: "workflow.flow",
    name: "Inspect validation result",
    intent: "Open a scenario, validate workflow assets, and inspect findings.",
    safety: "Observer",
    rank: "Best for run/do queries",
  },
  {
    type: "workflow.test",
    name: "Prove routed mutating safety",
    intent: "Validate that mutating workflow cases refuse without isolation proof.",
    safety: "Mutating guarded",
    rank: "Best for prove/validate queries",
  },
  {
    type: "workflow.fragment",
    name: "Select scenario action",
    intent: "Reusable navigation action used by inventory and run-detail flows.",
    safety: "Observer",
    rank: "Hidden unless fragments are requested",
  },
];

export const timeline: RunEvent[] = [
  {
    time: "00:00",
    label: "Catalog scan",
    detail: "Cases, flows, actions, seeds, and registry entries loaded.",
    status: "passed",
  },
  {
    time: "00:04",
    label: "Static validation",
    detail: "Maturity, metadata, requirements, selectors, and subflows checked.",
    status: "warned",
  },
  {
    time: "00:09",
    label: "Execution gate",
    detail: "Mutating flows held until confirmation and routed isolation are present.",
    status: "blocked",
  },
  {
    time: "00:11",
    label: "Artifacts written",
    detail: "latest.json and timeline.json references saved for the run.",
    status: "passed",
  },
];

export const findings: Finding[] = [
  {
    id: "WH-META-001",
    severity: "Medium",
    asset: "Apply deterministic fix",
    summary: "Mutating flow needs explicit confirmation metadata.",
    remediation: "Add safety confirmation text and routed isolation evidence before execution.",
  },
  {
    id: "WH-REG-002",
    severity: "Low",
    asset: "Routed workflow fixture",
    summary: "Registry entry is stale against scanned seed catalog.",
    remediation: "Preview and apply the registry rebuild fix.",
  },
  {
    id: "WH-REQ-003",
    severity: "Low",
    asset: "Workflow search inspection",
    summary: "Requirement evidence is present but lacks latest run artifact.",
    remediation: "Run the observer case and attach the generated artifact pointer.",
  },
];

export const fixPreviews: FixPreview[] = [
  {
    rule: "registry-rebuild",
    file: "bas/registry.json",
    change: "Reorder scanned entries and remove stale seed reference.",
    risk: "Mechanical",
  },
  {
    rule: "metadata-stub",
    file: "bas/flows/fixes/apply-registry-rebuild.json",
    change: "Add missing safety and requirement metadata stubs.",
    risk: "Review required",
  },
  {
    rule: "reset-normalize",
    file: "bas/cases/provider/validate-scenario.json",
    change: "Normalize legacy reset=database to reset=full.",
    risk: "Mechanical",
  },
];

export const pageIcons = {
  inventory: FileCheck2,
  search: Search,
  runs: PlayCircle,
  fixes: Wrench,
};

export const uiText = {
  overview: {
    scenarioLabel: "Scenario",
    scenarios: ["workflow-health", "browser-automation-studio", "test-genie"] as const,
    readySuffix: "ready",
    assetPosture: "Asset posture",
    latestRun: "Latest run",
    topFindings: "Top findings",
    assetHeaders: ["Kind", "Asset", "Safety", "Status"] as const,
  },
  inventory: {
    tabLabels: ["Cases", "Flows", "Actions", "Seeds", "Dependencies"] as const,
    catalog: "Catalog",
    countSuffix: "s",
    headers: ["Kind", "Name", "Path", "Requirements", "Safety", "Status"] as const,
  },
  search: {
    query: "Query",
    type: "Type",
    allLeaves: "All workflow leaves",
    flows: "Flows",
    tests: "Tests",
    fragments: "Fragments",
    rank: "Rank",
  },
  runs: {
    queue: "Run queue",
    timeline: "Timeline",
    runNames: ["Static validation", "Observer execution", "Mutating safety gate"] as const,
    runPrefix: "Run WH-2026-07-02-0",
    scenarioSuffix: " - workflow-health",
  },
  findings: {
    headers: ["Finding", "Severity", "Asset", "Summary", "Remediation"] as const,
  },
  fixes: {
    preview: "Preview",
    previewDescription: "Deterministic fixes are presented before any write happens.",
    applySelected: "Apply selected",
  },
};
