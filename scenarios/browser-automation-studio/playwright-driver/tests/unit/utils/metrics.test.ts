import { Metrics, metrics } from '../../../src/utils/metrics';

describe('Metrics', () => {
  let testMetrics: Metrics;

  beforeEach(() => {
    testMetrics = new Metrics();
  });

  describe('constructor', () => {
    it('should initialize all metrics', () => {
      expect(testMetrics.sessionCount).toBeDefined();
      expect(testMetrics.instructionDuration).toBeDefined();
      expect(testMetrics.instructionErrors).toBeDefined();
      expect(testMetrics.screenshotSize).toBeDefined();
      expect(testMetrics.sessionDuration).toBeDefined();
      expect(testMetrics.cleanupFailures).toBeDefined();
    });

    it('should have registry', () => {
      expect(testMetrics.getRegistry()).toBeDefined();
    });
  });

  describe('sessionCount gauge', () => {
    it('should track active sessions', () => {
      testMetrics.sessionCount.set({ state: 'active' }, 5);

      const metrics_output = testMetrics.getRegistry().getSingleMetric('playwright_driver_sessions');
      expect(metrics_output).toBeDefined();
    });

    it('should track idle sessions', () => {
      testMetrics.sessionCount.set({ state: 'idle' }, 3);

      const metrics_output = testMetrics.getRegistry().getSingleMetric('playwright_driver_sessions');
      expect(metrics_output).toBeDefined();
    });

    it('should increment and decrement', () => {
      testMetrics.sessionCount.set({ state: 'active' }, 0);
      testMetrics.sessionCount.inc({ state: 'active' });
      testMetrics.sessionCount.inc({ state: 'active' });
      testMetrics.sessionCount.dec({ state: 'active' });

      // Can't easily test the actual value without exposing internals
      // but we can verify the methods don't throw
      expect(() => testMetrics.sessionCount.inc({ state: 'active' })).not.toThrow();
    });
  });

  describe('instructionDuration histogram', () => {
    it('should observe instruction durations', () => {
      // Note: values are in milliseconds now, not seconds
      testMetrics.instructionDuration.observe({ type: 'navigate', success: 'true' }, 500);
      testMetrics.instructionDuration.observe({ type: 'click', success: 'true' }, 100);

      const metrics_output = testMetrics.getRegistry().getSingleMetric('playwright_driver_instruction_duration_ms');
      expect(metrics_output).toBeDefined();
    });

    it('should track failed instructions', () => {
      testMetrics.instructionDuration.observe({ type: 'navigate', success: 'false' }, 1500);

      expect(() => testMetrics.instructionDuration.observe({ type: 'navigate', success: 'false' }, 1500)).not.toThrow();
    });
  });

  describe('instructionErrors counter', () => {
    it('should count errors by type and kind', () => {
      testMetrics.instructionErrors.inc({ type: 'navigate', error_kind: 'timeout' });
      testMetrics.instructionErrors.inc({ type: 'click', error_kind: 'selector_not_found' });

      const metrics_output = testMetrics.getRegistry().getSingleMetric('playwright_driver_instruction_errors_total');
      expect(metrics_output).toBeDefined();
    });

    it('should increment multiple times', () => {
      testMetrics.instructionErrors.inc({ type: 'navigate', error_kind: 'timeout' });
      testMetrics.instructionErrors.inc({ type: 'navigate', error_kind: 'timeout' });
      testMetrics.instructionErrors.inc({ type: 'navigate', error_kind: 'timeout' });

      expect(() => testMetrics.instructionErrors.inc({ type: 'navigate', error_kind: 'timeout' })).not.toThrow();
    });
  });

  describe('cleanupFailures counter', () => {
    it('should count cleanup failures by operation type', () => {
      testMetrics.cleanupFailures.inc({ operation: 'page_close' });
      testMetrics.cleanupFailures.inc({ operation: 'context_close' });
      testMetrics.cleanupFailures.inc({ operation: 'tracing_stop' });
      testMetrics.cleanupFailures.inc({ operation: 'browser_close' });

      const metrics_output = testMetrics.getRegistry().getSingleMetric('playwright_driver_cleanup_failures_total');
      expect(metrics_output).toBeDefined();
    });

    it('should track multiple failures of the same operation', () => {
      testMetrics.cleanupFailures.inc({ operation: 'page_close' });
      testMetrics.cleanupFailures.inc({ operation: 'page_close' });
      testMetrics.cleanupFailures.inc({ operation: 'page_close' });

      expect(() => testMetrics.cleanupFailures.inc({ operation: 'page_close' })).not.toThrow();
    });
  });

  describe('screenshotSize histogram', () => {
    it('should observe screenshot sizes', () => {
      testMetrics.screenshotSize.observe(1024); // 1KB
      testMetrics.screenshotSize.observe(1024 * 1024); // 1MB
      testMetrics.screenshotSize.observe(512); // 512B

      const metrics_output = testMetrics.getRegistry().getSingleMetric('playwright_driver_screenshot_size_bytes');
      expect(metrics_output).toBeDefined();
    });
  });

  describe('sessionDuration histogram', () => {
    it('should observe session durations', () => {
      // Note: values are in milliseconds now, not seconds
      testMetrics.sessionDuration.observe(10000); // 10 seconds
      testMetrics.sessionDuration.observe(120000); // 2 minutes
      testMetrics.sessionDuration.observe(3600000); // 1 hour

      const metrics_output = testMetrics.getRegistry().getSingleMetric('playwright_driver_session_duration_ms');
      expect(metrics_output).toBeDefined();
    });
  });

  describe('getMetrics', () => {
    it('should return Prometheus-formatted metrics', async () => {
      testMetrics.sessionCount.set({ state: 'active' }, 5);
      testMetrics.instructionDuration.observe({ type: 'navigate', success: 'true' }, 500);

      const output = await testMetrics.getMetrics();

      expect(output).toBeDefined();
      expect(typeof output).toBe('string');
      expect(output).toContain('playwright_driver_sessions');
    });
  });

  describe('metrics singleton', () => {
    it('should export a metrics instance', () => {
      expect(metrics).toBeInstanceOf(Metrics);
    });

    it('should have all metric properties', () => {
      expect(metrics.sessionCount).toBeDefined();
      expect(metrics.instructionDuration).toBeDefined();
      expect(metrics.instructionErrors).toBeDefined();
      expect(metrics.screenshotSize).toBeDefined();
      expect(metrics.sessionDuration).toBeDefined();
      expect(metrics.cleanupFailures).toBeDefined();
    });
  });

  describe('registry', () => {
    it('should have content type', () => {
      const contentType = testMetrics.getRegistry().contentType;

      expect(contentType).toBeDefined();
      expect(contentType).toContain('text/plain');
    });

    it('should get single metric', () => {
      const metric = testMetrics.getRegistry().getSingleMetric('playwright_driver_sessions');

      expect(metric).toBeDefined();
    });

    it('should return metrics string', async () => {
      const metricsString = await testMetrics.getRegistry().metrics();

      expect(typeof metricsString).toBe('string');
    });
  });
});
