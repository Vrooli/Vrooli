// DOC: docs/reference/api-endpoints.md#scenario-list
// DOC: docs/reference/api-endpoints.md#documentation-health
import type {
  DocFileSearchResult,
  DocTextSearchMatch,
  DocUnifiedSearchResponse,
  ScenarioDocHealth,
  ScenarioSummary,
  MisplacedDoc,
} from "../services/documentationApi";

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
  fixCategory: string;
};

export type DocumentationSummaryView = {
  totalScenarios: number;
  coverageLabel: string;
  coveragePercentLabel: string;
  coverageTone: HealthTone;
  averageHealthLabel: string;
  averageHealthTone: HealthTone;
  manifestCoverageLabel: string;
  lastModifiedLabel: string;
};

export type DocSearchResultView = {
  id: string;
  title: string;
  path: string;
  snippet?: string;
  meta: string[];
  scoreLabel?: string;
  sourceLabel?: string;
};

export type DocSearchViewModel = {
  results: DocSearchResultView[];
  totalResults: number;
  tookMsLabel: string;
  displayQuery: string;
  hasResults: boolean;
};

const toSafeString = (value: unknown): string | undefined =>
  typeof value === "string" && value.trim() ? value : undefined;

const formatDate = (value?: string) => {
  if (!value) return "Unknown";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "Unknown" : date.toLocaleDateString();
};

const formatPercent = (value: number) => `${Math.round(value * 100)}%`;

const formatScore = (value?: number) => {
  if (value === undefined || !Number.isFinite(value)) return undefined;
  return `${(value * 100).toFixed(1)}%`;
};

const formatSize = (bytes?: number) => {
  if (bytes === undefined || !Number.isFinite(bytes)) return "";
  if (bytes < 1024) return `${bytes} B`;
  const kb = bytes / 1024;
  if (kb < 1024) return `${kb.toFixed(1)} KB`;
  const mb = kb / 1024;
  return `${mb.toFixed(1)} MB`;
};

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
    fixCategory: typeof health?.fix_category === "string" ? health.fix_category : "none",
  };
}

export function buildDocumentationSummaryView(
  scenarios: ScenarioSummary[] | undefined,
): DocumentationSummaryView {
  const normalized = Array.isArray(scenarios) ? scenarios : [];
  const total = normalized.length;
  const withDocs = normalized.filter((scenario) => scenario.doc_count > 0).length;
  const withManifest = normalized.filter((scenario) => scenario.has_manifest).length;
  const avgHealth =
    total === 0
      ? 0
      : normalized.reduce((sum, scenario) => sum + (scenario.health_score || 0), 0) / total;
  const latestModified = normalized.reduce((latest, scenario) => {
    const value = scenario.last_modified;
    if (!value) return latest;
    return !latest || value > latest ? value : latest;
  }, "" as string);

  const coverageRatio = total === 0 ? 0 : withDocs / total;
  const manifestRatio = total === 0 ? 0 : withManifest / total;

  return {
    totalScenarios: total,
    coverageLabel: `${withDocs.toLocaleString()} of ${total.toLocaleString()} documented`,
    coveragePercentLabel: formatPercent(coverageRatio),
    coverageTone: getHealthTone(coverageRatio),
    averageHealthLabel: formatPercent(avgHealth),
    averageHealthTone: getHealthTone(avgHealth),
    manifestCoverageLabel: `${formatPercent(manifestRatio)} have manifests`,
    lastModifiedLabel: formatDate(latestModified),
  };
}

export function buildFileSearchViewModel(
  results: DocFileSearchResult[] | undefined,
  pattern: string,
): DocSearchViewModel {
  const normalized = Array.isArray(results) ? results : [];
  const queryLabel = pattern.trim() || "pattern";
  const views = normalized.map((result, index) => {
    const title = result.relative_path || result.path;
    const meta = [
      result.scenario ? `Scenario: ${result.scenario}` : "",
      result.doc_type ? `Type: ${result.doc_type}` : "",
      formatSize(result.size),
    ].filter(Boolean);
    return {
      id: `${result.path}-${index}`,
      title,
      path: result.path,
      snippet: result.content_preview,
      meta,
    };
  });

  return {
    results: views,
    totalResults: views.length,
    tookMsLabel: "?ms",
    displayQuery: queryLabel,
    hasResults: views.length > 0,
  };
}

export function buildTextSearchViewModel(
  results: DocTextSearchMatch[] | undefined,
  query: string,
): DocSearchViewModel {
  const normalized = Array.isArray(results) ? results : [];
  const queryLabel = query.trim() || "query";
  const views = normalized.map((result, index) => {
    const title = result.relative_path || result.path;
    const meta = [
      result.scenario ? `Scenario: ${result.scenario}` : "",
      result.line_number ? `Line: ${result.line_number}` : "",
    ].filter(Boolean);
    const snippet = [result.context_before, result.content, result.context_after]
      .filter(Boolean)
      .join("\n");
    return {
      id: `${result.path}-${index}`,
      title,
      path: result.path,
      snippet,
      meta,
    };
  });

  return {
    results: views,
    totalResults: views.length,
    tookMsLabel: "?ms",
    displayQuery: queryLabel,
    hasResults: views.length > 0,
  };
}

export function buildUnifiedSearchViewModel(
  response: DocUnifiedSearchResponse | null | undefined,
  fallbackQuery: string,
): DocSearchViewModel {
  const normalized = Array.isArray(response?.results) ? response?.results : [];
  const displayQuery =
    toSafeString(response?.query) ?? (fallbackQuery.trim() ? fallbackQuery.trim() : "query");
  const views = normalized.map((result, index) => {
    const title = result.relative_path || result.path || result.id || `Result ${index + 1}`;
    const meta = [
      result.source ? `Source: ${result.source}` : "",
      result.scenario ? `Scenario: ${result.scenario}` : "",
      result.line_number ? `Line: ${result.line_number}` : "",
      result.doc_type ? `Type: ${result.doc_type}` : "",
    ].filter(Boolean);
    return {
      id: `${result.path ?? result.id ?? index}`,
      title,
      path: result.path ?? "",
      snippet: result.snippet ?? result.content,
      meta,
      scoreLabel: formatScore(result.score),
      sourceLabel: result.source,
    };
  });

  return {
    results: views,
    totalResults: views.length,
    tookMsLabel: response?.took_ms ? `${response.took_ms}ms` : "?ms",
    displayQuery,
    hasResults: views.length > 0,
  };
}

const toMisplacedDocView = (doc: MisplacedDoc): MisplacedDocView => ({
  actualPath: toSafeString(doc.actual_path) ?? "",
  expectedPath: toSafeString(doc.expected_path) ?? "",
  docType: toSafeString(doc.doc_type) ?? "",
  severity: toSafeString(doc.severity) ?? "info",
});
