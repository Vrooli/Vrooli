import axios, { AxiosError } from 'axios';
import type {
  App,
  Resource,
  LogEntry,
  ApiResponse,
  AppLogsResponse,
  AppViewStats,
  ResourceDetail,
  BridgeDiagnosticsReport,
  AppProxyMetadata,
  LocalhostUsageReport,
  CompleteDiagnostics,
  AppDocument,
  AppDocumentsList,
  AppDocumentMatch,
} from '@/types';
import type {
  ApiHealthResponse,
  UiHealthResponse,
  SystemStatusResponse,
  PreviewHealthDiagnosticsResponse,
  PreviewHealthDiagnosticsResult,
} from '@/types/health';
import { logger } from '@/services/logger';
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base';

const DEFAULT_API_PORT = (import.meta.env.VITE_API_PORT as string | undefined)?.trim() || '15100';

// CRITICAL: App-monitor must ALWAYS use an absolute API URL, never derive it from proxy paths
// When viewing app-monitor itself through a scenario preview, the api-base package would incorrectly
// derive the API URL as /apps/<scenario>/api/v1, breaking all API calls
const getExplicitApiUrl = (): string | undefined => {
  const envUrl = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.trim();
  if (envUrl) return envUrl;

  // Use the current page's origin (protocol + hostname + port from tunnel/proxy)
  // This works for:
  // - Local dev: http://localhost:15101
  // - Secure tunnel: https://app-monitor.itsagitime.com
  // - Production: https://yourdomain.com
  if (typeof window !== 'undefined' && window.location) {
    return window.location.origin;
  }

  // Fallback for SSR or edge cases
  return `http://127.0.0.1:${DEFAULT_API_PORT}`;
};

const API_BASE_URL = resolveApiBase({
  explicitUrl: getExplicitApiUrl(),
  defaultPort: DEFAULT_API_PORT,
  appendSuffix: true,
});

export const buildApiUrlWithBase = (path: string) => buildApiUrl(path, { baseUrl: API_BASE_URL });

// Create axios instance with default config
const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 35000, // 35s to accommodate 30s backend timeout for resource checks
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor for debugging
api.interceptors.request.use(
  (config) => {
    logger.logAPICall(config.method || 'GET', config.url || '', config.data);
    return config;
  },
  (error) => {
    logger.error('API Request error', error);
    return Promise.reject(error);
  }
);

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => {
    logger.logAPIResponse(response.config.url || '', response.status, response.data);
    return response;
  },
  (error: AxiosError<ApiResponse>) => {
    const message = error.response?.data?.error || error.message || 'Network error';
    logger.error('API Response error', { message, status: error.response?.status });

    // Enhanced error object
    const enhancedError = {
      ...error,
      message,
      status: error.response?.status,
      originalError: error,
    };

    return Promise.reject(enhancedError);
  }
);

export interface ReportIssueConsoleLogEntry {
  ts: number;
  level: string;
  source: 'console' | 'runtime';
  text: string;
}

export interface ReportIssueNetworkEntry {
  ts: number;
  kind: 'fetch' | 'xhr';
  method: string;
  url: string;
  status?: number;
  ok?: boolean;
  durationMs?: number;
  error?: string;
  requestId?: string;
}


export type ReportIssueAppStatusSeverity = 'ok' | 'warn' | 'error';

export interface ReportIssueAppStatus {
  appId: string;
  scenario: string;
  capturedAt?: string | null;
  statusLabel: string;
  severity: ReportIssueAppStatusSeverity;
  runtime?: string | null;
  processCount?: number | null;
  ports?: Record<string, number> | null;
  recommendations?: string[] | null;
  details: string[];
}

export interface ReportIssueHealthCheckEntry {
  id: string;
  name: string;
  status: 'pass' | 'warn' | 'fail';
  endpoint?: string | null;
  latency_ms?: number | null;
  latencyMs?: number | null;
  message?: string | null;
  code?: string | null;
  response?: string | null;
}

export interface BrowserlessFallbackConsoleLog {
  level: string;
  message: string;
  timestamp: string;
}

export interface BrowserlessFallbackNetworkRequest {
  requestId: string;
  url: string;
  method: string;
  resourceType: string;
  status?: number;
  ok?: boolean;
  statusText?: string;
  duration?: number;
  headers?: Record<string, string>;
  contentType?: string;
  failed?: boolean;
  error?: string;
  timestamp: string;
}

export interface BrowserlessFallbackPageStatus {
  whiteScreen: boolean;
  httpError?: {
    status: number;
    message: string;
  } | null;
  loadError?: string | null;
  moduleError?: string | null;
  resourceCount: number;
  loadTimeMs: number;
  performanceMetrics?: Record<string, any>;
}

