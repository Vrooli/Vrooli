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
      expect(snapshot?.selector).toBeUndefined();
    });

    it('should return undefined when DOM capture disabled', async () => {
      const configDisabled = createTestConfig({
        telemetry: {
          dom: {
            enabled: false,
            maxSizeBytes: 1 * 1024 * 1024,
          },
        },
      });

      const snapshot = await captureDOMSnapshot(mockPage, configDisabled);

      expect(mockPage.content).not.toHaveBeenCalled();
      expect(snapshot).toBeUndefined();
    });

    it('should return undefined when HTML exceeds size limit', async () => {
      const configSmallMax = createTestConfig({
        telemetry: {
          dom: {
            enabled: true,
            maxSizeBytes: 100,
          },
        },
      });

      const largeHTML = 'x'.repeat(200); // Exceeds 100 byte limit
      mockPage.content.mockResolvedValue(largeHTML);

      const snapshot = await captureDOMSnapshot(mockPage, configSmallMax);

      expect(snapshot).toBeUndefined();
    });

    it('should handle content() errors gracefully', async () => {
      mockPage.content.mockRejectedValue(new Error('Failed to get content'));

      const snapshot = await captureDOMSnapshot(mockPage, config);

      expect(snapshot).toBeUndefined();
    });

    it('should measure HTML size in bytes', async () => {
      const configSmallMax = createTestConfig({
        telemetry: {
          dom: {
            enabled: true,
            maxSizeBytes: 50,
          },
        },
      });

      // UTF-8 characters may be multiple bytes
      const html = 'test'; // 4 bytes
      mockPage.content.mockResolvedValue(html);

      const snapshot = await captureDOMSnapshot(mockPage, configSmallMax);

      expect(snapshot).toBeDefined();
      expect(snapshot?.html).toBe(html);
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
    it('should capture element outer HTML', async () => {
      const selector = '#test-element';
      const mockHTML = '<div id="test-element"><span>Content</span></div>';

      const mockLocator = {
        evaluate: jest.fn().mockResolvedValue(mockHTML),
      };
      mockPage.locator.mockReturnValue(mockLocator as any);

      const snapshot = await captureElementSnapshot(mockPage, selector, config);

      expect(mockPage.locator).toHaveBeenCalledWith(selector);
      expect(mockLocator.evaluate).toHaveBeenCalledWith(expect.any(Function));
      expect(snapshot).toBeDefined();
      expect(snapshot?.html).toBe(mockHTML);
      expect(snapshot?.selector).toBe(selector);
    });

    it('should return undefined when DOM capture disabled', async () => {
      const configDisabled = createTestConfig({
        telemetry: {
          dom: {
            enabled: false,
            maxSizeBytes: 1 * 1024 * 1024,
          },
        },
      });

      const snapshot = await captureElementSnapshot(mockPage, '#test', configDisabled);

      expect(mockPage.locator).not.toHaveBeenCalled();
      expect(snapshot).toBeUndefined();
    });

    it('should return undefined when element HTML exceeds size limit', async () => {
      const configSmallMax = createTestConfig({
        telemetry: {
          dom: {
            enabled: true,
            maxSizeBytes: 50,
          },
        },
      });

      const largeHTML = 'x'.repeat(100);
      const mockLocator = {
        evaluate: jest.fn().mockResolvedValue(largeHTML),
      };
      mockPage.locator.mockReturnValue(mockLocator as any);

      const snapshot = await captureElementSnapshot(mockPage, '#test', configSmallMax);

      expect(snapshot).toBeUndefined();
    });

    it('should handle element not found gracefully', async () => {
      const mockLocator = {
        evaluate: jest.fn().mockRejectedValue(new Error('Element not found')),
      };
      mockPage.locator.mockReturnValue(mockLocator as any);

      const snapshot = await captureElementSnapshot(mockPage, '#nonexistent', config);

      expect(snapshot).toBeUndefined();
    });

    it('should handle null outerHTML', async () => {
      const mockLocator = {
        evaluate: jest.fn().mockResolvedValue(null),
      };
      mockPage.locator.mockReturnValue(mockLocator as any);

      const snapshot = await captureElementSnapshot(mockPage, '#test', config);

      expect(snapshot).toBeUndefined();
    });

    it('should capture element with special characters in selector', async () => {
      const selector = 'div[data-test="value"]';
      const mockHTML = '<div data-test="value">Content</div>';

      const mockLocator = {
        evaluate: jest.fn().mockResolvedValue(mockHTML),
      };
      mockPage.locator.mockReturnValue(mockLocator as any);

      const snapshot = await captureElementSnapshot(mockPage, selector, config);

      expect(snapshot?.selector).toBe(selector);
      expect(snapshot?.html).toBe(mockHTML);
    });

    it('should use evaluate function to get outerHTML', async () => {
      const mockLocator = {
        evaluate: jest.fn().mockResolvedValue('<div>Test</div>'),
      };
      mockPage.locator.mockReturnValue(mockLocator as any);

      await captureElementSnapshot(mockPage, '#test', config);

      const evaluateArg = (mockLocator.evaluate as jest.Mock).mock.calls[0][0];
      expect(typeof evaluateArg).toBe('function');

      // Test the evaluate function
      const mockElement = { outerHTML: '<div>Expected</div>' };
      const result = evaluateArg(mockElement);
      expect(result).toBe('<div>Expected</div>');
    });
  });

  describe('size calculations', () => {
    it('should calculate byte size correctly for ASCII', async () => {
      const configSmallMax = createTestConfig({
        telemetry: {
          dom: {
            enabled: true,
            maxSizeBytes: 10,
          },
        },
      });

      mockPage.content.mockResolvedValue('12345'); // 5 bytes

      const snapshot = await captureDOMSnapshot(mockPage, configSmallMax);

      expect(snapshot).toBeDefined();
    });

    it('should calculate byte size correctly for UTF-8', async () => {
      const configSmallMax = createTestConfig({
        telemetry: {
          dom: {
            enabled: true,
            maxSizeBytes: 10,
          },
        },
      });

      // Emoji are multiple bytes in UTF-8
      mockPage.content.mockResolvedValue('😀'); // 4 bytes

      const snapshot = await captureDOMSnapshot(mockPage, configSmallMax);

      expect(snapshot).toBeDefined();
    });
  });
});
