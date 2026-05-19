// DOC: docs/reference/api-endpoints.md#scenario-list
// DOC: docs/reference/api-endpoints.md#scenario-documentation-tree
// DOC: docs/reference/api-endpoints.md#documentation-health
// DOC: docs/reference/api-endpoints.md#documentation-search
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
  fix_category: string;
}

export interface DocAutoFixMovedFile {
  from_path: string;
  to_path: string;
  doc_type: string;
}

export interface DocAutoFixSkippedFile {
  from_path: string;
  to_path: string;
  doc_type: string;
  reason: string;
}

export interface DocAutoFixResponse {
  scenario_name: string;
  moved: DocAutoFixMovedFile[];
  skipped: DocAutoFixSkippedFile[];
  health_before: number;
  health_after: number;
  dry_run: boolean;
}

export interface DocFileSearchRequest {
  pattern: string;
  scope?: string;
  scenario?: string;
  base_path?: string;
  limit?: number;
  include_content?: boolean;
}

export interface DocFileSearchResult {
  path: string;
  relative_path: string;
  scenario?: string;
  size?: number;
  modified_at?: string;
  doc_type?: string;
  content_preview?: string;
}

export interface DocTextSearchRequest {
  query: string;
  scope?: string;
  scenario?: string;
  base_path?: string;
  file_types?: string[];
  case_sensitive?: boolean;
  limit?: number;
  context_lines?: number;
}

export interface DocTextSearchMatch {
  path: string;
  relative_path: string;
  scenario?: string;
  line_number?: number;
  content: string;
  context_before?: string;
  context_after?: string;
}

export interface DocUnifiedSearchRequest {
  query?: string;
  pattern?: string;
  scope?: string;
  scenario?: string;
  base_path?: string;
  limit?: number;
  include_content?: boolean;
  file_types?: string[];
  case_sensitive?: boolean;
  context_lines?: number;
  use_semantic?: boolean;
  semantic_limit?: number;
  semantic_threshold?: number;
  semantic_collection?: string;
  semantic_namespaces?: string[];
  semantic_visibility?: string[];
  semantic_tags?: string[];
}

export interface DocUnifiedSearchResult {
  source: string;
  score?: number;
  path?: string;
  relative_path?: string;
  scenario?: string;
  line_number?: number;
  snippet?: string;
  doc_type?: string;
  content?: string;
  id?: string;
  metadata?: Record<string, unknown>;
}

export interface DocUnifiedSearchResponse {
  results: DocUnifiedSearchResult[];
  query: string;
  took_ms: number;
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

export interface DocHealRequest {
  scenario_name: string;
  issues?: string[];
  auto_approve?: boolean;
  dry_run?: boolean;
}

export interface DocHealFileDiff {
  path: string;
  operation: string;
  old_path?: string;
  diff: string;
}

export interface DocHealDiff {
  files: DocHealFileDiff[];
  summary: string;
}

export interface DocHealJob {
  job_id: string;
  scenario_name: string;
  status: string;
  progress?: string;
  started_at?: string;
  completed_at?: string;
  diff?: DocHealDiff;
  health_before?: number;
  health_after?: number;
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

// fetchScenarioDocHealth posts to the KnowledgeObservatoryService.DocHealth
// Connect-RPC endpoint (Connect protocol's JSON unary form: POST
// /<package>.<service>/<method> with `Content-Type: application/json`).
// The legacy REST GET endpoint was removed; this is the only doc-health
// surface.
export async function fetchScenarioDocHealth(scenarioName: string): Promise<ScenarioDocHealth> {
  const trimmed = scenarioName.trim();
  if (!trimmed) {
    throw new Error("Scenario name is required");
  }
  const url = buildApiUrl(
    "/knowledge_observatory.v1.KnowledgeObservatoryService/DocHealth",
    { baseUrl: API_BASE },
  );
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ scenarioName: trimmed }),
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

