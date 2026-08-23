// API client for Vrooli Autoheal
// [REQ:UI-HEALTH-001] [REQ:UI-EVENTS-001] [REQ:UI-REFRESH-001]
import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

export type HealthStatus = "ok" | "warning" | "critical" | "not-applicable";

// AI_CHECK: REACT_STABILITY_AUTOHEAL=1 | LAST: 2026-02-18
const HEALTH_STATUSES: ReadonlySet<HealthStatus> = new Set(["ok", "warning", "critical", "not-applicable"]);

export function isHealthStatus(value: unknown): value is HealthStatus {
  return typeof value === "string" && HEALTH_STATUSES.has(value as HealthStatus);
}

export function normalizeHealthStatus(value: unknown, fallback: HealthStatus = "ok"): HealthStatus {
  if (isHealthStatus(value)) {
    return value;
  }
  return fallback;
}

// Category groups related health checks for UI organization
export type CheckCategory = "infrastructure" | "resource" | "scenario" | "system";

// SubCheck represents a single sub-check within a compound health check
export interface SubCheck {
  name: string;
  passed: boolean;
  detail?: string;
}

// HealthMetrics provides structured health information beyond simple status
export interface HealthMetrics {
  score?: number; // 0-100, where 100 is fully healthy
  subChecks?: SubCheck[];
}

export interface HealthResult {
  checkId: string;
  status: HealthStatus;
  message: string;
  details?: Record<string, unknown>;
  metrics?: HealthMetrics;
  timestamp: string;
  duration: number;
}

export interface PlatformCapabilities {
  platform: "linux" | "windows" | "macos" | "other";
  supportsRdp: boolean;
  supportsSystemd: boolean;
  supportsLaunchd: boolean;
  supportsWindowsServices: boolean;
  isHeadlessServer: boolean;
  hasDocker: boolean;
  isWsl: boolean;
  supportsCloudflared: boolean;
}

export interface HealthSummary {
  total: number;
  ok: number;
  warning: number;
  critical: number;
  notApplicable?: number;
}

export interface StatusResponse {
  status: HealthStatus;
  platform: PlatformCapabilities;
  summary: HealthSummary;
	checks: HealthResult[];
	autoHealSkips?: ActionLog[];
	/** Latest failed or skipped recovery outcome keyed by check id. */
	autoHealIssues?: Record<string, ActionLog>;
	tickRunning?: boolean;
  tickStartedAt?: string | null;
  timestamp: string;
}

export interface TickResponse {
  success: boolean;
  status: HealthStatus;
  summary: HealthSummary;
  results: HealthResult[];
  timestamp: string;
}

export interface CheckInfo {
  id: string;
  title: string;
  description: string;
  importance: string;
  category: CheckCategory;
  intervalSeconds: number;
  platforms?: string[];
}

export interface HealthResponse {
  status: string;
  service: string;
  version: string;
  readiness: boolean;
  timestamp: string;
  dependencies: Record<string, string>;
}

// Recovery guidance from the API (machine-readable)
// [REQ:FAIL-SAFE-001]
export interface RecoveryInfo {
  /** Recommended recovery action: retry, fix_input, report, wait, none */
  action: "retry" | "fix_input" | "report" | "wait" | "none";
  /** Whether the client should attempt to retry */
  retryable: boolean;
  /** Human-readable recovery suggestion from the API */
  hint?: string;
}

// Structured error response from the API
export interface APIErrorResponse {
  success: false;
  error: string;
  message: string;
  recovery?: RecoveryInfo;
  requestId?: string;
  timestamp: string;
}

// Custom error class with structured error information.
// Recovery semantics come from the API when available, with sensible
// client-side defaults as fallback (e.g., for network errors).
export class APIError extends Error {
  code: string;
  requestId?: string;
  statusCode: number;
  isRetryable: boolean;
  recovery: RecoveryInfo;