export interface BrowserlessFallbackDiagnostics {
  consoleLogs: BrowserlessFallbackConsoleLog[];
  networkRequests: BrowserlessFallbackNetworkRequest[];
  pageStatus: BrowserlessFallbackPageStatus | null;
  screenshot?: string;
  source: string;
  capturedAt: string;
  url: string;
  title?: string;
}

export interface ReportIssueCapturePayload {
  id: string;
  type: 'page' | 'element';
  width: number;
  height: number;
  data: string;
  note?: string | null;
  selector?: string | null;
  tagName?: string | null;
  elementId?: string | null;
  classes?: string[] | null;
  label?: string | null;
  ariaDescription?: string | null;
  title?: string | null;
  role?: string | null;
  text?: string | null;
  boundingBox?: { x: number; y: number; width: number; height: number } | null;
  clip?: { x: number; y: number; width: number; height: number } | null;
  mode?: string | null;
  filename?: string | null;
  createdAt?: string | null;
}

// Re-export health types for convenience
export type { PreviewHealthDiagnosticsResponse, PreviewHealthDiagnosticsResult } from '@/types/health';

export interface ReportIssuePayload {
  message: string;
  primaryDescription?: string | null;
  includeDiagnosticsSummary?: boolean;
  includeScreenshot?: boolean;
  previewUrl?: string | null;
  appName?: string | null;
  scenarioName?: string | null;
  source?: string;
  screenshotData?: string | null;
  logs?: string[];
  logsTotal?: number;
  logsCapturedAt?: string | null;
  consoleLogs?: ReportIssueConsoleLogEntry[];
  consoleLogsTotal?: number;
  consoleLogsCapturedAt?: string | null;
  networkRequests?: ReportIssueNetworkEntry[];
  networkRequestsTotal?: number;
  networkCapturedAt?: string | null;
  healthChecks?: ReportIssueHealthCheckEntry[];
  healthChecksTotal?: number;
  healthChecksCapturedAt?: string | null;
  appStatusLines?: string[];
  appStatusLabel?: string | null;
  appStatusSeverity?: ReportIssueAppStatusSeverity | null;
  appStatusCapturedAt?: string | null;
  captures?: ReportIssueCapturePayload[];
}

export interface ScenarioIssueSummary {
  id: string;
  title: string;
  status: string;
  priority?: string;
  created_at?: string;
  updated_at?: string;
  reporter?: string;
  issue_url?: string;
}

export interface ScenarioIssuesSummary {
  scenario?: string;
  app_id?: string;
  issues?: ScenarioIssueSummary[];
  open_count?: number;
  active_count?: number;
  total_count?: number;
  tracker_url?: string;
  last_fetched?: string;
  from_cache?: boolean;
  stale?: boolean;
}

