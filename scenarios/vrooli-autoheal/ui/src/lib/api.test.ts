// API client error handling tests
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { APIError, sortChecksForDisplay, type HealthResult } from './api';
import * as api from './api';

const response = {
  ok: true,
  status: 200,
  json: async () => ({}),
  blob: async () => new Blob(["{}"], { type: "application/json" }),
};

describe('[REQ:FAIL-SAFE-001] APIError', () => {
  describe('constructor', () => {
    it('creates error with correct properties', () => {
      const error = new APIError('Test message', 'DATABASE_ERROR', 500, 'req-123');

      expect(error.message).toBe('Test message');
      expect(error.code).toBe('DATABASE_ERROR');
      expect(error.statusCode).toBe(500);
      expect(error.requestId).toBe('req-123');
      expect(error.name).toBe('APIError');
    });
  });

  describe('isRetryable', () => {
    it('marks 5xx errors as retryable', () => {
      expect(new APIError('', 'ERROR', 500).isRetryable).toBe(true);
      expect(new APIError('', 'ERROR', 502).isRetryable).toBe(true);
      expect(new APIError('', 'ERROR', 503).isRetryable).toBe(true);
    });

    it('marks network errors (status 0) as retryable', () => {
      expect(new APIError('', 'NETWORK_ERROR', 0).isRetryable).toBe(true);
    });

    it('marks timeout (408) as retryable', () => {
      expect(new APIError('', 'TIMEOUT', 408).isRetryable).toBe(true);
    });

    it('marks 4xx errors as non-retryable', () => {
      expect(new APIError('', 'ERROR', 400).isRetryable).toBe(false);
      expect(new APIError('', 'ERROR', 404).isRetryable).toBe(false);
      expect(new APIError('', 'ERROR', 422).isRetryable).toBe(false);
    });
  });

  describe('getUserMessage', () => {
    it('returns friendly message for DATABASE_ERROR', () => {
      const error = new APIError('Raw error', 'DATABASE_ERROR', 500);
      expect(error.getUserMessage()).toContain('Database');
    });

    it('returns original message for NOT_FOUND', () => {
      const error = new APIError("Check 'test-123' not found", 'NOT_FOUND', 404);
      expect(error.getUserMessage()).toBe("Check 'test-123' not found");
    });

    it('returns friendly message for TIMEOUT', () => {
      const error = new APIError('Raw error', 'TIMEOUT', 504);
      expect(error.getUserMessage()).toContain('took too long');
    });

    it('returns friendly message for SERVICE_UNAVAILABLE', () => {
      const error = new APIError('Raw error', 'SERVICE_UNAVAILABLE', 503);
      expect(error.getUserMessage()).toContain('unavailable');
    });

    it('returns generic message for unknown codes', () => {
      const error = new APIError('Raw error', 'UNKNOWN_CODE', 500);
      expect(error.getUserMessage()).toBe('Something went wrong. Please try again.');
    });
  });

  describe('getSuggestedAction', () => {
    it('suggests retry for retryable errors', () => {
      const error = new APIError('', 'DATABASE_ERROR', 500);
      expect(error.getSuggestedAction()).toContain('Try again');
    });

    it('suggests checking removal for NOT_FOUND', () => {
      const error = new APIError('', 'NOT_FOUND', 404);
      expect(error.getSuggestedAction()).toContain('may have been removed');
    });

    it('suggests checking scenario for non-retryable errors', () => {
      const error = new APIError('', 'UNKNOWN', 400);
      expect(error.getSuggestedAction()).toContain('scenario');
    });
  });
});

describe('sortChecksForDisplay', () => {
  it('applies deterministic ordering for dashboard rendering', () => {
    const input: Array<HealthResult & { title?: string; category?: string }> = [
      {
        checkId: 'scenario-zeta',
        status: 'warning',
        message: 'z',
        timestamp: '2026-02-19T00:00:00Z',
        duration: 1,
        title: 'Zeta Scenario',
        category: 'scenario',
      },
      {
        checkId: 'infra-beta',
        status: 'warning',
        message: 'b',
        timestamp: '2026-02-19T00:00:00Z',
        duration: 1,
        title: 'Beta Infra',
        category: 'infrastructure',
      },
      {
        checkId: 'resource-alpha',
        status: 'warning',
        message: 'a',
        timestamp: '2026-02-19T00:00:00Z',
        duration: 1,
        title: 'Alpha Resource',
        category: 'resource',
      },
      {
        checkId: 'critical-a',
        status: 'critical',
        message: 'crit',
        timestamp: '2026-02-19T00:00:00Z',
        duration: 1,
        category: 'infrastructure',
      },
      {
        checkId: 'ok-a',
        status: 'ok',
        message: 'ok',
        timestamp: '2026-02-19T00:00:00Z',
        duration: 1,
        category: 'infrastructure',
      },
    ];

    const sorted = sortChecksForDisplay(input).map((c) => c.checkId);
    expect(sorted).toEqual([
      'critical-a',
      'infra-beta',
      'resource-alpha',
      'scenario-zeta',
      'ok-a',
    ]);
  });
});