  constructor(
    message: string,
    code: string,
    statusCode: number,
    requestId?: string,
    recovery?: RecoveryInfo
  ) {
    super(message);
    this.name = "APIError";
    this.code = code;
    this.statusCode = statusCode;
    this.requestId = requestId;
    // Use API-provided recovery info when available; fall back to heuristic
    this.recovery = recovery ?? this.defaultRecovery(code, statusCode);
    this.isRetryable = this.recovery.retryable;
  }

  /** Derive recovery info when the API response doesn't include it (e.g., network errors). */
  private defaultRecovery(code: string, statusCode: number): RecoveryInfo {
    if (code === "NETWORK_ERROR" || statusCode === 0) {
      return { action: "retry", retryable: true, hint: "Check your network connection." };
    }
    if (statusCode === 408 || statusCode === 504) {
      return { action: "retry", retryable: true, hint: "The request timed out. Try again." };
    }
    if (statusCode >= 500) {
      return { action: "retry", retryable: true, hint: "A server error occurred." };
    }
    if (statusCode === 400) {
      return { action: "fix_input", retryable: false, hint: "Check the request data." };
    }
    if (statusCode === 404) {
      return { action: "none", retryable: false, hint: "The requested item was not found." };
    }
    if (statusCode === 409) {
      return { action: "wait", retryable: true, hint: "Wait for the current operation to finish." };
    }
    return { action: "none", retryable: false };
  }

  /** User-friendly error message. Prefers the API-provided hint when available. */
  getUserMessage(): string {
    // Prefer code-specific messages so user guidance stays stable.
    switch (this.code) {
      case "NETWORK_ERROR":
        return "Unable to connect to the API. Check your network connection.";
      case "DATABASE_ERROR":
        return "Database is temporarily unavailable. Your data is safe.";
      case "NOT_FOUND":
        return this.message;
      case "TIMEOUT":
        return "The request took too long. Please try again.";
      case "SERVICE_UNAVAILABLE":
        return "A required service is currently unavailable.";
      case "CONFLICT":
        return "Another operation is in progress. Please wait.";
      case "VALIDATION_ERROR":
        return this.message;
      default:
        return "Something went wrong. Please try again.";
    }
  }

  /** Suggested next action for the user, based on recovery semantics. */
  getSuggestedAction(): string {
    if (this.code === "NOT_FOUND") {
      return "This item may have been removed. If this persists, check if the scenario is running.";
    }

    if (!this.isRetryable && this.recovery.action !== "report") {
      return "If this persists, check if the scenario is running.";
    }

    switch (this.recovery.action) {
      case "retry":
        return "Try again in a few seconds.";
      case "fix_input":
        return "Check your input and try again.";
      case "report":
        return this.requestId
          ? `If this persists, report request ID: ${this.requestId}`
          : "If this persists, check if the scenario is running.";
      case "wait":
        return "Wait for the current operation to finish, then try again.";
      case "none":
      default:
        return "If this persists, check if the scenario is running.";
    }
  }
}

async function apiRequest<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const url = buildApiUrl(endpoint, { baseUrl: API_BASE });

  let res: Response;
  try {
    res = await fetch(url, {
      headers: { "Content-Type": "application/json" },
      cache: "no-store",
      ...options,
    });
  } catch {
    // Network error - API unreachable
    throw new APIError(
      "Unable to connect to the API",
      "NETWORK_ERROR",
      0
    );
  }

  if (!res.ok) {
    // Try to parse structured error response (includes recovery hints)
    try {
      const errorBody = (await res.json()) as APIErrorResponse;
      throw new APIError(
        errorBody.message || `Request failed: ${res.statusText}`,
        errorBody.error || "UNKNOWN_ERROR",
        res.status,
        errorBody.requestId,
        errorBody.recovery
      );
    } catch (parseErr) {
      // Failed to parse error response, use generic error with heuristic recovery
      if (parseErr instanceof APIError) throw parseErr;
      throw new APIError(
        `Request failed: ${res.statusText}`,
        "UNKNOWN_ERROR",
        res.status
      );
    }
  }

  return res.json() as Promise<T>;
}

export async function fetchHealth(): Promise<HealthResponse> {
  return apiRequest<HealthResponse>("/health");
}

