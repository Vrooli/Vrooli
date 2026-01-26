// DOC: docs/reference/api-endpoints.md#scenario-list
// DOC: docs/reference/api-endpoints.md#documentation-health
import type { ScenarioDocHealth, ScenarioSummary, MisplacedDoc } from "../services/documentationApi";

export type HealthTone = "good" | "medium" | "poor";

export type ScenarioSummaryView = {
  name: string;
  path: string;
  docCountLabel: string;
  healthScoreLabel: string;
  healthTone: HealthTone;
  hasManifest: boolean;
  hasReadme: boolean;
  lastModifiedLabel: string;
};

export type MisplacedDocView = {
  actualPath: string;
  expectedPath: string;
  docType: string;
  severity: string;
};

export type DocHealthViewModel = {
  healthScoreLabel: string;
  healthTone: HealthTone;
  totalDocsLabel: string;
  missingDocs: string[];
  extraDocs: string[];
  misplacedDocs: MisplacedDocView[];
  warningCount: number;
  hasIssues: boolean;
  canAutoFix: boolean;
};

const toSafeString = (value: unknown): string | undefined =>
  typeof value === "string" && value.trim() ? value : undefined;

const formatDate = (value?: string) => {
  if (!value) return "Unknown";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "Unknown" : date.toLocaleDateString();
};

const formatPercent = (value: number) => `${Math.round(value * 100)}%`;

export const getHealthTone = (score: number): HealthTone => {
  if (score >= 0.9) return "good";
  if (score >= 0.75) return "medium";
  return "poor";
};

export function buildScenarioSummaryViews(
  scenarios: ScenarioSummary[] | undefined,
  filter: string,
): ScenarioSummaryView[] {
  const normalized = Array.isArray(scenarios) ? scenarios : [];
  const trimmedFilter = filter.trim().toLowerCase();
  const filtered = trimmedFilter
    ? normalized.filter((scenario) => {
        const name = scenario.name.toLowerCase();
        const path = scenario.path.toLowerCase();
        return name.includes(trimmedFilter) || path.includes(trimmedFilter);
      })
    : normalized;

  return filtered.map((scenario) => {
    const score = typeof scenario.health_score === "number" ? scenario.health_score : 0;
    const docCount = typeof scenario.doc_count === "number" ? scenario.doc_count : 0;
    return {
      name: scenario.name,
      path: scenario.path,
      docCountLabel: `${docCount.toLocaleString()} docs`,
      healthScoreLabel: formatPercent(score),
      healthTone: getHealthTone(score),
      hasManifest: scenario.has_manifest,
      hasReadme: scenario.has_readme,
      lastModifiedLabel: formatDate(scenario.last_modified),
    };
  });
}

export function buildDocHealthViewModel(health?: ScenarioDocHealth | null): DocHealthViewModel {
  const score = typeof health?.health_score === "number" ? health.health_score : 0;
  const totalDocs = typeof health?.total_docs === "number" ? health.total_docs : 0;
  const missingDocs = Array.isArray(health?.missing_docs) ? health?.missing_docs : [];
  const extraDocs = Array.isArray(health?.extra_docs) ? health?.extra_docs : [];
  const misplacedDocs = Array.isArray(health?.misplaced_docs)
    ? health?.misplaced_docs.map(toMisplacedDocView)
    : [];
  const warningCount = Array.isArray(health?.warnings) ? health?.warnings.length : 0;

  return {
    healthScoreLabel: formatPercent(score),
    healthTone: getHealthTone(score),
    totalDocsLabel: `${totalDocs.toLocaleString()} docs`,
    missingDocs,
    extraDocs,
    misplacedDocs,
    warningCount,
    hasIssues: missingDocs.length > 0 || extraDocs.length > 0 || misplacedDocs.length > 0,
    canAutoFix: Boolean(health?.can_auto_fix),
  };
}

const toMisplacedDocView = (doc: MisplacedDoc): MisplacedDocView => ({
  actualPath: toSafeString(doc.actual_path) ?? "",
  expectedPath: toSafeString(doc.expected_path) ?? "",
  docType: toSafeString(doc.doc_type) ?? "",
  severity: toSafeString(doc.severity) ?? "info",
});
