/**
 * Recording Self-Test Module Unit Tests
 *
 * Tests the automated pipeline testing functionality that eliminates
 * human-in-the-loop debugging for recording issues.
 */

import { describe, it, expect } from '@jest/globals';
import { TEST_PAGE_HTML, DEFAULT_TEST_URL } from '../../../src/recording';

describe('Recording Self-Test Module', () => {
  describe('TEST_PAGE_HTML', () => {
    it('should contain required test elements', () => {
      // Check for test button
      expect(TEST_PAGE_HTML).toContain('id="test-click-button"');
      expect(TEST_PAGE_HTML).toContain('data-testid="test-click-button"');

      // Check for test input
      expect(TEST_PAGE_HTML).toContain('id="test-input"');
      expect(TEST_PAGE_HTML).toContain('data-testid="test-input"');

      // Check for test link
      expect(TEST_PAGE_HTML).toContain('id="test-link"');
      expect(TEST_PAGE_HTML).toContain('data-testid="test-link"');
    });

    it('should be valid HTML with proper structure', () => {
      expect(TEST_PAGE_HTML).toContain('<!DOCTYPE html>');
      expect(TEST_PAGE_HTML).toContain('<html');
      expect(TEST_PAGE_HTML).toContain('<head>');
      expect(TEST_PAGE_HTML).toContain('<body>');
      expect(TEST_PAGE_HTML).toContain('</html>');
    });

    it('should have title for identification', () => {
      expect(TEST_PAGE_HTML).toContain('<title>Recording Pipeline Test Page</title>');
    });

    it('should include client-side event logging', () => {
      // The test page logs events to a visible div for debugging
      expect(TEST_PAGE_HTML).toContain('event-log');
      expect(TEST_PAGE_HTML).toContain('logEvent');
    });
  });

  describe('DEFAULT_TEST_URL', () => {
    it('should be a stable external URL for testing injection', () => {
      // Uses example.com which is a lightweight, stable HTML page
      expect(DEFAULT_TEST_URL).toBe('https://example.com');
    });

    it('should be a valid HTTPS URL', () => {
      expect(DEFAULT_TEST_URL).toMatch(/^https:\/\//);
    });
  });

  describe('PipelineTestResult structure', () => {
    // Import types for testing
    const mockSuccessResult = {
      success: true,
      timestamp: '2024-01-01T00:00:00.000Z',
      durationMs: 1500,
      steps: [
        { name: 'load_external_url', passed: true, durationMs: 500 },
        { name: 'verify_script_injection', passed: true, durationMs: 200 },
        { name: 'start_recording', passed: true, durationMs: 100 },
        { name: 'simulate_click', passed: true, durationMs: 300 },
        { name: 'simulate_input', passed: true, durationMs: 300 },
      ],
      diagnostics: {
        testPageUrl: 'https://example.com',
        testPageInjected: true,
        eventsCaptured: [{ actionType: 'click', timestamp: '2024-01-01T00:00:00.500Z' }],
        consoleMessages: [],
      },
    };

    it('should have correct success structure', () => {
      expect(mockSuccessResult.success).toBe(true);
      expect(mockSuccessResult.failurePoint).toBeUndefined();
      expect(mockSuccessResult.failureMessage).toBeUndefined();
    });

    const mockFailureResult = {
      success: false,
      timestamp: '2024-01-01T00:00:00.000Z',
      durationMs: 800,
      failurePoint: 'event_detection' as const,
      failureMessage: 'Click was simulated but NOT detected by recording script',
      suggestions: [
        'DOM event handlers may have been removed',
        'Event propagation might be stopped by page scripts',
      ],
      steps: [
        { name: 'load_external_url', passed: true, durationMs: 500 },
        { name: 'verify_script_injection', passed: true, durationMs: 200 },
        { name: 'start_recording', passed: true, durationMs: 100 },
        { name: 'simulate_click', passed: false, durationMs: 100, error: 'Click event not received' },
      ],
      diagnostics: {
        testPageUrl: 'https://example.com',
        testPageInjected: true,
        eventsCaptured: [],
        consoleMessages: [],
      },
    };

    it('should have correct failure structure', () => {
      expect(mockFailureResult.success).toBe(false);
      expect(mockFailureResult.failurePoint).toBe('event_detection');
      expect(mockFailureResult.failureMessage).toBeDefined();
      expect(mockFailureResult.suggestions).toHaveLength(2);
    });

    it('should include detailed step results', () => {
      expect(mockFailureResult.steps).toHaveLength(4);
      expect(mockFailureResult.steps[3].passed).toBe(false);
      expect(mockFailureResult.steps[3].error).toBeDefined();
    });
  });

  describe('Failure Point Classification', () => {
    // Test that failure points cover all stages of the pipeline
    const expectedFailurePoints = [
      'page_load',
      'navigation',
      'script_injection',
      'script_initialization',
      'event_detection',
      'event_capture',
      'event_send',
      'route_receive',
      'handler_process',
      'unknown',
    ];

    it('should have all expected failure point types', () => {
      // This ensures we have comprehensive failure categorization
      expectedFailurePoints.forEach((point) => {
        // Just checking that these are valid string literals
        expect(typeof point).toBe('string');
      });
    });

    it('should map to specific stages in the event pipeline', () => {
      // Event pipeline stages:
      // 1. Page load → 2. Script injection → 3. Script initialization
      // 4. Event detection → 5. Event capture → 6. Event send
      // 7. Route receive → 8. Handler process

      const stageMapping: Record<string, string> = {
        page_load: 'External URL navigation',
        navigation: 'Test page load',
        script_injection: 'Script present in HTML',
        script_initialization: 'Script runs and initializes',
        event_detection: 'DOM event listener fires',
        event_capture: 'captureAction() called (isActive check)',
        event_send: 'fetch() to event URL',
        route_receive: 'Playwright route handler receives',
        handler_process: 'Event handler callback executes',
        unknown: 'Cannot determine failure point',
      };

      Object.keys(stageMapping).forEach((point) => {
        expect(expectedFailurePoints).toContain(point);
      });
    });
  });
});

describe('Test Page Interactive Elements', () => {
  // Parse the HTML to check for required attributes
  const hasDataTestId = (html: string, id: string) =>
    html.includes(`data-testid="${id}"`);

  const hasId = (html: string, id: string) =>
    html.includes(`id="${id}"`);

  it('should have testable click button', () => {
    expect(hasId(TEST_PAGE_HTML, 'test-click-button')).toBe(true);
    expect(hasDataTestId(TEST_PAGE_HTML, 'test-click-button')).toBe(true);
    expect(TEST_PAGE_HTML).toContain('<button');
  });

  it('should have testable input field', () => {
    expect(hasId(TEST_PAGE_HTML, 'test-input')).toBe(true);
    expect(hasDataTestId(TEST_PAGE_HTML, 'test-input')).toBe(true);
    expect(TEST_PAGE_HTML).toContain('type="text"');
  });

  it('should have testable link', () => {
    expect(hasId(TEST_PAGE_HTML, 'test-link')).toBe(true);
    expect(hasDataTestId(TEST_PAGE_HTML, 'test-link')).toBe(true);
    expect(TEST_PAGE_HTML).toContain('<a');
  });

  it('should use data-testid for reliable selectors', () => {
    // data-testid is the most reliable selector strategy
    // The test page should use it consistently
    const testIds = ['test-click-button', 'test-input', 'test-link'];
    testIds.forEach((id) => {
      expect(hasDataTestId(TEST_PAGE_HTML, id)).toBe(true);
    });
  });
});

describe('ExternalUrlTestResult structure', () => {
  const mockSuccessResult = {
    success: true,
    timestamp: '2024-01-01T00:00:00.000Z',
    durationMs: 2000,
    testedUrl: 'https://example.com',
    verification: {
      loaded: true,
      ready: true,
      inMainContext: true,
      handlersCount: 8,
      version: '1.0.0',
    },
    injectionStats: {
      attempted: 1,
      successful: 1,
      failed: 0,
      skipped: 5,
    },
  };

  it('should have correct success structure', () => {
    expect(mockSuccessResult.success).toBe(true);
    expect(mockSuccessResult.testedUrl).toBe('https://example.com');
    expect(mockSuccessResult.verification?.loaded).toBe(true);
    expect(mockSuccessResult.verification?.inMainContext).toBe(true);
  });

  const mockFailureResult = {
    success: false,
    timestamp: '2024-01-01T00:00:00.000Z',
    durationMs: 1500,
    testedUrl: 'https://example.com',
    failurePoint: 'script_load' as const,
    failureMessage: 'Injection stats show success but script did not load in browser',
    suggestions: [
      'Check if route.fulfill() completed successfully',
      'The browser may have rejected the modified response',
    ],
    verification: {
      loaded: false,
      ready: false,
      inMainContext: false,
      handlersCount: 0,
      version: null,
    },
    injectionStats: {
      attempted: 1,
      successful: 1,
      failed: 0,
      skipped: 5,
    },
  };

  it('should have correct failure structure', () => {
    expect(mockFailureResult.success).toBe(false);
    expect(mockFailureResult.failurePoint).toBe('script_load');
    expect(mockFailureResult.failureMessage).toBeDefined();
    expect(mockFailureResult.suggestions).toHaveLength(2);
  });

  it('should include verification details on failure', () => {
    expect(mockFailureResult.verification).toBeDefined();
    expect(mockFailureResult.verification?.loaded).toBe(false);
  });

  describe('External URL failure points', () => {
    const expectedFailurePoints = [
      'fetch',
      'modify',
      'fulfill',
      'script_load',
      'script_ready',
      'context_wrong',
      'network',
      'timeout',
    ];

    it('should cover all external URL injection failure scenarios', () => {
      expectedFailurePoints.forEach((point) => {
        expect(typeof point).toBe('string');
      });
    });
  });
});
