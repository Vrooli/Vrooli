// DOC: docs/reference/api-endpoints.md#scenario-list
// DOC: docs/reference/api-endpoints.md#scenario-documentation-tree
// DOC: docs/reference/api-endpoints.md#documentation-health
// DOC: docs/reference/api-endpoints.md#documentation-viewer
// DOC: docs/reference/api-endpoints.md#documentation-deep-search
import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";

type JsonRecord = Record<string, unknown>;

const API_BASE = resolveApiBase({ appendSuffix: false });

const isRecord = (value: unknown): value is JsonRecord =>
  typeof value === "object" && value !== null && !Array.isArray(value);

const toNonEmptyString = (value: unknown): string | undefined =>
  typeof value === "string" && value.trim() ? value : undefined;

const toFiniteNumber = (value: unknown): number | undefined =>
  typeof value === "number" && Number.isFinite(value) ? value : undefined;

const toBoolean = (value: unknown): boolean | undefined =>
  typeof value === "boolean" ? value : undefined;

const parseErrorMessage = (payload: unknown, fallback: string) => {
  if (isRecord(payload) && typeof payload.error === "string" && payload.error.trim()) {
    return payload.error;
  }
  return fallback;
};

export interface ScenarioSummary {
  name: string;
  path: string;
  doc_count: number;
  health_score: number;
  has_manifest: boolean;
  has_readme: boolean;
  last_modified?: string;
}

export interface DocWarning {
  type: string;
  message: string;
  expected_path?: string;
  severity: string;
}

export interface DocTreeNode {
  name: string;
  path: string;
  type: "file" | "directory";
  doc_type?: string;
  size?: number;
  modified_at?: string;
  warning?: DocWarning;
  children?: DocTreeNode[];
}

export interface MisplacedDoc {
  actual_path: string;
  expected_path: string;
  doc_type: string;
  severity: string;
}

export interface ScenarioDocHealth {
  scenario_name: string;
  health_score: number;
  total_docs: number;
  misplaced_docs: MisplacedDoc[];
  missing_docs: string[];
  extra_docs: string[];
  warnings: DocWarning[];
  can_auto_fix: boolean;
}

export interface DocResetConfig {
  max_age_days?: number;
  keep_min_entries?: number;
}

export interface DocContentResponse {
  path: string;
  content: string;
  format: string;
  doc_type?: string;
  size: number;
  modified_at: string;
  can_reset: boolean;
  reset_config?: DocResetConfig;
}

export interface DocResetRequest {
  path: string;
  max_age_days?: number;
  keep_min_entries?: number;
  preview_only?: boolean;
}

export interface DocResetResponse {
  path: string;
  doc_type: string;
  removed_count: number;
  kept_count: number;
  removed_entries?: string[];
  new_content?: string;
  preview_only: boolean;
}

export interface DeepSearchRequest {
  query: string;
  scope?: string;
  scenario?: string;
  base_path?: string;
  max_results?: number;
  follow_refs?: boolean;
  timeout_seconds?: number;
}

export interface DeepSearchResult {
  path: string;
  relevance: number;
  summary: string;
  match_reason: string;
  references?: string[];
  snippet?: string;
}

export interface DeepSearchJob {
  job_id: string;
  status: string;
  progress?: string;
  started_at?: string;
  completed_at?: string;
  results?: DeepSearchResult[];
  error?: string;
}

export async function fetchScenarioSummaries(): Promise<ScenarioSummary[]> {
  const url = buildApiUrl("/api/v1/scenarios", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });

  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    throw new Error(parseErrorMessage(errorPayload, `Scenario list failed: ${res.status}`));
  }

  const data = (await res.json().catch(() => null)) as unknown;
  if (!Array.isArray(data)) {
    throw new Error("Invalid scenario list response");
  }

  return data.flatMap((item, index) => {
    if (!isRecord(item)) return [];
    return [
      {
        name: toNonEmptyString(item.name) ?? `scenario-${index + 1}`,
        path: toNonEmptyString(item.path) ?? "",
        doc_count: toFiniteNumber(item.doc_count) ?? 0,
        health_score: toFiniteNumber(item.health_score) ?? 0,
        has_manifest: toBoolean(item.has_manifest) ?? false,
        has_readme: toBoolean(item.has_readme) ?? false,
        last_modified: toNonEmptyString(item.last_modified),
      },
    ];
  });
}