// App Management
export const appService = {
  // Get fast-loading summaries
  async getAppSummaries(): Promise<App[]> {
    return fetchArrayWithFallback(
      () => api.get<App[]>('/apps/summary'),
      'Failed to fetch app summaries',
    );
  },

  // Get all apps
  async getApps(): Promise<App[]> {
    return fetchArrayWithFallback(
      () => api.get<App[]>('/apps'),
      'Failed to fetch apps',
    );
  },

  // Get single app
  async getApp(id: string): Promise<App | null> {
    return fetchWrappedWithFallback(
      () => api.get<ApiResponse<App>>(`/apps/${id}`),
      `Failed to fetch app ${id}`,
    );
  },

  async getScenarioIssues(appId: string): Promise<ScenarioIssuesSummary | null> {
    const trimmed = appId.trim();
    if (!trimmed) {
      return null;
    }

    try {
      const { data } = await api.get<ApiResponse<ScenarioIssuesSummary>>(
        `/apps/${encodeURIComponent(trimmed)}/issues`,
      );
      if (data?.success === false) {
        logger.warn('Issue tracker responded with error', data?.error || data?.message);
        return null;
      }
      if (!data?.data) {
        return null;
      }
      return data.data;
    } catch (error) {
      logger.error(`Failed to load existing issues for ${trimmed}`, error);
      return null;
    }
  },

  // Control app (start/stop/restart)
  async controlApp(id: string, action: 'start' | 'stop' | 'restart'): Promise<boolean> {
    try {
      const { data } = await api.post<ApiResponse>(`/apps/${id}/${action}`);
      return data.success !== false;
    } catch (error) {
      logger.error(`Failed to ${action} app ${id}`, error);
      return false;
    }
  },

  // Get app logs
  async getAppLogs(appName: string, logType: 'lifecycle' | 'background' | 'both' = 'both'): Promise<AppLogsResponse> {
    try {
      const params = new URLSearchParams({ type: logType });
      const { data } = await api.get<AppLogsResponse>(`/logs/${appName}?${params}`);
      return {
        logs: Array.isArray(data?.logs) ? data.logs : [],
        streams: Array.isArray(data?.streams) ? data.streams : [],
        hasMore: data?.hasMore,
        error: data?.error,
      };
    } catch (error) {
      logger.error(`Failed to fetch logs for ${appName}`, error);
      return { logs: [], streams: [], error: 'Failed to fetch logs' };
    }
  },

  // Control all apps
  async controlAllApps(action: 'start' | 'stop' | 'restart'): Promise<boolean> {
    try {
      const { data } = await api.post<ApiResponse>(`/apps/all/${action}`);
      return data.success !== false;
    } catch (error) {
      logger.error(`Failed to ${action} all apps`, error);
      return false;
    }
  },

  async recordAppView(id: string): Promise<AppViewStats | null> {
    try {
      const { data } = await api.post<ApiResponse<AppViewStats>>(`/apps/${encodeURIComponent(id)}/view`);
      return data?.data ?? null;
    } catch (error) {
      logger.warn(`Failed to record view for app ${id}`, error);
      return null;
    }
  },

  async reportAppIssue(
    appId: string,
    payload: ReportIssuePayload,
  ): Promise<ApiResponse<{ issue_id?: string; issue_url?: string }>> {
    try {
      const { data} = await api.post<ApiResponse<{ issue_id?: string; issue_url?: string }>>(
        `/apps/${encodeURIComponent(appId)}/report`,
        payload,
      );
      return data;
    } catch (error) {
      logger.error(`Failed to report issue for app ${appId}`, error);
      throw error;
    }
  },

  async getFallbackDiagnostics(
    appId: string,
    url: string,
  ): Promise<BrowserlessFallbackDiagnostics | null> {
    try {
      const { data } = await api.post<ApiResponse<BrowserlessFallbackDiagnostics>>(
        `/apps/${encodeURIComponent(appId)}/fallback-diagnostics`,
        { url },
      );
      if (data?.success === false) {
        logger.warn(`Fallback diagnostics for ${appId} failed`, data?.error || data?.message);
        return null;
      }
      return data?.data ?? null;
    } catch (error) {
      logger.warn(`Failed to fetch fallback diagnostics for ${appId}`, error);
      return null;
    }
  },

  async getAppStatusSnapshot(appId: string): Promise<ReportIssueAppStatus | null> {
    const trimmed = (appId ?? '').trim();
    if (!trimmed) {
      return null;
    }

    try {
      const { data } = await api.get<ApiResponse<ReportIssueAppStatus>>(
        `/apps/${encodeURIComponent(trimmed)}/diagnostics/status`,
      );
      if (data?.success === false) {
        logger.warn(`Scenario status request for ${trimmed} failed`, data?.error || data?.message);
        return null;
      }
      return data?.data ?? null;
    } catch (error) {
      logger.warn(`Failed to fetch scenario status for ${trimmed}`, error);
      return null;
    }
  },

  async getIframeBridgeDiagnostics(
    appId: string,
  ): Promise<ApiResponse<BridgeDiagnosticsReport>> {
    try {
      const { data } = await api.get<ApiResponse<BridgeDiagnosticsReport>>(
        `/apps/${encodeURIComponent(appId)}/diagnostics/iframe-bridge`,
      );
      return data;
    } catch (error) {
      logger.error(`Failed to fetch iframe bridge diagnostics for ${appId}`, error);
      throw error;
    }
  },

  async getAppProxyMetadata(appId: string): Promise<AppProxyMetadata | null> {
    try {
      const { data } = await axios.get<ApiResponse<AppProxyMetadata>>(
        `/apps/${encodeURIComponent(appId)}/_proxy/metadata`,
        { baseURL: '' },
      );
      if (data?.success === false) {
        logger.warn(`Proxy metadata request for ${appId} returned error: ${data.error}`);
        return null;
      }
      return data?.data ?? null;
    } catch (error) {
      logger.warn(`Failed to fetch proxy metadata for ${appId}`, error);
      return null;
    }
  },

  async getAppLocalhostReport(appId: string): Promise<LocalhostUsageReport | null> {
    try {
      const { data } = await api.get<ApiResponse<LocalhostUsageReport>>(
        `/apps/${encodeURIComponent(appId)}/diagnostics/localhost`,
      );
      if (data?.success === false) {
        logger.warn(`Localhost usage diagnostic for ${appId} returned error: ${data.error}`);
        return null;
      }
      return data?.data ?? null;
    } catch (error) {
      logger.warn(`Failed to fetch localhost usage report for ${appId}`, error);
      return null;
    }
  },

  // Get complete diagnostics for an app
  async getCompleteDiagnostics(appId: string, fast = false): Promise<CompleteDiagnostics | null> {
    try {
      const params = fast ? '?fast=true' : '';
      const { data } = await api.get<ApiResponse<CompleteDiagnostics>>(`/apps/${encodeURIComponent(appId)}/diagnostics${params}`);
      return data.data ?? null;
    } catch (error) {
      logger.error(`Failed to fetch complete diagnostics for ${appId}`, error);
      return null;
    }
  },

  // Get list of documents for an app
  async getAppDocuments(appId: string): Promise<AppDocumentsList | null> {
    try {
      const { data } = await api.get<ApiResponse<AppDocumentsList>>(`/apps/${encodeURIComponent(appId)}/docs`);
      return data.data ?? null;
    } catch (error) {
      logger.warn(`Failed to fetch documents for ${appId}`, error);
      return null;
    }
  },

  // Get a specific document
  async getAppDocument(appId: string, docPath: string, render = true): Promise<AppDocument | null> {
    try {
      const params = render ? '?render=true' : '?render=false';
      const encodedPath = docPath.startsWith('/') ? docPath.substring(1) : docPath;
      const { data } = await api.get<ApiResponse<AppDocument>>(`/apps/${encodeURIComponent(appId)}/docs/${encodedPath}${params}`);
      return data.data ?? null;
    } catch (error) {
      logger.warn(`Failed to fetch document ${docPath} for ${appId}`, error);
      return null;
    }
  },

  // Search documents
  async searchAppDocuments(appId: string, query: string): Promise<AppDocumentMatch[]> {
    try {
      const { data} = await api.get<ApiResponse<AppDocumentMatch[]>>(`/apps/${encodeURIComponent(appId)}/docs-search?q=${encodeURIComponent(query)}`);
      return data.data ?? [];
    } catch (error) {
      logger.warn(`Failed to search documents for ${appId}`, error);
      return [];
    }
  },
};

