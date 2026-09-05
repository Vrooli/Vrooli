import { captureScreenshot, captureCompressedScreenshot } from '../../../src/telemetry/screenshot';
import { createMockPage, createTestConfig } from '../../helpers';

describe('Screenshot', () => {
  let mockPage: ReturnType<typeof createMockPage>;
  let config: ReturnType<typeof createTestConfig>;
  let screenshotMock: jest.MockedFunction<typeof mockPage.screenshot>;

  beforeEach(() => {
    mockPage = createMockPage();
    config = createTestConfig();
    // eslint-disable-next-line @typescript-eslint/unbound-method -- jest mock does not use this
    screenshotMock = mockPage.screenshot as jest.MockedFunction<typeof mockPage.screenshot>;
  });

  describe('captureScreenshot', () => {
    it('should capture PNG screenshot', async () => {
      const screenshot = await captureScreenshot(mockPage, config);

      expect(screenshotMock).toHaveBeenCalledWith(
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
          dom: { enabled: true, maxSizeBytes: 1 * 1024 * 1024 },
          console: { enabled: true, maxEntries: 100 },
          network: { enabled: true, maxEvents: 500 },
          har: { enabled: false },
          tracing: { enabled: false },
        },
      });

      await captureScreenshot(mockPage, configFullPage);

      expect(screenshotMock).toHaveBeenCalledWith(
        expect.objectContaining({
          fullPage: true,
        })
      );
    });

    it('should return base64 encoded screenshot', async () => {
      const mockBuffer = Buffer.from('fake-screenshot');
      screenshotMock.mockResolvedValue(mockBuffer);

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
          dom: { enabled: true, maxSizeBytes: 1 * 1024 * 1024 },
          console: { enabled: true, maxEntries: 100 },
          network: { enabled: true, maxEvents: 500 },
          har: { enabled: false },
          tracing: { enabled: false },
        },
      });

      const largeBuffer = Buffer.alloc(200); // Exceeds max
      const smallBuffer = Buffer.from('small');

      screenshotMock
        .mockResolvedValueOnce(largeBuffer) // First call (full page) too large
        .mockResolvedValueOnce(smallBuffer); // Second call (viewport) acceptable

      const screenshot = await captureScreenshot(mockPage, configSmallMax);

      expect(screenshotMock).toHaveBeenCalledTimes(2);
      const firstCall = screenshotMock.mock.calls[0]?.[0];
      const secondCall = screenshotMock.mock.calls[1]?.[0];

      expect(firstCall?.fullPage).toBe(true);
      expect(secondCall?.clip).toMatchObject({ x: 0, y: 0 });
      expect(screenshot?.base64).toBe(smallBuffer.toString('base64'));
    });

    it('should fall back to a bounded JPEG when a viewport PNG exceeds the size limit', async () => {
      const configSmallMax = createTestConfig({
        telemetry: {
          screenshot: {
            enabled: true,
            fullPage: false,
            quality: 80,
            maxSizeBytes: 100,
          },
          dom: { enabled: true, maxSizeBytes: 1 * 1024 * 1024 },
          console: { enabled: true, maxEntries: 100 },
          network: { enabled: true, maxEvents: 500 },
          har: { enabled: false },
          tracing: { enabled: false },
        },
      });

      const largeBuffer = Buffer.alloc(200);
      const compressedBuffer = Buffer.from('compressed');
      screenshotMock
        .mockResolvedValueOnce(largeBuffer)
        .mockResolvedValueOnce(compressedBuffer);

      const screenshot = await captureScreenshot(mockPage, configSmallMax);

      expect(screenshotMock).toHaveBeenCalledTimes(2);
      expect(screenshotMock.mock.calls[1]?.[0]).toEqual(expect.objectContaining({
        type: 'jpeg',
        quality: 80,
      }));
      expect(screenshot?.base64).toBe(compressedBuffer.toString('base64'));
      expect(screenshot?.media_type).toBe('image/jpeg');
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
          dom: { enabled: true, maxSizeBytes: 1 * 1024 * 1024 },
          console: { enabled: true, maxEntries: 100 },
          network: { enabled: true, maxEvents: 500 },
          har: { enabled: false },
          tracing: { enabled: false },
        },
      });

      const screenshot = await captureScreenshot(mockPage, configDisabled);

      expect(screenshotMock).not.toHaveBeenCalled();
      expect(screenshot).toBeUndefined();
    });

    it('should handle screenshot errors gracefully', async () => {
      screenshotMock.mockRejectedValue(new Error('Screenshot failed'));

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
      screenshotMock.mockResolvedValue(mockBuffer);

      const screenshot = await captureCompressedScreenshot(mockPage, 80, false);

      expect(screenshotMock).toHaveBeenCalledWith(
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

      screenshotMock
        .mockResolvedValueOnce(largeBuffer) // First call with quality 80
        .mockResolvedValueOnce(smallBuffer); // Second call with quality 60

      const screenshot = await captureCompressedScreenshot(mockPage, 80, false, 100);

      expect(screenshotMock).toHaveBeenCalledTimes(2);
      expect(screenshot?.base64).toBe(smallBuffer.toString('base64'));
    });

    it('should return undefined when still too large after quality reduction', async () => {
      const largeBuffer = Buffer.alloc(200);

      screenshotMock.mockResolvedValue(largeBuffer);

      const screenshot = await captureCompressedScreenshot(mockPage, 50, false, 100);

      expect(screenshot).toBeUndefined();
    });
  });
});
