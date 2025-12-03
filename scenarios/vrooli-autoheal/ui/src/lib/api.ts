// API client for Vrooli Autoheal
// [REQ:UI-HEALTH-001] [REQ:UI-EVENTS-001] [REQ:UI-REFRESH-001]
import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

export type HealthStatus = "ok" | "warning" | "critical";

// Category groups related health checks for UI organization
export type CheckCategory = "infrastructure" | "resource" | "scenario";

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
}

export interface StatusResponse {
  status: HealthStatus;
  platform: PlatformCapabilities;
  summary: HealthSummary;
  checks: HealthResult[];
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

// Structured error response from the API
// [REQ:FAIL-SAFE-001]
export interface APIErrorResponse {
  success: false;
  error: string;
  message: string;
  requestId?: string;
  timestamp: string;
}

// Custom error class with structured error information
export class APIError extends Error {
  code: string;
  requestId?: string;
  statusCode: number;
  isRetryable: boolean;

  constructor(
    message: string,
    code: string,
    statusCode: number,
    requestId?: string
  ) {
    super(message);
    this.name = "APIError";
    this.code = code;
    this.statusCode = statusCode;
    this.requestId = requestId;
    // Determine if error is retryable based on status code
    this.isRetryable = statusCode >= 500 || statusCode === 0 || statusCode === 408;
  }

  // User-friendly error message based on error code
  getUserMessage(): string {
    switch (this.code) {
      case "DATABASE_ERROR":
        return "Database is temporarily unavailable. Your data is safe.";
      case "NOT_FOUND":
        return this.message; // Already user-friendly
      case "TIMEOUT":
        return "The request took too long. Please try again.";
      case "SERVICE_UNAVAILABLE":
        return "A required service is currently unavailable.";
      default:
        return "Something went wrong. Please try again.";
    }
  }

  // Suggested action for the user
  getSuggestedAction(): string {
    if (this.isRetryable) {
      return "Try again in a few seconds.";
    }
    switch (this.code) {
      case "NOT_FOUND":
        return "The requested item may have been removed.";
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
  } catch (err) {
    // Network error - API unreachable
    throw new APIError(
      "Unable to connect to the API",
      "NETWORK_ERROR",
      0
    );
  }

  if (!res.ok) {
    // Try to parse structured error response
    try {
      const errorBody = (await res.json()) as APIErrorResponse;
      throw new APIError(
        errorBody.message || `Request failed: ${res.statusText}`,
        errorBody.error || "UNKNOWN_ERROR",
        res.status,
        errorBody.requestId
      );
    } catch (parseErr) {
      // Failed to parse error response, use generic error
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
  };
}

export async function fetchTimeline(): Promise<TimelineResponse> {
  return apiRequest<TimelineResponse>("/timeline");
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
// Incidents API - Status transitions
// ============================================================================

export interface Incident {
  timestamp: string;
  checkId: string;
  fromStatus: string;
  toStatus: string;
  message: string;
}

export interface IncidentsResponse {
  incidents: Incident[];
  windowHours: number;
  total: number;
}

export async function fetchIncidents(hours = 24, limit = 50): Promise<IncidentsResponse> {
  return apiRequest<IncidentsResponse>(`/incidents?hours=${hours}&limit=${limit}`);
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
};

/**
 * Groups an array of health results by their status.
 * This is the central decision point for UI display grouping.
 *
 * @param checks - Array of health check results
 * @returns Object with checks grouped by status (critical, warning, ok)
 */
export function groupChecksByStatus(checks: HealthResult[]): {
  critical: HealthResult[];
  warning: HealthResult[];
  ok: HealthResult[];
} {
  return {
    critical: checks.filter((c) => c.status === "critical"),
    warning: checks.filter((c) => c.status === "warning"),
    ok: checks.filter((c) => c.status === "ok"),
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
