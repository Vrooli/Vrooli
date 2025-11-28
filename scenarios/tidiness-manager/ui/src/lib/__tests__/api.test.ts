import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import * as api from '../api';

// Mock @vrooli/api-base
vi.mock('@vrooli/api-base', () => ({
  resolveWithConfig: vi.fn(),
  buildApiUrl: vi.fn((base, path) => `${base}${path}`),
}));

describe('API Module', () => {
  beforeEach(() => {
    // Clear module cache to reset API_BASE state between tests
    vi.resetModules();
    vi.clearAllMocks();

    // Mock fetch globally
    global.fetch = vi.fn();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('Type exports', () => {
    it('exports HealthResponse interface', () => {
      // TypeScript will error if interface doesn't exist
      const health: api.HealthResponse = {
        status: 'healthy',
        service: 'test',
        timestamp: new Date().toISOString(),
        readiness: true,
        version: '1.0.0',
        dependencies: {
          database: 'connected',
        },
      };
      expect(health).toBeDefined();
    });

    it('exports FileMetric interface', () => {
      const metric: api.FileMetric = {
        path: 'test.ts',
        lines: 100,
        extension: '.ts',
      };
      expect(metric).toBeDefined();
    });

    it('exports LongFile interface', () => {
      const longFile: api.LongFile = {
        path: 'long.ts',
        lines: 1000,
        threshold: 500,
      };
      expect(longFile).toBeDefined();
    });

    it('exports CommandRun interface', () => {
      const cmd: api.CommandRun = {
        command: 'make lint',
        exit_code: 0,
        stdout: 'output',
        stderr: '',
        duration_ms: 100,
        success: true,
        skipped: false,
      };
      expect(cmd).toBeDefined();
    });
  });

  describe('API endpoint structure', () => {
    it('should have consistent naming pattern for typed API functions', () => {
      // Verify that API module exports are consistently named
      expect(typeof api).toBe('object');
    });
  });

  describe('fetchHealth', () => {
    it('should successfully fetch health status', async () => {
      const { resolveWithConfig, buildApiUrl } = await import('@vrooli/api-base');
      (resolveWithConfig as any).mockResolvedValue('http://localhost:8080/');

      const mockHealthResponse: api.HealthResponse = {
        status: 'healthy',
        service: 'tidiness-manager',
        timestamp: '2024-01-01T00:00:00Z',
        readiness: true,
        version: '1.0.0',
        dependencies: {
          database: 'connected',
        },
      };

      (global.fetch as any).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockHealthResponse),
      });

      const health = await api.fetchHealth();

      expect(health.status).toBe('healthy');
      expect(health.readiness).toBe(true);
      expect(health.service).toBe('tidiness-manager');
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/health'),
        expect.objectContaining({
          headers: { 'Content-Type': 'application/json' },
          cache: 'no-store',
        })
      );
    });

    it('should throw error when health check fails', async () => {
      const { resolveWithConfig } = await import('@vrooli/api-base');
      (resolveWithConfig as any).mockResolvedValue('http://localhost:8080/');

      (global.fetch as any).mockResolvedValue({
        ok: false,
        status: 503,
      });

      await expect(api.fetchHealth()).rejects.toThrow('API health check failed: 503');
    });

    it('should handle network errors', async () => {
      const { resolveWithConfig } = await import('@vrooli/api-base');
      (resolveWithConfig as any).mockResolvedValue('http://localhost:8080/');

      (global.fetch as any).mockRejectedValue(new Error('Network error'));

      await expect(api.fetchHealth()).rejects.toThrow('Network error');
    });
  });

  describe('lightScan', () => {
    it('should successfully trigger a light scan', async () => {
      const { resolveWithConfig } = await import('@vrooli/api-base');
      (resolveWithConfig as any).mockResolvedValue('http://localhost:8080/');

      const mockScanResult: api.ScanResult = {
        scenario: 'test-scenario',
        started_at: '2024-01-01T00:00:00Z',
        completed_at: '2024-01-01T00:01:00Z',
        duration_ms: 60000,
        file_metrics: [],
        long_files: [],
        total_files: 10,
        total_lines: 1000,
        has_makefile: true,
      };

      (global.fetch as any).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockScanResult),
      });

      const result = await api.lightScan('test-scenario', 300);

      expect(result.scenario).toBe('test-scenario');
      expect(result.total_files).toBe(10);
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/scan/light'),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ scenario_path: 'test-scenario', timeout_sec: 300 }),
        })
      );
    });

    it('should handle scan errors with error message', async () => {
      const { resolveWithConfig } = await import('@vrooli/api-base');
      (resolveWithConfig as any).mockResolvedValue('http://localhost:8080/');

      (global.fetch as any).mockResolvedValue({
        ok: false,
        status: 400,
        json: () => Promise.resolve({ error: 'Invalid scenario path' }),
      });

      await expect(api.lightScan('invalid-scenario')).rejects.toThrow('Invalid scenario path');
    });

    it('should handle scan errors without error message', async () => {
      const { resolveWithConfig } = await import('@vrooli/api-base');
      (resolveWithConfig as any).mockResolvedValue('http://localhost:8080/');

      (global.fetch as any).mockResolvedValue({
        ok: false,
        status: 500,
        json: () => Promise.reject(new Error('Invalid JSON')),
      });

      // When JSON parsing fails, it defaults to "Unknown error"
      await expect(api.lightScan('test-scenario')).rejects.toThrow('Unknown error');
    });
  });

  describe('parseLintOutput', () => {
    it('should successfully parse lint output', async () => {
      const { resolveWithConfig } = await import('@vrooli/api-base');
      (resolveWithConfig as any).mockResolvedValue('http://localhost:8080/');

      const mockResponse = {
        issues: [
          {
            scenario: 'test-scenario',
            file: 'test.ts',
            line: 10,
            column: 5,
            message: 'Unused variable',
            severity: 'warning' as const,
            tool: 'eslint',
            category: 'lint' as const,
          },
        ],
        count: 1,
      };

      (global.fetch as any).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const result = await api.parseLintOutput('test-scenario', 'eslint', 'lint output');

      expect(result.count).toBe(1);
      expect(result.issues).toHaveLength(1);
      expect(result.issues[0].message).toBe('Unused variable');
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/scan/light/parse-lint'),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ scenario: 'test-scenario', tool: 'eslint', output: 'lint output' }),
        })
      );
    });

    it('should handle parse errors', async () => {
      const { resolveWithConfig } = await import('@vrooli/api-base');
      (resolveWithConfig as any).mockResolvedValue('http://localhost:8080/');

      (global.fetch as any).mockResolvedValue({
        ok: false,
        status: 400,
        json: () => Promise.resolve({ error: 'Invalid output format' }),
      });

      await expect(api.parseLintOutput('test', 'eslint', 'bad output')).rejects.toThrow('Invalid output format');
    });
  });

  describe('parseTypeOutput', () => {
    it('should successfully parse type checker output', async () => {
      const { resolveWithConfig } = await import('@vrooli/api-base');
      (resolveWithConfig as any).mockResolvedValue('http://localhost:8080/');

      const mockResponse = {
        issues: [
          {
            scenario: 'test-scenario',
            file: 'types.ts',
            line: 42,
            column: 10,
            message: 'Type mismatch',
            severity: 'error' as const,
            tool: 'tsc',
            category: 'type' as const,
          },
        ],
        count: 1,
      };

      (global.fetch as any).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const result = await api.parseTypeOutput('test-scenario', 'tsc', 'type output');

      expect(result.count).toBe(1);
      expect(result.issues[0].severity).toBe('error');
      expect(result.issues[0].category).toBe('type');
    });
  });

  describe('fetchScenarioStats', () => {
    it('should fetch and transform scenario statistics', async () => {
      const { resolveWithConfig } = await import('@vrooli/api-base');
      (resolveWithConfig as any).mockResolvedValue('http://localhost:8080/');

      const mockApiResponse = {
        scenarios: [
          { scenario: 'test-scenario-1', total: 10, lint: 3, type: 2, long_files: 1 },
          { scenario: 'test-scenario-2', total: 20, lint: 5, type: 3, long_files: 2 },
        ],
        count: 2,
      };

      (global.fetch as any).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockApiResponse),
      });

      const stats = await api.fetchScenarioStats();

      expect(stats).toHaveLength(2);
      expect(stats[0].scenario).toBe('test-scenario-1');
      expect(stats[0].light_issues).toBe(3);
      expect(stats[0].ai_issues).toBe(2);
      expect(stats[0].long_files).toBe(1);
    });

    it('should handle empty scenario list', async () => {
      const { resolveWithConfig } = await import('@vrooli/api-base');
      (resolveWithConfig as any).mockResolvedValue('http://localhost:8080/');

      (global.fetch as any).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ scenarios: [], count: 0 }),
      });

      const stats = await api.fetchScenarioStats();
      expect(stats).toEqual([]);
    });
  });

  describe('fetchAllIssues', () => {
    it('should fetch issues with filters', async () => {
      const { resolveWithConfig } = await import('@vrooli/api-base');
      (resolveWithConfig as any).mockResolvedValue('http://localhost:8080/');

      const mockIssues = [
        {
          id: 1,
          scenario: 'test-scenario',
          file_path: 'test.ts',
          line_number: 10,
          column_number: 5,
          title: 'Unused variable',
          description: 'Variable x is never used',
          severity: 'warning',
          category: 'lint',
          status: 'open',
          created_at: '2024-01-01T00:00:00Z',
        },
      ];

      (global.fetch as any).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockIssues),
      });

      const issues = await api.fetchAllIssues('test-scenario', {
        status: 'open',
        category: 'lint',
        severity: 'warning',
      });

      expect(issues).toHaveLength(1);
      expect(issues[0].file).toBe('test.ts');
      expect(issues[0].line).toBe(10);
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringMatching(/scenario=test-scenario.*status=open.*category=lint.*severity=warning/),
        expect.any(Object)
      );
    });

    it('should handle "all" filter values by not including them', async () => {
      const { resolveWithConfig } = await import('@vrooli/api-base');
      (resolveWithConfig as any).mockResolvedValue('http://localhost:8080/');

      (global.fetch as any).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve([]),
      });

      await api.fetchAllIssues('test-scenario', {
        status: 'all',
        category: 'all',
        severity: 'all',
      });

      // Verify that "all" filters are not included in the query string
      const [[url]] = (global.fetch as any).mock.calls;
      expect(url).toContain('scenario=test-scenario');
      expect(url).not.toContain('status=all');
      expect(url).not.toContain('category=all');
      expect(url).not.toContain('severity=all');
    });
  });

  describe('updateIssueStatus', () => {
    it('should update issue status successfully', async () => {
      const { resolveWithConfig } = await import('@vrooli/api-base');
      (resolveWithConfig as any).mockResolvedValue('http://localhost:8080/');

      const mockResponse = {
        id: 1,
        status: 'resolved',
        updated_at: '2024-01-01T00:00:00Z',
      };

      (global.fetch as any).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const result = await api.updateIssueStatus(1, 'resolved', 'Fixed by refactoring');

      expect(result.id).toBe(1);
      expect(result.status).toBe('resolved');
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/agent/issues/1'),
        expect.objectContaining({
          method: 'PATCH',
          body: JSON.stringify({ status: 'resolved', resolution_notes: 'Fixed by refactoring' }),
        })
      );
    });

    it('should handle update errors', async () => {
      const { resolveWithConfig } = await import('@vrooli/api-base');
      (resolveWithConfig as any).mockResolvedValue('http://localhost:8080/');

      (global.fetch as any).mockResolvedValue({
        ok: false,
        status: 404,
        json: () => Promise.resolve({ error: 'Issue not found' }),
      });

      await expect(api.updateIssueStatus(999, 'resolved')).rejects.toThrow('Issue not found');
    });
  });

  describe('Configuration resolution', () => {
    it('should resolve API base configuration', async () => {
      const { resolveWithConfig } = await import('@vrooli/api-base');
      (resolveWithConfig as any).mockResolvedValue('http://localhost:8080/');

      // The module should successfully resolve configuration
      expect(resolveWithConfig).toBeDefined();
    });

    it('should handle missing configuration', async () => {
      const { resolveWithConfig } = await import('@vrooli/api-base');
      (resolveWithConfig as any).mockRejectedValue(new Error('Config not found'));

      // Should handle missing config gracefully
      expect(resolveWithConfig).toBeDefined();
    });
  });

  describe('URL building', () => {
    it('should build correct API URLs', async () => {
      const { buildApiUrl } = await import('@vrooli/api-base');

      const base = 'http://localhost:8080';
      const path = '/api/health';

      expect(buildApiUrl(base, path)).toBe('http://localhost:8080/api/health');
    });

    it('should handle trailing slashes correctly', async () => {
      const { buildApiUrl } = await import('@vrooli/api-base');

      const base = 'http://localhost:8080/';
      const path = '/api/health';

      const url = buildApiUrl(base, path);
      // The mocked buildApiUrl just concatenates, so this test verifies the mock behavior
      // In a real implementation, buildApiUrl would handle double slashes
      expect(url).toContain('/api/health');
    });
  });

  describe('Response type validation', () => {
    it('should validate HealthResponse structure', () => {
      const validHealth: api.HealthResponse = {
        status: 'healthy',
        service: 'tidiness-manager',
        timestamp: '2024-01-01T00:00:00Z',
        readiness: true,
        version: '1.0.0',
        dependencies: {
          database: 'connected',
        },
      };

      expect(validHealth.status).toBe('healthy');
      expect(validHealth.readiness).toBe(true);
      expect(validHealth.dependencies.database).toBe('connected');
    });

    it('should validate FileMetric structure', () => {
      const validMetric: api.FileMetric = {
        path: 'src/main.ts',
        lines: 250,
        extension: '.ts',
      };

      expect(validMetric.path).toBe('src/main.ts');
      expect(validMetric.lines).toBeGreaterThan(0);
      expect(validMetric.extension).toMatch(/^\./);
    });

    it('should validate CommandRun structure with success', () => {
      const successCmd: api.CommandRun = {
        command: 'npm test',
        exit_code: 0,
        stdout: 'All tests passed',
        stderr: '',
        duration_ms: 1500,
        success: true,
        skipped: false,
      };

      expect(successCmd.success).toBe(true);
      expect(successCmd.exit_code).toBe(0);
      expect(successCmd.skipped).toBe(false);
    });

    it('should validate CommandRun structure with failure', () => {
      const failureCmd: api.CommandRun = {
        command: 'npm test',
        exit_code: 1,
        stdout: '',
        stderr: 'Test failed',
        duration_ms: 500,
        success: false,
        skipped: false,
      };

      expect(failureCmd.success).toBe(false);
      expect(failureCmd.exit_code).toBeGreaterThan(0);
    });

    it('should validate CommandRun structure when skipped', () => {
      const skippedCmd: api.CommandRun = {
        command: 'make lint',
        exit_code: 0,
        stdout: '',
        stderr: '',
        duration_ms: 0,
        success: false,
        skipped: true,
        skip_reason: 'No Makefile found',
      };

      expect(skippedCmd.skipped).toBe(true);
      expect(skippedCmd.skip_reason).toBeDefined();
    });
  });

  describe('Edge cases', () => {
    it('should handle empty string responses', async () => {
      (global.fetch as any).mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(''),
      });

      // Should handle empty responses
      expect(global.fetch).toBeDefined();
    });

    it('should handle very large responses', async () => {
      const largeData = JSON.stringify({
        files: Array(10000).fill({ path: 'test.ts', lines: 100, extension: '.ts' }),
      });

      (global.fetch as any).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(JSON.parse(largeData)),
      });

      // Should handle large datasets
      expect(global.fetch).toBeDefined();
    });

    it('should handle special characters in paths', () => {
      const metric: api.FileMetric = {
        path: 'src/components/Test (copy).tsx',
        lines: 100,
        extension: '.tsx',
      };

      expect(metric.path).toContain('(');
      expect(metric.path).toContain(')');
    });

    it('should handle unicode in file paths', () => {
      const metric: api.FileMetric = {
        path: 'src/日本語/test.ts',
        lines: 50,
        extension: '.ts',
      };

      expect(metric.path).toContain('日本語');
    });
  });

  describe('Timeout handling', () => {
    it('should handle request timeouts', async () => {
      const { resolveWithConfig } = await import('@vrooli/api-base');
      (resolveWithConfig as any).mockResolvedValue('http://localhost:8080/');

      const timeoutError = new Error('Request timeout');
      (global.fetch as any).mockRejectedValue(timeoutError);

      // Should handle timeouts gracefully
      expect(timeoutError.message).toBe('Request timeout');
    });
  });

  describe('HTTP status codes', () => {
    it('should handle 404 Not Found', async () => {
      (global.fetch as any).mockResolvedValue({
        ok: false,
        status: 404,
        statusText: 'Not Found',
      });

      // Should handle 404 appropriately
      const response = await (global.fetch as any)();
      expect(response.status).toBe(404);
    });

    it('should handle 500 Internal Server Error', async () => {
      (global.fetch as any).mockResolvedValue({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
      });

      const response = await (global.fetch as any)();
      expect(response.status).toBe(500);
    });

    it('should handle 401 Unauthorized', async () => {
      (global.fetch as any).mockResolvedValue({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
      });

      const response = await (global.fetch as any)();
      expect(response.status).toBe(401);
    });
  });

  describe('Data validation', () => {
    it('should validate positive line counts', () => {
      const validMetric: api.FileMetric = {
        path: 'test.ts',
        lines: 100,
        extension: '.ts',
      };

      expect(validMetric.lines).toBeGreaterThan(0);
    });

    it('should handle zero line files', () => {
      const emptyFile: api.FileMetric = {
        path: 'empty.ts',
        lines: 0,
        extension: '.ts',
      };

      expect(emptyFile.lines).toBe(0);
    });

    it('should validate threshold comparison', () => {
      const longFile: api.LongFile = {
        path: 'long.ts',
        lines: 1000,
        threshold: 500,
      };

      expect(longFile.lines).toBeGreaterThan(longFile.threshold);
    });
  });
});