  // Proto uses camelCase by default for JSON serialization.
  const scenarioNameOut = toNonEmptyString(data.scenarioName) ?? trimmed;
  const healthScore = toFiniteNumber(data.healthScore) ?? 0;
  const totalDocs = toFiniteNumber(data.totalDocs) ?? 0;
  const misplaced = normalizeMisplacedDocsProto(data.misplacedDocs);
  const missing = normalizeMissingDocIdentifiers(data.missingDocs);
  const extra = normalizeStringList(data.extraDocs);
  const temporary = normalizeStringList(data.temporaryDocs);
  const warnings = normalizeFindings(data.contractFindings)
    .concat(normalizeFindings(data.contentFindings))
    .concat(normalizeFindings(data.referenceFindings))
    .concat(normalizeFindings(data.manifestFindings));

  return {
    scenario_name: scenarioNameOut,
    health_score: healthScore,
    total_docs: totalDocs,
    misplaced_docs: misplaced,
    missing_docs: missing,
    extra_docs: extra,
    warnings,
    can_auto_fix: misplaced.length > 0,
    fix_category: deriveFixCategory(misplaced.length, missing.length, extra.length, temporary.length),
  };
}

function normalizeMisplacedDocsProto(raw: unknown): MisplacedDoc[] {
  if (!Array.isArray(raw)) return [];
  const out: MisplacedDoc[] = [];
  for (const item of raw) {
    if (!isRecord(item)) continue;
    const actual = toNonEmptyString(item.actualPath);
    const expected = toNonEmptyString(item.expectedPath);
    const docType = toNonEmptyString(item.docType) ?? "";
    const severity = severityProtoToString(toNonEmptyString(item.severity));
    if (!actual || !expected) continue;
    out.push({ actual_path: actual, expected_path: expected, doc_type: docType, severity });
  }
  return out;
}

function normalizeMissingDocIdentifiers(raw: unknown): string[] {
  if (!Array.isArray(raw)) {
    return [];
  }
  const out: string[] = [];
  for (const entry of raw) {
    if (!isRecord(entry)) continue;
    const docType = toNonEmptyString(entry.docType);
    if (docType) out.push(docType);
  }
  return out;
}

function normalizeFindings(raw: unknown): DocWarning[] {
  if (!Array.isArray(raw)) {
    return [];
  }
  const out: DocWarning[] = [];
  for (const entry of raw) {
    if (!isRecord(entry)) continue;
    const code = toNonEmptyString(entry.code) ?? "finding";
    const rawMessage = toNonEmptyString(entry.message) ?? "";
    const path = toNonEmptyString(entry.path);
    const docType = toNonEmptyString(entry.docType);
    const severity = severityProtoToString(toNonEmptyString(entry.severity));
    const parts: string[] = [rawMessage];
    if (path) parts.push(`(${path})`);
    if (docType) parts.push(`[${docType}]`);
    out.push({ type: code, message: parts.filter(Boolean).join(" "), severity });
  }
  return out;
}

function severityProtoToString(value: string | null | undefined): string {
  switch (value) {
    case "DOC_HEALTH_SEVERITY_INFO":
      return "info";
    case "DOC_HEALTH_SEVERITY_WARNING":
      return "warning";
    case "DOC_HEALTH_SEVERITY_FAILURE":
      return "error";
    default:
      return "warning";
  }
}

function deriveFixCategory(misplaced: number, missing: number, extra: number, temporary: number): string {
  const hasMisplaced = misplaced > 0;
  const hasAgent = missing > 0 || extra > 0 || temporary > 0;
  if (hasMisplaced && !hasAgent) return "all_auto";
  if (!hasMisplaced && hasAgent) return "all_agent";
  if (hasMisplaced && hasAgent) return "mixed";
  return "none";
}

export async function searchDocFiles(request: DocFileSearchRequest): Promise<DocFileSearchResult[]> {
  const url = buildApiUrl("/api/v1/docs/search/files", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    throw new Error(parseErrorMessage(errorPayload, `Doc file search failed: ${res.status}`));
  }
  const data = (await res.json().catch(() => null)) as unknown;
  return normalizeDocFileResults(data);
}

export async function searchDocText(request: DocTextSearchRequest): Promise<DocTextSearchMatch[]> {
  const url = buildApiUrl("/api/v1/docs/search/text", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    throw new Error(parseErrorMessage(errorPayload, `Doc text search failed: ${res.status}`));
  }
  const data = (await res.json().catch(() => null)) as unknown;
  return normalizeDocTextResults(data);
}

