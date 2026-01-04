/**
 * Unit Tests for Observability Collector
 *
 * Tests the collector that aggregates data from all subsystems.
 */

import { describe, beforeEach, it, expect, jest } from '@jest/globals';
import {
  ObservabilityCollector,
  createObservabilityCollector,
} from '../../../src/observability/collector';
import type {
  ObservabilityDependencies,
  SessionSummary,
  BrowserStatusSummary,
  CleanupStatus,
} from '../../../src/observability/types';

// Mock the playwright provider
jest.mock('../../../src/playwright', () => ({
  playwrightProvider: {
    name: 'rebrowser-playwright',
    capabilities: {
      evaluateIsolated: true,
      exposeBindingIsolated: true,
      hasAntiDetection: true,
    },
  },
}));

// Mock logger
jest.mock('../../../src/utils', () => ({
  logger: {
    info: jest.fn(),
    warn: jest.fn(),
    error: jest.fn(),
    debug: jest.fn(),
  },
  scopedLog: jest.fn((context: string, event: string) => `${context}: ${event}`),
  LogContext: {
    HEALTH: 'health',
  },
}));

// Helper to create mock dependencies
function createMockDeps(overrides: Partial<{
  sessionSummary: Partial<SessionSummary>;
  browserStatus: Partial<BrowserStatusSummary>;
  cleanupStatus: Partial<CleanupStatus>;
}> = {}): ObservabilityDependencies {
  return {
    getSessionSummary: jest.fn().mockReturnValue({
      total: 5,
      active: 3,
      idle: 2,
      active_recordings: 1,
      idle_timeout_ms: 300000,
      capacity: 10,
      ...overrides.sessionSummary,
    }),
    getBrowserStatus: jest.fn().mockReturnValue({
      healthy: true,
      version: '120.0.6099.109',
      ...overrides.browserStatus,
    }),
    getCleanupStatus: jest.fn().mockReturnValue({
      is_running: false,
      interval_ms: 60000,
      ...overrides.cleanupStatus,
    }),
    getMetricsConfig: jest.fn().mockReturnValue({
      enabled: true,
      port: 9090,
    }),
  };
}

