import { captureDOMSnapshot, captureElementSnapshot } from '../../../src/telemetry/dom';
import { createMockPage, createTestConfig } from '../../helpers';

describe('DOM Telemetry', () => {
  let mockPage: ReturnType<typeof createMockPage>;
  let config: ReturnType<typeof createTestConfig>;

  beforeEach(() => {
    mockPage = createMockPage();
    config = createTestConfig();
  });

  describe('captureDOMSnapshot', () => {
    it('should capture full page HTML', async () => {
      const mockHTML = '<html><body><h1>Test</h1></body></html>';
      mockPage.content.mockResolvedValue(mockHTML);

      const snapshot = await captureDOMSnapshot(mockPage, config);

      expect(mockPage.content).toHaveBeenCalled();
      expect(snapshot).toBeDefined();
      expect(snapshot?.html).toBe(mockHTML);
    });

    it('should include preview of HTML', async () => {
      const mockHTML = '<html><body><h1>Test</h1></body></html>';
      mockPage.content.mockResolvedValue(mockHTML);

      const snapshot = await captureDOMSnapshot(mockPage, config);

      expect(snapshot?.preview).toBeDefined();
      expect(mockHTML.startsWith(snapshot?.preview || '')).toBe(true);
    });

    it('should include collected_at timestamp', async () => {
      const mockHTML = '<html><body>Test</body></html>';
      mockPage.content.mockResolvedValue(mockHTML);

      const before = new Date().toISOString();
      const snapshot = await captureDOMSnapshot(mockPage, config);
      const after = new Date().toISOString();

      expect(snapshot?.collected_at).toBeDefined();
      expect(snapshot?.collected_at! >= before).toBe(true);
      expect(snapshot?.collected_at! <= after).toBe(true);
    });

    it('should return undefined when DOM capture disabled', async () => {
      const configDisabled = createTestConfig({
        telemetry: {
          screenshot: { enabled: true, fullPage: false, quality: 80, maxSizeBytes: 5 * 1024 * 1024 },
          video: { enabled: false },
          dom: { enabled: false, maxSizeBytes: 1 * 1024 * 1024 },
          console: { enabled: true, maxEntries: 100 },
          network: { enabled: true, maxEvents: 500 },
          har: { enabled: false },
          tracing: { enabled: false },
        },
      });

      const snapshot = await captureDOMSnapshot(mockPage, configDisabled);

      expect(mockPage.content).not.toHaveBeenCalled();
      expect(snapshot).toBeUndefined();
    });

    it('should truncate HTML that exceeds size limit', async () => {
      const configSmallMax = createTestConfig({
        telemetry: {
          screenshot: { enabled: true, fullPage: false, quality: 80, maxSizeBytes: 5 * 1024 * 1024 },
          video: { enabled: false },
          dom: { enabled: true, maxSizeBytes: 100 },
          console: { enabled: true, maxEntries: 100 },
          network: { enabled: true, maxEvents: 500 },
          har: { enabled: false },
          tracing: { enabled: false },
        },
      });

      const largeHTML = 'x'.repeat(200); // Exceeds 100 byte limit
      mockPage.content.mockResolvedValue(largeHTML);

      const snapshot = await captureDOMSnapshot(mockPage, configSmallMax);

      expect(snapshot).toBeDefined();
      expect(snapshot?.truncated).toBe(true);
      expect(snapshot?.html?.length).toBe(100);
    });

    it('should handle content() errors gracefully', async () => {
      mockPage.content.mockRejectedValue(new Error('Failed to get content'));

      const snapshot = await captureDOMSnapshot(mockPage, config);

      expect(snapshot).toBeUndefined();
    });

    it('should capture with default max size', async () => {
      const defaultConfig = createTestConfig();
      const html = '<html><body>Test</body></html>';
      mockPage.content.mockResolvedValue(html);

      const snapshot = await captureDOMSnapshot(mockPage, defaultConfig);

      expect(snapshot).toBeDefined();
    });
  });

  describe('captureElementSnapshot', () => {
    it('should capture element innerHTML', async () => {
      const selector = '#test-element';
      const mockHTML = '<span>Content</span>';

      const mockLocator = mockPage.locator(selector);
      mockLocator.first().innerHTML.mockResolvedValue(mockHTML);

      const snapshot = await captureElementSnapshot(mockPage, selector, config);

      expect(mockPage.locator).toHaveBeenCalledWith(selector);
      expect(snapshot).toBeDefined();
      expect(snapshot?.html).toBe(mockHTML);
    });

    it('should return undefined when DOM capture disabled', async () => {
      const configDisabled = createTestConfig({
        telemetry: {
          screenshot: { enabled: true, fullPage: false, quality: 80, maxSizeBytes: 5 * 1024 * 1024 },
          video: { enabled: false },
          dom: { enabled: false, maxSizeBytes: 1 * 1024 * 1024 },
          console: { enabled: true, maxEntries: 100 },
          network: { enabled: true, maxEvents: 500 },
          har: { enabled: false },
          tracing: { enabled: false },
        },
      });

      const snapshot = await captureElementSnapshot(mockPage, '#test', configDisabled);

      expect(snapshot).toBeUndefined();
    });

    it('should truncate element HTML that exceeds size limit', async () => {
      const configSmallMax = createTestConfig({
        telemetry: {
          screenshot: { enabled: true, fullPage: false, quality: 80, maxSizeBytes: 5 * 1024 * 1024 },
          video: { enabled: false },
          dom: { enabled: true, maxSizeBytes: 50 },
          console: { enabled: true, maxEntries: 100 },
          network: { enabled: true, maxEvents: 500 },
          har: { enabled: false },
          tracing: { enabled: false },
        },
      });

      const largeHTML = 'x'.repeat(100);
      const mockLocator = mockPage.locator('#test');
      mockLocator.first().innerHTML.mockResolvedValue(largeHTML);

      const snapshot = await captureElementSnapshot(mockPage, '#test', configSmallMax);

      expect(snapshot).toBeDefined();
      expect(snapshot?.truncated).toBe(true);
      expect(snapshot?.html?.length).toBe(50);
    });

    it('should handle element not found gracefully', async () => {
      const mockLocator = mockPage.locator('#nonexistent');
      mockLocator.first().innerHTML.mockRejectedValue(new Error('Element not found'));

      const snapshot = await captureElementSnapshot(mockPage, '#nonexistent', config);

      expect(snapshot).toBeUndefined();
    });

    it('should include collected_at timestamp', async () => {
      const mockHTML = '<span>Content</span>';
      const mockLocator = mockPage.locator('#test');
      mockLocator.first().innerHTML.mockResolvedValue(mockHTML);

      const snapshot = await captureElementSnapshot(mockPage, '#test', config);

      expect(snapshot?.collected_at).toBeDefined();
    });
  });

  describe('size calculations', () => {
    it('should calculate byte size correctly for ASCII', async () => {
      const configSmallMax = createTestConfig({
        telemetry: {
          screenshot: { enabled: true, fullPage: false, quality: 80, maxSizeBytes: 5 * 1024 * 1024 },
          video: { enabled: false },
          dom: { enabled: true, maxSizeBytes: 10 },
          console: { enabled: true, maxEntries: 100 },
          network: { enabled: true, maxEvents: 500 },
          har: { enabled: false },
          tracing: { enabled: false },
        },
      });

      mockPage.content.mockResolvedValue('12345'); // 5 bytes

      const snapshot = await captureDOMSnapshot(mockPage, configSmallMax);

      expect(snapshot).toBeDefined();
      expect(snapshot?.truncated).toBeFalsy();
    });
  });
});