export async function searchDocUnified(request: DocUnifiedSearchRequest): Promise<DocUnifiedSearchResponse> {
  const url = buildApiUrl("/api/v1/docs/search/unified", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    throw new Error(parseErrorMessage(errorPayload, `Doc unified search failed: ${res.status}`));
  }
  const data = (await res.json().catch(() => null)) as unknown;
  if (!isRecord(data)) {
    throw new Error("Invalid unified search response");
  }
  return {
    results: normalizeUnifiedResults(data.results),
    query: toNonEmptyString(data.query) ?? request.query ?? "",
    took_ms: toFiniteNumber(data.took_ms) ?? 0,
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

export async function startDocHealing(payload: DocHealRequest): Promise<DocHealJob> {
  const scenario = payload.scenario_name.trim();
  if (!scenario) {
    throw new Error("Scenario name is required");
  }
  const url = buildApiUrl(`/api/v1/scenarios/${encodeURIComponent(scenario)}/docs/heal`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    throw new Error(parseErrorMessage(errorPayload, `Doc healing failed: ${res.status}`));
  }
  const data = (await res.json().catch(() => null)) as unknown;
  return normalizeDocHealJob(data);
}

export async function fetchDocHealingJob(jobId: string): Promise<DocHealJob> {
  const trimmed = jobId.trim();
  if (!trimmed) {
    throw new Error("Job ID is required");
  }
  const url = buildApiUrl(`/api/v1/docs/heal/${encodeURIComponent(trimmed)}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    throw new Error(parseErrorMessage(errorPayload, `Doc healing status failed: ${res.status}`));
  }
  const data = (await res.json().catch(() => null)) as unknown;
  return normalizeDocHealJob(data);
}

export async function approveDocHealing(jobId: string, actor?: string): Promise<DocHealJob> {
  const trimmed = jobId.trim();
  if (!trimmed) {
    throw new Error("Job ID is required");
  }
  const url = buildApiUrl(`/api/v1/docs/heal/${encodeURIComponent(trimmed)}/approve`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ actor: actor?.trim() || undefined }),
  });
  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    throw new Error(parseErrorMessage(errorPayload, `Doc healing approve failed: ${res.status}`));
  }
  const data = (await res.json().catch(() => null)) as unknown;
  return normalizeDocHealJob(data);
}

export async function rejectDocHealing(jobId: string, actor?: string, reason?: string): Promise<DocHealJob> {
  const trimmed = jobId.trim();
  if (!trimmed) {
    throw new Error("Job ID is required");
  }
  const url = buildApiUrl(`/api/v1/docs/heal/${encodeURIComponent(trimmed)}/reject`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ actor: actor?.trim() || undefined, reason: reason?.trim() || undefined }),
  });
  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    throw new Error(parseErrorMessage(errorPayload, `Doc healing reject failed: ${res.status}`));
  }
  const data = (await res.json().catch(() => null)) as unknown;
  return normalizeDocHealJob(data);
}

export async function autoFixDocs(scenarioName: string, dryRun?: boolean): Promise<DocAutoFixResponse> {
  const trimmed = scenarioName.trim();
  if (!trimmed) {
    throw new Error("Scenario name is required");
  }
  const url = buildApiUrl(`/api/v1/scenarios/${encodeURIComponent(trimmed)}/docs/autofix`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ dry_run: dryRun ?? false }),
  });
  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    throw new Error(parseErrorMessage(errorPayload, `Doc auto-fix failed: ${res.status}`));
  }
  const data = (await res.json().catch(() => null)) as unknown;
  return normalizeDocAutoFixResponse(data);
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

const normalizeDocHealJob = (value: unknown): DocHealJob => {
  if (!isRecord(value)) {
    throw new Error("Invalid doc healing response");
  }
  return {
    job_id: toNonEmptyString(value.job_id) ?? "",
    scenario_name: toNonEmptyString(value.scenario_name) ?? "",
    status: toNonEmptyString(value.status) ?? "unknown",
    progress: toNonEmptyString(value.progress),
    started_at: toNonEmptyString(value.started_at),
    completed_at: toNonEmptyString(value.completed_at),
    error: toNonEmptyString(value.error),
    health_before: toFiniteNumber(value.health_before),
    health_after: toFiniteNumber(value.health_after),
    diff: normalizeDocHealDiff(value.diff),
  };
};

const normalizeDocHealDiff = (value: unknown): DocHealDiff | undefined => {
  if (!isRecord(value)) return undefined;
  return {
    summary: toNonEmptyString(value.summary) ?? "",
    files: normalizeDocHealFiles(value.files),
  };
};

const normalizeDocHealFiles = (value: unknown): DocHealFileDiff[] => {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    if (!isRecord(item)) return [];
    const path = toNonEmptyString(item.path);
    const operation = toNonEmptyString(item.operation);
    const diff = typeof item.diff === "string" ? item.diff : undefined;
    if (!path || !operation || diff === undefined) return [];
    return [
      {
        path,
        operation,
        old_path: toNonEmptyString(item.old_path),
        diff,
      },
    ];
  });
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

const normalizeDocFileResults = (value: unknown): DocFileSearchResult[] => {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    if (!isRecord(item)) return [];
    const path = toNonEmptyString(item.path);
    const relativePath = toNonEmptyString(item.relative_path);
    if (!path || !relativePath) return [];
    return [
      {
        path,
        relative_path: relativePath,
        scenario: toNonEmptyString(item.scenario),
        size: toFiniteNumber(item.size),
        modified_at: toNonEmptyString(item.modified_at),
        doc_type: toNonEmptyString(item.doc_type),
        content_preview: toNonEmptyString(item.content_preview),
      },
    ];
  });
};

const normalizeDocTextResults = (value: unknown): DocTextSearchMatch[] => {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    if (!isRecord(item)) return [];
    const path = toNonEmptyString(item.path);
    const relativePath = toNonEmptyString(item.relative_path);
    const content = toNonEmptyString(item.content);
    if (!path || !relativePath || content === undefined) return [];
    return [
      {
        path,
        relative_path: relativePath,
        scenario: toNonEmptyString(item.scenario),
        line_number: toFiniteNumber(item.line_number),
        content,
        context_before: toNonEmptyString(item.context_before),
        context_after: toNonEmptyString(item.context_after),
      },
    ];
  });
};

const normalizeUnifiedResults = (value: unknown): DocUnifiedSearchResult[] => {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    if (!isRecord(item)) return [];
    const source = toNonEmptyString(item.source);
    if (!source) return [];
    return [
      {
        source,
        score: toFiniteNumber(item.score),
        path: toNonEmptyString(item.path),
        relative_path: toNonEmptyString(item.relative_path),
        scenario: toNonEmptyString(item.scenario),
        line_number: toFiniteNumber(item.line_number),
        snippet: toNonEmptyString(item.snippet),
        doc_type: toNonEmptyString(item.doc_type),
        content: toNonEmptyString(item.content),
        id: toNonEmptyString(item.id),
        metadata: isRecord(item.metadata) ? item.metadata : undefined,
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

const normalizeDocAutoFixResponse = (value: unknown): DocAutoFixResponse => {
  if (!isRecord(value)) {
    throw new Error("Invalid auto-fix response");
  }
  return {
    scenario_name: toNonEmptyString(value.scenario_name) ?? "",
    moved: normalizeAutoFixMoved(value.moved),
    skipped: normalizeAutoFixSkipped(value.skipped),
    health_before: toFiniteNumber(value.health_before) ?? 0,
    health_after: toFiniteNumber(value.health_after) ?? 0,
    dry_run: toBoolean(value.dry_run) ?? false,
  };
};

const normalizeAutoFixMoved = (value: unknown): DocAutoFixMovedFile[] => {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    if (!isRecord(item)) return [];
    const from = toNonEmptyString(item.from_path);
    const to = toNonEmptyString(item.to_path);
    if (!from || !to) return [];
    return [{ from_path: from, to_path: to, doc_type: toNonEmptyString(item.doc_type) ?? "" }];
  });
};

const normalizeAutoFixSkipped = (value: unknown): DocAutoFixSkippedFile[] => {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    if (!isRecord(item)) return [];
    const from = toNonEmptyString(item.from_path);
    const to = toNonEmptyString(item.to_path);
    if (!from || !to) return [];
    return [
      {
        from_path: from,
        to_path: to,
        doc_type: toNonEmptyString(item.doc_type) ?? "",
        reason: toNonEmptyString(item.reason) ?? "",
      },
    ];
  });
};