export async function fetchStatus(): Promise<StatusResponse> {
  return apiRequest<StatusResponse>("/status");
}

export async function fetchPlatform(): Promise<PlatformCapabilities> {
  return apiRequest<PlatformCapabilities>("/platform");
}

export async function fetchChecks(): Promise<CheckInfo[]> {
  return apiRequest<CheckInfo[]>("/checks");
}

export async function runTick(force = false): Promise<TickResponse> {
  const endpoint = force ? "/tick?force=true" : "/tick";
  return apiRequest<TickResponse>(endpoint, { method: "POST" });
}

export interface HistoryEntry {
  checkId: string;
  status: HealthStatus;
  message: string;
  details?: Record<string, unknown>;
  timestamp: string;
  duration: number;
}

export interface CheckHistoryResponse {
  checkId: string;
  history: HistoryEntry[];
  count: number;
}

export async function fetchCheckHistory(checkId: string): Promise<CheckHistoryResponse> {
  return apiRequest<CheckHistoryResponse>(`/checks/${encodeURIComponent(checkId)}/history`);
}

// Timeline API types
export interface TimelineEvent {
  checkId: string;
  status: HealthStatus;
  message: string;
  details?: Record<string, unknown>;
  timestamp: string;
}

export interface TimelineResponse {
  events: TimelineEvent[];
  count: number;
  summary: {
    ok: number;
    warning: number;
    critical: number;
    notApplicable?: number;
  };
}

export async function fetchTimeline(): Promise<TimelineResponse> {
  return apiRequest<TimelineResponse>("/timeline");
}

export type SystemEventSeverity = "info" | "warning" | "critical";

export interface SystemEvent {
  id: number;
  fingerprint: string;
  occurredAt: string;
  ingestedAt?: string;
  source: string;
  platform: string;
  category: string;
  severity: SystemEventSeverity;
  title: string;
  summary: string;
  bootId?: string;
  details?: Record<string, unknown>;
}

export interface SystemEventSourceStatus {
  source: string;
  platform: string;
  status: "ok" | "unsupported" | "degraded" | "failed";
  lastIngestedAt?: string;
  lastError?: string;
  capabilities?: Record<string, unknown>;
}

export interface SystemEventCorrelation {
  title: string;
  summary: string;
  rationale: string;
  eventIds: number[];
  eventSources: string[];
  timeDelta?: string;
  confidence: string;
}

export interface SystemEventsResponse {
  events: SystemEvent[];
  count: number;
  sources: SystemEventSourceStatus[];
  correlations?: SystemEventCorrelation[];
}

export interface SystemEventsRefreshResponse {
  ingested: number;
  deduped: number;
  sources: SystemEventSourceStatus[];
  durationMs: number;
}

export interface SystemEventsParams {
  since?: string;
  until?: string;
  category?: string;
  severity?: string;
  source?: string;
  limit?: number;
  correlate?: boolean;
}

export async function fetchSystemEvents(params: SystemEventsParams = {}): Promise<SystemEventsResponse> {
  const query = new URLSearchParams();
  if (params.since) query.set("since", params.since);
  if (params.until) query.set("until", params.until);
  if (params.category) query.set("category", params.category);
  if (params.severity) query.set("severity", params.severity);
  if (params.source) query.set("source", params.source);
  if (params.limit) query.set("limit", String(params.limit));
  if (params.correlate) query.set("correlate", "true");
  const suffix = query.toString();
  return apiRequest<SystemEventsResponse>(`/system-events${suffix ? `?${suffix}` : ""}`);
}

export async function refreshSystemEvents(): Promise<SystemEventsRefreshResponse> {
  return apiRequest<SystemEventsRefreshResponse>("/system-events/refresh", { method: "POST" });
}

// Uptime stats API types
export interface UptimeStatsResponse {
  totalEvents: number;
  okEvents: number;
  warningEvents: number;
  criticalEvents: number;
  uptimePercentage: number;
  windowHours: number;
}

