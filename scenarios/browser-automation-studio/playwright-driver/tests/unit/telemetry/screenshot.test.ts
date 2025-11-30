import { captureScreenshot, captureCompressedScreenshot } from '../../../src/telemetry/screenshot';
import { createMockPage, createTestConfig } from '../../helpers';

describe('Screenshot', () => {
  let mockPage: ReturnType<typeof createMockPage>;
  let config: ReturnType<typeof createTestConfig>;

  beforeEach(() => {
    mockPage = createMockPage();
    config = createTestConfig();
  });

  describe('captureScreenshot', () => {
    it('should capture PNG screenshot', async () => {
      const screenshot = await captureScreenshot(mockPage, config);

      expect(mockPage.screenshot).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'png',
        })
      );
      expect(screenshot).toBeDefined();
      expect(screenshot?.media_type).toBe('image/png');
    });

    it('should capture full page when configured', async () => {
      const configFullPage = createTestConfig({
        telemetry: {
          screenshot: {
            enabled: true,
            fullPage: true,
            quality: 80,
            maxSizeBytes: 5 * 1024 * 1024,
          },
          video: { enabled: false },
          dom: { enabled: true, maxSizeBytes: 1 * 1024 * 1024 },
          console: { enabled: true, maxEntries: 100 },
          network: { enabled: true, maxEvents: 500 },
          har: { enabled: false },
          tracing: { enabled: false },
        },
      });

      await captureScreenshot(mockPage, configFullPage);

      expect(mockPage.screenshot).toHaveBeenCalledWith(
        expect.objectContaining({
          fullPage: true,
        })
      );
    });

    it('should return base64 encoded screenshot', async () => {
      const mockBuffer = Buffer.from('fake-screenshot');
      mockPage.screenshot.mockResolvedValue(mockBuffer);

      const screenshot = await captureScreenshot(mockPage, config);

      expect(screenshot?.base64).toBe(mockBuffer.toString('base64'));
    });

    it('should include viewport dimensions', async () => {
      mockPage.viewportSize.mockReturnValue({ width: 1920, height: 1080 });

      const screenshot = await captureScreenshot(mockPage, config);

      expect(screenshot?.width).toBe(1920);
      expect(screenshot?.height).toBe(1080);
    });

    it('should handle viewport-only screenshot when full page too large', async () => {
      const configSmallMax = createTestConfig({
        telemetry: {
          screenshot: {
            enabled: true,
            fullPage: true,
            quality: 80,
            maxSizeBytes: 100, // Very small limit
          },
          video: { enabled: false },
          dom: { enabled: true, maxSizeBytes: 1 * 1024 * 1024 },
          console: { enabled: true, maxEntries: 100 },
          network: { enabled: true, maxEvents: 500 },
          har: { enabled: false },
          tracing: { enabled: false },
        },
      });

      const largeBuffer = Buffer.alloc(200); // Exceeds max
      const smallBuffer = Buffer.from('small');

      mockPage.screenshot
        .mockResolvedValueOnce(largeBuffer) // First call (full page) too large
        .mockResolvedValueOnce(smallBuffer); // Second call (viewport) acceptable

      const screenshot = await captureScreenshot(mockPage, configSmallMax);

      expect(mockPage.screenshot).toHaveBeenCalledTimes(2);
      expect(mockPage.screenshot).toHaveBeenNthCalledWith(
        1,
        expect.objectContaining({ fullPage: true })
      );
      expect(mockPage.screenshot).toHaveBeenNthCalledWith(
        2,
        expect.objectContaining({ fullPage: false })
      );
      expect(screenshot?.base64).toBe(smallBuffer.toString('base64'));
    });

    it('should return undefined when screenshot exceeds size limit', async () => {
      const configSmallMax = createTestConfig({
        telemetry: {
          screenshot: {
            enabled: true,
            fullPage: false,
            quality: 80,
            maxSizeBytes: 100,
          },
          video: { enabled: false },
          dom: { enabled: true, maxSizeBytes: 1 * 1024 * 1024 },
          console: { enabled: true, maxEntries: 100 },
          network: { enabled: true, maxEvents: 500 },
          har: { enabled: false },
          tracing: { enabled: false },
        },
      });

      const largeBuffer = Buffer.alloc(200);
      mockPage.screenshot.mockResolvedValue(largeBuffer);

      const screenshot = await captureScreenshot(mockPage, configSmallMax);

      expect(screenshot).toBeUndefined();
    });

    it('should return undefined when screenshots disabled', async () => {
      const configDisabled = createTestConfig({
        telemetry: {
          screenshot: {
            enabled: false,
            fullPage: false,
            quality: 80,
            maxSizeBytes: 5 * 1024 * 1024,
          },
          video: { enabled: false },
          dom: { enabled: true, maxSizeBytes: 1 * 1024 * 1024 },
          console: { enabled: true, maxEntries: 100 },
          network: { enabled: true, maxEvents: 500 },
          har: { enabled: false },
          tracing: { enabled: false },
        },
      });

      const screenshot = await captureScreenshot(mockPage, configDisabled);

      expect(mockPage.screenshot).not.toHaveBeenCalled();
      expect(screenshot).toBeUndefined();
    });

    it('should handle screenshot errors gracefully', async () => {
      mockPage.screenshot.mockRejectedValue(new Error('Screenshot failed'));

      const screenshot = await captureScreenshot(mockPage, config);

      expect(screenshot).toBeUndefined();
    });

    it('should handle page with no viewport', async () => {
      mockPage.viewportSize.mockReturnValue(null);

      const screenshot = await captureScreenshot(mockPage, config);

      expect(screenshot?.width).toBe(0);
      expect(screenshot?.height).toBe(0);
    });
  });

  describe('captureCompressedScreenshot', () => {
    it('should capture JPEG screenshot', async () => {
      const mockBuffer = Buffer.from('jpeg-screenshot');
      mockPage.screenshot.mockResolvedValue(mockBuffer);

      const screenshot = await captureCompressedScreenshot(mockPage, 80, false);

      expect(mockPage.screenshot).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'jpeg',
          quality: 80,
        })
      );
      expect(screenshot?.media_type).toBe('image/jpeg');
    });

    it('should reduce quality when screenshot too large', async () => {
      const largeBuffer = Buffer.alloc(200);
      const smallBuffer = Buffer.from('small');

      mockPage.screenshot
        .mockResolvedValueOnce(largeBuffer) // First call with quality 80
        .mockResolvedValueOnce(smallBuffer); // Second call with quality 60

      const screenshot = await captureCompressedScreenshot(mockPage, 80, false, 100);

      expect(mockPage.screenshot).toHaveBeenCalledTimes(2);
      expect(screenshot?.base64).toBe(smallBuffer.toString('base64'));
    });

    it('should return undefined when still too large after quality reduction', async () => {
      const largeBuffer = Buffer.alloc(200);

      mockPage.screenshot.mockResolvedValue(largeBuffer);

      const screenshot = await captureCompressedScreenshot(mockPage, 50, false, 100);

      expect(screenshot).toBeUndefined();
    });
  });
});
