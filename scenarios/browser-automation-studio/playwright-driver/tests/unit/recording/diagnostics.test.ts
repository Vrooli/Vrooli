/**
 * Unit Tests for Recording Diagnostics
 *
 * Tests the diagnostic system that helps identify recording issues.
 */

import { describe, beforeEach, it, expect, jest } from '@jest/globals';
import {
  runRecordingDiagnostics,
  isRecordingReady,
  formatDiagnosticReport,
  RecordingDiagnosticLevel,
  DiagnosticSeverity,
  DIAGNOSTIC_CODES,
} from '../../../src/recording';
import type { InjectionStats, InjectionVerification } from '../../../src/recording';
import type { Page, BrowserContext } from 'rebrowser-playwright';

// Mock the verification module
jest.mock('../../../src/recording/validation/verification', () => ({
  verifyScriptInjection: jest.fn(),
}));

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

// Import the mocked function
import { verifyScriptInjection } from '../../../src/recording/validation/verification';

const mockVerifyScriptInjection = verifyScriptInjection as jest.MockedFunction<
  typeof verifyScriptInjection
>;

// Helper to create mock page
function createMockPage(): jest.Mocked<Page> {
  return {
    evaluate: jest.fn().mockResolvedValue(undefined),
    on: jest.fn(),
    context: jest.fn().mockReturnValue({
      newCDPSession: jest.fn().mockResolvedValue({
        send: jest.fn().mockResolvedValue({ result: { type: 'string', value: '{}' } }),
        detach: jest.fn().mockResolvedValue(undefined),
      }),
    }),
  } as unknown as jest.Mocked<Page>;
}

// Helper to create mock context
function createMockContext(): jest.Mocked<BrowserContext> {
  return {} as unknown as jest.Mocked<BrowserContext>;
}

// Helper to create mock context initializer
function createMockContextInitializer(stats: Partial<InjectionStats> = {}) {
  return {
    getInjectionStats: jest.fn().mockReturnValue({
      attempted: 1,
      successful: 1,
      failed: 0,
      skipped: 0,
      methods: { head: 1 },
      ...stats,
    }),
    isInitialized: jest.fn().mockReturnValue(true),
  };
}

// Helper to create good verification result
function createGoodVerification(): InjectionVerification {
  return {
    loaded: true,
    loadTime: Date.now(),
    version: '1.0.0',
    ready: true,
    handlersCount: 10,
    inMainContext: true,
  };
}