export async function fetchUptimeStats(): Promise<UptimeStatsResponse> {
  return apiRequest<UptimeStatsResponse>("/uptime");
}

// Uptime history API types (time-bucketed data)
export interface UptimeHistoryBucket {
  timestamp: string;
  total: number;
  ok: number;
  warning: number;
  critical: number;
}

export interface UptimeHistoryResponse {
  buckets: UptimeHistoryBucket[];
  overall: {
    uptimePercentage: number;
    totalEvents: number;
  };
  windowHours: number;
  bucketCount: number;
}

export async function fetchUptimeHistory(hours = 24, buckets = 24): Promise<UptimeHistoryResponse> {
  return apiRequest<UptimeHistoryResponse>(`/uptime/history?hours=${hours}&buckets=${buckets}`);
}

// ============================================================================
// Check Trends API - Per-check trend data
// ============================================================================

export interface CheckTrend {
  checkId: string;
  total: number;
  ok: number;
  warning: number;
  critical: number;
  uptimePercent: number;
  currentStatus: string;
  recentStatuses: string[];
  lastChecked: string;
}

export interface CheckTrendsResponse {
  trends: CheckTrend[];
  windowHours: number;
  totalChecks: number;
}

export async function fetchCheckTrends(hours = 24): Promise<CheckTrendsResponse> {
  return apiRequest<CheckTrendsResponse>(`/checks/trends?hours=${hours}`);
}

// ============================================================================
// Transitions API - Status transitions
// ============================================================================

export interface Transition {
  timestamp: string;
  checkId: string;
  fromStatus: string;
  toStatus: string;
  message: string;
}

export interface TransitionsResponse {
  transitions: Transition[];
  windowHours: number;
  total: number;
}

export async function fetchTransitions(hours = 24, limit = 50): Promise<TransitionsResponse> {
  return apiRequest<TransitionsResponse>(`/transitions?hours=${hours}&limit=${limit}`);
}

// ============================================================================
// Incidents API - Durable incident workflow
// ============================================================================

export type IncidentSeverity = "info" | "warning" | "critical";
export type IncidentStatus = "open" | "acknowledged" | "resolved" | "ignored";
export type IncidentType =
  | "host_integrity"
  | "unclean_boot"
  | "resource_failure"
  | "scenario_failure"
  | "autoheal_failure"
  | "manual";

export interface Incident {
  id: string;
  fingerprint: string;
  type: IncidentType;
  severity: IncidentSeverity;
  status: IncidentStatus;
  title: string;
  summary: string;
  detectedAt: string;
  lastSeenAt: string;
  updatedAt: string;
  resolvedAt?: string;
  acknowledgedAt?: string;
  ignoredAt?: string;
  bootId?: string;
  previousBootId?: string;
  sourceCheckIds?: string[];
  sourceResultIds?: string[];
  evidence?: Record<string, unknown>;
  recommendations?: string[];
  eventCount: number;
  observationCount: number;
  operatorNotes?: string;
}

export interface IncidentObservation {
  id: number;
  incidentId: string;
  observedAt: string;
  sourceCheckId?: string;
  severity: IncidentSeverity;
  status?: string;
  message: string;
  evidence?: Record<string, unknown>;
}

export interface IncidentsResponse {
  incidents: Incident[];
  total: number;
  filters: Record<string, unknown>;
}

export async function fetchIncidents(params: {
  status?: IncidentStatus | "";
  severity?: IncidentSeverity | "";
  type?: IncidentType | "";
  limit?: number;
} = {}): Promise<IncidentsResponse> {
  const query = new URLSearchParams();
  if (params.status) query.set("status", params.status);
  if (params.severity) query.set("severity", params.severity);
  if (params.type) query.set("type", params.type);
  query.set("limit", String(params.limit ?? 50));
  return apiRequest<IncidentsResponse>(`/incidents?${query.toString()}`);
}

export async function fetchIncident(id: string): Promise<Incident> {
  return apiRequest<Incident>(`/incidents/${encodeURIComponent(id)}`);
}

