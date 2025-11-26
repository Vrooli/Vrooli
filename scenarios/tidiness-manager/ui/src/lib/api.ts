import { resolveWithConfig, buildApiUrl } from "@vrooli/api-base";

// Lazily resolve API base using @vrooli/api-base's built-in resolution
// This handles proxy detection, /config endpoint fetching, and __VROOLI_CONFIG__ injection
let API_BASE: string | null = null;
let API_BASE_PROMISE: Promise<string> | null = null;

async function resolveApiBaseOnce(): Promise<string> {
  if (API_BASE) return API_BASE;
  if (API_BASE_PROMISE) return API_BASE_PROMISE;

  API_BASE_PROMISE = (async () => {
    try {
      // Use resolveWithConfig which:
      // 1. Checks for __VROOLI_CONFIG__ injected global
      // 2. Falls back to proxy metadata detection
      // 3. Falls back to /config endpoint fetch (absolute path)
      const resolved = await resolveWithConfig({
        appendSuffix: true,
        configEndpoint: '/config'  // Use absolute path
      });
      console.log('[tidiness-manager] resolveWithConfig returned:', resolved);

      if (!resolved || typeof resolved !== 'string' || resolved.trim() === '') {
        throw new Error('No valid API base could be resolved');
      }

      API_BASE = resolved;
      return resolved;
    } catch (e) {
      console.error('[tidiness-manager] Failed to resolve API base:', e);
      throw e;
    }
  })();

  return API_BASE_PROMISE;
}

// Helper to get API base (blocks until resolved)
async function getApiBase(): Promise<string> {
  return resolveApiBaseOnce();
}

// ============================================================================
// Type Definitions
// ============================================================================

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
  readiness: boolean;
  version: string;
  dependencies: {
    database: string;
  };
}

export interface FileMetric {
  path: string;
  lines: number;
  extension: string;
}

export interface LongFile {
  path: string;
  lines: number;
  threshold: number;
}

export interface CommandRun {
  command: string;
  exit_code: number;
  stdout: string;
  stderr: string;
  duration_ms: number;
  success: boolean;
  skipped: boolean;
  skip_reason?: string;
}

export interface ScanResult {
  scenario: string;
  started_at: string;
  completed_at: string;
  duration_ms: number;
  lint_output?: CommandRun;
  type_output?: CommandRun;
  file_metrics: FileMetric[];
  long_files: LongFile[];
  total_files: number;
  total_lines: number;
  has_makefile: boolean;
}

export interface Issue {
  id?: number;
  scenario: string;
  file: string;
  line: number;
  column: number;
  message: string;
  rule?: string;
  severity: "error" | "warning";
  tool: string;
  category: "lint" | "type" | "ai";
  status?: "open" | "resolved" | "ignored";
  resolved_at?: string;
  resolved_by?: string;
  resolution_notes?: string;
  created_at?: string;
}

export interface Campaign {
  id: string;
  scenario: string;
  status: "created" | "active" | "paused" | "completed" | "error" | "terminated";
  created_at: string;
  updated_at: string;
  completed_at?: string;
  max_sessions: number;
  current_session: number;
  files_total: number;
  files_visited: number;
  error_reason?: string;
  config: {
    max_files_per_session: number;
    priority_rules?: string[];
    category_focus?: string[];
  };
}

export interface ScenarioStats {
  scenario: string;
  light_issues: number;
  ai_issues: number;
  long_files: number;
  visit_percent: number;
  campaign_status: "none" | "active" | "paused" | "completed" | "error";
  total_files: number;
  total_lines: number;
  last_scan?: string;
}

export interface FileStats {
  path: string;
  lines: number;
  lint_issues: number;
  type_issues: number;
  ai_issues: number;
  visit_count: number;
  is_long_file: boolean;
  last_visited?: string;
  extension: string;
}

// ============================================================================
// API Client Functions
// ============================================================================

export async function fetchHealth(): Promise<HealthResponse> {
  const apiBase = await getApiBase();
  const res = await fetch(buildApiUrl("/health", { baseUrl: apiBase }), {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`API health check failed: ${res.status}`);
  }

  return res.json();
}

export async function lightScan(scenarioPath: string, timeoutSec?: number): Promise<ScanResult> {
  const apiBase = await getApiBase();
  const res = await fetch(buildApiUrl("/scan/light", { baseUrl: apiBase }), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ scenario_path: scenarioPath, timeout_sec: timeoutSec }),
  });

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(error.error || `Light scan failed: ${res.status}`);
  }

  return res.json();
}

