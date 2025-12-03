// API client for Vrooli Autoheal
// [REQ:UI-HEALTH-001] [REQ:UI-EVENTS-001] [REQ:UI-REFRESH-001]
import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

export type HealthStatus = "ok" | "warning" | "critical";

export interface HealthResult {
  checkId: string;
  status: HealthStatus;
  message: string;
  details?: Record<string, unknown>;
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
  description: string;
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

async function apiRequest<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const url = buildApiUrl(endpoint, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
    ...options,
  });

  if (!res.ok) {
    throw new Error(`API request failed: ${res.status} ${res.statusText}`);
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
