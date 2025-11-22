// Application Types
export interface AppDependency {
  name: string;
  type: string;
  description?: string;
  required: boolean;
  enabled: boolean;
  status: string;
  running: boolean;
  healthy: boolean;
  installed: boolean;
  note?: string;
}

export interface App {
  id: string;
  name: string;
  scenario_name: string;
  path: string;
  created_at: string;
  updated_at: string;
  status: 'running' | 'stopped' | 'error' | 'degraded' | 'healthy' | 'unknown' | 'unhealthy';
  port_mappings: Record<string, number>;
  environment: Record<string, unknown>;
  config: Record<string, unknown>;
  // Legacy fields for compatibility
  port?: string | number;
  uptime?: string;
  description?: string;
  type?: 'scenario' | 'resource';
  runtime?: string;
  tags?: string[];
  health_status?: string;
  is_partial?: boolean;
  view_count?: number;
  first_viewed_at?: string | null;
  last_viewed_at?: string | null;
  // Detailed insights (populated by GetApp for single app queries)
  tech_stack?: string[];
  dependencies?: AppDependency[];
  // Completeness metrics (populated by Pass 3 hydration)
  completeness_score?: number;
  completeness_classification?: string;
}

export interface AppProxyPortInfo {
  port: number;
  label?: string | null;
  slug?: string | null;
  path: string;
  aliases?: string[];
  source?: string;
  isPrimary?: boolean;
}

export interface AppProxyMetadata {
  appId: string;
  generatedAt: number;
  hosts: string[];
  primary: AppProxyPortInfo;
  ports: AppProxyPortInfo[];
}

export interface LocalhostUsageFinding {
  file_path: string;
  line: number;
  snippet: string;
  pattern: string;
}

export interface LocalhostUsageReport {
  scenario: string;
  checked_at: string;
  files_scanned: number;
  findings: LocalhostUsageFinding[];
  warnings?: string[];
}

export interface AppViewStats {
  scenario_name: string;
  view_count: number;
  first_viewed_at?: string | null;
  last_viewed_at?: string | null;
}

// Resource Types
export interface Resource {
  id: string;
  name: string;
  type: string;
  status: 'online' | 'offline' | 'error' | 'stopped' | 'unknown' | 'unregistered';
  description?: string;
  icon?: string;
  enabled?: boolean;
  running?: boolean;
  enabled_known?: boolean;
  status_detail?: string;
}

export interface ResourceStatusSummary {
  id: string;
  name: string;
  type: string;
  status: Resource['status'] | string;
  enabled?: boolean;
  enabledKnown?: boolean;
  running?: boolean;
  statusDetail?: string;
}

export interface ResourcePaths {
  serviceConfig?: string;
  runtimeConfig?: string;
  capabilities?: string;
  schema?: string;
}

export interface ResourceDetail {
  id: string;
  name: string;
  category?: string;
  description?: string;
  summary: ResourceStatusSummary;
  cliStatus?: Record<string, unknown>;
  serviceConfig?: Record<string, unknown>;
  runtimeConfig?: Record<string, unknown>;
  capabilityMetadata?: Record<string, unknown>;
  schema?: Record<string, unknown>;
  paths?: ResourcePaths;
}

// System Metrics
export interface SystemMetrics {
  cpu: number;
  memory: number;
  disk: number;
  network: number;
  timestamp?: string;
}

// Log Entry
export interface LogEntry {
  timestamp: string;
  level: 'info' | 'warning' | 'error' | 'debug' | 'success' | 'log';
  message: string;
  app?: string;
  type?: 'lifecycle' | 'background';
}

// WebSocket Message Types
export interface WSMessage {
  type: 'connection' | 'app_update' | 'metric_update' | 'log_entry' | 'command_response' | 'error';
  payload: unknown;
}

// API Response Types
export interface ApiResponse<T = unknown> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
  warning?: string;
}

export interface BridgeRuleViolation {
  rule_id?: string;
  type: string;
  title: string;
  description: string;
  file_path: string;
  line: number;
  recommendation: string;
  severity: string;
  standard?: string;
}

export interface BridgeRuleReport {
  rule_id: string;
  name?: string;
  scenario: string;
  files_scanned: number;
  duration_ms: number;
  warning?: string;
  warnings?: string[];
  targets?: string[];
  violations: BridgeRuleViolation[];
  checked_at: string;
}

export interface BridgeDiagnosticsReport {
  scenario: string;
  checked_at: string;
  files_scanned: number;
  duration_ms: number;
  warning?: string;
  warnings?: string[];
  targets?: string[];
  violations: BridgeRuleViolation[];
  results: BridgeRuleReport[];
}

export interface ScenarioAuditorArtifactRef {
  path: string;
  checksum?: string;
  size_bytes?: number;
  created_at?: string;
}