export async function parseLintOutput(scenario: string, tool: string, output: string): Promise<{ issues: Issue[]; count: number }> {
  const apiBase = await getApiBase();
  const res = await fetch(buildApiUrl("/scan/light/parse-lint", { baseUrl: apiBase }), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ scenario, tool, output }),
  });

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(error.error || `Parse lint failed: ${res.status}`);
  }

  return res.json();
}

export async function parseTypeOutput(scenario: string, tool: string, output: string): Promise<{ issues: Issue[]; count: number }> {
  const apiBase = await getApiBase();
  const res = await fetch(buildApiUrl("/scan/light/parse-type", { baseUrl: apiBase }), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ scenario, tool, output }),
  });

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(error.error || `Parse type failed: ${res.status}`);
  }

  return res.json();
}

// ============================================================================
// Mock Data & Future API Functions (will be replaced when backend ready)
// ============================================================================

// Mock scenarios for dashboard
const MOCK_SCENARIOS: ScenarioStats[] = [
  { scenario: "picker-wheel", light_issues: 3, ai_issues: 1, long_files: 2, visit_percent: 75, campaign_status: "active", total_files: 15, total_lines: 2400, last_scan: "2025-01-20T10:30:00Z" },
  { scenario: "tidiness-manager", light_issues: 8, ai_issues: 4, long_files: 5, visit_percent: 60, campaign_status: "completed", total_files: 32, total_lines: 5800, last_scan: "2025-01-19T14:20:00Z" },
  { scenario: "landing-manager", light_issues: 12, ai_issues: 7, long_files: 8, visit_percent: 45, campaign_status: "none", total_files: 48, total_lines: 9200, last_scan: "2025-01-18T09:15:00Z" },
  { scenario: "deployment-manager", light_issues: 5, ai_issues: 2, long_files: 3, visit_percent: 80, campaign_status: "paused", total_files: 28, total_lines: 4100, last_scan: "2025-01-21T08:00:00Z" },
  { scenario: "scenario-auditor", light_issues: 2, ai_issues: 0, long_files: 1, visit_percent: 95, campaign_status: "none", total_files: 18, total_lines: 2100, last_scan: "2025-01-20T16:45:00Z" },
];

export async function fetchScenarioStats(): Promise<ScenarioStats[]> {
  const apiBase = await getApiBase();
  const res = await fetch(buildApiUrl("/agent/scenarios", { baseUrl: apiBase }), {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch scenario stats: ${res.status}`);
  }

  const data = await res.json();

  // Transform API response to match UI expectations
  // API returns: { scenarios: [{ scenario, total, lint, type, long_files }], count }
  // UI expects: ScenarioStats[] with light_issues, ai_issues, etc.
  return (data.scenarios || []).map((s: any) => ({
    scenario: s.scenario,
    light_issues: s.lint || 0,
    ai_issues: s.type || 0,
    long_files: s.long_files || 0,
    visit_percent: 0, // TODO: Calculate from visited-tracker integration
    campaign_status: "none" as const, // TODO: Join with campaigns table
    total_files: 0, // TODO: Join with file_metrics table
    total_lines: 0, // TODO: Join with file_metrics table
  }));
}

export async function fetchScenarioDetail(scenarioName: string): Promise<{ stats: ScenarioStats; files: FileStats[] }> {
  const apiBase = await getApiBase();
  const res = await fetch(buildApiUrl(`/agent/scenarios/${encodeURIComponent(scenarioName)}`, { baseUrl: apiBase }), {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch scenario detail: ${res.status}`);
  }

  const data = await res.json();

  // API returns: { stats: {...}, files: [{...}] }
  return {
    stats: data.stats,
    files: data.files
  };
}

export async function fetchFileIssues(scenarioName: string, filePath: string): Promise<Issue[]> {
  // TODO: Replace with real API call when backend ready
  await new Promise(resolve => setTimeout(resolve, 150));

  const mockIssues: Issue[] = [
    { scenario: scenarioName, file: filePath, line: 42, column: 10, message: "Variable 'result' is never used", rule: "no-unused-vars", severity: "warning", tool: "eslint", category: "lint", status: "open", created_at: "2025-01-20T10:00:00Z" },
    { scenario: scenarioName, file: filePath, line: 95, column: 5, message: "Function complexity too high (15/10)", rule: "complexity", severity: "error", tool: "eslint", category: "lint", status: "open", created_at: "2025-01-20T10:00:00Z" },
    { scenario: scenarioName, file: filePath, line: 128, column: 20, message: "Type 'string | undefined' is not assignable to type 'string'", severity: "error", tool: "tsc", category: "type", status: "open", created_at: "2025-01-20T10:00:00Z" },
    { scenario: scenarioName, file: filePath, line: 156, column: 8, message: "Consider extracting this logic into a separate function for better maintainability", severity: "warning", tool: "claude-code", category: "ai", status: "open", created_at: "2025-01-20T11:30:00Z" },
  ];

  return mockIssues;
}

