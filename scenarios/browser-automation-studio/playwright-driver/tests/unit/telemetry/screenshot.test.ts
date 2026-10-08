import { captureScreenshot, captureCompressedScreenshot } from '../../../src/telemetry/screenshot';
import { createMockPage, createTestConfig } from '../../helpers';

function png(width: number, height: number): Buffer {
  const buffer = Buffer.alloc(200);
  Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]).copy(buffer);
  buffer.write('IHDR', 12);
  buffer.writeUInt32BE(width, 16);
  buffer.writeUInt32BE(height, 20);
  return buffer;
}

function jpeg(width: number, height: number, size = 200): Buffer {
  const header = Buffer.from([
    0xff, 0xd8, 0xff, 0xc0, 0x00, 0x11, 0x08,
    (height >> 8) & 0xff, height & 0xff,
    (width >> 8) & 0xff, width & 0xff,
    0x03, 0x01, 0x11, 0x00, 0x02, 0x11, 0x00, 0x03, 0x11, 0x00,
    0xff, 0xd9,
  ]);
  const buffer = Buffer.alloc(Math.max(size, header.length));
  header.copy(buffer);
  return buffer;
}

describe('Screenshot', () => {
  let mockPage: ReturnType<typeof createMockPage>;
  let config: ReturnType<typeof createTestConfig>;
  let screenshotMock: jest.MockedFunction<typeof mockPage.screenshot>;

  beforeEach(() => {
    mockPage = createMockPage();
    config = createTestConfig();
    // eslint-disable-next-line @typescript-eslint/unbound-method -- jest mock does not use this
    screenshotMock = mockPage.screenshot as jest.MockedFunction<typeof mockPage.screenshot>;
    screenshotMock.mockResolvedValue(png(1280, 720));
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
      expect(screenshot).toMatchObject({
        media_type: 'image/png',
        requested_extent: 'full_page',
        actual_extent: 'full_page',
        degraded: false,
      });
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
      const mockBuffer = png(320, 240);
      screenshotMock.mockResolvedValue(mockBuffer);

      const screenshot = await captureScreenshot(mockPage, config);

      expect(screenshot?.base64).toBe(mockBuffer.toString('base64'));
    });

    it('reports the captured raster dimensions rather than the CSS viewport', async () => {
      mockPage.viewportSize.mockReturnValue({ width: 1920, height: 1080 });
      screenshotMock.mockResolvedValue(png(3840, 2160));

      const screenshot = await captureScreenshot(mockPage, config);

      expect(screenshot?.width).toBe(3840);
      expect(screenshot?.height).toBe(2160);
    });

    it('preserves full-page extent when a large PNG can be degraded to JPEG', async () => {
      const configSmallMax = createTestConfig({
        telemetry: {
          screenshot: {
            enabled: true,
            fullPage: true,
            quality: 50,
            maxSizeBytes: 100, // Very small limit
          },
          dom: { enabled: true, maxSizeBytes: 1 * 1024 * 1024 },
          console: { enabled: true, maxEntries: 100 },
          network: { enabled: true, maxEvents: 500 },
          har: { enabled: false },
          tracing: { enabled: false },
        },
      });

      const largeBuffer = png(2880, 4224);
      const smallBuffer = jpeg(2880, 4224, 24);

      screenshotMock
        .mockResolvedValueOnce(largeBuffer) // First call (full page) too large
        .mockResolvedValueOnce(smallBuffer); // JPEG retry retains full-page extent

      const screenshot = await captureScreenshot(mockPage, configSmallMax);

      expect(screenshotMock).toHaveBeenCalledTimes(2);
      const firstCall = screenshotMock.mock.calls[0]?.[0];
      const secondCall = screenshotMock.mock.calls[1]?.[0];

      expect(firstCall?.fullPage).toBe(true);
      expect(secondCall).toMatchObject({ type: 'jpeg', fullPage: true });
      expect(screenshot?.base64).toBe(smallBuffer.toString('base64'));
      expect(screenshot).toMatchObject({
        width: 2880,
        height: 4224,
        media_type: 'image/jpeg',
        requested_extent: 'full_page',
        actual_extent: 'full_page',
        degraded: true,
      });
    });

    it('fails an oversized full-page raster instead of returning a misleading viewport crop', async () => {
      const configSmallMax = createTestConfig({
        telemetry: {
          screenshot: {
            enabled: true,
            fullPage: true,
            quality: 50,
            maxSizeBytes: 100,
          },
          dom: { enabled: true, maxSizeBytes: 1 * 1024 * 1024 },
          console: { enabled: true, maxEntries: 100 },
          network: { enabled: true, maxEvents: 500 },
          har: { enabled: false },
          tracing: { enabled: false },
        },
      });
      screenshotMock.mockResolvedValue(png(2850, 12384));

      await expect(captureScreenshot(mockPage, configSmallMax)).rejects.toMatchObject({
        name: 'ScreenshotExtentRejectedError',
        requestedExtent: 'full_page',
        width: 2850,
        height: 12384,
      });
      expect(screenshotMock).toHaveBeenCalledTimes(2);
      expect(screenshotMock.mock.calls[0]?.[0]?.fullPage).toBe(true);
      expect(screenshotMock.mock.calls[1]?.[0]).toMatchObject({ type: 'jpeg', fullPage: true });
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

      const largeBuffer = png(1280, 720);
      const compressedBuffer = jpeg(1280, 720, 24);
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

      expect(screenshot?.width).toBe(1280);
      expect(screenshot?.height).toBe(720);
    });
  });

  describe('captureCompressedScreenshot', () => {
    it('should capture JPEG screenshot', async () => {
      const mockBuffer = jpeg(1280, 720);
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
      const largeBuffer = jpeg(1280, 720, 200);
      const smallBuffer = jpeg(1280, 720, 24);

      screenshotMock
        .mockResolvedValueOnce(largeBuffer) // First call with quality 80
        .mockResolvedValueOnce(smallBuffer); // Second call with quality 60

      const screenshot = await captureCompressedScreenshot(mockPage, 80, false, 100);

      expect(screenshotMock).toHaveBeenCalledTimes(2);
      expect(screenshot?.base64).toBe(smallBuffer.toString('base64'));
    });

    it('rejects an oversized requested extent with actual raster geometry after quality reduction', async () => {
      const largeBuffer = jpeg(1280, 720);

      screenshotMock.mockResolvedValue(largeBuffer);

      await expect(captureCompressedScreenshot(mockPage, 50, true, 100)).rejects.toMatchObject({
        name: 'ScreenshotExtentRejectedError',
        requestedExtent: 'full_page',
        width: 1280,
        height: 720,
        bytes: largeBuffer.length,
        maxBytes: 100,
      });
    });
  });
});
