import type {
  ActionResult,
  CheckActionsResponse,
  CheckHistoryResponse,
  CheckInfo,
  CheckTrend,
  CheckTrendsResponse,
  Config,
  DefaultsResponse,
  HealthResult,
  HistoryEntry,
  StatusResponse,
  TimelineEvent,
  TimelineResponse,
  Transition,
  TransitionsResponse,
  UptimeHistoryResponse,
  UptimeStatsResponse,
  WatchdogStatus,
} from "../../lib/api";

const DEFAULT_TIMESTAMP = "2024-01-01T12:00:00Z";

export function createHistoryEntry(overrides: Partial<HistoryEntry> = {}): HistoryEntry {
  return {
    checkId: "test-check",
    status: "ok",
    message: "OK",
    timestamp: DEFAULT_TIMESTAMP,
    duration: 100,
    ...overrides,
  };
}

export function createCheckHistoryResponse(
  overrides: Partial<CheckHistoryResponse> = {}
): CheckHistoryResponse {
  const history = overrides.history ?? [];
  return {
    checkId: "test-check",
    history,
    count: history.length,
    ...overrides,
  };
}

export function createTimelineEvent(overrides: Partial<TimelineEvent> = {}): TimelineEvent {
  return {
    checkId: "test-check",
    status: "ok",
    message: "Event message",
    timestamp: DEFAULT_TIMESTAMP,
    ...overrides,
  };
}

export function createTimelineResponse(
  overrides: Partial<TimelineResponse> = {}
): TimelineResponse {
  const events = overrides.events ?? [];
  return {
    events,
    count: events.length,
    summary: { ok: 0, warning: 0, critical: 0 },
    ...overrides,
  };
}

export function createUptimeStatsResponse(
  overrides: Partial<UptimeStatsResponse> = {}
): UptimeStatsResponse {
  return {
    totalEvents: 100,
    okEvents: 95,
    warningEvents: 3,
    criticalEvents: 2,
    uptimePercentage: 95,
    windowHours: 24,
    ...overrides,
  };
}

export function createUptimeHistoryResponse(
  overrides: Partial<UptimeHistoryResponse> = {}
): UptimeHistoryResponse {
  return {
    buckets: [],
    overall: { uptimePercentage: 95, totalEvents: 100 },
    windowHours: 24,
    bucketCount: 24,
    ...overrides,
  };
}

export function createCheckTrend(overrides: Partial<CheckTrend> = {}): CheckTrend {
  return {
    checkId: "test-check",
    total: 100,
    ok: 95,
    warning: 3,
    critical: 2,
    uptimePercent: 95,
    currentStatus: "ok",
    recentStatuses: ["ok"],
    lastChecked: DEFAULT_TIMESTAMP,
    ...overrides,
  };
}

export function createCheckTrendsResponse(
  overrides: Partial<CheckTrendsResponse> = {}
): CheckTrendsResponse {
  const trends = overrides.trends ?? [];
  return {
    trends,
    windowHours: 24,
    totalChecks: trends.length,
    ...overrides,
  };
}

export function createIncident(overrides: Partial<Transition> = {}): Transition {
  return {
    timestamp: DEFAULT_TIMESTAMP,
    checkId: "test-check",
    fromStatus: "ok",
    toStatus: "warning",
    message: "Incident message",
    ...overrides,
  };
}

export function createIncidentsResponse(
  overrides: Partial<TransitionsResponse> = {}
): TransitionsResponse {
  const transitions = overrides.transitions ?? [];
  return {
    transitions,
    windowHours: 24,
    total: transitions.length,
    ...overrides,
  };
}

export function createWatchdogStatus(
  overrides: Partial<WatchdogStatus> = {}
): WatchdogStatus {
  return {
    loopRunning: true,
    watchdogType: "systemd",
    watchdogInstalled: true,
    watchdogEnabled: true,
    watchdogRunning: true,
    bootProtectionActive: true,
    canInstall: true,
    servicePath: "/etc/systemd/system/vrooli-autoheal.service",
    protectionLevel: "full",
    lingeringEnabled: false,
    ...overrides,
  };
}

export function createCheckInfo(overrides: Partial<CheckInfo> = {}): CheckInfo {
  return {
    id: "infra-network",
    title: "Internet Connection",
    description: "Network connectivity check",
    importance: "Required for external API calls",
    category: "infrastructure",
    intervalSeconds: 30,
    ...overrides,
  };
}

export function createHealthResult(overrides: Partial<HealthResult> = {}): HealthResult {
  return {
    checkId: "test-check",
    status: "ok",
    message: "Check passed",
    timestamp: DEFAULT_TIMESTAMP,
    duration: 10,
    ...overrides,
  };
}

export function createStatusResponse(overrides: Partial<StatusResponse> = {}): StatusResponse {
  return {
    status: "ok",
    platform: {
      platform: "linux",
      supportsRdp: false,
      supportsSystemd: true,
      supportsLaunchd: false,
      supportsWindowsServices: false,
      isHeadlessServer: false,
      hasDocker: true,
      isWsl: false,
      supportsCloudflared: true,
    },
    summary: {
      total: 1,
      ok: 1,
      warning: 0,
      critical: 0,
    },
    checks: [createHealthResult()],
    timestamp: DEFAULT_TIMESTAMP,
    ...overrides,
  };
}

export function createConfig(overrides: Partial<Config> = {}): Config {
  return {
    version: "1.0",
    global: {
      gracePeriodSeconds: 10,
      tickIntervalSeconds: 30,
      verifyDelaySeconds: 20,
      maxRestartAttempts: 3,
      restartCooldownSeconds: 60,
      historyRetentionHours: 24,
    },
    checks: {},
    ui: {
      autoRefreshSeconds: 30,
      theme: "dark",
      showDisabledChecks: false,
      defaultTab: "dashboard",
    },
    ...overrides,
  };
}

export function createDefaultsResponse(
  overrides: Partial<DefaultsResponse> = {}
): DefaultsResponse {
  return {
    global: {
      gracePeriodSeconds: 10,
      tickIntervalSeconds: 30,
      verifyDelaySeconds: 20,
      maxRestartAttempts: 3,
      restartCooldownSeconds: 60,
      historyRetentionHours: 24,
    },
    checks: {},
    ui: {
      autoRefreshSeconds: 30,
      theme: "dark",
      showDisabledChecks: false,
      defaultTab: "dashboard",
    },
    ...overrides,
  };
}

export function createCheckActionsResponse(
  overrides: Partial<CheckActionsResponse> = {}
): CheckActionsResponse {
  return {
    checkId: "test-check",
    actions: [],
    ...overrides,
  };
}

export function createActionResult(overrides: Partial<ActionResult> = {}): ActionResult {
  return {
    actionId: "heal",
    checkId: "test-check",
    success: true,
    message: "Recovery executed",
    timestamp: DEFAULT_TIMESTAMP,
    duration: 100,
    ...overrides,
  };
}
