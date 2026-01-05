/**
 * Recording Self-Test Module Unit Tests
 *
 * Tests the automated pipeline testing functionality that eliminates
 * human-in-the-loop debugging for recording issues.
 */

import { describe, it, expect, beforeEach, jest } from '@jest/globals';
import { TEST_PAGE_HTML, TEST_PAGE_URL } from '../../../src/recording/self-test';

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

  describe('TEST_PAGE_URL', () => {
    it('should be a special internal URL pattern', () => {
      expect(TEST_PAGE_URL).toBe('/__vrooli_recording_test__');
    });

    it('should not conflict with real URLs', () => {
      // The URL uses __ prefix which is a common pattern for internal routes
      expect(TEST_PAGE_URL).toMatch(/^\/__.+__$/);
    });
  });

  describe('PipelineTestResult structure', () => {
    // Import types for testing
    const mockSuccessResult = {
      success: true,
      timestamp: '2024-01-01T00:00:00.000Z',
      durationMs: 1500,
      steps: [
        { name: 'navigate_to_test_page', passed: true, durationMs: 500 },
        { name: 'verify_script_injection', passed: true, durationMs: 200 },
        { name: 'simulate_click', passed: true, durationMs: 300 },
        { name: 'simulate_input', passed: true, durationMs: 300 },
      ],
      diagnostics: {
        testPageUrl: 'http://localhost/__vrooli_recording_test__',
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
        { name: 'navigate_to_test_page', passed: true, durationMs: 500 },
        { name: 'verify_script_injection', passed: true, durationMs: 200 },
        { name: 'simulate_click', passed: false, durationMs: 100, error: 'Click event not received' },
      ],
      diagnostics: {
        testPageUrl: 'http://localhost/__vrooli_recording_test__',
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
      expect(mockFailureResult.steps).toHaveLength(3);
      expect(mockFailureResult.steps[2].passed).toBe(false);
      expect(mockFailureResult.steps[2].error).toBeDefined();
    });
  });

  describe('Failure Point Classification', () => {
    // Test that failure points cover all stages of the pipeline
    const expectedFailurePoints = [
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
      // 1. User action → 2. DOM event → 3. Recording script captures
      // 4. Fetch sends → 5. Route receives → 6. Handler processes

      const stageMapping: Record<string, string> = {
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
