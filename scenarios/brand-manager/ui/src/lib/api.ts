import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

// --- Structured API Error ---

/** Mirrors the Go apierr.Error JSON shape for programmatic error handling. */
export interface ApiError {
  code: "validation" | "not_found" | "conflict" | "internal" | "dependency";
  message: string;
  recovery?: string;
}

/**
 * Thrown by API functions when the server returns a non-OK response.
 * Carries the structured error from the server when available.
 */
export class ApiRequestError extends Error {
  status: number;
  apiError?: ApiError;

  constructor(status: number, apiError?: ApiError) {
    const msg = apiError?.message ?? `Request failed with status ${status}`;
    super(msg);
    this.name = "ApiRequestError";
    this.status = status;
    this.apiError = apiError;
  }

  /** User-friendly recovery hint from the server, if available. */
  get recovery(): string | undefined {
    return this.apiError?.recovery;
  }

  /** Whether the error is transient and retrying may help. */
  get isRetryable(): boolean {
    if (!this.apiError) return this.status >= 500;
    return this.apiError.code === "internal" || this.apiError.code === "dependency";
  }
}

/** Type guard for structured API errors from the server. */
function isApiErrorShape(body: unknown): body is ApiError {
  if (typeof body !== "object" || body === null) return false;
  if (!("code" in body) || !("message" in body)) return false;
  const { code, message } = body as { code: unknown; message: unknown };
  return typeof code === "string" && typeof message === "string";
}

/** Parse a successful JSON response with the expected type. */
async function parseJson<T>(res: Response): Promise<T> {
  // eslint-disable-next-line @typescript-eslint/no-unsafe-return -- res.json() returns unknown at runtime; caller validates shape
  return res.json();
}

/** Parse a non-OK response into an ApiRequestError. */
async function throwApiError(res: Response): Promise<never> {
  let apiError: ApiError | undefined;
  try {
    const body: unknown = await res.json();
    if (isApiErrorShape(body)) {
      apiError = body;
    }
  } catch {
    // Response wasn't JSON — fall through to generic error
  }
  throw new ApiRequestError(res.status, apiError);
}

// --- Request helpers ---

/** Execute a fetch and handle errors/parsing. Returns parsed JSON or throws ApiRequestError. */
async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) return throwApiError(res);
  if (res.status === 204) return undefined as T;
  return parseJson<T>(res);
}

/** POST/PUT with JSON body. */
async function jsonMutate<T>(url: string, method: string, body: unknown): Promise<T> {
  return handleResponse<T>(await fetch(url, {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  }));
}

// --- Health ---

export async function fetchHealth(): Promise<{ status: string; service: string; timestamp: string }> {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  return handleResponse(await fetch(url, { headers: { "Content-Type": "application/json" }, cache: "no-store" }));
}

// --- Brand Types ---

export interface BrandIdentity {
  display_name?: string;
  tagline?: string;
  logo_path?: string;
  favicon_path?: string;
  icon_path?: string;
}

export interface BrandColors {
  primary?: string;
  secondary?: string;
  accent?: string;
  background?: string;
  surface?: string;
  text?: string;
  error?: string;
}

export interface BrandTypography {
  heading_font?: string;
  body_font?: string;
  mono_font?: string;
  base_font_size?: string;
}

export interface BrandVoice {
  tone?: string;
  style?: string;
  keywords?: string[];
}