export async function fetchIncidentObservations(id: string): Promise<{ observations: IncidentObservation[]; total: number }> {
  return apiRequest<{ observations: IncidentObservation[]; total: number }>(`/incidents/${encodeURIComponent(id)}/observations`);
}

export async function updateIncidentStatus(id: string, action: "acknowledge" | "resolve" | "ignore", note = ""): Promise<Incident> {
  return apiRequest<Incident>(`/incidents/${encodeURIComponent(id)}/${action}`, {
    method: "POST",
    body: JSON.stringify({ note }),
  });
}

// ============================================================================
// Watchdog API - OS-level service status
// [REQ:WATCH-DETECT-001]
// ============================================================================

export type WatchdogType = "" | "systemd" | "launchd" | "windows-task";
export type ProtectionLevel = "full" | "partial" | "none";

export interface WatchdogStatus {
  loopRunning: boolean;
  watchdogType: WatchdogType;
  watchdogInstalled: boolean;
  watchdogEnabled: boolean;
  watchdogRunning: boolean;
  bootProtectionActive: boolean;
  canInstall: boolean;
  servicePath?: string;
  lastError?: string;
  protectionLevel: ProtectionLevel;
  /** Whether systemd lingering is enabled (Linux user services only) */
  lingeringEnabled: boolean;
  /** Current username, for displaying fix commands */
  username?: string;
  /** Whether this is a user-level service vs system-level */
  isUserService?: boolean;
}

export interface WatchdogTemplateResponse {
  platform: string;
  template: string;
  instructions: string;
  oneLiner: string;
}

export async function fetchWatchdogStatus(refresh = false): Promise<WatchdogStatus> {
  const endpoint = refresh ? "/watchdog?refresh=true" : "/watchdog";
  return apiRequest<WatchdogStatus>(endpoint);
}

export async function fetchWatchdogTemplate(): Promise<WatchdogTemplateResponse> {
  return apiRequest<WatchdogTemplateResponse>("/watchdog/template");
}

// ============================================================================
// Watchdog Installation API
// [REQ:WATCH-INSTALL-001]
// ============================================================================

export interface InstallOptions {
  /** Install as system-wide service (requires root/admin) */
  useSystemService?: boolean;
  /** Automatically enable lingering for user services on Linux */
  enableLingering?: boolean;
}

export interface InstallResult {
  success: boolean;
  message: string;
  servicePath?: string;
  needsLinger?: boolean;
  lingerCommand?: string;
  error?: string;
}

export interface UninstallResult {
  success: boolean;
  message: string;
  error?: string;
}

export interface InstallStatus {
  installed: boolean;
  enabled: boolean;
  running: boolean;
  bootProtected: boolean;
  servicePath?: string;
  watchdogType: string;
  canInstall: boolean;
  needsLinger: boolean;
  lingerCommand?: string;
  protectionLevel: string;
  lastChecked: string;
  recommendedSetup: string;
}

/** Install the watchdog service */
export async function installWatchdog(opts: InstallOptions = {}): Promise<InstallResult> {
  return apiRequest<InstallResult>("/watchdog/install", {
    method: "POST",
    body: JSON.stringify(opts),
  });
}

/** Uninstall the watchdog service */
export async function uninstallWatchdog(): Promise<UninstallResult> {
  return apiRequest<UninstallResult>("/watchdog/uninstall", {
    method: "POST",
  });
}

/** Enable lingering for user services (Linux only) */
export async function enableLingering(): Promise<InstallResult> {
  return apiRequest<InstallResult>("/watchdog/linger", {
    method: "POST",
  });
}

/** Get detailed installation status */
export async function fetchInstallStatus(): Promise<InstallStatus> {
  return apiRequest<InstallStatus>("/watchdog/status");
}

// ============================================================================
// Status Classification Helpers
// These provide a central place for status-based decisions in the UI
// ============================================================================

/**
 * Status severity order - higher number means more severe.
 * Used for sorting checks to display most severe first.
 */