describe('ObservabilityCollector', () => {
  describe('collect - quick depth', () => {
    it('should return status ok when browser is healthy', async () => {
      const deps = createMockDeps();
      const collector = new ObservabilityCollector(deps);

      const result = await collector.collect('quick');

      expect(result.status).toBe('ok');
      expect(result.ready).toBe(true);
      expect(result.depth).toBe('quick');
      expect(result.summary.sessions).toBe(5);
      expect(result.summary.recordings).toBe(1);
      expect(result.summary.browser_connected).toBe(true);
    });

    it('should return status degraded when browser is not verified yet', async () => {
      const deps = createMockDeps({
        browserStatus: {
          healthy: false,
          error: 'Browser not yet verified',
        },
      });
      const collector = new ObservabilityCollector(deps);

      const result = await collector.collect('quick');

      expect(result.status).toBe('degraded');
      expect(result.ready).toBe(false);
    });

    it('should return status error when browser has critical error', async () => {
      const deps = createMockDeps({
        browserStatus: {
          healthy: false,
          error: 'Chromium not found',
        },
      });
      const collector = new ObservabilityCollector(deps);

      const result = await collector.collect('quick');

      expect(result.status).toBe('error');
      expect(result.ready).toBe(false);
    });

    it('should include uptime and version', async () => {
      const deps = createMockDeps();
      const collector = new ObservabilityCollector(deps);

      const result = await collector.collect('quick');

      expect(result.version).toBeDefined();
      expect(result.uptime_ms).toBeGreaterThanOrEqual(0);
      expect(result.timestamp).toBeDefined();
    });
  });

  describe('collect - standard depth', () => {
    it('should include components for standard depth', async () => {
      const deps = createMockDeps();
      const collector = new ObservabilityCollector(deps);

      const result = await collector.collect('standard');

      expect(result.depth).toBe('standard');
      expect(result.components).toBeDefined();
      expect(result.components?.browser).toBeDefined();
      expect(result.components?.sessions).toBeDefined();
      expect(result.components?.recording).toBeDefined();
      expect(result.components?.cleanup).toBeDefined();
      expect(result.components?.metrics).toBeDefined();
    });

    it('should include browser component details', async () => {
      const deps = createMockDeps();
      const collector = new ObservabilityCollector(deps);

      const result = await collector.collect('standard');

      const browser = result.components?.browser;
      expect(browser?.status).toBe('healthy');
      expect(browser?.version).toBe('120.0.6099.109');
      expect(browser?.connected).toBe(true);
      expect(browser?.provider).toBe('rebrowser-playwright');
      expect(browser?.capabilities?.evaluate_isolated).toBe(true);
    });

    it('should include sessions component details', async () => {
      const deps = createMockDeps();
      const collector = new ObservabilityCollector(deps);

      const result = await collector.collect('standard');

      const sessions = result.components?.sessions;
      expect(sessions?.status).toBe('healthy');
      expect(sessions?.active).toBe(3);
      expect(sessions?.idle).toBe(2);
      expect(sessions?.total).toBe(5);
      expect(sessions?.capacity).toBe(10);
      expect(sessions?.active_recordings).toBe(1);
    });

    it('should show degraded status when sessions near capacity', async () => {
      const deps = createMockDeps({
        sessionSummary: {
          total: 9,
          active: 8,
          idle: 1,
          capacity: 10,
        },
      });
      const collector = new ObservabilityCollector(deps);

      const result = await collector.collect('standard');

      expect(result.components?.sessions.status).toBe('degraded');
      expect(result.components?.sessions.message).toContain('High utilization');
    });

    it('should show error status when at capacity', async () => {
      const deps = createMockDeps({
        sessionSummary: {
          total: 10,
          active: 10,
          idle: 0,
          capacity: 10,
        },
      });
      const collector = new ObservabilityCollector(deps);

      const result = await collector.collect('standard');

      expect(result.components?.sessions.status).toBe('error');
      expect(result.components?.sessions.hint).toContain('Close unused sessions');
    });

    it('should include cleanup component details', async () => {
      const deps = createMockDeps({
        cleanupStatus: {
          is_running: true,
          interval_ms: 60000,
        },
      });
      const collector = new ObservabilityCollector(deps);

      const result = await collector.collect('standard');

      const cleanup = result.components?.cleanup;
      expect(cleanup?.is_running).toBe(true);
      expect(cleanup?.interval_ms).toBe(60000);
    });

    it('should include metrics component details', async () => {
      const deps = createMockDeps();
      const collector = new ObservabilityCollector(deps);

      const result = await collector.collect('standard');

      const metricsComp = result.components?.metrics;
      expect(metricsComp?.enabled).toBe(true);
      expect(metricsComp?.port).toBe(9090);
      expect(metricsComp?.endpoint).toBe('/metrics');
    });

    it('should include config summary', async () => {
      const deps = createMockDeps();
      const collector = new ObservabilityCollector(deps);

      const result = await collector.collect('standard');

      expect(result.config).toBeDefined();
      expect(result.config?.summary).toBeDefined();
    });
  });

  describe('collect - deep depth', () => {
    it('should include diagnostics for deep depth', async () => {
      const deps = createMockDeps();
      const collector = new ObservabilityCollector(deps);

      const result = await collector.collect('deep');

      expect(result.depth).toBe('deep');
      expect(result.components).toBeDefined();
      expect(result.diagnostics).toBeDefined();
    });

    it('should run recording diagnostics when available', async () => {
      const mockDiagnostics = {
        ready: true,
        timestamp: new Date().toISOString(),
        durationMs: 50,
        level: 'full' as const,
        issues: [],
        provider: {
          name: 'rebrowser-playwright',
          evaluateIsolated: true,
          exposeBindingIsolated: true,
        },
      };

      const deps = createMockDeps();
      deps.runRecordingDiagnostics = jest.fn().mockResolvedValue(mockDiagnostics);

      const collector = new ObservabilityCollector(deps);
      const result = await collector.collect('deep');

      expect(deps.runRecordingDiagnostics).toHaveBeenCalled();
      expect(result.diagnostics?.recording).toEqual(mockDiagnostics);
    });
  });

  describe('browser error hints', () => {
    it('should provide hint for Chromium not found', async () => {
      const deps = createMockDeps({
        browserStatus: {
          healthy: false,
          error: 'Chromium executable not found',
        },
      });
      const collector = new ObservabilityCollector(deps);

      const result = await collector.collect('standard');

      expect(result.components?.browser.hint).toContain('npx playwright install chromium');
    });

    it('should provide hint for sandbox issues', async () => {
      const deps = createMockDeps({
        browserStatus: {
          healthy: false,
          error: 'Failed to launch browser: sandbox error',
        },
      });
      const collector = new ObservabilityCollector(deps);

      const result = await collector.collect('standard');

      expect(result.components?.browser.hint).toContain('sandbox');
    });

    it('should provide hint for memory issues', async () => {
      const deps = createMockDeps({
        browserStatus: {
          healthy: false,
          error: 'Out of memory',
        },
      });
      const collector = new ObservabilityCollector(deps);

      const result = await collector.collect('standard');

      expect(result.components?.browser.hint).toContain('memory');
    });
  });
});

describe('createObservabilityCollector', () => {
  it('should create a collector instance', () => {
    const deps = createMockDeps();
    const collector = createObservabilityCollector(deps);

    expect(collector).toBeInstanceOf(ObservabilityCollector);
  });
});