export interface Brand {
  id: string;
  name: string;
  description?: string;
  identity?: BrandIdentity;
  colors?: BrandColors;
  typography?: BrandTypography;
  voice?: BrandVoice;
  notes?: string;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface BrandVersion {
  id: string;
  brand_id: string;
  version: number;
  snapshot: string;
  created_at: string;
}

export interface Assignment {
  id: string;
  brand_id: string;
  scenario_name: string;
  brand_version: number;
  elements?: string[];
  applied_at: string;
}

export interface ContrastPairResult {
  foreground: string;
  background: string;
  ratio: number;
  aa_normal: boolean;
  aa_large: boolean;
}

export interface BrandContrastResult {
  pairs: ContrastPairResult[] | null;
  pass_all: boolean;
}

// --- Brand CRUD --- [REQ:BM-REQ-API-BRANDS]

export async function fetchBrands(name?: string): Promise<Brand[]> {
  const params = new URLSearchParams();
  if (name) params.set("name", name);
  return handleResponse(await fetch(buildApiUrl(`/brands?${params}`, { baseUrl: API_BASE })));
}

export async function fetchBrand(id: string): Promise<Brand> {
  return handleResponse(await fetch(buildApiUrl(`/brands/${id}`, { baseUrl: API_BASE })));
}

export async function createBrand(brand: Partial<Brand>): Promise<Brand> {
  return jsonMutate(buildApiUrl("/brands", { baseUrl: API_BASE }), "POST", brand);
}

export async function updateBrand(id: string, brand: Partial<Brand>): Promise<Brand> {
  return jsonMutate(buildApiUrl(`/brands/${id}`, { baseUrl: API_BASE }), "PUT", brand);
}

export async function deleteBrand(id: string): Promise<void> {
  return handleResponse(await fetch(buildApiUrl(`/brands/${id}`, { baseUrl: API_BASE }), { method: "DELETE" }));
}

// --- Versions --- [REQ:BM-REQ-API-VERSIONS]

export async function fetchVersions(brandId: string): Promise<BrandVersion[]> {
  return handleResponse(await fetch(buildApiUrl(`/brands/${brandId}/versions`, { baseUrl: API_BASE })));
}

// --- Assignments --- [REQ:BM-REQ-API-ASSIGN]

export async function createAssignment(brandId: string, scenarioName: string, elements?: string[]): Promise<Assignment> {
  return jsonMutate(buildApiUrl("/assignments", { baseUrl: API_BASE }), "POST", {
    brand_id: brandId, scenario_name: scenarioName, elements,
  });
}

export async function deleteAssignment(id: string): Promise<void> {
  return handleResponse(await fetch(buildApiUrl(`/assignments/${id}`, { baseUrl: API_BASE }), { method: "DELETE" }));
}

// --- Scenario Status --- [REQ:BM-REQ-API-STATUS]

export interface ScenarioStatus {
  scenario: string;
  has_brand: boolean;
  brand_id: string | null;
  brand_version: number | null;
  elements?: string[];
  applied_at?: string;
}

export async function fetchScenarioStatus(name: string): Promise<ScenarioStatus> {
  return handleResponse(await fetch(buildApiUrl(`/scenarios/${name}/status`, { baseUrl: API_BASE })));
}

// --- WCAG Contrast --- [REQ:BM-REQ-WCAG-CALC] [REQ:BM-REQ-WCAG-VALIDATE]

export async function checkContrast(fg: string, bg: string): Promise<ContrastPairResult> {
  return jsonMutate(buildApiUrl("/contrast/check", { baseUrl: API_BASE }), "POST", { foreground: fg, background: bg });
}

export async function checkBrandContrast(colors: BrandColors): Promise<BrandContrastResult> {
  return jsonMutate(buildApiUrl("/contrast/brand", { baseUrl: API_BASE }), "POST", colors);
}

// --- Theme Preview --- [REQ:BM-REQ-UI-THEME]

export interface ThemePreviewResult {
  brand_id: string;
  css: string;
  tokens: Record<string, string>;
  mode: string;
}

export async function fetchThemePreview(brandId: string, mode: "light" | "dark" = "light"): Promise<ThemePreviewResult> {
  return handleResponse(await fetch(buildApiUrl(`/brands/${brandId}/theme-preview?mode=${mode}`, { baseUrl: API_BASE })));
}

// --- Apply Preview --- [REQ:BM-REQ-UI-APPLY]

export interface ApplyAction {
  type: string;
  file: string;
  element: string;
}

export interface ApplyPreviewResult {
  scenario: string;
  brand_id: string;
  brand_version: number;
  applied: ApplyAction[];
  skipped?: { element: string; reason: string }[];
  dry_run: boolean;
}

export async function fetchApplyPreview(brandId: string, scenarioName: string, elements?: string[]): Promise<ApplyPreviewResult> {
  return jsonMutate(buildApiUrl(`/brands/${brandId}/apply/preview`, { baseUrl: API_BASE }), "POST", {
    scenario_name: scenarioName, elements,
  });
}

// --- Generate Options --- [REQ:BM-REQ-UI-GENERATE]

export interface GenerateProvider {
  id: string;
  name: string;
  description: string;
  available: boolean;
  capabilities: string[];
  requires?: string;
}

export interface GenerateOptionsResult {
  providers: GenerateProvider[];
  elements: string[];
}

export async function fetchGenerateOptions(): Promise<GenerateOptionsResult> {
  return handleResponse(await fetch(buildApiUrl("/brands/generate/options", { baseUrl: API_BASE })));
}

// --- Standards --- [REQ:BM-REQ-API-STANDARDS]

export interface StandardsResult {
  rules: { id: string; name: string; description: string; severity: string }[];
}

export async function fetchStandards(): Promise<StandardsResult> {
  return handleResponse(await fetch(buildApiUrl("/standards", { baseUrl: API_BASE })));
}

// --- Scanner --- [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON]

export interface ScanFinding {
  file: string;
  element: string;
  type: string;
  line?: number;
  value?: string;
}

export interface ScanResult {
  scenario: string;
  findings: ScanFinding[];
  summary: { total: number; css: number; json: number };
}

export async function scanScenario(scenario: string): Promise<ScanResult> {
  return handleResponse(await fetch(buildApiUrl(`/scan/${encodeURIComponent(scenario)}`, { baseUrl: API_BASE })));
}

// --- Audit --- [REQ:BM-REQ-AUDIT-ENDPOINT]

export interface AuditRule {
  id: string;
  name: string;
  description: string;
  severity: string;
}

export interface AuditResult {
  scenario: string;
  results: { rule_id: string; passed: boolean; message: string }[];
  pass_all: boolean;
}

export async function fetchAuditRules(): Promise<{ rules: AuditRule[] }> {
  return handleResponse(await fetch(buildApiUrl("/audit/rules", { baseUrl: API_BASE })));
}

export async function evaluateScenario(scenario: string): Promise<AuditResult> {
  return jsonMutate(buildApiUrl(`/audit/evaluate/${encodeURIComponent(scenario)}`, { baseUrl: API_BASE }), "POST", {});
}

// --- Agent-Assisted Application --- [REQ:BM-REQ-AGENT-SPAWN] [REQ:BM-REQ-AGENT-INSTRUCT] [REQ:BM-REQ-AGENT-VALIDATE]

export interface AgentApplyRequest {
  scenario_name: string;
  elements?: string[];
  prompt?: string;
}

export interface AgentApplyResult {
  scenario: string;
  brand_id: string;
  brand_version: number;
  agent_id?: string;
  status: string;
  elements: string[];
  instructions: string;
  dry_run?: boolean;
}

export async function agentApply(brandId: string, req: AgentApplyRequest, dryRun = false): Promise<AgentApplyResult> {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (dryRun) headers["X-Dry-Run"] = "true";
  return handleResponse(await fetch(buildApiUrl(`/brands/${brandId}/agent-apply`, { baseUrl: API_BASE }), {
    method: "POST",
    headers,
    body: JSON.stringify(req),
  }));
}

export interface AgentValidateRequest {
  scenario_name: string;
  elements?: string[];
}

export interface AgentValidateResult {
  scenario: string;
  valid: boolean;
  expected: string[];
  found: string[];
  missing: string[];
  scan_report?: ScanResult;
}

export async function agentValidate(brandId: string, req: AgentValidateRequest): Promise<AgentValidateResult> {
  return jsonMutate(buildApiUrl(`/brands/${brandId}/agent-validate`, { baseUrl: API_BASE }), "POST", req);
}

// --- Lighthouse WCAG Audit --- [REQ:BM-REQ-LIGHTHOUSE]

export interface LighthouseRequest {
  scenario_name: string;
  url?: string;
}

export interface LighthouseResult {
  scenario: string;
  brand_id: string;
  url: string;
  score: number;
  passed: boolean;
  threshold: number;
  status: string;
  error_message?: string;
}

export async function lighthouseAudit(brandId: string, req: LighthouseRequest, dryRun = false): Promise<LighthouseResult> {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (dryRun) headers["X-Dry-Run"] = "true";
  return handleResponse(await fetch(buildApiUrl(`/brands/${brandId}/lighthouse`, { baseUrl: API_BASE }), {
    method: "POST",
    headers,
    body: JSON.stringify(req),
  }));
}