export const STATUS_SEVERITY: Record<HealthStatus, number> = {
  critical: 2,
  warning: 1,
  ok: 0,
  "not-applicable": -1,
};

/**
 * Groups an array of health results by their status.
 * This is the central decision point for UI display grouping.
 *
 * @param checks - Array of health check results
 * @returns Object with checks grouped by status (critical, warning, ok)
 */
export function groupChecksByStatus<T extends HealthResult>(checks: T[]): {
  critical: T[];
  warning: T[];
  ok: T[];
  notApplicable: T[];
} {
  return {
    critical: checks.filter((c) => c.status === "critical"),
    warning: checks.filter((c) => c.status === "warning"),
    ok: checks.filter((c) => c.status === "ok"),
    notApplicable: checks.filter((c) => c.status === "not-applicable"),
  };
}

/**
 * Sorts checks by severity (critical first, then warning, then ok).
 * Within each severity level, maintains original order.
 */
export function sortChecksBySeverity(checks: HealthResult[]): HealthResult[] {
  return [...checks].sort(
    (a, b) => STATUS_SEVERITY[b.status] - STATUS_SEVERITY[a.status]
  );
}

/**
 * Stable UI ordering for health checks so cards don't "shuffle" between refreshes.
 * Sort priority:
 *   1) severity (critical > warning > ok)
 *   2) category (infrastructure > resource > scenario > unknown)
 *   3) display name (title if present, otherwise checkId)
 *   4) checkId (final deterministic tie-breaker)
 */
export function sortChecksForDisplay<T extends HealthResult & { title?: string; category?: string }>(checks: T[]): T[] {
  const categoryOrder: Record<string, number> = {
    infrastructure: 0,
    resource: 1,
    scenario: 2,
  };

  return [...checks].sort((a, b) => {
    const severityDiff = STATUS_SEVERITY[b.status] - STATUS_SEVERITY[a.status];
    if (severityDiff !== 0) {
      return severityDiff;
    }

    const categoryA = a.category ? (categoryOrder[a.category] ?? 99) : 99;
    const categoryB = b.category ? (categoryOrder[b.category] ?? 99) : 99;
    if (categoryA !== categoryB) {
      return categoryA - categoryB;
    }

    const nameA = (a.title ?? a.checkId).toLowerCase();
    const nameB = (b.title ?? b.checkId).toLowerCase();
    const nameDiff = nameA.localeCompare(nameB);
    if (nameDiff !== 0) {
      return nameDiff;
    }

    return a.checkId.localeCompare(b.checkId);
  });
}

/**
 * Determines the overall status from a summary object.
 * Decision logic:
 *   - Any critical → "critical"
 *   - Any warning (no critical) → "warning"
 *   - All ok → "ok"
 */
export function overallStatusFromSummary(summary: HealthSummary): HealthStatus {
  if (summary.critical > 0) return "critical";
  if (summary.warning > 0) return "warning";
  return "ok";
}

/**
 * Maps status to emoji for document title/notifications.
 * Decision: ✓ for ok, ⚠ for warning, ✗ for critical
 */
export function statusToEmoji(status: HealthStatus): string {
  switch (status) {
    case "ok":
      return "\u2713"; // ✓
    case "warning":
      return "\u26A0"; // ⚠
    case "critical":
      return "\u2717"; // ✗
    default:
      return "\u2753"; // ?
  }
}

// ============================================================================
// Recovery Actions API
// [REQ:HEAL-ACTION-001]
// ============================================================================

export interface RecoveryAction {
  id: string;
  name: string;
  description: string;
  dangerous: boolean;
  available: boolean;
}

export interface CheckActionsResponse {
  checkId: string;
  actions: RecoveryAction[];
}

export interface ActionResult {
  actionId: string;
  checkId: string;
  success: boolean;
  message: string;
  output?: string;
  error?: string;
  timestamp: string;
  duration: number;
}

export interface ActionLog {
  id: number;
  checkId: string;
  actionId: string;
  success: boolean;
  message: string;
  output?: string;
  error?: string;
  durationMs: number;
  timestamp: string;
}

