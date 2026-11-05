import {
  createTypedInstruction,
  createMockPage,
  createMockContext,
  createTestConfig,
} from '../../helpers';
import { ScreenshotHandler } from '../../../src/handlers/screenshot';
import type { HandlerContext } from '../../../src/handlers/base';
import { logger, metrics } from '../../../src/utils';

function png(width = 1280, height = 720): Buffer {
  const buffer = Buffer.alloc(24);
  Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]).copy(buffer);
  buffer.write('IHDR', 12);
  buffer.writeUInt32BE(width, 16);
  buffer.writeUInt32BE(height, 20);
  return buffer;
}

describe('ScreenshotHandler', () => {
  let handler: ScreenshotHandler;
  let mockPage: ReturnType<typeof createMockPage>;
  let context: HandlerContext;

  beforeEach(() => {
    handler = new ScreenshotHandler();
    mockPage = createMockPage();
    mockPage.screenshot = jest.fn().mockResolvedValue(png());
    context = {
      page: mockPage,
      browserContext: createMockContext(),
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'test-session',
    };
  });

  it('should capture screenshot', async () => {
    const instruction = createTypedInstruction('screenshot', {}, { nodeId: 'node-1' });

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.screenshot).toBeDefined();
  });

  it('should capture full page screenshot with quality option', async () => {
    const instruction = createTypedInstruction(
      'screenshot',
      { quality: 80, fullPage: true },
      { nodeId: 'node-1' }
    );

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.screenshot).toBeDefined();
    expect(mockPage.screenshot).toHaveBeenCalledWith(expect.objectContaining({ type: 'jpeg', fullPage: true }));
  });

  it('uses the configured extent consistently when fullPage is omitted', async () => {
    context.config.telemetry.screenshot.fullPage = false;
    const instruction = createTypedInstruction('screenshot', { quality: 80 }, { nodeId: 'node-default' });

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(mockPage.screenshot).toHaveBeenCalledWith(expect.objectContaining({ type: 'jpeg', clip: { x: 0, y: 0, width: 1280, height: 720 } }));
  });

  it('returns an actionable failure for full-page capture during native video', async () => {
    mockPage.video = jest.fn().mockReturnValue({});
    const instruction = createTypedInstruction('screenshot', { fullPage: true }, { nodeId: 'node-video' });

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(false);
    expect(result.error).toMatchObject({
      code: 'FULL_PAGE_SCREENSHOT_UNAVAILABLE_DURING_VIDEO',
      kind: 'orchestration',
      retryable: false,
    });
    expect(result.error?.message).toContain('capture the full page in a separate session');
    expect(mockPage.screenshot).not.toHaveBeenCalled();
  });

  it('returns requested and actual extent details when full-page raster still exceeds the budget', async () => {
    context.config.telemetry.screenshot.maxSizeBytes = 1;
    mockPage.screenshot = jest.fn().mockResolvedValue(png(2850, 12384));
    const instruction = createTypedInstruction('screenshot', { quality: 80, fullPage: true }, { nodeId: 'node-oversize' });

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(false);
    expect(result.error).toMatchObject({ code: 'SCREENSHOT_EXTENT_REJECTED', retryable: false });
    expect(result.error?.message).toContain('requested_extent=full_page');
    expect(result.error?.message).toContain('actual_extent=not_captured');
    expect(result.error?.message).toContain('raster=2850x12384');
  });

  it('should capture the requested element when a selector is provided', async () => {
    mockPage.locator('[data-testid="story-root"]').first().screenshot.mockResolvedValue(png(200, 100));
    const instruction = createTypedInstruction(
      'screenshot',
      { selector: '[data-testid="story-root"]', fullPage: false },
      { nodeId: 'node-selector' }
    );

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(mockPage.locator).toHaveBeenCalledWith('[data-testid="story-root"]');
    expect(mockPage.locator('[data-testid="story-root"]').first().screenshot).toHaveBeenCalledWith({
      type: 'png',
    });
  });
});