export interface ScenarioAuditorViolationExcerpt {
  id: string;
  severity: string;
  rule_id?: string;
  title?: string;
  file_path?: string;
  line_number?: number;
  scenario?: string;
  source?: string;
  recommendation?: string;
}

export interface ScenarioAuditorSummary {
  total: number;
  by_severity: Record<string, number>;
  by_rule?: Array<Record<string, unknown>>;
  highest_severity: string;
  top_violations: ScenarioAuditorViolationExcerpt[];
  artifact?: ScenarioAuditorArtifactRef | null;
  recommended_steps?: string[];
  generated_at: string;
}

export interface AppLogStream {
  key: string;
  label: string;
  type: 'lifecycle' | 'background';
  phase?: string;
  step?: string;
  command?: string;
  lines: string[];
}

// App Logs Response
export interface AppLogsResponse {
  logs: string[];
  streams?: AppLogStream[];
  hasMore?: boolean;
  error?: string;
}

// Terminal Command
export interface TerminalCommand {
  command: string;
  timestamp: string;
  output?: string;
  error?: string;
}

// View Types
export type ViewType = 'apps' | 'logs' | 'resources' | 'terminal';
export type AppViewMode = 'grid' | 'list';

// Filter/Sort Options
export interface FilterOptions {
  search?: string;
  status?: string[];
  type?: string[];
}

export interface SortOptions {
  field: 'name' | 'status' | 'port' | 'cpu' | 'memory';
  direction: 'asc' | 'desc';
}

// Complete Diagnostics Types
export interface DiagnosticWarning {
  source: string; // "health", "bridge", "localhost", "issues", "status"
  severity: string; // "warn", "error", "info"
  message: string;
  file_path?: string;
  line?: number;
}

export interface TechStackResource {
  type: string; // "postgres", "redis", "ollama", etc.
  enabled: boolean;
  required: boolean;
  purpose?: string;
}

export interface TechStackInfo {
  runtime?: string; // "Go", "Node.js", etc.
  processes: number;
  resources?: TechStackResource[];
  tags?: string[];
  dependencies?: Record<string, unknown>;
  ports?: Record<string, number>;
}

export interface AppDocumentInfo {
  name: string;
  path: string;
  size: number;
  is_markdown: boolean;
  modified_at: string;
}

export interface AppDocumentsList {
  root_docs: AppDocumentInfo[];
  docs_docs: AppDocumentInfo[];
  total: number;
}

export interface AppDocument {
  name: string;
  path: string;
  size: number;
  is_markdown: boolean;
  modified_at: string;
  content: string;
  rendered_html?: string;
}

export interface AppDocumentMatch {
  document: AppDocumentInfo;
  matches: Array<{
    line_number: number;
    line: string;
    context?: string;
  }>;
  score: number;
}

export interface AppScenarioStatus {
  app_id: string;
  scenario: string;
  captured_at?: string;
  status_label: string;
  severity: 'ok' | 'warn' | 'error';
  runtime?: string;
  process_count?: number;
  ports?: Record<string, number>;
  recommendations?: string[];
  details: string[];
}

export interface AppHealthCheck {
  id: string;
  name: string;
  status: string; // "pass", "fail", "warn"
  endpoint?: string;
  latency_ms?: number;
  message?: string;
  code?: string;
  response?: string;
}

export interface AppHealthDiagnostics {
  app_id: string;
  app_name?: string;
  scenario?: string;
  captured_at: string;
  ports?: Record<string, number>;
  checks: AppHealthCheck[];
  errors?: string[];
}

export interface AppIssuesSummary {
  scenario: string;
  app_id: string;
  issues: Array<{
    id: string;
    title: string;
    status: string;
    priority: string;
    created_at: string;
    updated_at: string;
    reporter: string;
    issue_url?: string;
  }>;
  open_count: number;
  active_count: number;
  total_count: number;
  tracker_url?: string;
  last_fetched: string;
  from_cache: boolean;
  stale: boolean;
}

export interface CompleteDiagnostics {
  app_id: string;
  scenario: string;
  captured_at: string;

  // Status & Health
  scenario_status?: AppScenarioStatus;
  health_checks?: AppHealthDiagnostics;

  // Issues
  issues?: AppIssuesSummary;

  // Compliance
  bridge_rules?: BridgeDiagnosticsReport;
  localhost_usage?: LocalhostUsageReport;
  auditor_summary?: ScenarioAuditorSummary;

  // Metadata
  tech_stack?: TechStackInfo;
  documents?: AppDocumentsList;

  // Aggregated Summary
  warnings: DiagnosticWarning[];
  severity: 'ok' | 'warn' | 'error' | 'unknown' | 'degraded';
  summary?: string;
}

// Completeness Score Types
export interface CompletenessScore {
  scenario: string;
  details: string[];
}