export interface ActionLogsResponse {
  logs: ActionLog[];
  total: number;
}

export async function fetchCheckActions(checkId: string): Promise<CheckActionsResponse> {
  return apiRequest<CheckActionsResponse>(`/checks/${encodeURIComponent(checkId)}/actions`);
}

export async function executeAction(checkId: string, actionId: string): Promise<ActionResult> {
  return apiRequest<ActionResult>(
    `/checks/${encodeURIComponent(checkId)}/actions/${encodeURIComponent(actionId)}`,
    { method: "POST" }
  );
}

export async function fetchActionHistory(checkId?: string): Promise<ActionLogsResponse> {
  const endpoint = checkId
    ? `/actions/history?checkId=${encodeURIComponent(checkId)}`
    : "/actions/history";
  return apiRequest<ActionLogsResponse>(endpoint);
}

// ============================================================================
// Configuration API
// [REQ:CONFIG-*]
// ============================================================================

export interface Thresholds {
  warningPercent?: number;
  criticalPercent?: number;
  warningCount?: number;
  criticalCount?: number;
  partitions?: string[];
}

export interface CheckSettings {
  tunnelTestUrl?: string;
  cleanPortsBeforeRestart?: boolean;
  captureLogsOnFailure?: boolean;
  logLinesToCapture?: number;
}

export interface CheckConfig {
  enabled?: boolean;
  autoHeal?: boolean;
  intervalSeconds?: number;
  thresholds?: Thresholds;
  settings?: CheckSettings;
}

export interface GlobalConfig {
  gracePeriodSeconds: number;
  tickIntervalSeconds: number;
  verifyDelaySeconds: number;
  maxRestartAttempts: number;
  restartCooldownSeconds: number;
  historyRetentionHours: number;
}

export interface UIConfig {
  autoRefreshSeconds: number;
  theme: "system" | "light" | "dark";
  showDisabledChecks: boolean;
  defaultTab: "dashboard" | "trends" | "docs";
}

export interface Config {
  version: string;
  global: GlobalConfig;
  checks?: Record<string, CheckConfig>; // Optional - may be empty/undefined if no overrides
  ui: UIConfig;
}

export interface ValidationError {
  path: string;
  message: string;
}

export interface ValidationResult {
  valid: boolean;
  errors?: ValidationError[];
}

export interface ConfigResponse {
  success: boolean;
  message: string;
  config: Config;
}

export interface CheckConfigResponse {
  checkId: string;
  config: {
    enabled: boolean;
    autoHeal: boolean;
    intervalSeconds: number;
    thresholds: Thresholds;
    settings: CheckSettings;
  };
}

export interface CheckDefaults {
  enabled: boolean;
  autoHeal: boolean;
  intervalSeconds: number;
  thresholds?: Thresholds;
}

export interface DefaultsResponse {
  global: GlobalConfig;
  ui: UIConfig;
  checks: Record<string, CheckDefaults>;
}

// Fetch current configuration
export async function fetchConfig(): Promise<Config> {
  return apiRequest<Config>("/config");
}

// Update entire configuration
export async function updateConfig(config: Config): Promise<ConfigResponse> {
  return apiRequest<ConfigResponse>("/config", {
    method: "PUT",
    body: JSON.stringify(config),
  });
}

// Validate configuration without saving
export async function validateConfig(config: Config): Promise<ValidationResult> {
  return apiRequest<ValidationResult>("/config/validate", {
    method: "POST",
    body: JSON.stringify(config),
  });
}

// Get JSON schema
export async function fetchConfigSchema(): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>("/config/schema");
}

// Export configuration as downloadable JSON
export async function exportConfig(): Promise<Blob> {
  const url = buildApiUrl("/config/export", { baseUrl: API_BASE });
  const res = await fetch(url);
  if (!res.ok) {
    throw new APIError("Failed to export configuration", "EXPORT_ERROR", res.status);
  }
  return res.blob();
}

// Import configuration from JSON
export async function importConfig(configJson: string): Promise<ConfigResponse> {
  return apiRequest<ConfigResponse>("/config/import", {
    method: "POST",
    body: configJson,
  });
}

