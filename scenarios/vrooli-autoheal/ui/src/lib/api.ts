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