// Resource Management
export const resourceService = {
  // Get all resources
  async getResources(): Promise<Resource[]> {
    return fetchArrayWithFallback(
      () => api.get<Resource[]>('/resources'),
      'Failed to fetch resources',
    );
  },

  // Get resource status
  async getResourceStatus(id: string): Promise<Resource | null> {
    return fetchWrappedWithFallback(
      () => api.get<ApiResponse<Resource>>(`/resources/${id}/status`),
      `Failed to fetch resource ${id}`,
    );
  },

  async getResourceDetails(id: string): Promise<ResourceDetail | null> {
    return fetchWrappedWithFallback(
      () => api.get<ApiResponse<ResourceDetail>>(`/resources/${encodeURIComponent(id)}`),
      `Failed to fetch resource ${id} details`,
    );
  },

  // Start a resource
  async startResource(id: string): Promise<ApiResponse<Resource>> {
    try {
      const { data } = await api.post<ApiResponse<Resource>>(`/resources/${encodeURIComponent(id)}/start`);
      return data ?? { success: true };
    } catch (error) {
      const message = (error as { message?: string })?.message || `Failed to start resource ${id}`;
      logger.error(`Failed to start resource ${id}`, error);
      return { success: false, error: message };
    }
  },

  // Stop a resource
  async stopResource(id: string): Promise<ApiResponse<Resource>> {
    try {
      const { data } = await api.post<ApiResponse<Resource>>(`/resources/${encodeURIComponent(id)}/stop`);
      return data ?? { success: true };
    } catch (error) {
      const message = (error as { message?: string })?.message || `Failed to stop resource ${id}`;
      logger.error(`Failed to stop resource ${id}`, error);
      return { success: false, error: message };
    }
  },
};

/**
 * Generic error-handling wrapper for API fetches that return arrays
 * Consolidates the try-catch-log-return pattern used throughout the service
 */
async function fetchArrayWithFallback<T>(
  fetcher: () => Promise<{ data: T[] }>,
  errorMsg: string,
): Promise<T[]> {
  try {
    const { data } = await fetcher();
    return data || [];
  } catch (error) {
    logger.error(errorMsg, error);
    return [];
  }
}

/**
 * Generic error-handling wrapper for API fetches that return wrapped responses
 * Consolidates the try-catch-log-return pattern used throughout the service
 */