// Get default values for all settings
export async function fetchDefaults(): Promise<DefaultsResponse> {
  return apiRequest<DefaultsResponse>("/config/defaults");
}

// Get global configuration
export async function fetchGlobalConfig(): Promise<GlobalConfig> {
  return apiRequest<GlobalConfig>("/config/global");
}

// Get UI configuration
export async function fetchUIConfig(): Promise<UIConfig> {
  return apiRequest<UIConfig>("/config/ui");
}

// Get effective configuration for a specific check
export async function fetchCheckConfig(checkId: string): Promise<CheckConfigResponse> {
  return apiRequest<CheckConfigResponse>(`/config/checks/${encodeURIComponent(checkId)}`);
}

// Update check enabled state
export async function setCheckEnabled(checkId: string, enabled: boolean): Promise<{ success: boolean; checkId: string; enabled: boolean }> {
  return apiRequest(`/config/checks/${encodeURIComponent(checkId)}/enabled`, {
    method: "PUT",
    body: JSON.stringify({ enabled }),
  });
}

// Update check autoHeal state
export async function setCheckAutoHeal(checkId: string, autoHeal: boolean): Promise<{ success: boolean; checkId: string; autoHeal: boolean }> {
  return apiRequest(`/config/checks/${encodeURIComponent(checkId)}/autoheal`, {
    method: "PUT",
    body: JSON.stringify({ autoHeal }),
  });
}

// Bulk update all checks
export async function bulkUpdateChecks(action: "enableAll" | "disableAll" | "autoHealAll" | "disableAutoHealAll"): Promise<ConfigResponse> {
  return apiRequest<ConfigResponse>("/config/checks/bulk", {
    method: "PUT",
    body: JSON.stringify({ action }),
  });
}

// ============================================================================
// Monitoring Configuration API
// Configure which scenarios and resources are monitored
// ============================================================================

export interface MonitoredScenario {
  critical: boolean;
}

export interface MonitoringConfig {
  scenarios: Record<string, MonitoredScenario>;
  resources: string[];
}

export interface MonitoringResponse {
  success: boolean;
  message: string;
  monitoring: MonitoringConfig;
}

// Fetch monitoring configuration
export async function fetchMonitoring(): Promise<MonitoringConfig> {
  return apiRequest<MonitoringConfig>("/config/monitoring");
}

// Update entire monitoring configuration
export async function updateMonitoring(monitoring: MonitoringConfig): Promise<MonitoringResponse> {
  return apiRequest<MonitoringResponse>("/config/monitoring", {
    method: "PUT",
    body: JSON.stringify(monitoring),
  });
}

// Add a scenario to monitoring
export async function addScenario(name: string, critical: boolean): Promise<MonitoringResponse> {
  return apiRequest<MonitoringResponse>("/config/monitoring/scenarios", {
    method: "POST",
    body: JSON.stringify({ name, critical }),
  });
}

// Remove a scenario from monitoring
export async function removeScenario(name: string): Promise<MonitoringResponse> {
  return apiRequest<MonitoringResponse>(`/config/monitoring/scenarios/${encodeURIComponent(name)}`, {
    method: "DELETE",
  });
}

// Set scenario criticality
export async function setScenarioCritical(name: string, critical: boolean): Promise<MonitoringResponse> {
  return apiRequest<MonitoringResponse>(`/config/monitoring/scenarios/${encodeURIComponent(name)}/critical`, {
    method: "PUT",
    body: JSON.stringify({ critical }),
  });
}

// Add a resource to monitoring
export async function addResource(name: string): Promise<MonitoringResponse> {
  return apiRequest<MonitoringResponse>("/config/monitoring/resources", {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

// Remove a resource from monitoring
export async function removeResource(name: string): Promise<MonitoringResponse> {
  return apiRequest<MonitoringResponse>(`/config/monitoring/resources/${encodeURIComponent(name)}`, {
    method: "DELETE",
  });
}
