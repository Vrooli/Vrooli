import { captureScreenshot } from '../../../src/telemetry/screenshot';
import { createMockPage, createTestConfig } from '../../helpers';

describe('Screenshot', () => {
  let mockPage: ReturnType<typeof createMockPage>;
  let config: ReturnType<typeof createTestConfig>;

  beforeEach(() => {
    mockPage = createMockPage();
    config = createTestConfig();
  });

  describe('captureScreenshot', () => {
    it('should capture PNG screenshot by default', async () => {
      const screenshot = await captureScreenshot(mockPage, config);

      expect(mockPage.screenshot).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'png',
          fullPage: false,
        })
      );
      expect(screenshot).toBeDefined();
      expect(screenshot?.format).toBe('png');
    });

    it('should capture JPEG screenshot when configured', async () => {
      const configJPEG = createTestConfig({
        telemetry: {
          screenshot: {
            enabled: true,
            format: 'jpeg',
            quality: 80,
            fullPage: false,
            maxSizeBytes: 5 * 1024 * 1024,
          },
        },
      });

      const screenshot = await captureScreenshot(mockPage, configJPEG);

      expect(mockPage.screenshot).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'jpeg',
          quality: 80,
        })
      );
      expect(screenshot?.format).toBe('jpeg');
    });

    it('should capture full page when configured', async () => {
      const configFullPage = createTestConfig({
        telemetry: {
          screenshot: {
            enabled: true,
            format: 'png',
            quality: 80,
            fullPage: true,
            maxSizeBytes: 5 * 1024 * 1024,
          },
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

      expect(screenshot?.data).toBe(mockBuffer.toString('base64'));
    });

    it('should include viewport dimensions', async () => {
      mockPage.viewport.mockReturnValue({ width: 1920, height: 1080 });

      const screenshot = await captureScreenshot(mockPage, config);

      expect(screenshot?.viewport_width).toBe(1920);
      expect(screenshot?.viewport_height).toBe(1080);
    });

    it('should handle viewport-only screenshot when full page too large', async () => {
      const configSmallMax = createTestConfig({
        telemetry: {
          screenshot: {
            enabled: true,
            format: 'png',
            quality: 80,
            fullPage: true,
            maxSizeBytes: 100, // Very small limit
          },
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
      expect(screenshot?.data).toBe(smallBuffer.toString('base64'));
    });

    it('should return undefined when screenshot exceeds size limit', async () => {
      const configSmallMax = createTestConfig({
        telemetry: {
          screenshot: {
            enabled: true,
            format: 'png',
            quality: 80,
            fullPage: false,
            maxSizeBytes: 100,
          },
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
            format: 'png',
            quality: 80,
            fullPage: false,
            maxSizeBytes: 5 * 1024 * 1024,
          },
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
      mockPage.viewport.mockReturnValue(null);

      const screenshot = await captureScreenshot(mockPage, config);

      expect(screenshot?.viewport_width).toBe(0);
      expect(screenshot?.viewport_height).toBe(0);
    });

    it('should apply quality setting for JPEG', async () => {
      const configQuality = createTestConfig({
        telemetry: {
          screenshot: {
            enabled: true,
            format: 'jpeg',
            quality: 90,
            fullPage: false,
            maxSizeBytes: 5 * 1024 * 1024,
          },
        },
      });

      await captureScreenshot(mockPage, configQuality);

      expect(mockPage.screenshot).toHaveBeenCalledWith(
        expect.objectContaining({
          quality: 90,
        })
      );
    });

    it('should not include quality for PNG', async () => {
      const configPNG = createTestConfig({
        telemetry: {
          screenshot: {
            enabled: true,
            format: 'png',
            quality: 80,
            fullPage: false,
            maxSizeBytes: 5 * 1024 * 1024,
          },
        },
      });

      await captureScreenshot(mockPage, configPNG);

      const call = (mockPage.screenshot as jest.Mock).mock.calls[0][0];
      expect(call.quality).toBeUndefined();
    });
  });
});