export async function fetchScenarioDocTree(scenarioName: string): Promise<DocTreeNode> {
  const trimmed = scenarioName.trim();
  if (!trimmed) {
    throw new Error("Scenario name is required");
  }
  const url = buildApiUrl(`/api/v1/scenarios/${encodeURIComponent(trimmed)}/docs`, {
    baseUrl: API_BASE,
  });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });

  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    throw new Error(parseErrorMessage(errorPayload, `Doc tree failed: ${res.status}`));
  }

  const data = (await res.json().catch(() => null)) as unknown;
  return normalizeDocTreeNode(data);
}

export async function fetchScenarioDocHealth(scenarioName: string): Promise<ScenarioDocHealth> {
  const trimmed = scenarioName.trim();
  if (!trimmed) {
    throw new Error("Scenario name is required");
  }
  const url = buildApiUrl(`/api/v1/scenarios/${encodeURIComponent(trimmed)}/docs/health`, {
    baseUrl: API_BASE,
  });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });

  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    throw new Error(parseErrorMessage(errorPayload, `Doc health failed: ${res.status}`));
  }

  const data = (await res.json().catch(() => null)) as unknown;
  if (!isRecord(data)) {
    throw new Error("Invalid doc health response");
  }

  return {
    scenario_name: toNonEmptyString(data.scenario_name) ?? trimmed,
    health_score: toFiniteNumber(data.health_score) ?? 0,
    total_docs: toFiniteNumber(data.total_docs) ?? 0,
    misplaced_docs: normalizeMisplacedDocs(data.misplaced_docs),
    missing_docs: normalizeStringList(data.missing_docs),
    extra_docs: normalizeStringList(data.extra_docs),
    warnings: normalizeWarnings(data.warnings),
    can_auto_fix: toBoolean(data.can_auto_fix) ?? false,
  };
}

export async function fetchDocContent(path: string, format?: string): Promise<DocContentResponse> {
  const trimmed = path.trim();
  if (!trimmed) {
    throw new Error("Document path is required");
  }
  const url = new URL(buildApiUrl("/api/v1/docs/content", { baseUrl: API_BASE }));
  url.searchParams.set("path", trimmed);
  if (format && format.trim()) {
    url.searchParams.set("format", format.trim());
  }

  const res = await fetch(url.toString(), {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });

  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    throw new Error(parseErrorMessage(errorPayload, `Doc content failed: ${res.status}`));
  }

  const data = (await res.json().catch(() => null)) as unknown;
  if (!isRecord(data)) {
    throw new Error("Invalid doc content response");
  }

  return {
    path: toNonEmptyString(data.path) ?? trimmed,
    content: toNonEmptyString(data.content) ?? "",
    format: toNonEmptyString(data.format) ?? "raw",
    doc_type: toNonEmptyString(data.doc_type),
    size: toFiniteNumber(data.size) ?? 0,
    modified_at: toNonEmptyString(data.modified_at) ?? "",
    can_reset: toBoolean(data.can_reset) ?? false,
    reset_config: normalizeResetConfig(data.reset_config),
  };
}

export async function startDeepSearch(payload: DeepSearchRequest): Promise<DeepSearchJob> {
  const url = buildApiUrl("/api/v1/docs/search/deep", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    throw new Error(parseErrorMessage(errorPayload, `Deep search failed: ${res.status}`));
  }
  const data = (await res.json().catch(() => null)) as unknown;
  return normalizeDeepSearchJob(data);
}