async function fetchWrappedWithFallback<T>(
  fetcher: () => Promise<{ data: ApiResponse<T> }>,
  errorMsg: string,
): Promise<T | null> {
  try {
    const { data } = await fetcher();
    return data?.data ?? null;
  } catch (error) {
    logger.error(errorMsg, error);
    return null;
  }
}

// Metrics are now handled by system-monitor iframe

// System Logs
export const logService = {
  // Get system logs
  async getSystemLogs(level?: string): Promise<LogEntry[]> {
    return fetchWrappedWithFallback(
      () => {
        const params = level ? `?level=${level}` : '';
        return api.get<ApiResponse<LogEntry[]>>(`/logs${params}`);
      },
      'Failed to fetch system logs',
    ).then(result => result ?? []);
  },

  // Stream logs (returns EventSource)
  streamLogs(onMessage: (log: LogEntry) => void): EventSource {
    const eventSource = new EventSource(buildApiUrlWithBase('/logs/stream'));
    
    eventSource.onmessage = (event) => {
      try {
        const log = JSON.parse(event.data) as LogEntry;
        onMessage(log);
      } catch (error) {
        logger.error('Failed to parse log stream message', error);
      }
    };
    
    eventSource.onerror = (error) => {
      logger.error('Log stream error', error);
    };
    
    return eventSource;
  },
};

// Health Check Service
export const healthService = {
  /**
   * Get lightweight system status for quick polling
   * Optimized to avoid expensive full data fetches
   */
  async getSystemStatus(): Promise<SystemStatusResponse | null> {
    try {
      const { data } = await api.get<SystemStatusResponse>('/system/status');
      return data;
    } catch (error) {
      logger.error('Failed to fetch system status', error);
      return null;
    }
  },

  async checkHealth(): Promise<ApiHealthResponse | null> {
    try {
      const result = await healthService.checkHealthWithMeta();
      return result.data;
    } catch (error) {
      logger.error('Health check failed', error);
      return null;
    }
  },

  async checkHealthWithMeta(): Promise<{ url: string; data: ApiHealthResponse | null }> {
    const url = buildApiUrlWithBase('/health');
    const { data } = await axios.get<ApiHealthResponse>(url, { timeout: 10000 });
    return { url, data: data ?? null };
  },

  async checkUiHealth(): Promise<{ url: string; data: UiHealthResponse | null }> {
    const origin = typeof window !== 'undefined' && window.location ? window.location.origin : '';
    const url = origin ? `${origin.replace(/\/+$/, '')}/health` : '/health';
    const { data } = await axios.get<UiHealthResponse>(url, { timeout: 10000 });
    return { url, data: data ?? null };
  },

  async checkPreviewHealth(appId: string): Promise<PreviewHealthDiagnosticsResult> {
    const trimmed = appId.trim();
    if (!trimmed) {
      throw new Error('App identifier is required to run preview health diagnostics.');
    }

    try {
      const { data } = await api.get<ApiResponse<PreviewHealthDiagnosticsResponse>>(
        `/apps/${encodeURIComponent(trimmed)}/diagnostics/health`,
      );

      if (!data?.success) {
        throw new Error(data?.error || data?.message || 'Failed to gather preview health diagnostics.');
      }

      const payload = data.data;
      const checksRaw = Array.isArray(payload?.checks) ? payload?.checks ?? [] : [];
      const checks = checksRaw.map((entry) => {
        const status: 'pass' | 'warn' | 'fail' = entry.status === 'pass'
          ? 'pass'
          : entry.status === 'warn'
            ? 'warn'
            : 'fail';

        return {
          id: entry.id,
          name: entry.name,
          status,
          endpoint: entry.endpoint ?? null,
          latencyMs: typeof entry.latencyMs === 'number' ? Math.round(entry.latencyMs) : entry.latencyMs ?? null,
          message: entry.message ?? null,
          code: entry.code ?? null,
          response: entry.response ?? null,
        };
      });

      const errors: string[] = [];
      if (Array.isArray(payload?.errors)) {
        payload?.errors.forEach((value) => {
          const normalized = typeof value === 'string' ? value.trim() : '';
          if (normalized) {
            errors.push(normalized);
          }
        });
      }
      if (data?.warning) {
        const warning = data.warning.trim();
        if (warning) {
          errors.push(warning);
        }
      }

      return {
        appId: payload?.app_id ?? trimmed,
        appName: payload?.app_name ?? null,
        scenario: payload?.scenario ?? null,
        capturedAt: payload?.captured_at ?? null,
        checks,
        errors,
        ports: payload?.ports ?? {},
      };
    } catch (error) {
      logger.warn('Failed to run preview health diagnostics', error);
      throw error;
    }
  },
};

// Terminal service removed - functionality not needed

export default api;