describe("API client endpoint wrappers", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(response));
  });

  it("constructs every read and mutation request", async () => {
    await api.fetchHealth();
    await api.fetchStatus();
    await api.fetchPlatform();
    await api.fetchChecks();
    await api.runTick();
    await api.runTick(true);
    await api.fetchCheckHistory("check/one");
    await api.fetchTimeline();
    await api.fetchSystemEvents();
    await api.fetchSystemEvents({ limit: 3, correlate: true });
    await api.refreshSystemEvents();
    await api.fetchUptimeStats();
    await api.fetchUptimeHistory(2, 3);
    await api.fetchCheckTrends(2);
    await api.fetchTransitions(2, 3);
    await api.fetchIncidents({ status: "open", severity: "critical", type: "manual", limit: 2 });
    await api.fetchIncident("inc/one");
    await api.fetchIncidentObservations("inc/one");
    await api.updateIncidentStatus("inc/one", "resolve", "done");
    await api.fetchWatchdogStatus(true);
    await api.fetchWatchdogTemplate();
    await api.installWatchdog({ useSystemService: true, enableLingering: true });
    await api.uninstallWatchdog();
    await api.enableLingering();
    await api.fetchInstallStatus();
    await api.fetchCheckActions("resource-postgres");
    await api.executeAction("resource-postgres", "restart");
    await api.fetchActionHistory();
    await api.fetchActionHistory("resource-postgres");
    await api.fetchConfig();
    await api.updateConfig({} as Parameters<typeof api.updateConfig>[0]);
    await api.validateConfig({} as Parameters<typeof api.validateConfig>[0]);
    await api.fetchConfigSchema();
    await api.exportConfig();
    await api.importConfig("{}");
    await api.fetchDefaults();
    await api.fetchGlobalConfig();
    await api.fetchUIConfig();
    await api.fetchCheckConfig("infra-dns");
    await api.setCheckEnabled("infra-dns", true);
    await api.setCheckAutoHeal("infra-dns", false);
    await api.bulkUpdateChecks("enableAll");
    await api.fetchMonitoring();
    await api.updateMonitoring({ scenarios: {}, resources: [] });
    await api.addScenario("demo", true);
    await api.removeScenario("demo");
    await api.setScenarioCritical("demo", false);
    await api.addResource("redis");
    await api.removeResource("redis");

    expect(fetch).toHaveBeenCalled();
  });

  it("classifies status values and check groups", () => {
    expect(api.isHealthStatus("ok")).toBe(true);
    expect(api.isHealthStatus("unexpected")).toBe(false);
    expect(api.normalizeHealthStatus("warning")).toBe("warning");
    expect(api.normalizeHealthStatus("unexpected", "critical")).toBe("critical");
    const checks = [
      { checkId: "a", status: "ok", message: "", timestamp: "", duration: 0 },
      { checkId: "b", status: "warning", message: "", timestamp: "", duration: 0 },
      { checkId: "c", status: "critical", message: "", timestamp: "", duration: 0 },
    ] as HealthResult[];
    expect(Object.keys(api.groupChecksByStatus(checks))).toHaveLength(4);
    expect(api.sortChecksBySeverity(checks)[0]?.status).toBe("critical");
    expect(api.overallStatusFromSummary({ total: 1, ok: 0, warning: 1, critical: 0 })).toBe("warning");
    expect(api.overallStatusFromSummary({ total: 1, ok: 1, warning: 0, critical: 0 })).toBe("ok");
    expect(api.statusToEmoji("ok")).toBe("✓");
    expect(api.statusToEmoji("critical")).toBe("✗");
  });
});