export async function fetchDeepSearchJob(jobId: string): Promise<DeepSearchJob> {
  const trimmed = jobId.trim();
  if (!trimmed) {
    throw new Error("Job ID is required");
  }
  const url = buildApiUrl(`/api/v1/docs/search/deep/${encodeURIComponent(trimmed)}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    throw new Error(parseErrorMessage(errorPayload, `Deep search status failed: ${res.status}`));
  }
  const data = (await res.json().catch(() => null)) as unknown;
  return normalizeDeepSearchJob(data);
}

export async function resetDocContent(request: DocResetRequest): Promise<DocResetResponse> {
  const trimmed = request.path.trim();
  if (!trimmed) {
    throw new Error("Document path is required");
  }
  const url = buildApiUrl("/api/v1/docs/reset", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      path: trimmed,
      max_age_days: request.max_age_days,
      keep_min_entries: request.keep_min_entries,
      preview_only: request.preview_only,
    }),
    cache: "no-store",
  });

  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    throw new Error(parseErrorMessage(errorPayload, `Doc reset failed: ${res.status}`));
  }

  const data = (await res.json().catch(() => null)) as unknown;
  if (!isRecord(data)) {
    throw new Error("Invalid doc reset response");
  }

  return {
    path: toNonEmptyString(data.path) ?? trimmed,
    doc_type: toNonEmptyString(data.doc_type) ?? "",
    removed_count: toFiniteNumber(data.removed_count) ?? 0,
    kept_count: toFiniteNumber(data.kept_count) ?? 0,
    removed_entries: normalizeStringList(data.removed_entries),
    new_content: toNonEmptyString(data.new_content),
    preview_only: toBoolean(data.preview_only) ?? false,
  };
}

const normalizeDeepSearchJob = (value: unknown): DeepSearchJob => {
  if (!isRecord(value)) {
    throw new Error("Invalid deep search response");
  }
  return {
    job_id: toNonEmptyString(value.job_id) ?? "",
    status: toNonEmptyString(value.status) ?? "unknown",
    progress: toNonEmptyString(value.progress),
    started_at: toNonEmptyString(value.started_at),
    completed_at: toNonEmptyString(value.completed_at),
    error: toNonEmptyString(value.error),
    results: normalizeDeepSearchResults(value.results),
  };
};

const normalizeDeepSearchResults = (value: unknown): DeepSearchResult[] => {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    if (!isRecord(item)) return [];
    const path = toNonEmptyString(item.path);
    const relevance = toFiniteNumber(item.relevance);
    const summary = toNonEmptyString(item.summary);
    const matchReason = toNonEmptyString(item.match_reason);
    if (!path || relevance === undefined || !summary || !matchReason) return [];
    return [
      {
        path,
        relevance,
        summary,
        match_reason: matchReason,
        references: normalizeStringList(item.references),
        snippet: toNonEmptyString(item.snippet),
      },
    ];
  });
};

const normalizeWarnings = (value: unknown): DocWarning[] => {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => normalizeWarning(item) ?? []);
};

const normalizeResetConfig = (value: unknown): DocResetConfig | undefined => {
  if (!isRecord(value)) return undefined;
  const maxAge = toFiniteNumber(value.max_age_days);
  const keepMin = toFiniteNumber(value.keep_min_entries);
  if (maxAge === undefined && keepMin === undefined) return undefined;
  return {
    max_age_days: maxAge,
    keep_min_entries: keepMin,
  };
};

const normalizeWarning = (value: unknown): DocWarning | undefined => {
  if (!isRecord(value)) return undefined;
  const type = toNonEmptyString(value.type);
  const message = toNonEmptyString(value.message);
  const severity = toNonEmptyString(value.severity);
  if (!type || !message || !severity) return undefined;
  return {
    type,
    message,
    severity,
    expected_path: toNonEmptyString(value.expected_path),
  };
};

const normalizeMisplacedDocs = (value: unknown): MisplacedDoc[] => {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    if (!isRecord(item)) return [];
    const actual = toNonEmptyString(item.actual_path);
    const expected = toNonEmptyString(item.expected_path);
    const docType = toNonEmptyString(item.doc_type);
    const severity = toNonEmptyString(item.severity);
    if (!actual || !expected || !docType || !severity) return [];
    return [
      {
        actual_path: actual,
        expected_path: expected,
        doc_type: docType,
        severity,
      },
    ];
  });
};

const normalizeStringList = (value: unknown): string[] => {
  if (!Array.isArray(value)) return [];
  return value.flatMap((entry) => (typeof entry === "string" && entry.trim() ? [entry] : []));
};

const normalizeDocTreeNode = (value: unknown): DocTreeNode => {
  if (!isRecord(value)) {
    throw new Error("Invalid doc tree response");
  }

  const name = toNonEmptyString(value.name) ?? "Unknown";
  const path = toNonEmptyString(value.path) ?? "";
  const typeRaw = toNonEmptyString(value.type);
  const type: DocTreeNode["type"] = typeRaw === "directory" ? "directory" : "file";
  const childrenRaw = Array.isArray(value.children) ? value.children : [];

  const node: DocTreeNode = {
    name,
    path,
    type,
  };

  const docType = toNonEmptyString(value.doc_type);
  if (docType) {
    node.doc_type = docType;
  }

  const size = toFiniteNumber(value.size);
  if (size !== undefined) {
    node.size = size;
  }

  const modifiedAt = toNonEmptyString(value.modified_at);
  if (modifiedAt) {
    node.modified_at = modifiedAt;
  }

  const warning = normalizeWarning(value.warning);
  if (warning) {
    node.warning = warning;
  }

  const children = childrenRaw.map((child) => normalizeDocTreeNode(child));
  if (children.length > 0) {
    node.children = children;
  }

  return node;
};