export async function fetchAllIssues(scenarioName: string, filters?: { status?: string; category?: string; severity?: string }): Promise<Issue[]> {
  const apiBase = await getApiBase();

  // Build query parameters
  const params = new URLSearchParams({ scenario: scenarioName });
  if (filters?.status && filters.status !== "all") {
    params.append("status", filters.status);
  }
  if (filters?.category && filters.category !== "all") {
    params.append("category", filters.category);
  }
  if (filters?.severity && filters.severity !== "all") {
    params.append("severity", filters.severity);
  }

  const res = await fetch(buildApiUrl(`/agent/issues?${params.toString()}`, { baseUrl: apiBase }), {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch issues: ${res.status}`);
  }

  const data = await res.json();

  // Transform API response to match UI expectations
  // API returns: [{ id, scenario, file_path, line_number, column_number, title, description, ... }]
  // UI expects: Issue[] with { id, scenario, file, line, column, message, ... }
  return (data || []).map((issue: any) => ({
    id: issue.id,
    scenario: issue.scenario,
    file: issue.file_path,
    line: issue.line_number || 0,
    column: issue.column_number || 0,
    message: issue.description || issue.title,
    rule: issue.title,
    severity: issue.severity === "error" ? "error" : "warning",
    tool: issue.category, // category serves as tool identifier
    category: issue.category === "lint" ? "lint" : issue.category === "type" ? "type" : "ai",
    status: issue.status || "open",
    resolution_notes: issue.resolution_notes,
    created_at: issue.created_at,
  }));
}

export async function updateIssueStatus(issueId: number, status: "open" | "resolved" | "ignored", resolutionNotes?: string): Promise<{ id: number; status: string; updated_at: string }> {
  const apiBase = await getApiBase();
  const res = await fetch(buildApiUrl(`/agent/issues/${issueId}`, { baseUrl: apiBase }), {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ status, resolution_notes: resolutionNotes || "" }),
  });

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(error.error || `Failed to update issue: ${res.status}`);
  }

  return res.json();
}

export async function fetchCampaigns(scenarioName?: string): Promise<Campaign[]> {
  // TODO: Replace with real API call when backend ready
  await new Promise(resolve => setTimeout(resolve, 200));

  const mockCampaigns: Campaign[] = [
    {
      id: "camp-001",
      scenario: "picker-wheel",
      status: "active",
      created_at: "2025-01-18T10:00:00Z",
      updated_at: "2025-01-20T14:30:00Z",
      max_sessions: 10,
      current_session: 3,
      files_total: 15,
      files_visited: 11,
      config: { max_files_per_session: 5, priority_rules: ["long_files_first"] },
    },
    {
      id: "camp-002",
      scenario: "tidiness-manager",
      status: "completed",
      created_at: "2025-01-15T09:00:00Z",
      updated_at: "2025-01-19T16:00:00Z",
      completed_at: "2025-01-19T16:00:00Z",
      max_sessions: 8,
      current_session: 8,
      files_total: 32,
      files_visited: 32,
      config: { max_files_per_session: 4 },
    },
  ];

  return scenarioName ? mockCampaigns.filter(c => c.scenario === scenarioName) : mockCampaigns;
}

export async function createCampaign(scenarioName: string, config: Campaign["config"]): Promise<Campaign> {
  // TODO: Replace with real API call when backend ready
  await new Promise(resolve => setTimeout(resolve, 150));
  return {
    id: `camp-${Date.now()}`,
    scenario: scenarioName,
    status: "created",
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    max_sessions: 10,
    current_session: 0,
    files_total: 0,
    files_visited: 0,
    config,
  };
}

export async function updateCampaignStatus(campaignId: string, status: "active" | "paused" | "terminated"): Promise<void> {
  // TODO: Replace with real API call when backend ready
  await new Promise(resolve => setTimeout(resolve, 100));
  console.log(`Updated campaign ${campaignId} to ${status}`);
}
