import { createTestInstruction, createMockPage, createTestConfig } from '../../helpers';
import { ScreenshotHandler } from '../../../src/handlers/screenshot';
import type { HandlerContext } from '../../../src/handlers/base';
import { logger, metrics } from '../../../src/utils';

describe('ScreenshotHandler', () => {
  let handler: ScreenshotHandler;
  let mockPage: ReturnType<typeof createMockPage>;
  let context: HandlerContext;

  beforeEach(() => {
    handler = new ScreenshotHandler();
    mockPage = createMockPage();
    context = {
      page: mockPage,
      context: {} as any,
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'test-session',
    };
  });

  it('should capture screenshot', async () => {
    const instruction = createTestInstruction({
      type: 'screenshot',
      params: {},
      node_id: 'node-1',
    });

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.screenshot).toBeDefined();
  });

  it('should capture full page screenshot with quality option', async () => {
    const instruction = createTestInstruction({
      type: 'screenshot',
      params: { quality: 80, fullPage: true },
      node_id: 'node-1',
    });

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.screenshot).toBeDefined();
  });
});