describe('Recording Diagnostics', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('runRecordingDiagnostics', () => {
    it('should return ready=true when everything is working', async () => {
      const page = createMockPage();
      const context = createMockContext();

      mockVerifyScriptInjection.mockResolvedValue(createGoodVerification());

      const result = await runRecordingDiagnostics(page, context, {
        level: RecordingDiagnosticLevel.STANDARD,
      });

      expect(result.ready).toBe(true);
      expect(result.issues).toHaveLength(0);
      expect(result.provider.name).toBe('rebrowser-playwright');
    });

    it('should detect when script is not loaded', async () => {
      const page = createMockPage();
      const context = createMockContext();

      mockVerifyScriptInjection.mockResolvedValue({
        loaded: false,
        loadTime: null,
        version: null,
        ready: false,
        handlersCount: 0,
        inMainContext: false,
        error: 'Script not found',
      });

      const result = await runRecordingDiagnostics(page, context);

      expect(result.ready).toBe(false);
      expect(result.issues.some((i) => i.code === DIAGNOSTIC_CODES.SCRIPT_NOT_LOADED)).toBe(true);
    });

    it('should detect when script has initialization error', async () => {
      const page = createMockPage();
      const context = createMockContext();

      mockVerifyScriptInjection.mockResolvedValue({
        loaded: true,
        loadTime: Date.now(),
        version: '1.0.0',
        ready: false,
        handlersCount: 0,
        inMainContext: true,
        initError: 'TypeError: Cannot read property x of undefined',
      });

      const result = await runRecordingDiagnostics(page, context);

      expect(result.ready).toBe(false);
      expect(result.issues.some((i) => i.code === DIAGNOSTIC_CODES.SCRIPT_INIT_ERROR)).toBe(true);
    });

    it('should detect when script is in wrong context', async () => {
      const page = createMockPage();
      const context = createMockContext();

      mockVerifyScriptInjection.mockResolvedValue({
        loaded: true,
        loadTime: Date.now(),
        version: '1.0.0',
        ready: true,
        handlersCount: 10,
        inMainContext: false, // Wrong context!
      });

      const result = await runRecordingDiagnostics(page, context, {
        level: RecordingDiagnosticLevel.STANDARD,
      });

      expect(result.ready).toBe(false);
      expect(result.issues.some((i) => i.code === DIAGNOSTIC_CODES.SCRIPT_WRONG_CONTEXT)).toBe(
        true
      );
    });

    it('should not check context for QUICK level', async () => {
      const page = createMockPage();
      const context = createMockContext();

      mockVerifyScriptInjection.mockResolvedValue({
        loaded: true,
        loadTime: Date.now(),
        version: '1.0.0',
        ready: true,
        handlersCount: 10,
        inMainContext: false, // Wrong context, but QUICK level doesn't check
      });

      const result = await runRecordingDiagnostics(page, context, {
        level: RecordingDiagnosticLevel.QUICK,
      });

      // QUICK level only checks loaded and ready, not context
      expect(result.ready).toBe(true);
      expect(result.issues.some((i) => i.code === DIAGNOSTIC_CODES.SCRIPT_WRONG_CONTEXT)).toBe(
        false
      );
    });

    it('should warn about low handler count', async () => {
      const page = createMockPage();
      const context = createMockContext();

      mockVerifyScriptInjection.mockResolvedValue({
        loaded: true,
        loadTime: Date.now(),
        version: '1.0.0',
        ready: true,
        handlersCount: 3, // Too low
        inMainContext: true,
      });

      const result = await runRecordingDiagnostics(page, context);

      // Low handler count is a warning, not an error
      expect(result.ready).toBe(true);
      expect(result.issues.some((i) => i.code === DIAGNOSTIC_CODES.SCRIPT_LOW_HANDLERS)).toBe(true);
      expect(
        result.issues.find((i) => i.code === DIAGNOSTIC_CODES.SCRIPT_LOW_HANDLERS)?.severity
      ).toBe(DiagnosticSeverity.WARNING);
    });

    it('should check injection stats when context initializer is provided', async () => {
      const page = createMockPage();
      const context = createMockContext();
      const initializer = createMockContextInitializer({
        attempted: 5,
        successful: 5,
        failed: 0,
      });

      mockVerifyScriptInjection.mockResolvedValue(createGoodVerification());

      const result = await runRecordingDiagnostics(page, context, {
        contextInitializer: initializer as any,
      });

      expect(result.injectionStats).toBeDefined();
      expect(result.injectionStats?.successful).toBe(5);
    });

    it('should warn about high injection failure rate', async () => {
      const page = createMockPage();
      const context = createMockContext();
      const initializer = createMockContextInitializer({
        attempted: 10,
        successful: 5,
        failed: 5, // 50% failure rate
      });

      mockVerifyScriptInjection.mockResolvedValue(createGoodVerification());

      const result = await runRecordingDiagnostics(page, context, {
        contextInitializer: initializer as any,
      });

      expect(
        result.issues.some((i) => i.code === DIAGNOSTIC_CODES.INJECTION_HIGH_FAILURE_RATE)
      ).toBe(true);
    });

    it('should error when all injections failed', async () => {
      const page = createMockPage();
      const context = createMockContext();
      const initializer = createMockContextInitializer({
        attempted: 5,
        successful: 0,
        failed: 5,
      });

      mockVerifyScriptInjection.mockResolvedValue(createGoodVerification());

      const result = await runRecordingDiagnostics(page, context, {
        contextInitializer: initializer as any,
      });

      expect(result.issues.some((i) => i.code === DIAGNOSTIC_CODES.INJECTION_ALL_FAILED)).toBe(
        true
      );
      expect(
        result.issues.find((i) => i.code === DIAGNOSTIC_CODES.INJECTION_ALL_FAILED)?.severity
      ).toBe(DiagnosticSeverity.ERROR);
    });
  });

  describe('isRecordingReady', () => {
    it('should return true when recording is ready', async () => {
      const page = createMockPage();

      mockVerifyScriptInjection.mockResolvedValue(createGoodVerification());

      const ready = await isRecordingReady(page);

      expect(ready).toBe(true);
    });

    it('should return false when script not loaded', async () => {
      const page = createMockPage();

      mockVerifyScriptInjection.mockResolvedValue({
        loaded: false,
        loadTime: null,
        version: null,
        ready: false,
        handlersCount: 0,
        inMainContext: false,
      });

      const ready = await isRecordingReady(page);

      expect(ready).toBe(false);
    });

    it('should return false when script in wrong context', async () => {
      const page = createMockPage();

      mockVerifyScriptInjection.mockResolvedValue({
        loaded: true,
        loadTime: Date.now(),
        version: '1.0.0',
        ready: true,
        handlersCount: 10,
        inMainContext: false,
      });

      const ready = await isRecordingReady(page);

      expect(ready).toBe(false);
    });

    it('should return false on verification error', async () => {
      const page = createMockPage();

      mockVerifyScriptInjection.mockRejectedValue(new Error('CDP not available'));

      const ready = await isRecordingReady(page);

      expect(ready).toBe(false);
    });
  });

  describe('formatDiagnosticReport', () => {
    it('should format a ready result correctly', () => {
      const result = {
        ready: true,
        timestamp: '2024-01-01T00:00:00.000Z',
        durationMs: 50,
        level: RecordingDiagnosticLevel.STANDARD,
        issues: [],
        provider: {
          name: 'rebrowser-playwright',
          evaluateIsolated: true,
          exposeBindingIsolated: true,
        },
        scriptVerification: {
          loaded: true,
          loadTime: Date.now(),
          version: '1.0.0',
          ready: true,
          handlersCount: 10,
          inMainContext: true,
        },
      };

      const report = formatDiagnosticReport(result);

      expect(report).toContain('READY');
      expect(report).toContain('rebrowser-playwright');
      expect(report).toContain('Issues: None');
    });

    it('should format issues correctly', () => {
      const result = {
        ready: false,
        timestamp: '2024-01-01T00:00:00.000Z',
        durationMs: 50,
        level: RecordingDiagnosticLevel.STANDARD,
        issues: [
          {
            code: DIAGNOSTIC_CODES.SCRIPT_NOT_LOADED,
            message: 'Recording script not loaded',
            severity: DiagnosticSeverity.ERROR,
            suggestion: 'Check HTML injection',
          },
        ],
        provider: {
          name: 'rebrowser-playwright',
          evaluateIsolated: true,
          exposeBindingIsolated: true,
        },
      };

      const report = formatDiagnosticReport(result);

      expect(report).toContain('NOT READY');
      expect(report).toContain(DIAGNOSTIC_CODES.SCRIPT_NOT_LOADED);
      expect(report).toContain('Recording script not loaded');
      expect(report).toContain('Check HTML injection');
    });
  });

  describe('DIAGNOSTIC_CODES', () => {
    it('should have unique codes', () => {
      const codes = Object.values(DIAGNOSTIC_CODES);
      const uniqueCodes = new Set(codes);
      expect(uniqueCodes.size).toBe(codes.length);
    });

    it('should have SCREAMING_SNAKE_CASE format', () => {
      for (const code of Object.values(DIAGNOSTIC_CODES)) {
        expect(code).toMatch(/^[A-Z][A-Z0-9_]*$/);
      }
    });

    it('should cover all major issue categories', () => {
      // Script issues
      expect(DIAGNOSTIC_CODES.SCRIPT_NOT_LOADED).toBeDefined();
      expect(DIAGNOSTIC_CODES.SCRIPT_NOT_READY).toBeDefined();
      expect(DIAGNOSTIC_CODES.SCRIPT_INIT_ERROR).toBeDefined();
      expect(DIAGNOSTIC_CODES.SCRIPT_WRONG_CONTEXT).toBeDefined();

      // Injection issues
      expect(DIAGNOSTIC_CODES.INJECTION_NO_ATTEMPTS).toBeDefined();
      expect(DIAGNOSTIC_CODES.INJECTION_ALL_FAILED).toBeDefined();

      // Event flow issues
      expect(DIAGNOSTIC_CODES.EVENT_NOT_RECEIVED).toBeDefined();
    });
  });
});

describe('DiagnosticSeverity', () => {
  it('should define severity levels', () => {
    expect(DiagnosticSeverity.ERROR).toBe('error');
    expect(DiagnosticSeverity.WARNING).toBe('warning');
    expect(DiagnosticSeverity.INFO).toBe('info');
  });
});

describe('RecordingDiagnosticLevel', () => {
  it('should define diagnostic levels', () => {
    expect(RecordingDiagnosticLevel.QUICK).toBe('quick');
    expect(RecordingDiagnosticLevel.STANDARD).toBe('standard');
    expect(RecordingDiagnosticLevel.FULL).toBe('full');
  });
});
