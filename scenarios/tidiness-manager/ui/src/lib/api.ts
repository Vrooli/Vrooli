import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

// Fallback to direct API URL if proxy metadata resolution fails
// In local development, use the API port from window.__VROOLI_CONFIG__ or default to localhost
const API_BASE = (() => {
  try {
    const resolved = resolveApiBase({ appendSuffix: true });
    console.log('[tidiness-manager] resolveApiBase returned:', resolved);

    const currentPort = window.location.port;
    const apiPort = (window as any).__VROOLI_CONFIG__?.apiPort || '16820';

    // If resolution returns empty, just a path, undefined, or points to the UI port (wrong!), use direct API port
    if (!resolved ||
        typeof resolved !== 'string' ||
        resolved.trim() === '' ||
        resolved.startsWith('/') ||
        (currentPort && resolved.includes(`:${currentPort}`))) {
      const fallback = `http://localhost:${apiPort}/api/v1`;
      console.log('[tidiness-manager] Using fallback API base (resolved pointed to UI port or was invalid):', fallback);
      return fallback;
    }

    console.log('[tidiness-manager] Using resolved API base:', resolved);
    return resolved;
  } catch (e) {
    // Fallback to default localhost API
    const fallback = 'http://localhost:16820/api/v1';
    console.warn('[tidiness-manager] Failed to resolve API base, using fallback:', fallback, e);
    return fallback;
  }
})();

console.log('[tidiness-manager] Final API_BASE:', API_BASE);

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
  const res = await fetch(buildApiUrl("/health", { baseUrl: API_BASE }), {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`API health check failed: ${res.status}`);
  }

  return res.json();
}

export async function lightScan(scenarioPath: string, timeoutSec?: number): Promise<ScanResult> {
  const res = await fetch(buildApiUrl("/scan/light", { baseUrl: API_BASE }), {
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
  const res = await fetch(buildApiUrl("/scan/light/parse-lint", { baseUrl: API_BASE }), {
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
  const res = await fetch(buildApiUrl("/scan/light/parse-type", { baseUrl: API_BASE }), {
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
  const res = await fetch(buildApiUrl("/agent/scenarios", { baseUrl: API_BASE }), {
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
  // TODO: Replace with real API call when backend ready
  await new Promise(resolve => setTimeout(resolve, 200));

  const stats = MOCK_SCENARIOS.find(s => s.scenario === scenarioName) || MOCK_SCENARIOS[0];

  const files: FileStats[] = [
    { path: "api/main.go", lines: 250, lint_issues: 3, type_issues: 0, ai_issues: 1, visit_count: 5, is_long_file: false, extension: ".go" },
    { path: "api/handlers.go", lines: 650, lint_issues: 5, type_issues: 2, ai_issues: 3, visit_count: 8, is_long_file: true, extension: ".go", last_visited: "2025-01-19T10:00:00Z" },
    { path: "ui/src/App.tsx", lines: 180, lint_issues: 1, type_issues: 1, ai_issues: 0, visit_count: 12, is_long_file: false, extension: ".tsx" },
    { path: "ui/src/components/Dashboard.tsx", lines: 380, lint_issues: 2, type_issues: 3, ai_issues: 2, visit_count: 3, is_long_file: false, extension: ".tsx" },
    { path: "api/light_scanner.go", lines: 320, lint_issues: 1, type_issues: 0, ai_issues: 1, visit_count: 2, is_long_file: false, extension: ".go" },
    { path: "api/parsers.go", lines: 180, lint_issues: 0, type_issues: 0, ai_issues: 0, visit_count: 7, is_long_file: false, extension: ".go" },
  ];

  return { stats, files };
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
  // TODO: Replace with real API call when backend ready
  await new Promise(resolve => setTimeout(resolve, 200));

  let issues: Issue[] = [
    { scenario: scenarioName, file: "api/main.go", line: 42, column: 10, message: "Variable 'result' is never used", rule: "no-unused-vars", severity: "warning", tool: "eslint", category: "lint", status: "open", created_at: "2025-01-20T10:00:00Z" },
    { scenario: scenarioName, file: "api/handlers.go", line: 95, column: 5, message: "Function complexity too high (15/10)", rule: "complexity", severity: "error", tool: "eslint", category: "lint", status: "open", created_at: "2025-01-20T10:00:00Z" },
    { scenario: scenarioName, file: "ui/src/App.tsx", line: 128, column: 20, message: "Type 'string | undefined' is not assignable to type 'string'", severity: "error", tool: "tsc", category: "type", status: "resolved", created_at: "2025-01-20T10:00:00Z", resolved_at: "2025-01-20T14:00:00Z" },
    { scenario: scenarioName, file: "api/handlers.go", line: 156, column: 8, message: "Consider extracting this logic into a separate function", severity: "warning", tool: "claude-code", category: "ai", status: "open", created_at: "2025-01-20T11:30:00Z" },
  ];

  if (filters?.status) {
    issues = issues.filter(i => i.status === filters.status);
  }
  if (filters?.category) {
    issues = issues.filter(i => i.category === filters.category);
  }
  if (filters?.severity) {
    issues = issues.filter(i => i.severity === filters.severity);
  }

  return issues;
}

export async function updateIssueStatus(scenarioName: string, filePath: string, line: number, status: "resolved" | "ignored", notes?: string): Promise<void> {
  // TODO: Replace with real API call when backend ready
  await new Promise(resolve => setTimeout(resolve, 100));
  console.log(`Updated issue at ${filePath}:${line} to ${status}`, notes);
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
