import { ScreenshotHandler } from '../../../src/handlers/screenshot';
import type { CompiledInstruction, HandlerContext } from '../../../src/types';
import { createMockPage, createTestConfig } from '../../helpers';
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
    const instruction: CompiledInstruction = {
      type: 'screenshot',
      params: {},
      node_id: 'node-1',
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.screenshot).toBeDefined();
  });

  it('should capture element screenshot when selector provided', async () => {
    const instruction: CompiledInstruction = {
      type: 'screenshot',
      params: { selector: '#element' },
      node_id: 'node-1',
    };

    const mockLocator = {
      screenshot: jest.fn().mockResolvedValue(Buffer.from('element-screenshot')),
    };
    mockPage.locator.mockReturnValue(mockLocator as any);

    const result = await handler.execute(instruction, context);

    expect(mockLocator.screenshot).toHaveBeenCalled();
    expect(result.success).toBe(true);
  });
});
